package model

import "time"

// SystemJobStatus stores the latest run state for one built-in background job.
type SystemJobStatus struct {
	JobName             string     `gorm:"column:job_name;primaryKey;size:64" json:"job_name"`
	DisplayName         string     `gorm:"column:display_name;size:100;not null" json:"display_name"`
	Status              string     `gorm:"column:status;size:20;not null" json:"status"`
	LastStartedAt       *time.Time `gorm:"column:last_started_at" json:"last_started_at"`
	LastFinishedAt      *time.Time `gorm:"column:last_finished_at" json:"last_finished_at"`
	LastSuccessAt       *time.Time `gorm:"column:last_success_at" json:"last_success_at"`
	NextRetryAt         *time.Time `gorm:"column:next_retry_at" json:"next_retry_at"`
	LastDurationMS      int64      `gorm:"column:last_duration_ms;not null" json:"last_duration_ms"`
	LastProcessedCount  int        `gorm:"column:last_processed_count;not null" json:"last_processed_count"`
	LastSuccessCount    int        `gorm:"column:last_success_count;not null" json:"last_success_count"`
	LastFailedCount     int        `gorm:"column:last_failed_count;not null" json:"last_failed_count"`
	LastAttemptCount    int        `gorm:"column:last_attempt_count;not null" json:"last_attempt_count"`
	ConsecutiveFailures int        `gorm:"column:consecutive_failures;not null" json:"consecutive_failures"`
	ManualRetryCount    int        `gorm:"column:manual_retry_count;not null" json:"manual_retry_count"`
	LastError           string     `gorm:"column:last_error;type:text" json:"last_error"`
	LockedBy            *string    `gorm:"column:locked_by;size:100" json:"-"`
	LockExpiresAt       *time.Time `gorm:"column:lock_expires_at" json:"-"`
	CreatedAt           time.Time  `gorm:"column:created_at;not null" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;not null" json:"updated_at"`
}

func (SystemJobStatus) TableName() string { return "system_job_status" }
