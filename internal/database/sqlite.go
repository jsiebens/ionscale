package database

import (
	"github.com/glebarez/sqlite"
	"github.com/jsiebens/ionscale/internal/config"
	"gorm.io/gorm"
)

func newSqliteDB(config *config.Database, g *gorm.Config) (*gorm.DB, dbLock, error) {
	db, err := gorm.Open(sqlite.Open(config.Url), g)
	if err != nil {
		return nil, nil, err
	}
	return db, &sqliteLock{}, nil
}

type sqliteLock struct {
}

func (s *sqliteLock) Lock() error {
	return nil
}

func (s *sqliteLock) UnlockErr(prevErr error) error {
	return prevErr
}
