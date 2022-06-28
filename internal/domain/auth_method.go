package domain

import (
	"context"
	"errors"
	"gorm.io/gorm"
)

type AuthMethod struct {
	ID           uint64 `gorm:"primary_key;autoIncrement:false"`
	Name         string `gorm:"type:varchar(64);unique_index"`
	Type         string
	Issuer       string
	ClientId     string
	ClientSecret string
}

func (r *repository) SaveAuthMethod(ctx context.Context, m *AuthMethod) error {
	tx := r.withContext(ctx).Save(m)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) ListAuthMethods(ctx context.Context) ([]AuthMethod, error) {
	var authMethods = []AuthMethod{}
	tx := r.withContext(ctx).Find(&authMethods)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return authMethods, nil
}

func (r *repository) GetAuthMethod(ctx context.Context, id uint64) (*AuthMethod, error) {
	var m AuthMethod
	tx := r.withContext(ctx).First(&m, "id = ?", id)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &m, nil
}

func (r *repository) DeleteAuthMethod(ctx context.Context, id uint64) error {
	tx := r.withContext(ctx).Delete(&AuthMethod{ID: id})
	return tx.Error
}
