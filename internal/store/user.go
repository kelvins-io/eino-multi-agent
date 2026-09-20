package store

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

func (s *Store) CreateUser(ctx context.Context, u *User) error {
	return s.db.WithContext(ctx).Create(u).Error
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	if err := s.db.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	if err := s.db.WithContext(ctx).First(&u, "username = ?", username).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
