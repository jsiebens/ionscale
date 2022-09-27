package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/jsiebens/ionscale/internal/domain"
	"gorm.io/gorm"
)

func m202209251530_add_autoallowips_column() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202209251530",
		Migrate: func(db *gorm.DB) error {
			type Machine struct {
				AutoAllowIPs domain.AllowIPs
			}

			return db.AutoMigrate(
				&Machine{},
			)
		},
		Rollback: nil,
	}
}
