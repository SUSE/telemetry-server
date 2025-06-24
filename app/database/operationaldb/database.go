package operationaldb

import (
	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry-server/app/database"
)

// Operational DB Tables
var operationalDbTables = database.DbTables{
	database.GetReportsStagingTableSpec(),
	database.GetClientsTableSpec(),
}

func GetTables() database.DbTables {
	return operationalDbTables
}

// Get a new Operational AppDb instance
func New(cfg *config.Config) (*database.AppDb, error) {
	return database.GetDb(
		"Operational",
		&cfg.DataBases.Operational,
		GetTables(),
	)
}
