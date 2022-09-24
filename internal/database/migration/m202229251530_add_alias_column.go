package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func m202229251530_add_alias_column() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202229251530a",
		Migrate: func(db *gorm.DB) error {
			type Tailnet struct {
				Alias *string `gorm:"type:varchar(64)"`
			}

			return db.AutoMigrate(
				&Tailnet{},
			)
		},
		Rollback: nil,
	}
}

func m202229251530_add_alias_column_constraint() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202229251530b",
		Migrate: func(db *gorm.DB) error {
			type Tailnet struct {
				Name  string  `gorm:"uniqueIndex"`
				Alias *string `gorm:"uniqueIndex"`
			}

			return db.AutoMigrate(
				&Tailnet{},
			)
		},
		Rollback: nil,
	}
}
