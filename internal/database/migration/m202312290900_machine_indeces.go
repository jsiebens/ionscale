package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func m202312290900_machine_indeces() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202312290900",
		Migrate: func(db *gorm.DB) error {
			type Machine struct {
				Name    string `gorm:"index:idx_tailnet_id_name,unique,priority:2"`
				NameIdx uint64 `gorm:"index:idx_tailnet_id_name,unique,sort:desc,priority:3"`
			}

			db.Migrator().DropIndex(&Machine{}, "idx_tailnet_id_name")

			return db.AutoMigrate(
				&Machine{},
			)
		},
		Rollback: nil,
	}
}
