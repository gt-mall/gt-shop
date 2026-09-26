package runner_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mall/admin_core/internal/model"
	"mall/admin_core/internal/system/job"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?cache=shared&_pragma=busy_timeout(5000)", t.TempDir()+"/jobs.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemJobStatus{}); err != nil {
		t.Fatalf("create job status table: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func options() job.ManagerOptions {
	return job.ManagerOptions{
		Interval: time.Hour, LeaseDuration: time.Second, LeaseRenewInterval: 100 * time.Millisecond,
		RetryDelays: []time.Duration{5 * time.Millisecond, 10 * time.Millisecond},
	}
}

func newManager(t *testing.T, db *gorm.DB, name string, run job.RunFunc) *job.Manager {
	t.Helper()
	manager := job.NewManager(db, options())
	if err := manager.Register(job.Job{Name: name, DisplayName: "测试任务", Run: run}); err != nil {
		t.Fatalf("register job: %v", err)
	}
	return manager
}

func start(t *testing.T, manager *job.Manager) {
	t.Helper()
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start manager: %v", err)
	}
	t.Cleanup(manager.Close)
}

func waitRow(t *testing.T, db *gorm.DB, name string, match func(model.SystemJobStatus) bool) model.SystemJobStatus {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var row model.SystemJobStatus
		if err := db.Where("job_name = ?", name).First(&row).Error; err == nil && match(row) {
			return row
		}
		time.Sleep(5 * time.Millisecond)
	}
	var row model.SystemJobStatus
	_ = db.Where("job_name = ?", name).First(&row).Error
	t.Fatalf("job %s did not reach expected state; current status=%s attempt=%d error=%q", name, row.Status, row.LastAttemptCount, row.LastError)
	return row
}

func TestFirstRunSuccessPersistsResult(t *testing.T) {
	db := openDB(t)
	manager := newManager(t, db, "job_success", func(context.Context) (job.RunResult, error) {
		return job.RunResult{ProcessedCount: 4, SuccessCount: 4}, nil
	})
	start(t, manager)

	row := waitRow(t, db, "job_success", func(row model.SystemJobStatus) bool { return row.Status == job.StatusSuccess })
	if row.LastAttemptCount != 1 || row.LastProcessedCount != 4 || row.LastSuccessCount != 4 || row.LastFailedCount != 0 {
		t.Fatalf("unexpected successful result: %+v", row)
	}
	if row.LastSuccessAt == nil || row.LastStartedAt == nil || row.LastFinishedAt == nil {
		t.Fatalf("success timestamps were not recorded: %+v", row)
	}
}

