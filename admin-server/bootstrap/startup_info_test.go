package bootstrap

import (
	"context"
	"errors"
	"net"
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
	if info.ExecutablePath == "" || info.ExecutableSize <= 0 || info.StaticSize <= 0 {
		t.Fatalf("runtime file metadata should be available: %+v", info)
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
	if strings.Count(plain, "✔ GT-Shop started successfully") != 1 || strings.Count(plain, strings.Repeat("─", 56)) != 3 {
		t.Fatalf("expected three separators and one success footer: %q", plain)
	}
	if !strings.Contains(plain, "GT-Shop") || !strings.Contains(plain, "production") || !strings.Contains(plain, "HTTP      ● Running") || !strings.Contains(plain, "MySQL     ● Connected") || !strings.Contains(plain, "Redis     ● Connected") {
		t.Fatalf("compact header or connection rows are missing: %q", plain)
	}
	positions := []int{
		strings.Index(plain, "HTTP"),
		strings.Index(plain, "MySQL"),
		strings.Index(plain, "Redis"),
		strings.Index(plain, "➜ Version"),
		strings.Index(plain, "➜ Size"),
		strings.Index(plain, "➜ Running Path"),
	}
	for i, pos := range positions {
		if pos < 0 || i > 0 && pos <= positions[i-1] {
			t.Fatalf("rows must follow the reference order: %q", plain)
		}
	}
	if !strings.Contains(plain, "127.0.0.1:3306/shop") || !strings.Contains(plain, "➜ Network") || !strings.Contains(plain, "➜ Running Path") {
		t.Fatalf("connection or runtime values are missing: %q", plain)
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
	if !strings.Contains(failedOutput, "● Failed") || !strings.Contains(failedOutput, "✘ GT-Shop started with unavailable dependencies") || strings.Contains(failedOutput, "✔ GT-Shop started successfully") {
		t.Fatalf("failed services must be clear in the compact output: %q", failedOutput)
	}
}

func TestStartupDisplayLongMetadata(t *testing.T) {
	info := StartupInfo{
		Version:        strings.Repeat("long-version-", 8),
		BuildTime:      "20260923.1721",
		GitCommit:      "a8f31c2",
		Environment:    strings.Repeat("production-", 8),
		Listen:         "[::]:8100",
		ExecutablePath: "/tmp/mall_server",
		ExecutableSize: 30 * 1024 * 1024,
		StaticSize:     12 * 1024 * 1024,
	}
	output := FormatStartupInfo(info)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 12 || displayWidth(lines[0]) != displayWidth(lines[1]) || lines[1] != lines[5] || lines[5] != lines[10] {
		t.Fatalf("separators should fit long metadata: %q", output)
	}
	if !strings.Contains(output, "mall_server  30.0 MB   Static  12.0 MB") || !strings.Contains(output, "(20260923.1721-a8f31c2)") {
		t.Fatalf("expected build metadata and file sizes: %q", output)
	}
}

func TestNetworkURLUsesBoundAddress(t *testing.T) {
	addr := &net.TCPAddr{IP: net.ParseIP("192.168.1.60"), Port: 8100}
	if got := NetworkURL(addr, ""); got != "http://192.168.1.60:8100/" {
		t.Fatalf("unexpected network URL: %q", got)
	}
	if got := NetworkURL(addr, "http://192.168.1.60:6100/"); got != "http://192.168.1.60:6100/" {
		t.Fatalf("public URL should take precedence: %q", got)
	}
	if got := NetworkURL(addr, "http://user:secret@192.168.1.60:6100/"); got != "http://192.168.1.60:8100/" {
		t.Fatalf("URLs with credentials must not be printed: %q", got)
	}
}
