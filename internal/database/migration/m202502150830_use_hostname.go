package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func m202502150830_use_hostname() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202502150830",
		Migrate: func(db *gorm.DB) error {
			type Machine struct {
				UseOSHostname bool `gorm:"default:true"`
			}

			if err := db.Migrator().AddColumn(&Machine{}, "UseOSHostname"); err != nil {
				return err
			}

			return nil
		},
		Rollback: nil,
	}
}
