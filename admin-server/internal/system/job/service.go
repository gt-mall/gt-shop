package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"mall/admin_core/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ManagerOptions struct {
	Interval           time.Duration
	LeaseDuration      time.Duration
	LeaseRenewInterval time.Duration
	RetryDelays        []time.Duration
	Now                func() time.Time
}

type Manager struct {
	db      *gorm.DB
	options ManagerOptions
	owner   string

	mu      sync.Mutex
	jobs    map[string]Job
	ordered []string
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	closed  bool
	wg      sync.WaitGroup
}

func NewManager(db *gorm.DB, options ...ManagerOptions) *Manager {
	config := ManagerOptions{
		Interval: time.Minute, LeaseDuration: 2 * time.Minute,
		LeaseRenewInterval: 30 * time.Second,
		RetryDelays:        []time.Duration{5 * time.Second, 30 * time.Second},
		Now:                time.Now,
	}
	if len(options) > 0 {
		provided := options[0]
		if provided.Interval > 0 {
			config.Interval = provided.Interval
		}
		if provided.LeaseDuration > 0 {
			config.LeaseDuration = provided.LeaseDuration
		}
		if provided.LeaseRenewInterval > 0 {
			config.LeaseRenewInterval = provided.LeaseRenewInterval
		}
		if provided.RetryDelays != nil {
			config.RetryDelays = append([]time.Duration(nil), provided.RetryDelays...)
		}
		if provided.Now != nil {
			config.Now = provided.Now
		}
	}
	if config.LeaseRenewInterval >= config.LeaseDuration {
		config.LeaseRenewInterval = config.LeaseDuration / 3
	}
	return &Manager{db: db, options: config, owner: randomID(), jobs: make(map[string]Job)}
}

func randomID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (m *Manager) Register(job Job) error {
	if job.Name == "" || job.DisplayName == "" || job.Run == nil {
		return errors.New("任务名称、显示名称和执行器不能为空")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started || m.closed {
		return ErrManagerClosed
	}
	if _, exists := m.jobs[job.Name]; exists {
		return fmt.Errorf("任务 %q 已注册", job.Name)
	}
	m.jobs[job.Name] = job
	m.ordered = append(m.ordered, job.Name)
	return nil
}

// Start checks the migrated table, idempotently registers each in-process task,
// runs an immediate sweep, then starts the one-minute scheduler.
func (m *Manager) Start(ctx context.Context) error {
	if m == nil || m.db == nil {
		return errors.New("任务管理器数据库不可用")
	}
	m.mu.Lock()
	if m.started || m.closed {
		m.mu.Unlock()
		return errors.New("任务管理器已启动或已关闭")
	}
	if len(m.jobs) == 0 {
		m.mu.Unlock()
		return errors.New("没有注册内置任务")
	}
	checkCtx, cancelCheck := context.WithTimeout(ctx, 5*time.Second)
	defer cancelCheck()
	if sqlDB, err := m.db.DB(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("获取任务数据库连接失败: %w", err)
	} else if err := sqlDB.PingContext(checkCtx); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("任务数据库不可用: %w", err)
	}
	if !m.db.WithContext(checkCtx).Migrator().HasTable(&model.SystemJobStatus{}) {
		m.mu.Unlock()
		return errors.New("system_job_status 不存在，请先执行数据库 Migration")
	}
	now := m.options.Now()
	for _, name := range m.ordered {
		job := m.jobs[name]
		row := model.SystemJobStatus{JobName: job.Name, DisplayName: job.DisplayName, Status: StatusIdle, CreatedAt: now, UpdatedAt: now}
		if err := m.db.WithContext(checkCtx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "job_name"}},
			DoUpdates: clause.Assignments(map[string]any{"display_name": job.DisplayName, "updated_at": now}),
		}).Create(&row).Error; err != nil {
			m.mu.Unlock()
			return fmt.Errorf("注册任务状态 %s 失败: %w", job.Name, err)
		}
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.started = true
	m.wg.Add(1)
	go m.scheduleLoop()
	jobNames := append([]string(nil), m.ordered...)
	m.mu.Unlock()

	for _, name := range jobNames {
		m.trigger(name)
	}
	return nil
}

// StartWhenReady retries initialization until the database and migration are
// available. The service context remains the lifetime of a successful manager.
func (m *Manager) StartWhenReady(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := m.Start(ctx); err == nil {
			return nil
		} else if ctx.Err() == nil {
			log.Printf("background task manager is unavailable: %v", err)
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (m *Manager) scheduleLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(m.options.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			names := append([]string(nil), m.ordered...)
			m.mu.Unlock()
			for _, name := range names {
				m.trigger(name)
			}
		}
	}
}

// Retry accepts a final failed job and starts it asynchronously under the same
// database lease used by scheduled executions.
func (m *Manager) Retry(jobName string) error {
	if m == nil {
		return ErrManagerClosed
	}
	queued, err := m.queue(jobName, true)
	if err != nil {
		return err
	}
	if !queued {
		return ErrJobLocked
	}
	return nil
}

