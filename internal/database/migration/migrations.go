package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
)

func Migrations() []*gormigrate.Migration {
	var migrations = []*gormigrate.Migration{
		m202209070900_initial_schema(),
		m202209251530_add_autoallowips_column(),
		m202209251532_add_alias_column(),
		m202229251530_add_alias_column_constraint(),
		m202210040828_add_derpmap_colum(),
		m202210070814_add_filesharing_and_servicecollection_columns(),
		m202210080700_ssh_action_request(),
		m202211031100_add_authorized_column(),
		m202212201300_add_user_id_column(),
	}
	return migrations
}
