package domain

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/jsiebens/ionscale/internal/util"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"sync"
	"tailscale.com/tailcfg"
)

var (
	_defaultDERPMapMu sync.RWMutex
	_defaultDERPMap   = WrapDERPMap(tailcfg.DERPMap{})
)

func SetDefaultDERPMap(v *tailcfg.DERPMap) {
	if v == nil {
		return
	}

	_defaultDERPMapMu.Lock()
	defer _defaultDERPMapMu.Unlock()
	_defaultDERPMap = WrapDERPMap(*v)
}

func GetDefaultDERPMap() DERPMap {
	_defaultDERPMapMu.RLock()
	defer _defaultDERPMapMu.RUnlock()
	return _defaultDERPMap
}

type DERPMap struct {
	Checksum string
	DERPMap  tailcfg.DERPMap
}

func (d DERPMap) GetDERPMap(_ context.Context) (*DERPMap, error) {
	return &d, nil
}

func WrapDERPMap(d tailcfg.DERPMap) DERPMap {
	return DERPMap{
		Checksum: util.Checksum(d),
		DERPMap:  d,
	}
}

func (hi *DERPMap) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, hi)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (hi DERPMap) Value() (driver.Value, error) {
	bytes, err := json.Marshal(hi)
	return bytes, err
}

// GormDataType gorm common data type
func (DERPMap) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (DERPMap) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}

type DefaultDERPMap interface {
	GetDERPMap(ctx context.Context) (*DERPMap, error)
}
