package domain

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	AccountRepository
	ApiKeyRepository
	SystemApiKeyRepository
	AuthKeyRepository
	MachineRepository
	TailnetRepository
	UserRepository
	AuthenticationRequestRepository
	RegistrationRequestRepository
	SSHActionRequestRepository

	GetControlKeys(ctx context.Context) (*ControlKeys, error)
	SetControlKeys(ctx context.Context, keys *ControlKeys) error

	GetJSONWebKeySet(ctx context.Context) (*JSONWebKeys, error)
	SetJSONWebKeySet(ctx context.Context, keys *JSONWebKeys) error

	Transaction(func(rp Repository) error) error
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

type repository struct {
	db *gorm.DB
}

func (r *repository) withContext(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Omit(clause.Associations)
}

func (r *repository) Transaction(action func(Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return action(NewRepository(tx))
	})
}
