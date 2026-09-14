package bootstrap

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"sort"

	"gorm.io/gorm"

	"mall/admin_core/data"
)

// RunMigrations applies each not-yet-applied SQL file in lexical order from embedded data.FS.
func RunMigrations(ctx context.Context, db *gorm.DB) error {
	return RunMigrationsFS(ctx, db, data.FS)
}

// RunMigrationsFS applies each not-yet-applied SQL file in lexical order from the provided fs.FS.
func RunMigrationsFS(ctx context.Context, db *gorm.DB, fsys fs.FS) error {
	if db == nil {
		return nil
	}

	db = db.WithContext(ctx)
	migrationsTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	if err := db.Exec(migrationsTableSQL).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(fsys, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, fileName := range files {
		var applied int64
		if err := db.Raw(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, fileName).Scan(&applied).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", fileName, err)
		}
		if applied > 0 {
			continue
		}

		log.Printf("applying migration: %s", fileName)
		sql, err := fs.ReadFile(fsys, path.Join("migrations", fileName))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", fileName, err)
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", fileName, err)
			}
			if err := tx.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, fileName).Error; err != nil {
				return fmt.Errorf("record migration %s: %w", fileName, err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("run migration %s: %w", fileName, err)
		}
		log.Printf("applied migration: %s", fileName)
	}

	return nil
}

// RunInit applies all migrations and seeds in order.
// Typically used during clean installation or dev database initialization.
func RunInit(ctx context.Context, db *gorm.DB) error {
	return RunInitFS(ctx, db, data.FS)
}

// RunInitFS applies migrations and seeds from the provided fs.FS.
func RunInitFS(ctx context.Context, db *gorm.DB, fsys fs.FS) error {
	if err := RunMigrationsFS(ctx, db, fsys); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	if err := RunSeedsFS(ctx, db, fsys); err != nil {
		return fmt.Errorf("run seeds: %w", err)
	}
	return nil
}

// AlignBaseline marks the baseline migration as applied in schema_migrations
// without re-executing CREATE TABLE statements, then applies seeds.
// This is used for existing development databases that already have all tables created.
func AlignBaseline(ctx context.Context, db *gorm.DB) error {
	return AlignBaselineFS(ctx, db, data.FS)
}

// AlignBaselineFS marks baseline migrations as applied using the provided fs.FS, then applies seeds.
func AlignBaselineFS(ctx context.Context, db *gorm.DB, fsys fs.FS) error {
	if db == nil {
		return nil
	}

	db = db.WithContext(ctx)
	migrationsTableSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
	if err := db.Exec(migrationsTableSQL).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	if err := db.Exec(`DELETE FROM schema_migrations`).Error; err != nil {
		return fmt.Errorf("reset schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(fsys, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			name := entry.Name()
			if err := db.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, name).Error; err != nil {
				return fmt.Errorf("record baseline %s: %w", name, err)
			}
			log.Printf("marked migration as applied: %s", name)
		}
	}

	if err := RunSeedsFS(ctx, db, fsys); err != nil {
		return fmt.Errorf("run seeds: %w", err)
	}
	return nil
}

// RunSeeds applies each SQL file in lexical order from embedded data.FS under seeds/.
// Seed execution is intended for initial setup or explicit seeding, not automated on every server start.
func RunSeeds(ctx context.Context, db *gorm.DB) error {
	return RunSeedsFS(ctx, db, data.FS)
}

// RunSeedsFS applies each SQL file in lexical order from the provided fs.FS under seeds/.
func RunSeedsFS(ctx context.Context, db *gorm.DB, fsys fs.FS) error {
	if db == nil {
		return nil
	}

	db = db.WithContext(ctx)
	entries, err := fs.ReadDir(fsys, "seeds")
	if err != nil {
		return fmt.Errorf("read seeds directory: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)

	for _, fileName := range files {
		log.Printf("applying seed: %s", fileName)
		sql, err := fs.ReadFile(fsys, path.Join("seeds", fileName))
		if err != nil {
			return fmt.Errorf("read seed %s: %w", fileName, err)
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("apply seed %s: %w", fileName, err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("run seed %s: %w", fileName, err)
		}
		log.Printf("applied seed: %s", fileName)
	}

	return nil
}
