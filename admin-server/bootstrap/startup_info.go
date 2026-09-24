package bootstrap

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"mall/admin_core/pkg/cache"
	"mall/admin_core/pkg/logger"
	pkgversion "mall/admin_core/pkg/version"
	"mall/admin_core/static"
)

// StartupInfo contains verified connection states and the values shown at startup.
type StartupInfo struct {
	Version        string
	BuildTime      string
	GitCommit      string
	ExecutablePath string
	ExecutableSize int64
	StaticSize     int64
	NetworkURL     string
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
	version := pkgversion.Get()
	info := StartupInfo{
		Version:        version.Version,
		BuildTime:      version.BuildTime,
		GitCommit:      version.GitCommit,
		StaticSize:     static.AssetsSize(),
		NetworkURL:     "http://127.0.0.1:" + config.Port + "/",
		Environment:    config.AppEnv,
		Listen:         "0.0.0.0:" + config.Port,
		DatabaseStatus: "not configured",
		RedisStatus:    "disabled",
		SchemaStatus:   "unavailable",
		SchemaVersion:  "-",
		LogsStatus:     "ok",
	}
	if executable, err := os.Executable(); err == nil {
		info.ExecutablePath = executable
		if stat, err := os.Stat(executable); err == nil {
			info.ExecutableSize = stat.Size()
		}
	}
	if info.Environment == "" {
		info.Environment = "production"
	}
	if config.DatabaseSet {
		info.DatabaseStatus = "failed"
		if dsn, err := mysqlDriver.ParseDSN(config.DatabaseDSN); err == nil {
			info.DatabaseTarget = dsn.Addr
			if dsn.DBName != "" {
				info.DatabaseTarget += "/" + dsn.DBName
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

// NetworkURL returns a local network address for the bound HTTP listener.
func NetworkURL(listener net.Addr, publicURL string) string {
	if publicURL != "" {
		parsed, err := url.Parse(strings.TrimSpace(publicURL))
		if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" {
			return parsed.String()
		}
	}
	host, port, err := net.SplitHostPort(listener.String())
	if err != nil {
		return "-"
	}
	if host == "" || host == "::" || host == "0.0.0.0" {
		host = "127.0.0.1"
		if addrs, err := net.InterfaceAddrs(); err == nil {
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if ok && ipNet.IP.To4() != nil && ipNet.IP.IsPrivate() && !ipNet.IP.IsLoopback() {
					host = ipNet.IP.String()
					break
				}
			}
		}
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}
