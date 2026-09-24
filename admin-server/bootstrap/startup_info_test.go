package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"mall/admin_core/pkg/cache"
	"mall/admin_core/pkg/logger"
)

type startupCache struct {
	cache.Cache
	pingErr error
}

func (c startupCache) IsDisabled() bool           { return false }
func (c startupCache) Ping(context.Context) error { return c.pingErr }

func TestCollectStartupInfoNotConfigured(t *testing.T) {
	info := CollectStartupInfo(context.Background(), Config{Port: "8100"}, nil, cache.NewNoop(), logger.DefaultConfig(), nil)
	if info.DatabaseStatus != "not configured" || info.RedisStatus != "disabled" || info.SchemaStatus != "unavailable" {
		t.Fatalf("unexpected connection states: %+v", info)
	}
	if info.LogsStatus != "ok" || info.LogsValue != "info · ./logs" {
		t.Fatalf("unexpected log state: %+v", info)
	}
}

func TestStartupDisplayUsesVerifiedStates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE schema_migrations (version VARCHAR(255) PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO schema_migrations(version) VALUES ('008_settings.sql')").Error; err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		AppEnv:       "production",
		Port:         "8100",
		DatabaseSet:  true,
		DatabaseDSN:  "user:secret@tcp(127.0.0.1:3306)/shop?parseTime=true",
		RedisEnabled: true,
		RedisAddr:    "127.0.0.1:6379",
	}
	info := CollectStartupInfo(context.Background(), cfg, db, startupCache{}, logger.DefaultConfig(), nil)
	if info.DatabaseStatus != "connected" || info.RedisStatus != "connected" || info.SchemaStatus != "ok" || info.SchemaVersion != "v8" {
		t.Fatalf("expected connected services and schema v8: %+v", info)
	}
	plain := FormatStartupInfo(info)
	if strings.Contains(plain, "secret") || strings.Contains(plain, "\x1b[") {
		t.Fatal("plain startup output must not contain credentials or ANSI escapes")
	}
	if strings.Count(plain, "╭") != 1 || strings.ContainsAny(plain, "┌█") || strings.Count(plain, "✔ Ready") != 1 {
		t.Fatalf("expected one compact header and ready footer: %q", plain)
	}
	if !strings.Contains(plain, "GT-Shop") || !strings.Contains(plain, "PRODUCTION") || !strings.Contains(plain, "HTTP      ● Running") || !strings.Contains(plain, "MySQL     ● Connected") || !strings.Contains(plain, "Redis     ● Connected") {
		t.Fatalf("compact header or connection rows are missing: %q", plain)
	}
	positions := []int{
		strings.Index(plain, "  HTTP"),
		strings.Index(plain, "  MySQL"),
		strings.Index(plain, "  Redis"),
		strings.Index(plain, "  Schema"),
		strings.Index(plain, "  Logs"),
		strings.Index(plain, "  ✔ Ready"),
	}
	for i, pos := range positions {
		if pos < 0 || i > 0 && pos <= positions[i-1] {
			t.Fatalf("rows must follow the reference order: %q", plain)
		}
	}
	if !strings.Contains(plain, "127.0.0.1:3306 / shop") || !strings.Contains(plain, "Schema    v8") || !strings.Contains(plain, "Logs      info · ./logs") {
		t.Fatalf("connection, schema or log values are missing: %q", plain)
	}
	colored := renderStartupInfo(info, true)
	if !strings.Contains(colored, colorPurple) || !strings.Contains(colored, colorGreen) {
		t.Fatal("terminal output should color the header and healthy states")
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	info = CollectStartupInfo(context.Background(), cfg, db, startupCache{pingErr: errors.New("connection refused")}, logger.DefaultConfig(), errors.New("log path invalid"))
	if info.DatabaseStatus != "failed" || info.RedisStatus != "failed" || info.SchemaStatus != "unavailable" || info.LogsStatus != "failed" {
		t.Fatalf("failed services must not appear healthy: %+v", info)
	}
	failedOutput := FormatStartupInfo(info)
	if !strings.Contains(failedOutput, "● Failed") || !strings.Contains(failedOutput, "✘ Startup incomplete") || strings.Contains(failedOutput, "✔ Ready") {
		t.Fatalf("failed services must be clear in the compact output: %q", failedOutput)
	}
}

func TestStartupDisplayLongMetadata(t *testing.T) {
	info := StartupInfo{
		Version:     strings.Repeat("long-version-", 8),
		Environment: strings.Repeat("production-", 8),
		Listen:      "[::]:8100",
	}
	output := FormatStartupInfo(info)
	var border, header, bottom string
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "╭"):
			border = line
		case strings.HasPrefix(line, "│"):
			header = line
		case strings.HasPrefix(line, "╰"):
			bottom = line
		}
	}
	if border == "" || header == "" || bottom == "" || len([]rune(border)) != len([]rune(header)) || len([]rune(border)) != len([]rune(bottom)) {
		t.Fatalf("header border should fit long metadata: %q", output)
	}
}
