package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/jsiebens/ionscale/internal/domain"
	"gorm.io/gorm"
	"time"
)

func m202209070900_initial_schema() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202209070900",
		Migrate: func(db *gorm.DB) error {
			// it's a good practice to copy the struct inside the function,
			// so side effects are prevented if the original struct changes during the time
			type ServerConfig struct {
				Key   string `gorm:"primary_key"`
				Value []byte
			}

			type Tailnet struct {
				ID        uint64 `gorm:"primary_key;autoIncrement:false"`
				Name      string `gorm:"type:varchar(64);unique_index"`
				DNSConfig domain.DNSConfig
				IAMPolicy domain.IAMPolicy
				ACLPolicy domain.ACLPolicy
			}

			type Account struct {
				ID         uint64 `gorm:"primary_key;autoIncrement:false"`
				ExternalID string
				LoginName  string
			}

			type User struct {
				ID        uint64 `gorm:"primary_key;autoIncrement:false"`
				Name      string
				UserType  domain.UserType
				TailnetID uint64
				Tailnet   Tailnet
				AccountID *uint64
				Account   *Account
			}

			type SystemApiKey struct {
				ID   uint64 `gorm:"primary_key;autoIncrement:false"`
				Key  string `gorm:"type:varchar(64);unique_index"`
				Hash string

				CreatedAt time.Time
				ExpiresAt *time.Time

				AccountID uint64
				Account   Account
			}

			type ApiKey struct {
				ID   uint64 `gorm:"primary_key;autoIncrement:false"`
				Key  string `gorm:"type:varchar(64);unique_index"`
				Hash string

				CreatedAt time.Time
				ExpiresAt *time.Time

				TailnetID uint64
				Tailnet   Tailnet

				UserID uint64
				User   User
			}

			type AuthKey struct {
				ID        uint64 `gorm:"primary_key;autoIncrement:false"`
				Key       string `gorm:"type:varchar(64);unique_index"`
				Hash      string
				Ephemeral bool
				Tags      domain.Tags

				CreatedAt time.Time
				ExpiresAt *time.Time

				TailnetID uint64
				Tailnet   Tailnet

				UserID uint64
				User   User
			}

			type Machine struct {
				ID                uint64 `gorm:"primary_key;autoIncrement:false"`
				Name              string
				NameIdx           uint64
				MachineKey        string
				NodeKey           string
				DiscoKey          string
				Ephemeral         bool
				RegisteredTags    domain.Tags
				Tags              domain.Tags
				KeyExpiryDisabled bool

				HostInfo  domain.HostInfo
				Endpoints domain.Endpoints
				AllowIPs  domain.AllowIPs

				IPv4 domain.IP
				IPv6 domain.IP

				CreatedAt time.Time
				ExpiresAt time.Time
				LastSeen  *time.Time

				UserID uint64
				User   User

				TailnetID uint64
				Tailnet   Tailnet
			}

			type RegistrationRequest struct {
				MachineKey    string `gorm:"primary_key;autoIncrement:false"`
				Key           string `gorm:"type:varchar(64);unique_index"`
				Data          domain.RegistrationRequestData
				CreatedAt     time.Time
				Authenticated bool
				Error         string
			}

			type AuthenticationRequest struct {
				Key       string `gorm:"primary_key;autoIncrement:false"`
				Token     string
				TailnetID *uint64
				Error     string
				CreatedAt time.Time
			}

			return db.AutoMigrate(
				&ServerConfig{},
				&Tailnet{},
				&Account{},
				&User{},
				&SystemApiKey{},
				&ApiKey{},
				&AuthKey{},
				&Machine{},
				&RegistrationRequest{},
				&AuthenticationRequest{},
			)
		},
		Rollback: nil,
	}
}
