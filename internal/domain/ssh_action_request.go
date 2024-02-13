package domain

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
)

type SSHActionRequestRepository interface {
	SaveSSHActionRequest(ctx context.Context, session *SSHActionRequest) error
	GetSSHActionRequest(ctx context.Context, key string) (*SSHActionRequest, error)
	DeleteSSHActionRequest(ctx context.Context, key string) error
}

type SSHActionRequest struct {
	Key          string `gorm:"primary_key"`
	Action       string
	SrcMachineID uint64
	DstMachineID uint64
	CreatedAt    time.Time
}

func (r *repository) SaveSSHActionRequest(ctx context.Context, session *SSHActionRequest) error {
	tx := r.withContext(ctx).Save(session)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) GetSSHActionRequest(ctx context.Context, key string) (*SSHActionRequest, error) {
	var m SSHActionRequest
	tx := r.withContext(ctx).Take(&m, "key = ?", key)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &m, nil
}

func (r *repository) DeleteSSHActionRequest(ctx context.Context, key string) error {
	tx := r.withContext(ctx).Delete(&SSHActionRequest{Key: key})
	return tx.Error
}
