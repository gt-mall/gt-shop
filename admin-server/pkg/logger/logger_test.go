package logger

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestInitAndDefaultConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gtshop_log_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subLogDir := filepath.Join(tmpDir, "nested", "logs")
	cfg := DefaultConfig()
	cfg.Path = subLogDir
	cfg.Console = false
	cfg.File = true

	// 1. 测试自动创建深层嵌套日志目录
	if err := Init(cfg); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer Close()

	if _, err := os.Stat(subLogDir); os.IsNotExist(err) {
		t.Fatalf("expected log dir %s to be created automatically", subLogDir)
	}

	// 2. 验证写入 Info 日志
	Info("server started", String("env", "test"), Int64("worker_id", 1))
	_ = Sync()

	appLogPath := filepath.Join(subLogDir, "app.log")
	errLogPath := filepath.Join(subLogDir, "error.log")

	appBytes, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("failed to read app.log: %v", err)
	}
	appContent := string(appBytes)
	if !strings.Contains(appContent, "server started") {
		t.Errorf("expected app.log to contain 'server started', got: %s", appContent)
	}

	// 解析 JSON 验证字段规范
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(appContent), "\n")
	if len(lines) == 0 {
		t.Fatalf("expected at least 1 log line")
	}
	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("app.log line is not valid JSON: %v, raw: %s", err, lines[0])
	}
	if logEntry["level"] != "info" {
		t.Errorf("expected level 'info', got: %v", logEntry["level"])
	}
	if logEntry["message"] != "server started" {
		t.Errorf("expected message 'server started', got: %v", logEntry["message"])
	}
	if logEntry["env"] != "test" {
		t.Errorf("expected env 'test', got: %v", logEntry["env"])
	}
	if _, ok := logEntry["time"]; !ok {
		t.Errorf("expected 'time' field in json log")
	}

	// 此时 error.log 应该存在（由于创建了 writer）且为空
	errBytes, err := os.ReadFile(errLogPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("unexpected error reading error.log: %v", err)
	}
	if len(errBytes) > 0 {
		t.Errorf("expected error.log to be empty for Info log, got: %s", string(errBytes))
	}
}

func TestErrorLogDualWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gtshop_log_err_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Level:      "info",
		Path:       tmpDir,
		Console:    false,
		File:       true,
		MaxSize:    10,
		MaxAge:     7,
		MaxBackups: 3,
		Compress:   false,
	}

	if err := Init(cfg); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer Close()

	testErr := errors.New("database connection refused")
	Error("database failed", Int64("order_id", 10001), Err(testErr))
	_ = Sync()

	appLogPath := filepath.Join(tmpDir, "app.log")
	errLogPath := filepath.Join(tmpDir, "error.log")

	// 1. 验证 app.log 包含该错误
	appBytes, err := os.ReadFile(appLogPath)
	if err != nil {
		t.Fatalf("failed to read app.log: %v", err)
	}
	appContent := string(appBytes)
	if !strings.Contains(appContent, "database failed") || !strings.Contains(appContent, "database connection refused") {
		t.Errorf("expected app.log to contain error details, got: %s", appContent)
	}

	// 2. 验证 error.log 同时包含该错误
	errBytes, err := os.ReadFile(errLogPath)
	if err != nil {
		t.Fatalf("failed to read error.log: %v", err)
	}
	errContent := string(errBytes)
	if !strings.Contains(errContent, "database failed") || !strings.Contains(errContent, "database connection refused") {
		t.Errorf("expected error.log to contain error details, got: %s", errContent)
	}

	// 验证 JSON 结构
	var errEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(errContent)), &errEntry); err != nil {
		t.Fatalf("error.log is not valid JSON: %v", err)
	}
	if errEntry["level"] != "error" {
		t.Errorf("expected level 'error', got: %v", errEntry["level"])
	}
	if errEntry["message"] != "database failed" {
		t.Errorf("expected message 'database failed', got: %v", errEntry["message"])
	}
	if val, ok := errEntry["order_id"].(float64); !ok || int64(val) != 10001 {
		t.Errorf("expected order_id 10001, got: %v", errEntry["order_id"])
	}
}

func TestLevelFiltering(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gtshop_log_lvl_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 当级别为 info 时，debug 不应该写入文件
	cfg := Config{
		Level:   "info",
		Path:    tmpDir,
		Console: false,
		File:    true,
	}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	Debug("debug detail should be ignored", String("foo", "bar"))
	_ = Sync()

	appBytes, _ := os.ReadFile(filepath.Join(tmpDir, "app.log"))
	if strings.Contains(string(appBytes), "debug detail should be ignored") {
		t.Errorf("debug log should NOT be written when level is info")
	}

	// 切换为 debug 级别
	cfg.Level = "debug"
	if err := Init(cfg); err != nil {
		t.Fatalf("Init() with debug error: %v", err)
	}
	defer Close()

	Debug("debug detail should appear", String("step", "2"))
	_ = Sync()

	appBytes2, _ := os.ReadFile(filepath.Join(tmpDir, "app.log"))
	if !strings.Contains(string(appBytes2), "debug detail should appear") {
		t.Errorf("debug log SHOULD be written when level is debug")
	}
}

