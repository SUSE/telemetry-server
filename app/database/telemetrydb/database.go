package telemetrydb

import (
	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry-server/app/database"
)

// Telemetry DB Tables
var telemetryDbTables = database.DbTables{
	database.GetDbVersionTableSpec(),
	database.GetCustomersTableSpec(),
	database.GetTagSetsTableSpec(),
	database.GetTelemetryTableSpec(),
}

func GetTables() database.DbTables {
	return telemetryDbTables
}

//
// Telemetry DB Migration
//

func TelemetryMigrator_20250602_36a45576(adb *database.AppDb) (err error) {
	// dummy initial migration
	return
}

// list of migration versions in order from oldest to newest
var telemetryDbMigrations = database.DbMigrations{
	&database.DbVersionMigration{
		Version:  "36a45576-7bfc-44db-9ecd-ff5bf03dea40",
		Date:     "2025-06-02",
		Migrator: TelemetryMigrator_20250602_36a45576,
	},
}

func GetMigrations() database.DbMigrations {
	return telemetryDbMigrations
}

// Get a new Telemetry AppDb instance
func New(cfg *config.Config) (*database.AppDb, error) {
	return database.GetDb(
		"Telemetry",
		&cfg.DataBases.Telemetry,
		GetTables(),
		GetMigrations(),
	)
}
