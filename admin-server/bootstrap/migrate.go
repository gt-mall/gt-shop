package bootstrap

import (
	"context"
	"fmt"
	"io/fs"
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
	}

	return nil
}
