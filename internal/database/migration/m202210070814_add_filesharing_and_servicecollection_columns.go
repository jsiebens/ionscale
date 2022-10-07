package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func m202210070814_add_filesharing_and_servicecollection_columns() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202210070814",
		Migrate: func(db *gorm.DB) error {
			type Tailnet struct {
				Name                     string  `gorm:"uniqueIndex"`
				Alias                    *string `gorm:"uniqueIndex"`
				ServiceCollectionEnabled bool
				FileSharingEnabled       bool
			}

			return db.AutoMigrate(
				&Tailnet{},
			)
		},
		Rollback: nil,
	}
}
