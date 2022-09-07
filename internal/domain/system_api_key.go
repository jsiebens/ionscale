package domain

import (
	"context"
	"errors"
	"fmt"
	"github.com/jsiebens/ionscale/internal/util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"strings"
	"time"
)

func CreateSystemApiKey(account *Account, expiresAt *time.Time) (string, *SystemApiKey) {
	key := util.RandStringBytes(12)
	pwd := util.RandStringBytes(22)
	value := fmt.Sprintf("sk_%s_%s", key, pwd)

	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	return value, &SystemApiKey{
		ID:        util.NextID(),
		Key:       key,
		Hash:      string(hash),
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,

		AccountID: account.ID,
	}
}

type SystemApiKey struct {
	ID   uint64 `gorm:"primary_key"`
	Key  string
	Hash string

	CreatedAt time.Time
	ExpiresAt *time.Time

	AccountID uint64
	Account   Account
}

func (r *repository) SaveSystemApiKey(ctx context.Context, key *SystemApiKey) error {
	tx := r.withContext(ctx).Save(key)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) LoadSystemApiKey(ctx context.Context, token string) (*SystemApiKey, error) {
	split := strings.Split(token, "_")
	if len(split) != 3 {
		return nil, nil
	}

	prefix := split[0]
	key := split[1]
	value := split[2]

	if prefix != "sk" {
		return nil, nil
	}

	var m SystemApiKey
	tx := r.withContext(ctx).Preload("Account").First(&m, "key = ?", key)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	if err := bcrypt.CompareHashAndPassword([]byte(m.Hash), []byte(value)); err != nil {
		return nil, nil
	}

	if !m.ExpiresAt.IsZero() && m.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	return &m, nil
}
