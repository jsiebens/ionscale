package domain

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"tailscale.com/tailcfg"
	"time"
)

type RegistrationRequest struct {
	MachineKey    string `gorm:"primary_key"`
	Key           string
	Data          RegistrationRequestData
	CreatedAt     time.Time
	Authenticated bool
	Error         string
}

func (r *RegistrationRequest) IsFinished() bool {
	return r.Authenticated || len(r.Error) != 0
}

type RegistrationRequestData tailcfg.RegisterRequest

func (hi *RegistrationRequestData) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, hi)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (hi RegistrationRequestData) Value() (driver.Value, error) {
	bytes, err := json.Marshal(hi)
	return bytes, err
}

// GormDataType gorm common data type
func (RegistrationRequestData) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (RegistrationRequestData) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}

func (r *repository) SaveRegistrationRequest(ctx context.Context, request *RegistrationRequest) error {
	tx := r.withContext(ctx).Save(request)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) GetRegistrationRequestByKey(ctx context.Context, key string) (*RegistrationRequest, error) {
	var m RegistrationRequest
	tx := r.withContext(ctx).First(&m, "key = ?", key)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &m, nil
}

func (r *repository) GetRegistrationRequestByMachineKey(ctx context.Context, key string) (*RegistrationRequest, error) {
	var m RegistrationRequest
	tx := r.withContext(ctx).First(&m, "machine_key = ?", key)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &m, nil
}
