package domain

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"tailscale.com/tailcfg"
)

type DERPMap struct {
	Checksum string
	DERPMap  tailcfg.DERPMap
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
	GetDERPMap(ctx context.Context) (*tailcfg.DERPMap, error)
}