func TestStartWhenReadyRecoversAfterStatusTableAppears(t *testing.T) {
	db := openDB(t)
	if err := db.Migrator().DropTable(&model.SystemJobStatus{}); err != nil {
		t.Fatal(err)
	}
	manager := newManager(t, db, "job_recover_start", func(context.Context) (job.RunResult, error) {
		return job.RunResult{ProcessedCount: 1, SuccessCount: 1}, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer func() { cancel(); manager.Close() }()
	if err := manager.Start(ctx); err == nil {
		t.Fatal("manager started before the status table existed")
	}
	done := make(chan error, 1)
	if _, err := manager.List(context.Background()); !errors.Is(err, job.ErrManagerClosed) {
		t.Fatalf("unstarted manager list error = %v, want ErrManagerClosed", err)
	}
	if err := db.AutoMigrate(&model.SystemJobStatus{}); err != nil {
		t.Fatal(err)
	}
	go func() { done <- manager.StartWhenReady(ctx, 20*time.Millisecond) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("manager did not recover: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("manager did not retry initialization")
	}
	waitRow(t, db, "job_recover_start", func(row model.SystemJobStatus) bool { return row.Status == job.StatusSuccess })
}

func TestFailedAttemptStatusWriteReleasesLease(t *testing.T) {
	db := openDB(t)
	var failOnce atomic.Bool
	failOnce.Store(true)
	if err := db.Callback().Update().Before("gorm:update").Register("fail_job_attempt_once", func(tx *gorm.DB) {
		values, ok := tx.Statement.Dest.(map[string]any)
		if !ok || tx.Statement.Table != "system_job_status" {
			return
		}
		if _, acquiring := values["locked_by"]; acquiring {
			return
		}
		if _, attempt := values["last_attempt_count"]; attempt && failOnce.CompareAndSwap(true, false) {
			tx.AddError(errors.New("injected attempt status failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	manager := newManager(t, db, "job_attempt_abort", func(context.Context) (job.RunResult, error) {
		calls.Add(1)
		return job.RunResult{}, nil
	})
	start(t, manager)
	row := waitRow(t, db, "job_attempt_abort", func(row model.SystemJobStatus) bool {
		return row.Status == job.StatusIdle && row.LastFinishedAt != nil
	})
	if failOnce.Load() || calls.Load() != 0 || row.LockedBy != nil || row.LockExpiresAt != nil {
		t.Fatalf("failed status write left an active task or lease: calls=%d row=%+v", calls.Load(), row)
	}
}

func TestFailedRetryStatusWriteReleasesLease(t *testing.T) {
	db := openDB(t)
	var failOnce atomic.Bool
	failOnce.Store(true)
	if err := db.Callback().Update().Before("gorm:update").Register("fail_job_retry_once", func(tx *gorm.DB) {
		values, ok := tx.Statement.Dest.(map[string]any)
		if !ok || tx.Statement.Table != "system_job_status" || values["status"] != job.StatusRetrying {
			return
		}
		if failOnce.CompareAndSwap(true, false) {
			tx.AddError(errors.New("injected retry status failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	manager := newManager(t, db, "job_retry_abort", func(context.Context) (job.RunResult, error) {
		calls.Add(1)
		return job.RunResult{ProcessedCount: 1, FailedCount: 1}, errors.New("temporary failure")
	})
	start(t, manager)
	row := waitRow(t, db, "job_retry_abort", func(row model.SystemJobStatus) bool {
		return row.Status == job.StatusIdle && row.LastFinishedAt != nil
	})
	if failOnce.Load() || calls.Load() != 1 || row.LockedBy != nil || row.LockExpiresAt != nil {
		t.Fatalf("failed retry status write left an active task or lease: calls=%d row=%+v", calls.Load(), row)
	}
}

func TestFailedFinalStatusWriteReleasesLease(t *testing.T) {
	db := openDB(t)
	var failOnce atomic.Bool
	failOnce.Store(true)
	if err := db.Callback().Update().Before("gorm:update").Register("fail_job_finish_once", func(tx *gorm.DB) {
		values, ok := tx.Statement.Dest.(map[string]any)
		if !ok || tx.Statement.Table != "system_job_status" || values["status"] != job.StatusSuccess {
			return
		}
		if failOnce.CompareAndSwap(true, false) {
			tx.AddError(errors.New("injected final status failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	manager := newManager(t, db, "job_finish_abort", func(context.Context) (job.RunResult, error) {
		return job.RunResult{ProcessedCount: 1, SuccessCount: 1}, nil
	})
	start(t, manager)
	row := waitRow(t, db, "job_finish_abort", func(row model.SystemJobStatus) bool {
		return row.Status == job.StatusIdle && row.LastFinishedAt != nil
	})
	if failOnce.Load() || row.LockedBy != nil || row.LockExpiresAt != nil {
		t.Fatalf("failed final status write left a lease: %+v", row)
	}
}

func TestRetrySucceedsOnSecondAttempt(t *testing.T) {
	db := openDB(t)
	var calls atomic.Int32
	manager := newManager(t, db, "job_second_try", func(context.Context) (job.RunResult, error) {
		if calls.Add(1) == 1 {
			return job.RunResult{ProcessedCount: 2, SuccessCount: 1, FailedCount: 1}, errors.New("temporary failure")
		}
		return job.RunResult{ProcessedCount: 2, SuccessCount: 2}, nil
	})
	start(t, manager)
	row := waitRow(t, db, "job_second_try", func(row model.SystemJobStatus) bool { return row.Status == job.StatusSuccess })
	if row.LastAttemptCount != 2 || row.ConsecutiveFailures != 0 || row.LastError != "" {
		t.Fatalf("unexpected retry result: %+v", row)
	}
}

func TestThreeFailedAttemptsSetFinalFailure(t *testing.T) {
	db := openDB(t)
	var calls atomic.Int32
	manager := newManager(t, db, "job_all_fail", func(context.Context) (job.RunResult, error) {
		calls.Add(1)
		return job.RunResult{ProcessedCount: 3, SuccessCount: 1, FailedCount: 2}, errors.New("provider unavailable")
	})
	start(t, manager)
	row := waitRow(t, db, "job_all_fail", func(row model.SystemJobStatus) bool { return row.Status == job.StatusFailed })
	if calls.Load() != 3 || row.LastAttemptCount != 3 || row.ConsecutiveFailures != 1 || row.LastFailedCount != 2 {
		t.Fatalf("unexpected final failure: calls=%d row=%+v", calls.Load(), row)
	}
	if row.LockedBy != nil || row.LockExpiresAt != nil || row.NextRetryAt != nil {
		t.Fatalf("failure did not release its lease: %+v", row)
	}
}

func TestManualRetryRunsSameExecutorAndCanSucceed(t *testing.T) {
	db := openDB(t)
	var calls atomic.Int32
	manager := newManager(t, db, "job_manual", func(context.Context) (job.RunResult, error) {
		if calls.Add(1) <= 3 {
			return job.RunResult{}, errors.New("first cycle failure")
		}
		return job.RunResult{ProcessedCount: 1, SuccessCount: 1}, nil
	})
	start(t, manager)
	waitRow(t, db, "job_manual", func(row model.SystemJobStatus) bool { return row.Status == job.StatusFailed })
	if err := manager.Retry("job_manual"); err != nil {
		t.Fatalf("retry final failure: %v", err)
	}
	row := waitRow(t, db, "job_manual", func(row model.SystemJobStatus) bool {
		return row.Status == job.StatusSuccess && row.ManualRetryCount == 1
	})
	if row.LastAttemptCount != 1 || calls.Load() != 4 {
		t.Fatalf("manual retry did not begin a new executor cycle: calls=%d row=%+v", calls.Load(), row)
	}
}

func TestRunningJobRejectsManualRetry(t *testing.T) {
	db := openDB(t)
	entered := make(chan struct{}, 1)
	manager := newManager(t, db, "job_running", func(ctx context.Context) (job.RunResult, error) {
		entered <- struct{}{}
		<-ctx.Done()
		return job.RunResult{}, ctx.Err()
	})
	start(t, manager)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("job did not start")
	}
	if err := manager.Retry("job_running"); !errors.Is(err, job.ErrNotFailed) {
		t.Fatalf("retry error = %v, want ErrNotFailed", err)
	}
}

func TestTwoManagersCannotRunSameLeasedJob(t *testing.T) {
	db := openDB(t)
	var calls atomic.Int32
	entered := make(chan struct{}, 1)
	first := newManager(t, db, "job_shared", func(ctx context.Context) (job.RunResult, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-ctx.Done()
		return job.RunResult{}, ctx.Err()
	})
	start(t, first)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first manager did not start job")
	}
	second := newManager(t, db, "job_shared", func(context.Context) (job.RunResult, error) {
		calls.Add(1)
		return job.RunResult{}, nil
	})
	start(t, second)
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("job ran %d times across managers, want one lease holder", calls.Load())
	}
}

func TestExpiredLeaseCanBeRecovered(t *testing.T) {
	db := openDB(t)
	expired := time.Now().Add(-time.Minute)
	owner := "dead-instance"
	if err := db.Create(&model.SystemJobStatus{JobName: "job_recover", DisplayName: "测试任务", Status: job.StatusRunning,
		LockedBy: &owner, LockExpiresAt: &expired, CreatedAt: time.Now(), UpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	manager := newManager(t, db, "job_recover", func(context.Context) (job.RunResult, error) {
		return job.RunResult{ProcessedCount: 1, SuccessCount: 1}, nil
	})
	start(t, manager)
	row := waitRow(t, db, "job_recover", func(row model.SystemJobStatus) bool { return row.Status == job.StatusSuccess })
	if row.LastAttemptCount != 1 {
		t.Fatalf("expired lease was not recovered: %+v", row)
	}
}

func TestCancelDuringRetryWaitDoesNotRepeatOrRecordFailure(t *testing.T) {
	db := openDB(t)
	var calls atomic.Int32
	config := options()
	config.RetryDelays = []time.Duration{250 * time.Millisecond, 10 * time.Millisecond}
	manager := job.NewManager(db, config)
	if err := manager.Register(job.Job{Name: "job_cancel_wait", DisplayName: "测试任务", Run: func(context.Context) (job.RunResult, error) {
		calls.Add(1)
		return job.RunResult{}, errors.New("transient")
	}}); err != nil {
		t.Fatal(err)
	}
	start(t, manager)
	waitRow(t, db, "job_cancel_wait", func(row model.SystemJobStatus) bool { return row.Status == job.StatusRetrying })
	manager.Close()
	row := waitRow(t, db, "job_cancel_wait", func(row model.SystemJobStatus) bool { return row.Status == job.StatusIdle })
	if calls.Load() != 1 || row.ConsecutiveFailures != 0 {
		t.Fatalf("canceled retry was recorded as a failed cycle: calls=%d row=%+v", calls.Load(), row)
	}
}

func TestPermanentErrorSkipsFastRetries(t *testing.T) {
	db := openDB(t)
	var calls atomic.Int32
	manager := newManager(t, db, "job_no_retry", func(context.Context) (job.RunResult, error) {
		calls.Add(1)
		return job.RunResult{}, job.NoRetry(errors.New("支付密钥未配置"))
	})
	start(t, manager)
	row := waitRow(t, db, "job_no_retry", func(row model.SystemJobStatus) bool { return row.Status == job.StatusFailed })
	if calls.Load() != 1 || !strings.Contains(row.LastError, "未配置") {
		t.Fatalf("permanent configuration failure was retried or lost: calls=%d row=%+v", calls.Load(), row)
	}
}

func TestErrorSummaryIsBoundedAndRedactsSecretsAndProviderBodies(t *testing.T) {
	db := openDB(t)
	message := "app_secret=do-not-store Bearer abc123 {\"private_key\":\"pem-data\"} " + strings.Repeat("x", 700)
	manager := newManager(t, db, "job_redact", func(context.Context) (job.RunResult, error) {
		return job.RunResult{}, job.NoRetry(errors.New(message))
	})
	start(t, manager)
	row := waitRow(t, db, "job_redact", func(row model.SystemJobStatus) bool { return row.Status == job.StatusFailed })
	if strings.Contains(row.LastError, "do-not-store") || strings.Contains(row.LastError, "abc123") || strings.Contains(row.LastError, "pem-data") {
		t.Fatalf("sensitive content persisted: %q", row.LastError)
	}
	if len(row.LastError) > 515 || !strings.Contains(row.LastError, "外部响应内容已隐藏") {
		t.Fatalf("error summary was not redacted and bounded: length=%d value=%q", len(row.LastError), row.LastError)
	}
}

func TestTaskWaitingForRetryDoesNotBlockOtherTasks(t *testing.T) {
	db := openDB(t)
	config := options()
	config.RetryDelays = []time.Duration{250 * time.Millisecond, 10 * time.Millisecond}
	manager := job.NewManager(db, config)
	if err := manager.Register(job.Job{Name: "job_waiting", DisplayName: "等待任务", Run: func(context.Context) (job.RunResult, error) {
		return job.RunResult{}, errors.New("retry later")
	}}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(job.Job{Name: "job_other", DisplayName: "独立任务", Run: func(context.Context) (job.RunResult, error) {
		return job.RunResult{ProcessedCount: 1, SuccessCount: 1}, nil
	}}); err != nil {
		t.Fatal(err)
	}
	start(t, manager)
	waitRow(t, db, "job_waiting", func(row model.SystemJobStatus) bool { return row.Status == job.StatusRetrying })
	waitRow(t, db, "job_other", func(row model.SystemJobStatus) bool { return row.Status == job.StatusSuccess })
}

func TestRegisterIsIdempotentAcrossManagerRestarts(t *testing.T) {
	db := openDB(t)
	for i := 0; i < 2; i++ {
		manager := newManager(t, db, "job_once", func(context.Context) (job.RunResult, error) { return job.RunResult{}, nil })
		if err := manager.Start(context.Background()); err != nil {
			t.Fatalf("start manager %d: %v", i, err)
		}
		waitRow(t, db, "job_once", func(row model.SystemJobStatus) bool { return row.Status == job.StatusSuccess })
		manager.Close()
	}
	var count int64
	if err := db.Model(&model.SystemJobStatus{}).Where("job_name = ?", "job_once").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("registered row count = %d, want 1", count)
	}
}
