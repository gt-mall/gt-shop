package bootstrap

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"mall/admin_core/pkg/cache"
	"mall/admin_core/pkg/logger"
	pkgversion "mall/admin_core/pkg/version"
)

// StartupInfo contains verified connection states and the values shown at startup.
type StartupInfo struct {
	Version        string
	Environment    string
	Listen         string
	DatabaseStatus string
	DatabaseTarget string
	RedisStatus    string
	RedisTarget    string
	SchemaStatus   string
	SchemaVersion  string
	LogsStatus     string
	LogsValue      string
}

// CollectStartupInfo checks connections after startup setup has completed.
func CollectStartupInfo(ctx context.Context, config Config, db *gorm.DB, c cache.Cache, logCfg logger.Config, logErr error) StartupInfo {
	info := StartupInfo{
		Version:        pkgversion.Get().Version,
		Environment:    config.AppEnv,
		Listen:         "0.0.0.0:" + config.Port,
		DatabaseStatus: "not configured",
		RedisStatus:    "disabled",
		SchemaStatus:   "unavailable",
		SchemaVersion:  "-",
		LogsStatus:     "ok",
	}
	if info.Environment == "" {
		info.Environment = "production"
	}
	if config.DatabaseSet {
		info.DatabaseStatus = "failed"
		if dsn, err := mysqlDriver.ParseDSN(config.DatabaseDSN); err == nil {
			info.DatabaseTarget = dsn.Addr
			if dsn.DBName != "" {
				info.DatabaseTarget += " / " + dsn.DBName
			}
		}
		if db != nil {
			if sqlDB, err := db.DB(); err == nil {
				pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				if sqlDB.PingContext(pingCtx) == nil {
					info.DatabaseStatus = "connected"
					info.SchemaStatus, info.SchemaVersion = schemaInfo(pingCtx, db)
				}
				cancel()
			}
		}
	}
	if config.RedisEnabled {
		info.RedisStatus = "failed"
		info.RedisTarget = config.RedisAddr
		if c != nil && !c.IsDisabled() {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			if c.Ping(pingCtx) == nil {
				info.RedisStatus = "connected"
			}
			cancel()
		}
	}
	level := strings.TrimSpace(logCfg.Level)
	if level == "" {
		level = "info"
	}
	switch {
	case logErr != nil:
		info.LogsStatus = "failed"
	case !logCfg.Console && !logCfg.File:
		info.LogsStatus = "disabled"
	}
	target := "disabled"
	if logCfg.File {
		target = strings.TrimSpace(logCfg.Path)
		if target == "" {
			target = "./logs"
		}
	} else if logCfg.Console {
		target = "console"
	}
	info.LogsValue = level + " · " + target
	return info
}

func schemaInfo(ctx context.Context, db *gorm.DB) (status, version string) {
	if !db.WithContext(ctx).Migrator().HasTable("schema_migrations") {
		return "missing", "-"
	}
	var latest string
	if err := db.WithContext(ctx).Raw("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&latest).Error; err != nil {
		return "failed", "-"
	}
	if latest == "" {
		return "empty", "-"
	}
	prefix, _, _ := strings.Cut(latest, "_")
	if number, err := strconv.Atoi(prefix); err == nil {
		return "ok", fmt.Sprintf("v%d", number)
	}
	return "ok", latest
}