func (m *Manager) queue(jobName string, manual bool) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobName]
	if !ok {
		return false, ErrUnknownJob
	}
	if !m.started || m.closed || m.ctx == nil || m.ctx.Err() != nil {
		return false, ErrManagerClosed
	}
	token := m.owner + ":" + randomID()
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	acquired, err := m.acquire(ctx, job, token, manual)
	cancel()
	if err != nil || !acquired {
		return acquired, err
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.execute(job, token)
	}()
	return true, nil
}

func (m *Manager) acquire(ctx context.Context, job Job, token string, manual bool) (bool, error) {
	now := m.options.Now()
	values := map[string]any{
		"locked_by": token, "lock_expires_at": now.Add(m.options.LeaseDuration),
		"status": StatusRunning, "last_started_at": now, "last_finished_at": nil,
		"next_retry_at": nil, "last_attempt_count": 0,
		"last_processed_count": 0, "last_success_count": 0, "last_failed_count": 0,
		"last_error": "", "updated_at": now,
	}
	query := m.db.WithContext(ctx).Model(&model.SystemJobStatus{}).Where("job_name = ?", job.Name)
	if manual {
		query = query.Where("status = ?", StatusFailed)
		values["manual_retry_count"] = gorm.Expr("manual_retry_count + 1")
	}
	if m.db.Dialector.Name() == "mysql" {
		values["lock_expires_at"] = gorm.Expr("DATE_ADD(NOW(6), INTERVAL ? MICROSECOND)", m.options.LeaseDuration.Microseconds())
		query = query.Where("(locked_by IS NULL OR lock_expires_at IS NULL OR lock_expires_at <= NOW(6))")
	} else {
		query = query.Where("(locked_by IS NULL OR lock_expires_at IS NULL OR lock_expires_at <= ?)", now)
	}
	result := query.Updates(values)
	if result.Error != nil {
		return false, fmt.Errorf("获取任务租约失败: %w", result.Error)
	}
	if result.RowsAffected == 1 {
		return true, nil
	}
	if !manual {
		return false, nil
	}
	var current model.SystemJobStatus
	if err := m.db.WithContext(ctx).Select("status", "lock_expires_at").Where("job_name = ?", job.Name).First(&current).Error; err != nil {
		return false, fmt.Errorf("读取任务状态失败: %w", err)
	}
	if current.Status != StatusFailed {
		return false, ErrNotFailed
	}
	return false, ErrJobLocked
}

func (m *Manager) List(ctx context.Context) ([]JobStatusResponse, error) {
	if m == nil || m.db == nil {
		return nil, errors.New("任务状态数据库不可用")
	}
	m.mu.Lock()
	if !m.started || m.closed || m.ctx == nil || m.ctx.Err() != nil {
		m.mu.Unlock()
		return nil, ErrManagerClosed
	}
	names := append([]string(nil), m.ordered...)
	m.mu.Unlock()
	if len(names) == 0 {
		return []JobStatusResponse{}, nil
	}
	var rows []model.SystemJobStatus
	if err := m.db.WithContext(ctx).Where("job_name IN ?", names).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询任务状态失败: %w", err)
	}
	byName := make(map[string]model.SystemJobStatus, len(rows))
	for _, row := range rows {
		byName[row.JobName] = row
	}
	result := make([]JobStatusResponse, 0, len(names))
	for _, name := range names {
		row, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("任务状态 %s 未注册", name)
		}
		result = append(result, statusResponse(row))
	}
	return result, nil
}

// Close cancels active attempts, stops scheduling and waits for the workers.
func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	cancel := m.cancel
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	m.wg.Wait()
}

func (m *Manager) trigger(jobName string) {
	if _, err := m.queue(jobName, false); err != nil && !errors.Is(err, ErrManagerClosed) {
		log.Printf("job scheduling failed job=%s", jobName)
	}
}

type Service struct {
	db      *gorm.DB
	manager *Manager
}

func NewService(db *gorm.DB, manager *Manager) *Service {
	return &Service{db: db, manager: manager}
}

func (s *Service) List(ctx context.Context) ([]JobStatusResponse, error) {
	if s.manager != nil {
		return s.manager.List(ctx)
	}
	if s.db == nil {
		return nil, errors.New("任务状态数据库不可用")
	}
	names := []string{PaymentReconcileJob, OrderTimeoutCloseJob, OrderAutoReceiveJob, RefundReconcileJob}
	var rows []model.SystemJobStatus
	if err := s.db.WithContext(ctx).Where("job_name IN ?", names).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("查询任务状态失败: %w", err)
	}
	byName := make(map[string]model.SystemJobStatus, len(rows))
	for _, row := range rows {
		byName[row.JobName] = row
	}
	result := make([]JobStatusResponse, 0, len(names))
	for _, name := range names {
		row, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("任务状态 %s 未注册", name)
		}
		result = append(result, statusResponse(row))
	}
	return result, nil
}

func (s *Service) Retry(jobName string) error {
	if s.manager == nil {
		return ErrManagerClosed
	}
	return s.manager.Retry(jobName)
}
