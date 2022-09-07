package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
)

func Migrations() []*gormigrate.Migration {
	var migrations = []*gormigrate.Migration{
		m202209070900_initial_schema(),
	}
	return migrations
}
