package bootstrap_test

import (
	"os"
	"testing"

	"mall/admin_core/bootstrap"
)

func TestLoadConfig_BizNoWorkerID_Default(t *testing.T) {
	os.Unsetenv("BIZ_NO_WORKER_ID")
	cfg := bootstrap.LoadConfig()
	if cfg.BizNoWorkerID != 0 {
		t.Errorf("expected default BizNoWorkerID=0, got %d", cfg.BizNoWorkerID)
	}
}

func TestLoadConfig_BizNoWorkerID_Configured(t *testing.T) {
	t.Setenv("BIZ_NO_WORKER_ID", "12")
	cfg := bootstrap.LoadConfig()
	if cfg.BizNoWorkerID != 12 {
		t.Errorf("expected BizNoWorkerID=12, got %d", cfg.BizNoWorkerID)
	}
}

func TestLoadConfig_AppTimezone_Default(t *testing.T) {
	os.Unsetenv("APP_TIMEZONE")
	cfg := bootstrap.LoadConfig()
	if cfg.AppTimezone != "Asia/Shanghai" {
		t.Errorf("expected default AppTimezone=Asia/Shanghai, got %s", cfg.AppTimezone)
	}
}

func TestLoadConfig_AppTimezone_Configured(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "UTC")
	cfg := bootstrap.LoadConfig()
	if cfg.AppTimezone != "UTC" {
		t.Errorf("expected AppTimezone=UTC, got %s", cfg.AppTimezone)
	}
}
