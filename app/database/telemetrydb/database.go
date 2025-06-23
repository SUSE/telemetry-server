package telemetrydb

import (
	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry-server/app/database"
)

// Telemetry DB Tables
var telemetryDbTables = database.DbTables{
	database.GetCustomersTableSpec(),
	database.GetTagSetsTableSpec(),
	database.GetTelemetryTableSpec(),
}

func GetTables() database.DbTables {
	return telemetryDbTables
}

// Get a new Telemetry AppDb instance
func New(cfg *config.Config) (*database.AppDb, error) {
	return database.GetDb(
		"Telemetry",
		&cfg.DataBases.Telemetry,
		GetTables(),
	)
}
