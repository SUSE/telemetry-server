package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry-server/app/database"
	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

func (a *App) StageTelemetryReport(reqBody []byte, rHeader *telemetrylib.TelemetryReportHeader) (stagingId int64, err error) {
	// Stores the report body in the operational database's reports table

	// create a ReportStagingTableRow struct

	reportStagingRow := new(database.ReportStagingTableRow)
	if err = reportStagingRow.SetupDB(a.OperationalDB); err != nil {
		slog.Error("ReportStagingTableRow.SetupDB failed", slog.String("error", err.Error()))
		return
	}

	reportStagingRow.Init(
		rHeader.ReportClientId,
		rHeader.ReportId,
		reqBody,
	)

	stagingId, err = reportStagingRow.Insert()
	if err != nil {
		slog.Error("staged report insert failed", slog.String("report", reportStagingRow.ReportIdentifer()), slog.String("error", err.Error()))
	}

	return
}

func (a *App) ProcessStagedReports() error {
	var errs []error

	reportRow := new(database.ReportStagingTableRow)
	reportRow.SetupDB(a.OperationalDB)

	for reportRow.FirstUnallocated() {
		err := a.ProcessStagedReport(reportRow)
		if err != nil {
			slog.Error(
				"report processing failed",
				slog.Int64("id", reportRow.Id),
				slog.String("reportId", reportRow.ReportId),
				slog.String("error", err.Error()),
			)
			errs = append(errs, fmt.Errorf("staged report processing failed: %w", err))
			continue
		}
		err = reportRow.Delete()
		if err != nil {
			slog.Error(
				"delete of processed report failed",
				slog.Int64("id", reportRow.Id),
				slog.String("reportId", reportRow.ReportId),
				slog.String("error", err.Error()),
			)
			errs = append(errs, fmt.Errorf("staged report deletion failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (a *App) ProcessStagedReport(reportRow *database.ReportStagingTableRow) (err error) {
	slog.Info("Processing", slog.String("report", reportRow.ReportIdentifer()))

	var report telemetrylib.TelemetryReport
	var reportData []byte

	switch t := reportRow.Data.(type) {
	case []byte: // sqlite3
		reportData = reportRow.Data.([]byte)
	case string: // postgresql
		reportData = []byte(reportRow.Data.(string))
	default:
		err = fmt.Errorf("unsupported type: %T", t)
		slog.Error(
			"reportRow.Data type unmatched",
			slog.String("error", err.Error()),
		)
		return
	}

	err = json.Unmarshal(reportData, &report)
	if err != nil {
		slog.Error("data unmarshal failed", slog.String("report", reportRow.ReportIdentifer()), slog.String("error", err.Error()))
		return
	}

	err = a.ProcessTelemetryReport(&report)
	if err != nil {
		slog.Error(
			"Failed to process telemetry report",
		)
	}

	return
}
