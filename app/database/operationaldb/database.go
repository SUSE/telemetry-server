package operationaldb

import (
	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry-server/app/database"
)

// Operational DB Tables
var operationalDbTables = database.DbTables{
	database.GetDbVersionTableSpec(),
	database.GetReportsStagingTableSpec(),
	database.GetClientsTableSpec(),
}

func GetTables() database.DbTables {
	return operationalDbTables
}

//
// Operational DB Migration
//

// list of migration versions in order from oldest to newest
var operationalDbMigrations = database.DbMigrations{
	&database.DbVersionMigration{
		Version:  "046c87f9-ec02-44a2-b2f5-57498eb35fff",
		Date:     "2025-06-02",
		Migrator: OperationalMigrator_20250602_046c87f9,
	},
}

func OperationalMigrator_20250602_046c87f9(adb *database.AppDb) (err error) {
	// dummy initial migration
	return
}

func GetMigrations() database.DbMigrations {
	return operationalDbMigrations
}

// Get a new Operational AppDb instance
func New(cfg *config.Config) (*database.AppDb, error) {
	return database.GetDb(
		"Operational",
		&cfg.DataBases.Operational,
		GetTables(),
		GetMigrations(),
	)
}