func TestFieldsSupport(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gtshop_log_fields_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Level:   "debug",
		Path:    tmpDir,
		Console: false,
		File:    true,
	}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer Close()

	Warn(
		"field test",
		String("str_key", "hello"),
		Int("int_key", 123),
		Int64("int64_key", 9876543210),
		Bool("bool_key", true),
		Any("map_key", map[string]string{"k": "v"}),
		Float64("float_key", 3.14),
		Uint64("uint_key", 999),
		Strings("slice_key", []string{"a", "b"}),
		Duration("dur_key", 5*time.Second),
	)
	_ = Sync()

	appBytes, err := os.ReadFile(filepath.Join(tmpDir, "app.log"))
	if err != nil {
		t.Fatalf("read app.log error: %v", err)
	}

	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(appBytes))), &entry); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if entry["str_key"] != "hello" {
		t.Errorf("str_key mismatch")
	}
	if int(entry["int_key"].(float64)) != 123 {
		t.Errorf("int_key mismatch")
	}
	if int64(entry["int64_key"].(float64)) != 9876543210 {
		t.Errorf("int64_key mismatch")
	}
	if entry["bool_key"] != true {
		t.Errorf("bool_key mismatch")
	}
}

func TestFormatLogging(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gtshop_log_fmt_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Level:   "debug",
		Path:    tmpDir,
		Console: false,
		File:    true,
	}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer Close()

	Infof("user %s created order %d", "alice", 12345)
	Warnf("sku %d low inventory %d", 55, 3)
	Errorf("failed with code %d", 500)
	_ = Sync()

	appBytes, _ := os.ReadFile(filepath.Join(tmpDir, "app.log"))
	appContent := string(appBytes)
	if !strings.Contains(appContent, "user alice created order 12345") {
		t.Errorf("expected app.log to contain formatted string")
	}
	if !strings.Contains(appContent, "sku 55 low inventory 3") {
		t.Errorf("expected app.log to contain formatted string")
	}

	errBytes, _ := os.ReadFile(filepath.Join(tmpDir, "error.log"))
	if !strings.Contains(string(errBytes), "failed with code 500") {
		t.Errorf("expected error.log to contain formatted string")
	}
}

