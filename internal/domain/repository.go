package domain

import (
	"context"
	"gorm.io/gorm"
	"tailscale.com/tailcfg"
	"time"
)

type Repository interface {
	GetControlKeys(ctx context.Context) (*ControlKeys, error)
	SetControlKeys(ctx context.Context, v *ControlKeys) error

	GetDERPMap(ctx context.Context) (*tailcfg.DERPMap, error)
	SetDERPMap(ctx context.Context, v *tailcfg.DERPMap) error

	SaveAuthMethod(ctx context.Context, m *AuthMethod) error
	ListAuthMethods(ctx context.Context) ([]AuthMethod, error)
	GetAuthMethod(ctx context.Context, id uint64) (*AuthMethod, error)

	GetAccount(ctx context.Context, accountID uint64) (*Account, error)
	GetOrCreateAccount(ctx context.Context, authMethodID uint64, externalID, loginName string) (*Account, bool, error)

	SaveTailnet(ctx context.Context, tailnet *Tailnet) error
	GetOrCreateTailnet(ctx context.Context, name string) (*Tailnet, bool, error)
	GetTailnet(ctx context.Context, id uint64) (*Tailnet, error)
	ListTailnets(ctx context.Context) ([]Tailnet, error)
	DeleteTailnet(ctx context.Context, id uint64) error

	GetDNSConfig(ctx context.Context, tailnetID uint64) (*DNSConfig, error)
	SetDNSConfig(ctx context.Context, tailnetID uint64, config *DNSConfig) error
	DeleteDNSConfig(ctx context.Context, tailnetID uint64) error
	GetACLPolicy(ctx context.Context, tailnetID uint64) (*ACLPolicy, error)
	SetACLPolicy(ctx context.Context, tailnetID uint64, policy *ACLPolicy) error
	DeleteACLPolicy(ctx context.Context, tailnetID uint64) error

	SaveApiKey(ctx context.Context, key *ApiKey) error
	LoadApiKey(ctx context.Context, key string) (*ApiKey, error)

	GetAuthKey(ctx context.Context, id uint64) (*AuthKey, error)
	SaveAuthKey(ctx context.Context, key *AuthKey) error
	DeleteAuthKey(ctx context.Context, id uint64) (bool, error)
	DeleteAuthKeysByTailnet(ctx context.Context, tailnetID uint64) error
	ListAuthKeys(ctx context.Context, tailnetID uint64) ([]AuthKey, error)
	ListAuthKeysByTailnetAndUser(ctx context.Context, tailnetID, userID uint64) ([]AuthKey, error)
	LoadAuthKey(ctx context.Context, key string) (*AuthKey, error)

	GetOrCreateServiceUser(ctx context.Context, tailnet *Tailnet) (*User, bool, error)
	ListUsers(ctx context.Context, tailnetID uint64) (Users, error)
	GetOrCreateUserWithAccount(ctx context.Context, tailnet *Tailnet, account *Account) (*User, bool, error)
	DeleteUsersByTailnet(ctx context.Context, tailnetID uint64) error

	SaveMachine(ctx context.Context, m *Machine) error
	DeleteMachine(ctx context.Context, id uint64) (bool, error)
	GetMachine(ctx context.Context, id uint64) (*Machine, error)
	GetMachineByKey(ctx context.Context, tailnetID uint64, key string) (*Machine, error)
	GetMachineByKeys(ctx context.Context, machineKey string, nodeKey string) (*Machine, error)
	CountMachinesWithIPv4(ctx context.Context, ip string) (int64, error)
	GetNextMachineNameIndex(ctx context.Context, tailnetID uint64, name string) (uint64, error)
	ListMachineByTailnet(ctx context.Context, tailnetID uint64) (Machines, error)
	CountMachineByTailnet(ctx context.Context, tailnetID uint64) (int64, error)
	DeleteMachineByTailnet(ctx context.Context, tailnetID uint64) error
	ListMachinePeers(ctx context.Context, tailnetID uint64, key string) (Machines, error)
	ListInactiveEphemeralMachines(ctx context.Context, checkpoint time.Time) (Machines, error)
	SetMachineLastSeen(ctx context.Context, machineID uint64) error
	ExpireMachineByAuthMethod(ctx context.Context, tailnetID, authMethodID uint64) (int64, error)

	SaveRegistrationRequest(ctx context.Context, request *RegistrationRequest) error
	GetRegistrationRequestByKey(ctx context.Context, key string) (*RegistrationRequest, error)
	GetRegistrationRequestByMachineKey(ctx context.Context, key string) (*RegistrationRequest, error)

	SaveAuthenticationRequest(ctx context.Context, session *AuthenticationRequest) error
	GetAuthenticationRequest(ctx context.Context, key string) (*AuthenticationRequest, error)
	DeleteAuthenticationRequest(ctx context.Context, key string) error

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
	return r.db.WithContext(ctx)
}

func (r *repository) Transaction(action func(Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return action(NewRepository(tx))
	})
}
