package database

import (
	"context"
	"fmt"
	"hash/crc32"

	"github.com/hashicorp/go-multierror"
	"github.com/jsiebens/ionscale/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newPostgresDB(config *config.Database, g *gorm.Config) (*gorm.DB, dbLock, error) {
	db, err := gorm.Open(postgres.Open(config.Url), g)
	if err != nil {
		return nil, nil, err
	}

	return db, &pgLock{db: db}, nil
}

type pgLock struct {
	db *gorm.DB
}

func (s *pgLock) Lock() error {
	d, _ := s.db.DB()

	query := `SELECT pg_advisory_lock($1)`
	id := s.generateAdvisoryLockId()
	if _, err := d.ExecContext(context.Background(), query, id); err != nil {
		return err
	}

	return nil
}

func (s *pgLock) UnlockErr(prevErr error) error {
	if err := s.unlock(); err != nil {
		return multierror.Append(prevErr, err)
	}
	return prevErr
}

func (s *pgLock) unlock() error {
	d, _ := s.db.DB()

	query := `SELECT pg_advisory_unlock($1)`
	if _, err := d.ExecContext(context.Background(), query, s.generateAdvisoryLockId()); err != nil {
		return err
	}

	return nil
}

const advisoryLockIDSalt uint = 1486364155

func (s *pgLock) generateAdvisoryLockId() string {
	sum := crc32.ChecksumIEEE([]byte("ionscale_migration"))
	sum = sum * uint32(advisoryLockIDSalt)
	return fmt.Sprint(sum)
}
