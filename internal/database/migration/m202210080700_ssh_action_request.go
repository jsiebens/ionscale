package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"time"
)

func m202210080700_ssh_action_request() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202210080700",
		Migrate: func(db *gorm.DB) error {
			type Tailnet struct {
				Name       string  `gorm:"uniqueIndex"`
				Alias      *string `gorm:"uniqueIndex"`
				SSHEnabled bool
			}

			type SSHActionRequest struct {
				Key          string `gorm:"primary_key"`
				Action       string
				SrcMachineID uint64
				DstMachineID uint64
				CreatedAt    time.Time
			}

			return db.AutoMigrate(
				&Tailnet{},
				&SSHActionRequest{},
			)
		},
		Rollback: nil,
	}
}
