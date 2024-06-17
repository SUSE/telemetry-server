package app

// Telemetry DB Tables
var dbTablesTelemetry = map[string]string{
	"clients": clientsTableColumns,
	"tagSets": tagSetsTableColumns,

	// default telemetry storage table
	"telemetryData": defaultTelemetryTableColumns,
}

// telemetry type specific transform tables
var dbTablesXform = map[string]string{
	"telemetrySccHwInfo": sccHwInfoTelemetryTableColumns,
}

// Staging DB Tables
var dbTablesStaging = map[string]string{
	"reports": reportsTableColumns,
}
