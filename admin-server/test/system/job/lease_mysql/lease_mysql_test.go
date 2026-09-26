package leasemysql_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"mall/admin_core/bootstrap"
	"mall/admin_core/internal/model"
	"mall/admin_core/internal/system/job"
)

func TestInnoDBLeaseAllowsOnlyOneManager(t *testing.T) {
	dsn, err := bootstrap.ValidateTestDSN(os.Getenv("TEST_DATABASE_DSN"), os.Getenv("DATABASE_DSN"))
	if errors.Is(err, bootstrap.ErrTestDSNEmpty) {
		t.Skip("TEST_DATABASE_DSN is not configured")
	}
	if errors.Is(err, bootstrap.ErrTestDSNEqualsDev) {
		t.Skip("TEST_DATABASE_DSN points to the normal database; refusing MySQL integration test")
	}
	if err != nil {
		t.Skip("TEST_DATABASE_DSN is not a valid MySQL DSN")
	}
	if !strings.Contains(strings.ToLower(bootstrap.TestDBName(dsn)), "test") {
		t.Skip("TEST_DATABASE_DSN must target a disposable database whose name includes test")
	}

	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open explicit test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping explicit test database: %v", err)
	}
	var engine string
	if err := db.Raw("SELECT ENGINE FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", "system_job_status").Scan(&engine).Error; err != nil {
		t.Fatal(err)
	}
	if engine == "" {
		t.Skip("system_job_status is missing; run Migration 009 against the disposable test database first")
	}
	if !strings.EqualFold(engine, "InnoDB") {
		t.Fatalf("system_job_status engine = %q, want InnoDB", engine)
	}

	jobName := fmt.Sprintf("lease_test_%d", time.Now().UnixNano())
	if err := db.Create(&model.SystemJobStatus{JobName: jobName, DisplayName: "租约并发测试", Status: job.StatusIdle}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Where("job_name = ?", jobName).Delete(&model.SystemJobStatus{}).Error })

	var calls atomic.Int32
	entered := make(chan struct{}, 2)
	makeManager := func() *job.Manager {
		manager := job.NewManager(db, job.ManagerOptions{Interval: time.Hour, LeaseDuration: time.Second, LeaseRenewInterval: 100 * time.Millisecond})
		if err := manager.Register(job.Job{Name: jobName, DisplayName: "租约并发测试", Run: func(ctx context.Context) (job.RunResult, error) {
			calls.Add(1)
			entered <- struct{}{}
			<-ctx.Done()
			return job.RunResult{}, ctx.Err()
		}}); err != nil {
			t.Fatal(err)
		}
		return manager
	}
	first, second := makeManager(), makeManager()
	if err := first.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first manager did not acquire the lease")
	}
	if err := second.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	time.Sleep(150 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("executor ran %d times across managers; expected one InnoDB lease holder", got)
	}
}
