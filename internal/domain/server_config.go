package domain

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	tkey "tailscale.com/types/key"
	"time"
)

type configKey string

const (
	derpMapConfigKey     configKey = "derp_map"
	controlKeysConfigKey configKey = "control_keys"
	jwksConfigKey        configKey = "jwks"
)

type JSONWebKeys struct {
	Key JSONWebKey
}

type JSONWebKey struct {
	Id         string
	PrivateKey rsa.PrivateKey
	CreatedAt  time.Time
}

func (j JSONWebKey) Public() crypto.PublicKey {
	return j.PrivateKey.Public()
}

type ServerConfig struct {
	Key   configKey `gorm:"primary_key"`
	Value []byte
}

type ControlKeys struct {
	ControlKey       tkey.MachinePrivate
	LegacyControlKey tkey.MachinePrivate
}

func (r *repository) GetControlKeys(ctx context.Context) (*ControlKeys, error) {
	var m ControlKeys
	err := r.getServerConfig(ctx, controlKeysConfigKey, &m)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (r *repository) SetControlKeys(ctx context.Context, v *ControlKeys) error {
	return r.setServerConfig(ctx, controlKeysConfigKey, v)
}

func (r *repository) GetJSONWebKeySet(ctx context.Context) (*JSONWebKeys, error) {
	var m JSONWebKeys
	err := r.getServerConfig(ctx, jwksConfigKey, &m)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func (r *repository) SetJSONWebKeySet(ctx context.Context, v *JSONWebKeys) error {
	return r.setServerConfig(ctx, jwksConfigKey, v)
}

func (r *repository) getServerConfig(ctx context.Context, s configKey, v interface{}) error {
	var m ServerConfig
	tx := r.withContext(ctx).Take(&m, "key = ?", s)

	if tx.Error != nil {
		return tx.Error
	}

	err := json.Unmarshal(m.Value, v)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) setServerConfig(ctx context.Context, s configKey, v interface{}) error {
	marshal, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c := &ServerConfig{
		Key:   s,
		Value: marshal,
	}
	tx := r.withContext(ctx).Save(c)

	return tx.Error
}
