package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func m202212201300_add_user_id_column() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202212201300",
		Migrate: func(db *gorm.DB) error {
			type RegistrationRequest struct {
				Key    string `gorm:"type:varchar(64);uniqueIndex"`
				UserID uint64
			}

			return db.AutoMigrate(
				&RegistrationRequest{},
			)
		},
		Rollback: nil,
	}
}
