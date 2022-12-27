package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/jsiebens/ionscale/internal/domain"
	"gorm.io/gorm"
)

func m202212270800_machine_indeces() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202212270800",
		Migrate: func(db *gorm.DB) error {
			type Machine struct {
				ID         uint64 `gorm:"primaryKey;autoIncrement:false;index:idx_tailnet_id_id,priority:2"`
				MachineKey string `gorm:"index:idx_machine_keys"`
				NodeKey    string `gorm:"index:idx_machine_keys"`

				Name    string `gorm:"index:idx_tailnet_id_name,priority:2"`
				NameIdx uint64 `gorm:"index:idx_tailnet_id_name,sort:desc,priority:3"`

				TailnetID uint64 `gorm:"index:idx_tailnet_id_id,priority:1;index:idx_tailnet_id_name,priority:1"`

				IPv4 domain.IP `gorm:"index:idx_ipv4"`
			}

			return db.AutoMigrate(
				&Machine{},
			)
		},
		Rollback: nil,
	}
}
