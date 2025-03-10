package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/jsiebens/ionscale/internal/database/migration"
	"github.com/jsiebens/ionscale/internal/util"
	"go.uber.org/zap"
	"tailscale.com/types/key"
	"time"

	"github.com/jsiebens/ionscale/internal/config"
	"github.com/jsiebens/ionscale/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/prometheus"
)

type dbLock interface {
	Lock() error
	UnlockErr(error) error
}

func OpenDB(config *config.Database, logger *zap.Logger) (*sql.DB, domain.Repository, error) {
	db, lock, err := createDB(config, logger)
	if err != nil {
		return nil, nil, err
	}

	_ = db.Use(prometheus.New(prometheus.Config{StartServer: false}))

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime.Std())
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime.Std())

	repository := domain.NewRepository(db)

	if err := lock.Lock(); err != nil {
		return nil, nil, err
	}

	if err := lock.UnlockErr(migrate(db)); err != nil {
		return nil, nil, err
	}

	return sqlDB, repository, nil
}

func createDB(config *config.Database, logger *zap.Logger) (*gorm.DB, dbLock, error) {
	gormConfig := &gorm.Config{
		Logger: &GormLoggerAdapter{logger: logger.Sugar()},
	}

	switch config.Type {
	case "sqlite", "sqlite3":
		return newSqliteDB(config, gormConfig)
	case "postgres", "postgresql":
		return newPostgresDB(config, gormConfig)
	}

	return nil, nil, fmt.Errorf("invalid database type '%s'", config.Type)
}

func migrate(db *gorm.DB) error {
	m := gormigrate.New(db, gormigrate.DefaultOptions, migration.Migrations())

	if err := m.Migrate(); err != nil {
		return err
	}

	ctx := context.Background()
	repository := domain.NewRepository(db)

	if err := createServerKey(ctx, repository); err != nil {
		return err
	}

	if err := createJSONWebKeySet(ctx, repository); err != nil {
		return err
	}

	return nil
}

func createServerKey(ctx context.Context, repository domain.Repository) error {
	serverKey, err := repository.GetControlKeys(ctx)
	if err != nil {
		return err
	}
	if serverKey != nil {
		return nil
	}

	keys := domain.ControlKeys{
		ControlKey:       key.NewMachine(),
		LegacyControlKey: key.NewMachine(),
	}
	if err := repository.SetControlKeys(ctx, &keys); err != nil {
		return err
	}

	return nil
}

func createJSONWebKeySet(ctx context.Context, repository domain.Repository) error {
	jwks, err := repository.GetJSONWebKeySet(ctx)
	if err != nil {
		return err
	}
	if jwks != nil {
		return nil
	}

	privateKey, id, err := util.NewPrivateKey()
	if err != nil {
		return err
	}

	jsonWebKey := domain.JSONWebKey{Id: id, PrivateKey: *privateKey}

	if err := repository.SetJSONWebKeySet(ctx, &domain.JSONWebKeys{Key: jsonWebKey}); err != nil {
		return err
	}

	return nil
}

type GormLoggerAdapter struct {
	logger *zap.SugaredLogger
}

func (g *GormLoggerAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

func (g *GormLoggerAdapter) Info(ctx context.Context, s string, i ...interface{}) {
	g.logger.Infow(s, i)
}

func (g *GormLoggerAdapter) Warn(ctx context.Context, s string, i ...interface{}) {
	g.logger.Warnw(s, i)
}

func (g *GormLoggerAdapter) Error(ctx context.Context, s string, i ...interface{}) {
	g.logger.Error(s, i)
}

func (g *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if g.logger.Level().Enabled(zap.DebugLevel) {
		elapsed := time.Since(begin)
		switch {
		case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
			sql, rows := fc()
			if rows == -1 {
				g.logger.Debugw("Error executing query", "sql", sql, "start_time", begin.Format(time.RFC3339), "duration", elapsed, "err", err)
			} else {
				g.logger.Debugw("Error executing query", "sql", sql, "start_time", begin.Format(time.RFC3339), "duration", elapsed, "rows", rows, "err", err)
			}
		default:
			sql, rows := fc()
			if rows == -1 {
				g.logger.Debugw("Statement executed", "sql", sql, "start_time", begin.Format(time.RFC3339), "duration", elapsed)
			} else {
				g.logger.Debugw("Statement executed", "sql", sql, "start_time", begin.Format(time.RFC3339), "duration", elapsed, "rows", rows)
			}
		}
	}
}
