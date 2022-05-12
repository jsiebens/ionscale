package domain

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
)

type TailnetConfig struct {
	Key       string `gorm:"primary_key"`
	TailnetID uint64 `gorm:"primary_key;autoIncrement:false"`
	Value     []byte
}

type DNSConfig struct {
	MagicDNS         bool
	OverrideLocalDNS bool
	Nameservers      []string
	Routes           map[string][]string
}

func (r *repository) GetDNSConfig(ctx context.Context, tailnetID uint64) (*DNSConfig, error) {
	var m DNSConfig
	err := r.getConfig(ctx, "dns_config", tailnetID, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *repository) SetDNSConfig(ctx context.Context, tailnetID uint64, config *DNSConfig) error {
	return r.setConfig(ctx, "dns_config", tailnetID, config)
}

func (r *repository) SetACLPolicy(ctx context.Context, tailnetID uint64, policy *ACLPolicy) error {
	if err := r.setConfig(ctx, "acl_policy", tailnetID, policy); err != nil {
		return err
	}
	return nil
}

func (r *repository) GetACLPolicy(ctx context.Context, tailnetID uint64) (*ACLPolicy, error) {
	var p = defaultPolicy()
	err := r.getConfig(ctx, "acl_policy", tailnetID, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) getConfig(ctx context.Context, s string, tailnetID uint64, v interface{}) error {
	var m TailnetConfig
	tx := r.withContext(ctx).Take(&m, "key = ? AND tailnet_id = ?", s, tailnetID)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil
	}

	if tx.Error != nil {
		return tx.Error
	}

	err := json.Unmarshal(m.Value, v)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) setConfig(ctx context.Context, s string, tailnetID uint64, v interface{}) error {
	marshal, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c := &TailnetConfig{
		Key:       s,
		Value:     marshal,
		TailnetID: tailnetID,
	}
	tx := r.withContext(ctx).Save(c)

	return tx.Error
}
