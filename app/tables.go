package app

// Operational DB Tables
var operationalTables = []TableSpec{
	reportsStagingTableSpec,
	clientsTableSpec,
}

// Telemetry DB Tables
var telemetryTables = []TableSpec{
	customersTableSpec,
	tagSetsTableSpec,
	telemetryTableSpec,
}
