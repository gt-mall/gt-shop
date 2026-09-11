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
