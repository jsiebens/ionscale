package domain

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"net/mail"
	"strings"
	"tailscale.com/util/dnsname"
)

type Tailnet struct {
	ID                          uint64 `gorm:"primary_key"`
	Name                        string
	DNSConfig                   DNSConfig
	IAMPolicy                   HuJSON[IAMPolicy]
	ACLPolicy                   HuJSON[ACLPolicy]
	DERPMap                     DERPMap
	ServiceCollectionEnabled    bool
	FileSharingEnabled          bool
	SSHEnabled                  bool
	MachineAuthorizationEnabled bool
}

type TailnetRepository interface {
	SaveTailnet(ctx context.Context, tailnet *Tailnet) error
	GetTailnet(ctx context.Context, id uint64) (*Tailnet, error)
	GetTailnetByName(ctx context.Context, name string) (*Tailnet, error)
	ListTailnets(ctx context.Context) ([]Tailnet, error)
	DeleteTailnet(ctx context.Context, id uint64) error
}

func (t Tailnet) GetDERPMap(ctx context.Context, fallback DefaultDERPMap) (*DERPMap, error) {
	if t.DERPMap.Checksum == "" {
		return fallback.GetDERPMap(ctx)
	} else {
		return &t.DERPMap, nil
	}
}

func SanitizeTailnetName(name string) string {
	name = strings.ToLower(name)

	a, err := mail.ParseAddress(name)
	if err == nil && a.Address == name {
		s := strings.Split(name, "@")
		return strings.Join([]string{dnsname.SanitizeLabel(s[0]), s[1]}, ".")
	}

	labels := strings.Split(name, ".")
	for i, s := range labels {
		labels[i] = dnsname.SanitizeLabel(s)
	}

	return strings.Join(labels, ".")
}

func (r *repository) SaveTailnet(ctx context.Context, tailnet *Tailnet) error {
	tx := r.withContext(ctx).Save(tailnet)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) GetTailnet(ctx context.Context, id uint64) (*Tailnet, error) {
	var t Tailnet
	tx := r.withContext(ctx).Take(&t, "id = ?", id)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &t, nil
}

func (r *repository) GetTailnetByName(ctx context.Context, name string) (*Tailnet, error) {
	var t Tailnet
	tx := r.withContext(ctx).Take(&t, "name = ?", name)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &t, nil
}

func (r *repository) ListTailnets(ctx context.Context) ([]Tailnet, error) {
	var tailnets = []Tailnet{}
	tx := r.withContext(ctx).Find(&tailnets)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tailnets, nil
}

func (r *repository) DeleteTailnet(ctx context.Context, id uint64) error {
	tx := r.withContext(ctx).Delete(&Tailnet{ID: id})
	return tx.Error
}
