package domain

import (
	"context"
	"errors"
	"github.com/jsiebens/ionscale/internal/util"
	"gorm.io/gorm"
	"time"
)

type Account struct {
	ID         uint64 `gorm:"primary_key"`
	ExternalID string
	LoginName  string
}

func (r *repository) GetOrCreateAccount(ctx context.Context, externalID, loginName string) (*Account, bool, error) {
	account := &Account{}
	id := util.NextID()

	tx := r.withContext(ctx).
		Where(Account{ExternalID: externalID}).
		Attrs(Account{ID: id, LoginName: loginName}).
		FirstOrCreate(account)

	if tx.Error != nil {
		return nil, false, tx.Error
	}

	return account, account.ID == id, nil
}

func (r *repository) GetAccount(ctx context.Context, id uint64) (*Account, error) {
	var account Account
	tx := r.withContext(ctx).Take(&account, "id = ?", id)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &account, nil
}

func (r *repository) SetAccountLastAuthenticated(ctx context.Context, accountID uint64) error {
	now := time.Now().UTC()
	tx := r.withContext(ctx).
		Model(Account{}).
		Where("id = ?", accountID).
		Updates(map[string]interface{}{"last_authenticated": &now})

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}
