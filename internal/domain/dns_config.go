package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DNSConfig struct {
	HttpsCertsEnabled bool                `json:"http_certs"`
	MagicDNS          bool                `json:"magic_dns"`
	OverrideLocalDNS  bool                `json:"override_local_dns"`
	Nameservers       []string            `json:"nameservers"`
	Routes            map[string][]string `json:"routes"`
	SearchDomains     []string            `json:"search_domains"`
}

func (i *DNSConfig) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, i)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (i DNSConfig) Value() (driver.Value, error) {
	bytes, err := json.Marshal(i)
	return bytes, err
}

// GormDataType gorm common data type
func (DNSConfig) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (DNSConfig) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}
