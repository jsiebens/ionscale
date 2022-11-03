package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func m202211031100_add_authorized_column() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202211031100",
		Migrate: func(db *gorm.DB) error {
			type Tailnet struct {
				Name                        string `gorm:"uniqueIndex"`
				MachineAuthorizationEnabled bool
			}

			type AuthKey struct {
				PreAuthorized bool
			}

			type Machine struct {
				Authorized bool `gorm:"default:true"`
			}

			return db.AutoMigrate(
				&Tailnet{},
				&AuthKey{},
				&Machine{},
			)
		},
		Rollback: nil,
	}
}
