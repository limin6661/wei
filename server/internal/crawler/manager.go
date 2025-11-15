package crawler

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"wechat2rss/internal/config"
	"wechat2rss/internal/models"
)

// Manager polls pending tasks and dispatches them to the executor.
type Manager struct {
	cfg      *config.Config
	db       *gorm.DB
	ticker   *time.Ticker
	executor Executor
	running  chan struct{}
}

func NewManager(cfg *config.Config, db *gorm.DB) *Manager {
	return newManagerWithTicker(cfg, db, time.Duration(cfg.TaskPollInterval)*time.Second, NewArticleExecutor(db))
}

func newManagerWithTicker(cfg *config.Config, db *gorm.DB, interval time.Duration, executor Executor) *Manager {
	concurrency := cfg.CrawlerConcurrent
	if concurrency < 1 {
		concurrency = 1
	}
	return &Manager{
		cfg:      cfg,
		db:       db,
		ticker:   time.NewTicker(interval),
		executor: executor,
		running:  make(chan struct{}, concurrency),
	}
}

// Start begins polling pending tasks.
func (m *Manager) Start(ctx context.Context) {
	log.Printf("crawler manager started (interval=%ds, chromium=%s, concurrency=%d)",
		m.cfg.TaskPollInterval, m.cfg.ChromiumPath, m.cfg.CrawlerConcurrent)
	for {
		select {
		case <-ctx.Done():
			log.Println("crawler manager stopping")
			m.ticker.Stop()
			return
		case <-m.ticker.C:
			m.pollOnce(ctx)
		}
	}
}

func (m *Manager) pollOnce(ctx context.Context) {
	for i := 0; i < m.cfg.CrawlerConcurrent; i++ {
		task, err := m.claimNextTask()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		if err != nil {
			log.Printf("claim task error: %v", err)
			return
		}
		m.running <- struct{}{}
		go func(task models.Task) {
			defer func() { <-m.running }()
			m.executeTask(ctx, &task)
		}(task)
	}
}

func (m *Manager) claimNextTask() (models.Task, error) {
	var task models.Task
	err := m.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ?", models.TaskStatusPending).
			Order("id").
			Preload("Account").
			First(&task).Error; err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&models.Task{}).
			Where("id = ?", task.ID).
			Updates(map[string]any{
				"status":     models.TaskStatusRunning,
				"started_at": now,
				"error_msg":  "",
			}).Error; err != nil {
			return err
		}
		task.Status = models.TaskStatusRunning
		task.StartedAt = &now
		return nil
	})
	return task, err
}

func (m *Manager) executeTask(ctx context.Context, task *models.Task) {
	log.Printf("task %d started (account=%d)", task.ID, task.AccountID)
	m.logTask(task.ID, "info", "任务开始执行")

	taskCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	err := m.executor.Execute(taskCtx, task)

	finish := time.Now()
	if err == nil {
		if err := m.markSuccess(task.ID, finish); err != nil {
			log.Printf("task %d success but update failed: %v", task.ID, err)
		}
		m.logTask(task.ID, "info", "任务执行成功")
		log.Printf("task %d completed", task.ID)
		return
	}

	m.logTask(task.ID, "error", err.Error())
	if err := m.handleFailure(task.ID, task.RetryCount, err); err != nil {
		log.Printf("task %d failure update error: %v", task.ID, err)
	}
}

func (m *Manager) markSuccess(taskID uint, finish time.Time) error {
	return m.db.Model(&models.Task{}).
		Where("id = ?", taskID).
		Updates(map[string]any{
			"status":      models.TaskStatusSuccess,
			"finished_at": finish,
			"error_msg":   "",
		}).Error
}

func (m *Manager) handleFailure(taskID uint, retryCount int, execErr error) error {
	log.Printf("task %d failed (retry=%d): %v", taskID, retryCount, execErr)
	nextStatus := models.TaskStatusPending
	if retryCount+1 >= models.TaskMaxRetries {
		nextStatus = models.TaskStatusFailed
	}

	return m.db.Model(&models.Task{}).
		Where("id = ?", taskID).
		Updates(map[string]any{
			"status":      nextStatus,
			"retry_count": gorm.Expr("retry_count + 1"),
			"error_msg":   execErr.Error(),
			"finished_at": time.Now(),
		}).Error
}

func (m *Manager) logTask(taskID uint, level, msg string) {
	entry := models.TaskLog{
		TaskID:  taskID,
		Level:   level,
		Message: msg,
	}
	if err := m.db.Create(&entry).Error; err != nil {
		log.Printf("task %d log error: %v", taskID, err)
	}
}
