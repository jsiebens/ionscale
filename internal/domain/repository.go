package domain

import (
	"context"
	"gorm.io/gorm"
	"tailscale.com/tailcfg"
	"time"
)

type Repository interface {
	GetDERPMap(ctx context.Context) (*tailcfg.DERPMap, error)
	SetDERPMap(ctx context.Context, v *tailcfg.DERPMap) error

	GetOrCreateTailnet(ctx context.Context, name string) (*Tailnet, bool, error)
	GetTailnet(ctx context.Context, id uint64) (*Tailnet, error)
	ListTailnets(ctx context.Context) ([]Tailnet, error)

	SaveAuthKey(ctx context.Context, key *AuthKey) error
	DeleteAuthKey(ctx context.Context, id uint64) (bool, error)
	ListAuthKeys(ctx context.Context, tailnetID uint64) ([]AuthKey, error)
	LoadAuthKey(ctx context.Context, key string) (*AuthKey, error)

	GetOrCreateServiceUser(ctx context.Context, tailnet *Tailnet) (*User, bool, error)
	ListUsers(ctx context.Context, tailnetID uint64) (Users, error)

	SaveMachine(ctx context.Context, m *Machine) error
	DeleteMachine(ctx context.Context, id uint64) (bool, error)
	GetMachine(ctx context.Context, id uint64) (*Machine, error)
	GetMachineByKey(ctx context.Context, tailnetID uint64, key string) (*Machine, error)
	GetMachineByKeys(ctx context.Context, machineKey string, nodeKey string) (*Machine, error)
	CountMachinesWithIPv4(ctx context.Context, ip string) (int64, error)
	GetNextMachineNameIndex(ctx context.Context, tailnetID uint64, name string) (uint64, error)
	ListMachineByTailnet(ctx context.Context, tailnetID uint64) (Machines, error)
	ListMachinePeers(ctx context.Context, tailnetID uint64, key string) (Machines, error)
	ListInactiveEphemeralMachines(ctx context.Context, checkpoint time.Time) (Machines, error)
	SetMachineLastSeen(ctx context.Context, machineID uint64) error
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
	return r.db.WithContext(ctx)
}
