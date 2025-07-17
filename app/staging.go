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
	//
	// create an operationalDb transaction
	//
	odbTx, err := a.OperationalDB.StartTx()
	if err != nil {
		slog.Error(
			"Failed to start a tranaction to stage a telemetry report",
			slog.String("reportId", rHeader.ReportId),
			slog.String("error", err.Error()),
		)
		return
	}

	// defer a rollback of the operationalDb transaction
	defer func() {
		a.OperationalDB.RollbackTx(odbTx, "AuthenticateClient")
	}()

	//
	// Store the report body in the operational database's reports table
	//

	// create a ReportStagingTableRow struct
	reportStagingRow := new(database.ReportStagingTableRow)
	reportStagingRow.SetupDB(a.OperationalDB, odbTx)
	reportStagingRow.Init(
		rHeader.ReportClientId,
		rHeader.ReportId,
		reqBody,
	)

	stagingId, err = reportStagingRow.Insert()
	if err != nil {
		slog.Error(
			"staged report insert failed",
			slog.String("report", reportStagingRow.ReportIdentifer()),
			slog.String("error", err.Error()),
		)
		return
	}

	// commit the transaction
	if err = a.OperationalDB.CommitTx(odbTx); err != nil {
		return 0, err
	}

	return
}

func (a *App) ProcessStagedReports() error {
	var errs []error
	reportRow := new(database.ReportStagingTableRow)

	//
	// create an operationalDb transaction
	//
	odbTx, err := a.OperationalDB.StartTx()
	if err != nil {
		slog.Error(
			"Failed to start a tranaction to process staged telemetry reports",
			slog.String("error", err.Error()),
		)
		errs = append(errs, fmt.Errorf("staged report transaction start failed: %w", err))
		goto join_errors
	}

	// defer a rollback of the operationalDb transaction
	defer func() {
		a.OperationalDB.RollbackTx(odbTx, "AuthenticateClient")
	}()

	reportRow.SetupDB(a.OperationalDB, odbTx)

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

	// commit the transaction
	if err = a.OperationalDB.CommitTx(odbTx); err != nil {
		slog.Error(
			"Failed to commit a tranaction after processing staged telemetry reports",
			slog.String("error", err.Error()),
		)
		errs = append(errs, fmt.Errorf("staged report transaction commit failed: %w", err))
	}

join_errors:
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
