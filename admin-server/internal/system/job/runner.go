package job

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"mall/admin_core/internal/model"

	"gorm.io/gorm"
)

var (
	sensitiveValuePattern = regexp.MustCompile(`(?i)(access[_ -]?token|refresh[_ -]?token|app[_ -]?secret|secret|password|authorization|api[_ -]?key|private[_ -]?key|certificate|cert)\s*[:=]\s*("[^"]*"|'[^']*'|[^\s,;&]+)`)
	bearerPattern         = regexp.MustCompile(`(?i)bearer\s+[a-z0-9._~+/-]+=*`)
	pemPattern            = regexp.MustCompile(`(?s)-----BEGIN [^-]+-----.*?-----END [^-]+-----`)
	structuredBodyPattern = regexp.MustCompile(`(?s)(\{.*\}|\[.*\]|<\?xml.*|<html.*)`)
)

func (m *Manager) execute(job Job, token string) {
	startedAt := m.options.Now()
	runCtx, cancelRun := context.WithCancel(m.ctx)
	stopRenew := make(chan struct{})
	renewDone := make(chan struct{})
	go m.renewLease(runCtx, job.Name, token, cancelRun, stopRenew, renewDone)
	defer func() {
		close(stopRenew)
		<-renewDone
		cancelRun()
	}()

	maxAttempts := DefaultMaxAttempts
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if runCtx.Err() != nil {
			m.finishCanceled(job.Name, token, startedAt)
			return
		}
		if !m.updateAttempt(job.Name, token, attempt) {
			cancelRun()
			<-renewDone
			m.finishCanceled(job.Name, token, startedAt)
			return
		}

		result, runErr := runSafely(runCtx, job.Run)
		if runErr == nil && result.FailedCount > 0 {
			runErr = errors.New("部分记录处理失败")
		}
		if errors.Is(runErr, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
			m.finishCanceled(job.Name, token, startedAt)
			return
		}
		if runErr == nil {
			m.finishSuccess(job.Name, token, startedAt, result)
			return
		}

		if attempt < maxAttempts && !isPermanent(runErr) {
			delay := m.retryDelay(attempt)
			nextRetryAt := m.options.Now().Add(delay)
			if !m.markRetrying(job.Name, token, nextRetryAt, result, runErr) {
				cancelRun()
				<-renewDone
				m.finishCanceled(job.Name, token, startedAt)
				return
			}
			timer := time.NewTimer(delay)
			select {
			case <-runCtx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				m.finishCanceled(job.Name, token, startedAt)
				return
			case <-timer.C:
			}
			continue
		}
		m.finishFailure(job.Name, token, startedAt, result, runErr)
		return
	}
}

func runSafely(ctx context.Context, run RunFunc) (result RunResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = RunResult{}
			err = fmt.Errorf("任务执行异常 (%T)", recovered)
		}
	}()
	return run(ctx)
}

func (m *Manager) retryDelay(attempt int) time.Duration {
	if attempt <= 0 || attempt > len(m.options.RetryDelays) {
		return 0
	}
	return m.options.RetryDelays[attempt-1]
}

func (m *Manager) renewLease(ctx context.Context, jobName, token string, cancel context.CancelFunc, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(m.options.LeaseRenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-ticker.C:
			now := m.options.Now()
			updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Second)
			query := m.db.WithContext(updateCtx).Model(&model.SystemJobStatus{}).Where("job_name = ? AND locked_by = ?", jobName, token)
			expiresAt := any(now.Add(m.options.LeaseDuration))
			if m.db.Dialector.Name() == "mysql" {
				query = query.Where("lock_expires_at > NOW(6)")
				expiresAt = gorm.Expr("DATE_ADD(NOW(6), INTERVAL ? MICROSECOND)", m.options.LeaseDuration.Microseconds())
			} else {
				query = query.Where("lock_expires_at > ?", now)
			}
			result := query.Updates(map[string]any{"lock_expires_at": expiresAt, "updated_at": now})
			updateCancel()
			if result.Error != nil || result.RowsAffected != 1 {
				log.Printf("job lease renewal failed job=%s", jobName)
				cancel()
				return
			}
		}
	}
}

func (m *Manager) updateAttempt(jobName, token string, attempt int) bool {
	now := m.options.Now()
	ctx, cancel := statusWriteContext()
	defer cancel()
	result := m.activeLeaseQuery(ctx, jobName, token, now).
		Updates(map[string]any{"status": StatusRunning, "last_attempt_count": attempt, "next_retry_at": nil, "updated_at": now})
	if result.Error != nil || result.RowsAffected != 1 {
		if result.Error != nil {
			log.Printf("job attempt status update failed job=%s", jobName)
		}
		return false
	}
	return true
}

