package domain

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
)

type AuthenticationRequest struct {
	Key       string `gorm:"primary_key"`
	Token     string
	TailnetID *uint64
	Error     string
	CreatedAt time.Time
}

func (r *repository) SaveAuthenticationRequest(ctx context.Context, session *AuthenticationRequest) error {
	tx := r.withContext(ctx).Save(session)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) GetAuthenticationRequest(ctx context.Context, key string) (*AuthenticationRequest, error) {
	var m AuthenticationRequest
	tx := r.withContext(ctx).First(&m, "key = ?", key)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &m, nil
}

func (r *repository) DeleteAuthenticationRequest(ctx context.Context, key string) error {
	tx := r.withContext(ctx).Delete(&AuthenticationRequest{Key: key})
	return tx.Error
}
