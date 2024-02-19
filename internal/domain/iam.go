package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-bexpr"
	"github.com/mitchellh/pointerstructure"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
)

type Identity struct {
	UserID   string
	Username string
	Email    string
	Attr     map[string]interface{}
}

type IAMPolicy struct {
	Subs    []string            `json:"subs,omitempty"`
	Emails  []string            `json:"emails,omitempty"`
	Filters []string            `json:"filters,omitempty"`
	Roles   map[string]UserRole `json:"roles,omitempty"`
}

func (i *IAMPolicy) GetRole(user User) UserRole {
	if val, ok := i.Roles[user.Name]; ok {
		return val
	}
	return UserRoleMember
}

func (i *IAMPolicy) EvaluatePolicy(identity *Identity) (bool, error) {
	for _, sub := range i.Subs {
		if identity.UserID == sub {
			return true, nil
		}
	}

	for _, email := range i.Emails {
		if identity.Email == email {
			return true, nil
		}
	}

	for _, f := range i.Filters {
		if f == "*" {
			return true, nil
		}

		evaluator, err := bexpr.CreateEvaluator(f)
		if err != nil {
			return false, err
		}

		result, err := evaluator.Evaluate(identity.Attr)
		if err != nil && !errors.Is(err, pointerstructure.ErrNotFound) {
			return false, err
		}

		if result {
			return true, nil
		}
	}

	return false, nil
}

func (i *IAMPolicy) Equal(x *IAMPolicy) bool {
	if i == nil && x == nil {
		return true
	}
	if (i == nil) != (x == nil) {
		return false
	}
	return reflect.DeepEqual(i, x)
}

func (i *IAMPolicy) Scan(destination interface{}) error {
	switch value := destination.(type) {
	case []byte:
		return json.Unmarshal(value, i)
	default:
		return fmt.Errorf("unexpected data type %T", destination)
	}
}

func (i IAMPolicy) Value() (driver.Value, error) {
	bytes, err := json.Marshal(i)
	return bytes, err
}

// GormDataType gorm common data type
func (IAMPolicy) GormDataType() string {
	return "json"
}

// GormDBDataType gorm db data type
func (IAMPolicy) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	}
	return ""
}
