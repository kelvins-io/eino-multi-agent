package store

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type CheckpointStore struct {
	store *Store
}

func NewCheckpointStore(s *Store) *CheckpointStore {
	return &CheckpointStore{store: s}
}

func (c *CheckpointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	var row Checkpoint
	err := c.store.db.WithContext(ctx).First(&row, "id = ?", checkPointID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return row.Data, true, nil
}

func (c *CheckpointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	row := Checkpoint{ID: checkPointID, Data: checkPoint}
	return c.store.db.WithContext(ctx).Save(&row).Error
}

func (c *CheckpointStore) Delete(ctx context.Context, checkPointID string) error {
	return c.store.db.WithContext(ctx).Delete(&Checkpoint{}, "id = ?", checkPointID).Error
}
