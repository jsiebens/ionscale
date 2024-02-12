package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"time"
)

func m202402120800_user_last_authenticated() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202402120800",
		Migrate: func(db *gorm.DB) error {
			type User struct {
				LastAuthenticated *time.Time
			}

			return db.AutoMigrate(
				&User{},
			)
		},
		Rollback: nil,
	}
}
