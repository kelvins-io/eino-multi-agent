package store

import (
	"context"
	"time"
)

func (s *Store) CreateProject(ctx context.Context, p *Project) error {
	return s.db.WithContext(ctx).Create(p).Error
}

func (s *Store) GetProject(ctx context.Context, id string) (*Project, error) {
	var p Project
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ListProjects(ctx context.Context) ([]Project, error) {
	var items []Project
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (s *Store) SaveProject(ctx context.Context, p *Project) error {
	return s.db.WithContext(ctx).Save(p).Error
}

func (s *Store) CreateSchedule(ctx context.Context, item *Schedule) error {
	return s.db.WithContext(ctx).Create(item).Error
}

func (s *Store) GetSchedule(ctx context.Context, id string) (*Schedule, error) {
	var item Schedule
	if err := s.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) ListSchedules(ctx context.Context) ([]Schedule, error) {
	var items []Schedule
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (s *Store) SaveSchedule(ctx context.Context, item *Schedule) error {
	return s.db.WithContext(ctx).Save(item).Error
}

func (s *Store) DueSchedules(ctx context.Context, now time.Time) ([]Schedule, error) {
	var items []Schedule
	err := s.db.WithContext(ctx).Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now).Find(&items).Error
	return items, err
}

func (s *Store) CreateConnector(ctx context.Context, item *Connector) error {
	return s.db.WithContext(ctx).Create(item).Error
}

func (s *Store) GetConnector(ctx context.Context, id string) (*Connector, error) {
	var item Connector
	if err := s.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) ListConnectors(ctx context.Context) ([]Connector, error) {
	var items []Connector
	err := s.db.WithContext(ctx).Order("created_at DESC").Find(&items).Error
	return items, err
}

func (s *Store) SaveConnector(ctx context.Context, item *Connector) error {
	return s.db.WithContext(ctx).Save(item).Error
}

func (s *Store) EnabledConnectors(ctx context.Context) ([]Connector, error) {
	var items []Connector
	err := s.db.WithContext(ctx).Where("enabled = ?", true).Find(&items).Error
	return items, err
}

func (s *Store) ListInFlight(ctx context.Context) ([]Task, error) {
	var items []Task
	err := s.db.WithContext(ctx).
		Where("status IN ?", []string{StatusQueued, StatusRunning}).
		Order("created_at ASC").
		Find(&items).Error
	return items, err
}

func (s *Store) CreateAudit(ctx context.Context, item *AuditLog) error {
	return s.db.WithContext(ctx).Create(item).Error
}

func (s *Store) ListAudit(ctx context.Context, limit int) ([]AuditLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var items []AuditLog
	err := s.db.WithContext(ctx).Order("id DESC").Limit(limit).Find(&items).Error
	return items, err
}
