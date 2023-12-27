package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"time"
)

func m202312271200_account_last_authenticated() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202312271200",
		Migrate: func(db *gorm.DB) error {
			type Account struct {
				LastAuthenticated *time.Time
			}

			return db.AutoMigrate(
				&Account{},
			)
		},
		Rollback: nil,
	}
}
