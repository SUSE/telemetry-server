package app

// Telemetry DB Tables
var dbTablesTelemetry = map[string]string{
	// tag set storage table
	"tagSets": tagSetsTableColumns,

	// default telemetry storage table
	"telemetryData": defaultTelemetryTableColumns,
}

// Staging DB Tables
var dbTablesStaging = map[string]string{
	"reports": reportsTableColumns,
}

var operationalTables = []TableSpec{
	clientsTableSpec,
}

// telemetry type specific transform tables
var dbTablesXform = map[string]string{
	"telemetrySccHwInfo": sccHwInfoTelemetryTableColumns,
}
