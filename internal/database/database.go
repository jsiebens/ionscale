package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/hashicorp/go-hclog"
	"github.com/jsiebens/ionscale/internal/broker"
	"github.com/jsiebens/ionscale/internal/database/migration"
	"time"

	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type db interface {
	DB() *gorm.DB
	Lock() error
	Unlock() error
	UnlockErr(error) error
}

func OpenDB(config *config.Database, logger hclog.Logger) (domain.Repository, broker.Pubsub, error) {
	db, pubsub, err := createDB(config, logger)
	if err != nil {
		return nil, nil, err
	}

	repository := domain.NewRepository(db.DB())

	if err := db.Lock(); err != nil {
		return nil, nil, err
	}

	if err := db.UnlockErr(migrate(db.DB())); err != nil {
		return nil, nil, err
	}

	return repository, pubsub, nil
}

func createDB(config *config.Database, logger hclog.Logger) (db, broker.Pubsub, error) {
	gormConfig := &gorm.Config{
		Logger: &GormLoggerAdapter{logger: logger.Named("db")},
	}

	switch config.Type {
	case "sqlite", "sqlite3":
		db, err := newSqliteDB(config, gormConfig)
		return db, broker.NewPubsubInMemory(), err
	case "postgres", "postgresql":
		db, err := newPostgresDB(config, gormConfig)
		if err != nil {
			return nil, nil, err
		}
		stdDB, err := db.DB().DB()
		if err != nil {
			return nil, nil, err
		}
		pubsub, err := broker.NewPubsub(context.TODO(), stdDB, config.Url)
		if err != nil {
			return nil, nil, err
		}
		return db, pubsub, err
	}

	return nil, nil, fmt.Errorf("invalid database type '%s'", config.Type)
}

func migrate(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, migration.Migrations())

	if err := m.Migrate(); err != nil {
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
