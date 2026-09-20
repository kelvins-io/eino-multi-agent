package store

import (
	"context"
	"fmt"
	"time"

	"github.com/kelvins-io/eino-multi-agent/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	db *gorm.DB
}

func Open(cfg config.DatabaseConfig) (*Store, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := db.AutoMigrate(&Task{}, &TaskEvent{}, &Artifact{}, &Checkpoint{}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *gorm.DB { return s.db }

func (s *Store) CreateTask(ctx context.Context, task *Task) error {
	return s.db.WithContext(ctx).Create(task).Error
}

func (s *Store) GetTask(ctx context.Context, id string) (*Task, error) {
	var task Task
	if err := s.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Store) ListTasks(ctx context.Context, limit int) ([]Task, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var tasks []Task
	err := s.db.WithContext(ctx).Order("created_at DESC").Limit(limit).Find(&tasks).Error
	return tasks, err
}

func (s *Store) SaveTask(ctx context.Context, task *Task) error {
	return s.db.WithContext(ctx).Save(task).Error
}

func (s *Store) UpdateStatus(ctx context.Context, id, status string, extra func(*Task)) error {
	task, err := s.GetTask(ctx, id)
	if err != nil {
		return err
	}
	task.Status = status
	if extra != nil {
		extra(task)
	}
	return s.SaveTask(ctx, task)
}

func (s *Store) AppendEvent(ctx context.Context, ev *TaskEvent) error {
	return s.db.WithContext(ctx).Create(ev).Error
}

func (s *Store) ListEvents(ctx context.Context, taskID string, afterID uint64) ([]TaskEvent, error) {
	q := s.db.WithContext(ctx).Where("task_id = ?", taskID)
	if afterID > 0 {
		q = q.Where("id > ?", afterID)
	}
	var events []TaskEvent
	err := q.Order("id ASC").Find(&events).Error
	return events, err
}

func (s *Store) ReplaceArtifacts(ctx context.Context, taskID string, items []Artifact) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", taskID).Delete(&Artifact{}).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		return tx.Create(&items).Error
	})
}

func (s *Store) ListArtifacts(ctx context.Context, taskID string) ([]Artifact, error) {
	var items []Artifact
	err := s.db.WithContext(ctx).Where("task_id = ?", taskID).Order("id ASC").Find(&items).Error
	return items, err
}

func (s *Store) GetArtifact(ctx context.Context, taskID string, id uint64) (*Artifact, error) {
	var item Artifact
	if err := s.db.WithContext(ctx).Where("task_id = ? AND id = ?", taskID, id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
