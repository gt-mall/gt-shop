package bootstrap

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Config contains only process-level settings needed during startup.
type Config struct {
	Port          string
	DatabaseDSN   string
	DatabaseSet   bool
	AutoMigrate   bool
	MigrationsDir string
}

// LoadConfig reads configuration from environment variables. No credentials
// are hard-coded; an empty DATABASE_DSN keeps the demo runnable without a DB.
func LoadConfig() Config {
	port := strings.TrimSpace(os.Getenv("APP_PORT"))
	if port == "" {
		port = "8080"
	}
	databaseDSN := strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	migrationsDir := strings.TrimSpace(os.Getenv("MIGRATIONS_DIR"))
	if migrationsDir == "" {
		migrationsDir = "./migrations"
	}

	autoMigrate := true
	if raw := strings.TrimSpace(os.Getenv("AUTO_MIGRATE")); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			autoMigrate = parsed
		}
	}

	return Config{
		Port:          port,
		DatabaseDSN:   databaseDSN,
		DatabaseSet:   databaseDSN != "",
		AutoMigrate:   autoMigrate,
		MigrationsDir: migrationsDir,
	}
}

// NewDB creates and verifies a MySQL connection when DATABASE_DSN is set.
// The nil result is intentional for the no-database demo mode.
func NewDB(ctx context.Context, config Config) (*gorm.DB, error) {
	if !config.DatabaseSet {
		return nil, nil
	}

	dsnConfig, err := mysqlDriver.ParseDSN(config.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	dsnConfig.MultiStatements = true

	db, err := gorm.Open(gormmysql.Open(dsnConfig.FormatDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}
