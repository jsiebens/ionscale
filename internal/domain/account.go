package domain

import (
	"context"
	"errors"
	"github.com/jsiebens/ionscale/internal/util"
	"gorm.io/gorm"
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
