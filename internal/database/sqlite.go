package database

import (
	"github.com/glebarez/sqlite"
	"github.com/jsiebens/ionscale/internal/config"
	"gorm.io/gorm"
)

func newSqliteDB(config *config.Database, g *gorm.Config) (db, error) {
	db, err := gorm.Open(sqlite.Open(config.Url), g)
	if err != nil {
		return nil, err
	}

	return &Sqlite{
		db: db,
	}, nil
}

type Sqlite struct {
	db *gorm.DB
}

func (s *Sqlite) DB() *gorm.DB {
	return s.db
}

func (s *Sqlite) Lock() error {
	return nil
}

func (s *Sqlite) Unlock() error {
	return nil
}

func (s *Sqlite) UnlockErr(prevErr error) error {
	return prevErr
}
