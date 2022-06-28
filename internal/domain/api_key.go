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

func CreateApiKey(tailnet *Tailnet, user *User, expiresAt *time.Time) (string, *ApiKey) {
	key := util.RandStringBytes(12)
	pwd := util.RandStringBytes(22)
	value := fmt.Sprintf("%s_%s", key, pwd)

	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	return value, &ApiKey{
		ID:        util.NextID(),
		Key:       key,
		Hash:      string(hash),
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,

		TailnetID: tailnet.ID,
		UserID:    user.ID,
	}
}

type ApiKey struct {
	ID   uint64 `gorm:"primary_key;autoIncrement:false"`
	Key  string `gorm:"type:varchar(64);unique_index"`
	Hash string

	CreatedAt time.Time
	ExpiresAt *time.Time

	TailnetID uint64
	Tailnet   Tailnet

	UserID uint64
	User   User
}

func (r *repository) SaveApiKey(ctx context.Context, key *ApiKey) error {
	tx := r.withContext(ctx).Save(key)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) LoadApiKey(ctx context.Context, key string) (*ApiKey, error) {
	split := strings.Split(key, "_")
	if len(split) != 2 {
		return nil, nil
	}

	var m ApiKey
	tx := r.withContext(ctx).Preload("User").Preload("Tailnet").First(&m, "key = ?", split[0])

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	if err := bcrypt.CompareHashAndPassword([]byte(m.Hash), []byte(split[1])); err != nil {
		return nil, nil
	}

	if !m.ExpiresAt.IsZero() && m.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	return &m, nil
}

func (r *repository) DeleteApiKeysByAuthMethod(ctx context.Context, authMethodID uint64) (int64, error) {
	subQuery := r.withContext(ctx).
		Select("api_keys.id").
		Table("api_keys").
		Joins("JOIN users u on u.id = api_keys.user_id JOIN accounts a on a.id = u.account_id").
		Where("a.auth_method_id = ?", authMethodID)

	tx := r.withContext(ctx).
		Delete(&ApiKey{}, "id in (?)", subQuery)

	if tx.Error != nil {
		return 0, tx.Error
	}

	return tx.RowsAffected, nil
}
