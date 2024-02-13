package domain

import (
	"context"
	"encoding/json"
	"github.com/jsiebens/ionscale/internal/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http"
	"sync"
	"tailscale.com/tailcfg"
	"time"
)

type Repository interface {
	AccountRepository
	ApiKeyRepository
	SystemApiKeyRepository
	AuthKeyRepository
	MachineRepository
	TailnetRepository
	UserRepository
	AuthenticationRequestRepository
	RegistrationRequestRepository
	SSHActionRequestRepository

	GetControlKeys(ctx context.Context) (*ControlKeys, error)
	SetControlKeys(ctx context.Context, keys *ControlKeys) error

	GetJSONWebKeySet(ctx context.Context) (*JSONWebKeys, error)
	SetJSONWebKeySet(ctx context.Context, keys *JSONWebKeys) error

	GetDERPMap(ctx context.Context) (*DERPMap, error)
	SetDERPMap(ctx context.Context, v *DERPMap) error

	Transaction(func(rp Repository) error) error
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db:             db,
		defaultDERPMap: &derpMapCache{},
	}
}

type repository struct {
	db             *gorm.DB
	defaultDERPMap *derpMapCache
}

func (r *repository) withContext(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Omit(clause.Associations)
}

func (r *repository) Transaction(action func(Repository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return action(NewRepository(tx))
	})
}

type derpMapCache struct {
	sync.RWMutex
	value *DERPMap
}

func (d *derpMapCache) Get() (*DERPMap, error) {
	d.RLock()

	if d.value != nil {
		d.RUnlock()
		return d.value, nil
	}
	d.RUnlock()

	d.Lock()
	defer d.Unlock()

	getJson := func(url string, target interface{}) error {
		c := http.Client{Timeout: 5 * time.Second}
		r, err := c.Get(url)
		if err != nil {
			return err
		}
		defer r.Body.Close()

		return json.NewDecoder(r.Body).Decode(target)
	}

	m := &tailcfg.DERPMap{}
	if err := getJson("https://controlplane.tailscale.com/derpmap/default", m); err != nil {
		return nil, err
	}

	d.value = &DERPMap{
		Checksum: util.Checksum(m),
		DERPMap:  *m,
	}

	return d.value, nil
}
