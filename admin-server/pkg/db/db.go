package db

import (
	"context"

	"gorm.io/gorm"
)

// Ping is shared by future database-backed modules that need a small readiness check.
func Ping(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
