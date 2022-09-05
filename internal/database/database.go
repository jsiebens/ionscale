package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/hashicorp/go-hclog"
	"github.com/jsiebens/ionscale/internal/broker"
	"gorm.io/driver/postgres"
	"net/http"
	"tailscale.com/tailcfg"
	"time"

	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func OpenDB(config *config.Database, logger hclog.Logger) (*gorm.DB, domain.Repository, broker.Pubsub, error) {
	gormDB, pubsub, err := createDB(config, logger)
	if err != nil {
		return nil, nil, nil, err
	}

	repository := domain.NewRepository(gormDB)

	if err := migrate(gormDB, repository); err != nil {
		return nil, nil, nil, err
	}

	return gormDB, repository, pubsub, nil
}

func createDB(config *config.Database, logger hclog.Logger) (*gorm.DB, broker.Pubsub, error) {
	gormConfig := &gorm.Config{
		Logger: &GormLoggerAdapter{logger: logger.Named("db")},
	}

	switch config.Type {
	case "sqlite", "sqlite3":
		db, err := gorm.Open(sqlite.Open(config.Url), gormConfig)
		return db, broker.NewPubsubInMemory(), err
	case "postgres", "postgresql":
		gormDB, err := gorm.Open(postgres.Open(config.Url), gormConfig)
		if err != nil {
			return nil, nil, err
		}
		db, err := gormDB.DB()
		if err != nil {
			return nil, nil, err
		}
		pubsub, err := broker.NewPubsub(context.Background(), db, config.Url)
		if err != nil {
			return nil, nil, err
		}
		return gormDB, pubsub, err
	}

	return nil, nil, fmt.Errorf("invalid database type '%s'", config.Type)
}

func migrate(db *gorm.DB, repository domain.Repository) error {
	err := db.AutoMigrate(
		&domain.ServerConfig{},
		&domain.Tailnet{},
		&domain.Account{},
		&domain.User{},
		&domain.SystemApiKey{},
		&domain.ApiKey{},
		&domain.AuthKey{},
		&domain.Machine{},
		&domain.RegistrationRequest{},
		&domain.AuthenticationRequest{},
	)

	if err != nil {
		return err
	}

	if err := initializeDERPMap(repository); err != nil {
		return err
	}

	return nil
}

func initializeDERPMap(repository domain.Repository) error {
	ctx := context.Background()
	derpMap, err := repository.GetDERPMap(ctx)
	if err != nil {
		return err
	}
	if derpMap != nil {
		return nil
	}

	getJson := func(url string, target interface{}) error {
		c := http.Client{Timeout: 5 * time.Second}
		r, err := c.Get(url)
		if err != nil {
			return err
		}
		defer r.Body.Close()

		return json.NewDecoder(r.Body).Decode(target)
	}

	m := &tailcfg.DERPMap{}
	if err := getJson("https://controlplane.tailscale.com/derpmap/default", m); err != nil {
		return err
	}

	if err := repository.SetDERPMap(ctx, m); err != nil {
		return err
	}

	return nil
}

type GormLoggerAdapter struct {
	logger hclog.Logger
}

func (g *GormLoggerAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

func (g *GormLoggerAdapter) Info(ctx context.Context, s string, i ...interface{}) {
	g.logger.Info(s, i)
}

func (g *GormLoggerAdapter) Warn(ctx context.Context, s string, i ...interface{}) {
	g.logger.Warn(s, i)
}

func (g *GormLoggerAdapter) Error(ctx context.Context, s string, i ...interface{}) {
	g.logger.Error(s, i)
}

func (g *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		sql, rows := fc()
		if rows == -1 {
			g.logger.Error("Error executing query", "sql", sql, "start_time", begin.Format(time.RFC3339), "duration", elapsed, "err", err)
		} else {
			g.logger.Error("Error executing query", "sql", sql, "start_time", begin.Format(time.RFC3339), "duration", elapsed, "rows", rows, "err", err)
		}
	case g.logger.IsTrace():
		sql, rows := fc()
		if rows == -1 {
			g.logger.Trace("Statement executed", "sql", sql, "start_time", begin.Format(time.RFC3339), "duration", elapsed)
		} else {
			g.logger.Trace("Statement executed", "sql", sql, "start_time", begin.Format(time.RFC3339), "duration", elapsed, "rows", rows)
		}
	}
}
