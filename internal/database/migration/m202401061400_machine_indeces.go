package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func m202401061400_machine_indeces() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202401061400",
		Migrate: func(db *gorm.DB) error {
			type Machine struct {
				ID        uint64 `gorm:"primaryKey;autoIncrement:false;index:idx_tailnet_id_id,priority:2"`
				Name      string `gorm:"index:idx_tailnet_id_name,unique,priority:2"`
				NameIdx   uint64 `gorm:"index:idx_tailnet_id_name,unique,sort:desc,priority:3"`
				TailnetID uint64 `gorm:"index:idx_tailnet_id_id,priority:1;index:idx_tailnet_id_name,priority:1"`
			}

			db.Migrator().DropIndex(&Machine{}, "idx_tailnet_id_name")

			return db.AutoMigrate(
				&Machine{},
			)
		},
		Rollback: nil,
	}
}
