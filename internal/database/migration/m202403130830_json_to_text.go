package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func m202403130830_json_to_text() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202403130830",
		Migrate: func(db *gorm.DB) error {
			type Tailnet struct {
				IAMPolicy string
				ACLPolicy string
			}

			if err := db.Migrator().AlterColumn(&Tailnet{}, "IAMPolicy"); err != nil {
				return err
			}

			if err := db.Migrator().AlterColumn(&Tailnet{}, "ACLPolicy"); err != nil {
				return err
			}

			return nil
		},
		Rollback: nil,
	}
}