func TestConcurrentLoggingSafety(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gtshop_log_race_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := Config{
		Level:   "info",
		Path:    tmpDir,
		Console: false,
		File:    true,
	}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	defer Close()

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	for i := 0; i < workers; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if j%5 == 0 {
					Error("concurrent error", Int("worker", workerID), Int("seq", j), Err(errors.New("err")))
				} else if j%2 == 0 {
					Warn("concurrent warn", Int("worker", workerID), Int("seq", j))
				} else {
					Info("concurrent info", Int("worker", workerID), Int("seq", j))
				}
			}
		}()
	}

	wg.Wait()
	_ = Sync()

	appBytes, err := os.ReadFile(filepath.Join(tmpDir, "app.log"))
	if err != nil {
		t.Fatalf("read app.log error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(appBytes)), "\n")
	expectedCount := workers * iterations
	if len(lines) != expectedCount {
		t.Errorf("expected %d log lines, got: %d", expectedCount, len(lines))
	}
}

func TestRollingWriterProperties(t *testing.T) {
	cfg := Config{
		MaxSize:    50,
		MaxAge:     15,
		MaxBackups: 5,
		Compress:   true,
	}
	w := newRollingWriter("/tmp/test.log", cfg)
	if w.MaxSize != 50 || w.MaxAge != 15 || w.MaxBackups != 5 || !w.Compress {
		t.Errorf("newRollingWriter config mismatch: %+v", w)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("LOG_PATH", "/tmp/custom_logs")
	t.Setenv("LOG_CONSOLE", "false")
	t.Setenv("LOG_FILE", "true")
	t.Setenv("LOG_MAX_SIZE", "200")
	t.Setenv("LOG_MAX_AGE", "60")
	t.Setenv("LOG_MAX_BACKUPS", "20")
	t.Setenv("LOG_COMPRESS", "false")

	cfg := LoadConfigFromEnv()
	if cfg.Level != "warn" {
		t.Errorf("expected level warn, got: %s", cfg.Level)
	}
	if cfg.Path != "/tmp/custom_logs" {
		t.Errorf("expected path /tmp/custom_logs, got: %s", cfg.Path)
	}
	if cfg.Console != false {
		t.Errorf("expected console false")
	}
	if cfg.File != true {
		t.Errorf("expected file true")
	}
	if cfg.MaxSize != 200 {
		t.Errorf("expected maxSize 200, got: %d", cfg.MaxSize)
	}
	if cfg.MaxAge != 60 {
		t.Errorf("expected maxAge 60, got: %d", cfg.MaxAge)
	}
	if cfg.MaxBackups != 20 {
		t.Errorf("expected maxBackups 20, got: %d", cfg.MaxBackups)
	}
	if cfg.Compress != false {
		t.Errorf("expected compress false")
	}
}

func TestFallbackBeforeInit(t *testing.T) {
	// 验证全局 logger 的 L() 和 S() 不为空，直接调用不会 panic
	if L() == nil {
		t.Fatal("L() should not be nil")
	}
	if S() == nil {
		t.Fatal("S() should not be nil")
	}
	// 安全调用无 panic
	Info("fallback call before custom init")
	Debug("fallback debug call")
}

func TestInitRejectsUnwritableFilesWithoutSwitching(t *testing.T) {
	base := t.TempDir()
	goodPath := filepath.Join(base, "good")
	cfg := DefaultConfig()
	cfg.Path, cfg.Console = goodPath, false
	if err := Init(cfg); err != nil {
		t.Fatalf("initial Init() error: %v", err)
	}
	defer Close()

	for _, filename := range []string{"app.log", "error.log"} {
		t.Run(filename, func(t *testing.T) {
			badPath := filepath.Join(base, filename)
			if err := os.Mkdir(badPath, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(badPath, filename), 0700); err != nil {
				t.Fatal(err)
			}
			cfg.Path = badPath
			if err := Init(cfg); err == nil {
				t.Fatalf("Init() should reject %s as a directory", filename)
			}
			Info("previous output remains active")
		})
	}

	data, err := os.ReadFile(filepath.Join(goodPath, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(data), "previous output remains active"); count != 2 {
		t.Fatalf("expected both logs in previous output, got %d", count)
	}
}

func TestCachedLoggersFollowReinitAndClose(t *testing.T) {
	base := t.TempDir()
	firstPath := filepath.Join(base, "first")
	secondPath := filepath.Join(base, "second")
	cfg := DefaultConfig()
	cfg.Path, cfg.Console = firstPath, false
	if err := Init(cfg); err != nil {
		t.Fatal(err)
	}
	defer Close()

	cached := L().With(String("source", "cached"))
	cachedSugar := S()
	cfg.Path = secondPath
	if err := Init(cfg); err != nil {
		t.Fatal(err)
	}
	cached.Info("after reinit")
	cachedSugar.Infof("formatted after %s", "reinit")

	first, err := os.ReadFile(filepath.Join(firstPath, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 0 {
		t.Fatalf("old output received logs after reinit: %s", first)
	}
	second, err := os.ReadFile(filepath.Join(secondPath, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(second), `"source":"cached"`) ||
		!strings.Contains(string(second), "formatted after reinit") {
		t.Fatalf("cached loggers did not use the new output: %s", second)
	}

	if err := Close(); err != nil {
		t.Fatal(err)
	}
	cached.Info("after close")
	firstAfter, err := os.ReadFile(filepath.Join(firstPath, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	secondAfter, err := os.ReadFile(filepath.Join(secondPath, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(firstAfter) != len(first) || len(secondAfter) != len(second) {
		t.Fatal("Close() should not allow a cached logger to reopen a file output")
	}
}

func TestConcurrentReinitKeepsAllLogEntries(t *testing.T) {
	base := t.TempDir()
	cfg := DefaultConfig()
	cfg.Console = false
	cfg.Path = filepath.Join(base, "a")
	if err := Init(cfg); err != nil {
		t.Fatal(err)
	}
	defer Close()

	const workers, iterations = 8, 100
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				Info("reinit race", Int("worker", worker), Int("iteration", i))
			}
		}(worker)
	}
	for i := 0; i < 20; i++ {
		if i%2 == 0 {
			cfg.Path = filepath.Join(base, "b")
		} else {
			cfg.Path = filepath.Join(base, "a")
		}
		if err := Init(cfg); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	if err := Sync(); err != nil {
		t.Fatal(err)
	}

	total := 0
	for _, name := range []string{"a", "b"} {
		data, err := os.ReadFile(filepath.Join(base, name, "app.log"))
		if err != nil {
			t.Fatal(err)
		}
		total += strings.Count(string(data), "reinit race")
	}
	if total != workers*iterations {
		t.Fatalf("expected %d entries across both files, got %d", workers*iterations, total)
	}
}

func TestZeroMaxBackupsDisablesCountLimit(t *testing.T) {
	writer := newRollingWriter(filepath.Join(t.TempDir(), "app.log"), Config{MaxBackups: 0})
	if writer.MaxBackups != 0 {
		t.Fatalf("MaxBackups=0 should disable the count limit, got %d", writer.MaxBackups)
	}
}

func TestErrorFileRespectsConfiguredLevel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Path = t.TempDir()
	cfg.Console = false
	cfg.Level = "panic"
	if err := Init(cfg); err != nil {
		t.Fatal(err)
	}
	defer Close()

	Error("below configured level")
	for _, filename := range []string{"app.log", "error.log"} {
		data, err := os.ReadFile(filepath.Join(cfg.Path, filename))
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != 0 {
			t.Fatalf("%s should respect the panic level, got %s", filename, data)
		}
	}
}
