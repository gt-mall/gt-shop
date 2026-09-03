package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gorm.io/gorm"
)

// RunMigrations applies each not-yet-applied SQL file in lexical order.
// Keeping this small makes the first version easy to inspect and replace later.
func RunMigrations(ctx context.Context, db *gorm.DB, directory string) error {
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

	entries, err := os.ReadDir(directory)
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

		sql, err := os.ReadFile(filepath.Join(directory, fileName))
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
