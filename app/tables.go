package app

// Staging DB Tables
var stagingTables = []TableSpec{
	reportsStagingTableSpec,
}

// Operational DB Tables
var operationalTables = []TableSpec{
	clientsTableSpec,
}

// Telemetry DB Tables
var telemetryTables = []TableSpec{
	tagSetsTableSpec,
	telemetryTableSpec,
}
