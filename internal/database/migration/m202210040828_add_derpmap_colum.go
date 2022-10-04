package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/jsiebens/ionscale/internal/domain"
	"gorm.io/gorm"
)

func m202210040828_add_derpmap_colum() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202210040828",
		Migrate: func(db *gorm.DB) error {
			type Tailnet struct {
				Name    string  `gorm:"uniqueIndex"`
				Alias   *string `gorm:"uniqueIndex"`
				DERPMap domain.DERPMap
			}

			return db.AutoMigrate(
				&Tailnet{},
			)
		},
		Rollback: nil,
	}
}
