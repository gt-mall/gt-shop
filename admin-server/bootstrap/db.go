package bootstrap

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config contains only process-level settings needed during startup.
type Config struct {
	Port          string
	DatabaseURL   string
	DatabaseSet   bool
	AutoMigrate   bool
	MigrationsDir string
}

// LoadConfig reads configuration from environment variables. No credentials
// are hard-coded; an empty DATABASE_URL keeps the demo runnable without a DB.
func LoadConfig() Config {
	port := strings.TrimSpace(os.Getenv("APP_PORT"))
	if port == "" {
		port = "8080"
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
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
		DatabaseURL:   databaseURL,
		DatabaseSet:   databaseURL != "",
		AutoMigrate:   autoMigrate,
		MigrationsDir: migrationsDir,
	}
}

// NewPool creates and verifies a PostgreSQL pool when DATABASE_URL is set.
// The nil result is intentional for the no-database demo mode.
func NewPool(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	if !config.DatabaseSet {
		return nil, nil
	}

	pool, err := pgxpool.New(ctx, config.DatabaseURL)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
