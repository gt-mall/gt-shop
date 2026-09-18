package bootstrap

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	mysqlDriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"mall/admin_core/internal/analytics"
)

// Config contains only process-level settings needed during startup.
type Config struct {
	Port          string
	DatabaseDSN   string
	DatabaseSet   bool
	AutoMigrate   bool
	AppTimezone   string
	JWTSecret     string
	JWTExpireHour int
	BizNoWorkerID uint8
	RedisEnabled  bool
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// LoadConfig reads configuration from environment variables. No credentials
// are hard-coded; an empty DATABASE_DSN keeps the demo runnable without a DB.
func LoadConfig() Config {
	port := strings.TrimSpace(os.Getenv("APP_PORT"))
	if port == "" {
		port = "8100"
	}
	databaseDSN := strings.TrimSpace(os.Getenv("DATABASE_DSN"))

	appTimezone := strings.TrimSpace(os.Getenv("APP_TIMEZONE"))
	if appTimezone == "" {
		appTimezone = analytics.DefaultTimezone
	}
	loc, err := time.LoadLocation(appTimezone)
	if err != nil {
		log.Printf("failed to load APP_TIMEZONE %q, falling back to %s: %v", appTimezone, analytics.DefaultTimezone, err)
		loc, err = time.LoadLocation(analytics.DefaultTimezone)
		if err != nil {
			loc = time.FixedZone("CST", 8*3600)
		}
	}
	time.Local = loc
	analytics.SetBusinessLocation(loc)

	autoMigrate := true
	if raw := strings.TrimSpace(os.Getenv("AUTO_MIGRATE")); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			autoMigrate = parsed
		}
	}
	jwtExpireHour := 12
	if raw := strings.TrimSpace(os.Getenv("AUTH_JWT_EXPIRE_HOURS")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			jwtExpireHour = parsed
		}
	}

	var bizNoWorkerID uint8 = 0
	if raw := strings.TrimSpace(os.Getenv("BIZ_NO_WORKER_ID")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 8)
		if err != nil || parsed > 15 {
			log.Fatalf("invalid BIZ_NO_WORKER_ID %q: must be an integer between 0 and 15", raw)
		}
		bizNoWorkerID = uint8(parsed)
	}

	redisEnabled := false
	if raw := strings.TrimSpace(os.Getenv("REDIS_ENABLED")); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			redisEnabled = parsed
		}
	}
	redisAddr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	redisPassword := strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	redisDB := 0
	if raw := strings.TrimSpace(os.Getenv("REDIS_DB")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			redisDB = parsed
		}
	}

	return Config{
		Port:          port,
		DatabaseDSN:   databaseDSN,
		DatabaseSet:   databaseDSN != "",
		AutoMigrate:   autoMigrate,
		AppTimezone:   appTimezone,
		JWTSecret:     strings.TrimSpace(os.Getenv("AUTH_JWT_SECRET")),
		JWTExpireHour: jwtExpireHour,
		BizNoWorkerID: bizNoWorkerID,
		RedisEnabled:  redisEnabled,
		RedisAddr:     redisAddr,
		RedisPassword: redisPassword,
		RedisDB:       redisDB,
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
	dsnConfig.ParseTime = true
	if dsnConfig.Loc == nil || dsnConfig.Loc.String() == "" || dsnConfig.Loc == time.UTC {
		dsnConfig.Loc = analytics.BusinessLocation()
	}

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

var (
	// ErrTestDSNEmpty indicates that TEST_DATABASE_DSN is not configured.
	ErrTestDSNEmpty = errors.New("TEST_DATABASE_DSN is not configured")
	// ErrTestDSNEqualsDev indicates that TEST_DATABASE_DSN points to the same database as DATABASE_DSN.
	ErrTestDSNEqualsDev = errors.New("TEST_DATABASE_DSN must not equal DATABASE_DSN")
)

// ValidateTestDSN verifies that testDSN is configured and does not point to the normal development database.
// Integration tests must only connect to TEST_DATABASE_DSN and must never use or match DATABASE_DSN.
func ValidateTestDSN(testDSN, devDSN string) (string, error) {
	testDSN = strings.TrimSpace(testDSN)
	if testDSN == "" {
		return "", ErrTestDSNEmpty
	}

	devDSN = strings.TrimSpace(devDSN)
	if devDSN != "" {
		if testDSN == devDSN {
			return "", ErrTestDSNEqualsDev
		}

		cfgTest, errTest := mysqlDriver.ParseDSN(testDSN)
		cfgDev, errDev := mysqlDriver.ParseDSN(devDSN)
		if errTest == nil && errDev == nil {
			if cfgTest.Addr == cfgDev.Addr && cfgTest.DBName == cfgDev.DBName {
				return "", ErrTestDSNEqualsDev
			}
		}
	}

	return testDSN, nil
}

// TestDBName parses and returns the database name from the given DSN for logging confirmation.
func TestDBName(dsn string) string {
	if cfg, err := mysqlDriver.ParseDSN(strings.TrimSpace(dsn)); err == nil {
		return cfg.DBName
	}
	return ""
}
