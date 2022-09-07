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

func CreateAuthKey(tailnet *Tailnet, user *User, ephemeral bool, tags Tags, expiresAt *time.Time) (string, *AuthKey) {
	key := util.RandStringBytes(12)
	pwd := util.RandStringBytes(22)
	value := fmt.Sprintf("%s_%s", key, pwd)

	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	return value, &AuthKey{
		ID:        util.NextID(),
		Key:       key,
		Hash:      string(hash),
		Ephemeral: ephemeral,
		Tags:      tags,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: expiresAt,

		TailnetID: tailnet.ID,
		UserID:    user.ID,
	}
}

type AuthKey struct {
	ID        uint64 `gorm:"primary_key"`
	Key       string
	Hash      string
	Ephemeral bool
	Tags      Tags

	CreatedAt time.Time
	ExpiresAt *time.Time

	TailnetID uint64
	Tailnet   Tailnet

	UserID uint64
	User   User
}

func (r *repository) GetAuthKey(ctx context.Context, authKeyId uint64) (*AuthKey, error) {
	var t AuthKey
	tx := r.withContext(ctx).
		Preload("User").
		Preload("Tailnet").
		Take(&t, "id = ?", authKeyId)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &t, nil
}

func (r *repository) SaveAuthKey(ctx context.Context, key *AuthKey) error {
	tx := r.withContext(ctx).Save(key)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) DeleteAuthKey(ctx context.Context, id uint64) (bool, error) {
	tx := r.withContext(ctx).Delete(&AuthKey{}, id)
	return tx.RowsAffected == 1, tx.Error
}

func (r *repository) DeleteAuthKeysByTailnet(ctx context.Context, tailnetID uint64) error {
	tx := r.withContext(ctx).
		Where("tailnet_id = ?", tailnetID).
		Delete(&AuthKey{TailnetID: tailnetID})

	return tx.Error
}

func (r *repository) DeleteAuthKeysByUser(ctx context.Context, userID uint64) error {
	tx := r.withContext(ctx).
		Where("user_id = ?", userID).
		Delete(&AuthKey{UserID: userID})

	return tx.Error
}

func (r *repository) ListAuthKeys(ctx context.Context, tailnetID uint64) ([]AuthKey, error) {
	var authKeys = []AuthKey{}
	tx := (r.withContext(ctx).
		Preload("User").
		Preload("Tailnet")).
		Where("tailnet_id = ?", tailnetID).
		Find(&authKeys)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return authKeys, nil
}

func (r *repository) ListAuthKeysByTailnetAndUser(ctx context.Context, tailnetID, userID uint64) ([]AuthKey, error) {
	var authKeys = []AuthKey{}
	tx := (r.withContext(ctx).
		Preload("User").
		Preload("Tailnet")).
		Where("tailnet_id = ? and user_id = ?", tailnetID, userID).
		Find(&authKeys)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return authKeys, nil
}

func (r *repository) LoadAuthKey(ctx context.Context, key string) (*AuthKey, error) {
	split := strings.Split(key, "_")
	if len(split) != 2 {
		return nil, nil
	}

	var m AuthKey
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