func (m *Manager) markRetrying(jobName, token string, next time.Time, result RunResult, runErr error) bool {
	now := m.options.Now()
	ctx, cancel := statusWriteContext()
	defer cancel()
	update := m.activeLeaseQuery(ctx, jobName, token, now).
		Updates(map[string]any{"status": StatusRetrying, "next_retry_at": next, "last_processed_count": result.ProcessedCount,
			"last_success_count": result.SuccessCount, "last_failed_count": result.FailedCount,
			"last_error": sanitizeError(runErr.Error()), "updated_at": now})
	if update.Error != nil || update.RowsAffected != 1 {
		log.Printf("job retry status update failed job=%s", jobName)
		return false
	}
	return true
}

func statusWriteContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (m *Manager) activeLeaseQuery(ctx context.Context, jobName, token string, now time.Time) *gorm.DB {
	query := m.db.WithContext(ctx).Model(&model.SystemJobStatus{}).Where("job_name = ? AND locked_by = ?", jobName, token)
	if m.db.Dialector.Name() == "mysql" {
		return query.Where("lock_expires_at > NOW(6)")
	}
	return query.Where("lock_expires_at > ?", now)
}

func (m *Manager) finishSuccess(jobName, token string, started time.Time, result RunResult) {
	now := m.options.Now()
	ctx, cancel := statusWriteContext()
	defer cancel()
	update := m.db.WithContext(ctx).Model(&model.SystemJobStatus{}).Where("job_name = ? AND locked_by = ?", jobName, token).
		Updates(map[string]any{"status": StatusSuccess, "last_finished_at": now, "last_success_at": now,
			"last_duration_ms": now.Sub(started).Milliseconds(), "last_processed_count": result.ProcessedCount,
			"last_success_count": result.SuccessCount, "last_failed_count": result.FailedCount,
			"last_error": "", "next_retry_at": nil, "consecutive_failures": 0,
			"locked_by": nil, "lock_expires_at": nil, "updated_at": now})
	if update.Error != nil || update.RowsAffected != 1 {
		log.Printf("job success status update failed job=%s", jobName)
		if update.Error != nil {
			m.finishCanceled(jobName, token, started)
		}
	}
}

func (m *Manager) finishFailure(jobName, token string, started time.Time, result RunResult, runErr error) {
	now := m.options.Now()
	ctx, cancel := statusWriteContext()
	defer cancel()
	update := m.db.WithContext(ctx).Model(&model.SystemJobStatus{}).Where("job_name = ? AND locked_by = ?", jobName, token).
		Updates(map[string]any{"status": StatusFailed, "last_finished_at": now,
			"last_duration_ms": now.Sub(started).Milliseconds(), "last_processed_count": result.ProcessedCount,
			"last_success_count": result.SuccessCount, "last_failed_count": result.FailedCount,
			"last_error": sanitizeError(runErr.Error()), "next_retry_at": nil,
			"consecutive_failures": gorm.Expr("consecutive_failures + 1"),
			"locked_by":            nil, "lock_expires_at": nil, "updated_at": now})
	if update.Error != nil || update.RowsAffected != 1 {
		log.Printf("job failure status update failed job=%s", jobName)
		if update.Error != nil {
			m.finishCanceled(jobName, token, started)
		}
	}
}

func (m *Manager) finishCanceled(jobName, token string, started time.Time) {
	now := m.options.Now()
	ctx, cancel := statusWriteContext()
	defer cancel()
	update := m.db.WithContext(ctx).Model(&model.SystemJobStatus{}).Where("job_name = ? AND locked_by = ?", jobName, token).
		Updates(map[string]any{"status": StatusIdle, "last_finished_at": now,
			"last_duration_ms": now.Sub(started).Milliseconds(), "next_retry_at": nil,
			"last_error": "", "locked_by": nil, "lock_expires_at": nil, "updated_at": now})
	if update.Error != nil || update.RowsAffected != 1 {
		log.Printf("job cancellation status update failed job=%s", jobName)
	}
}

func sanitizeError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "任务执行失败"
	}
	value = pemPattern.ReplaceAllString(value, "[证书内容已隐藏]")
	value = bearerPattern.ReplaceAllString(value, "Bearer [已隐藏]")
	value = sensitiveValuePattern.ReplaceAllString(value, "$1=[已隐藏]")
	value = structuredBodyPattern.ReplaceAllString(value, "[外部响应内容已隐藏]")
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > maxStoredErrorLength {
		value = value[:maxStoredErrorLength]
		for !utf8.ValidString(value) {
			value = value[:len(value)-1]
		}
		value += "…"
	}
	return value
}
