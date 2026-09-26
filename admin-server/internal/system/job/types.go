package job

import (
	"context"
	"errors"
	"strings"
	"time"

	"mall/admin_core/internal/model"
)

const (
	StatusIdle     = "idle"
	StatusRunning  = "running"
	StatusRetrying = "retrying"
	StatusSuccess  = "success"
	StatusFailed   = "failed"

	PaymentReconcileJob  = "payment_reconcile"
	OrderTimeoutCloseJob = "order_timeout_close"
	OrderAutoReceiveJob  = "order_auto_receive"
	RefundReconcileJob   = "refund_reconcile"
	DefaultMaxAttempts   = 3
	maxStoredErrorLength = 512
)

var (
	ErrUnknownJob    = errors.New("未知任务")
	ErrNotFailed     = errors.New("任务仅允许在最终失败后人工重试")
	ErrJobLocked     = errors.New("任务租约仍有效，暂时不能重试")
	ErrManagerClosed = errors.New("任务管理器未运行")
)

// RunResult describes the work completed by one execution attempt.
type RunResult struct {
	ProcessedCount int
	SuccessCount   int
	FailedCount    int
}

type RunFunc func(context.Context) (RunResult, error)

// Job is one registered task and its existing business executor.
type Job struct {
	Name        string
	DisplayName string
	Run         RunFunc
}

// JobStatusResponse is the public monitoring view; lease ownership remains private.
type JobStatusResponse struct {
	JobName             string     `json:"job_name"`             // 内置任务唯一名称
	DisplayName         string     `json:"display_name"`         // 任务中文名称
	Status              string     `json:"status"`               // idle、running、retrying、success 或 failed
	LastStartedAt       *time.Time `json:"last_started_at"`      // 最近开始时间
	LastFinishedAt      *time.Time `json:"last_finished_at"`     // 最近结束时间
	LastSuccessAt       *time.Time `json:"last_success_at"`      // 最近成功时间
	NextRetryAt         *time.Time `json:"next_retry_at"`        // 下次自动重试时间
	LastDurationMS      int64      `json:"last_duration_ms"`     // 最近执行耗时，单位毫秒
	LastProcessedCount  int        `json:"last_processed_count"` // 最近处理数量
	LastSuccessCount    int        `json:"last_success_count"`   // 最近成功数量
	LastFailedCount     int        `json:"last_failed_count"`    // 最近失败数量
	LastAttemptCount    int        `json:"last_attempt_count"`   // 最近执行周期尝试次数
	ConsecutiveFailures int        `json:"consecutive_failures"` // 连续失败周期数
	ManualRetryCount    int        `json:"manual_retry_count"`   // 人工重试次数
	LastError           string     `json:"last_error"`           // 最近错误摘要，已脱敏
} // @name SystemJobStatus

func statusResponse(row model.SystemJobStatus) JobStatusResponse {
	return JobStatusResponse{
		JobName: row.JobName, DisplayName: row.DisplayName, Status: row.Status,
		LastStartedAt: row.LastStartedAt, LastFinishedAt: row.LastFinishedAt,
		LastSuccessAt: row.LastSuccessAt, NextRetryAt: row.NextRetryAt,
		LastDurationMS: row.LastDurationMS, LastProcessedCount: row.LastProcessedCount,
		LastSuccessCount: row.LastSuccessCount, LastFailedCount: row.LastFailedCount,
		LastAttemptCount: row.LastAttemptCount, ConsecutiveFailures: row.ConsecutiveFailures,
		ManualRetryCount: row.ManualRetryCount, LastError: row.LastError,
	}
}

type RetryAccepted struct {
	JobName string `json:"job_name"` // 已接受的任务名称
	Status  string `json:"status"`   // 当前任务状态
} // @name SystemJobRetryAccepted

// PermanentError marks configuration or validation failures that should wait for
// the next scheduled cycle or an administrator instead of being retried quickly.
type PermanentError struct{ Err error }

func (e PermanentError) Error() string {
	if e.Err == nil {
		return "任务因不可重试原因失败"
	}
	return e.Err.Error()
}

func (e PermanentError) Unwrap() error { return e.Err }

func NoRetry(err error) error {
	if err == nil {
		return nil
	}
	return PermanentError{Err: err}
}

func isPermanent(err error) bool {
	var permanent PermanentError
	if errors.As(err, &permanent) {
		return true
	}
	message := strings.ToLower(errString(err))
	for _, marker := range []string{"必须配置", "未配置", "配置密文损坏", "配置无法解密", "invalid configuration", "missing config", "真实退款渠道未配置"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func IsPermanentError(err error) bool { return isPermanent(err) }

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
