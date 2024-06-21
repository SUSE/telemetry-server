package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
	"github.com/SUSE/telemetry/pkg/types"
)

func (a *App) StageTelemetryReport(reqBody []byte, rHeader *telemetrylib.TelemetryReportHeader) (err error) {
	// Stores the report body into the staging database in the reports table

	// create a ReportStagingTableRow struct
	reportStagingRow := ReportStagingTableRow{
		ClientId:   fmt.Sprintf("%d", rHeader.ReportClientId),
		ReportId:   rHeader.ReportId,
		Data:       reqBody,
		ReceivedAt: types.Now().String(),
	}

	if err := reportStagingRow.Insert(a.StagingDB.Conn); err != nil {
		slog.Error("staged report insert failed", slog.String("report", reportStagingRow.ReportIdentifer()), slog.String("error", err.Error()))
		return err
	}

	return
}

func (a *App) ProcessStagedReports() {
	var reportRow = ReportStagingTableRow{}

	for reportRow.FirstUnallocated(a.StagingDB.Conn) {
		err := a.ProcessStagedReport(&reportRow)
		if err != nil {
			slog.Error("report processing failed", slog.String("error", err.Error()))
			continue
		}
		err = reportRow.Delete(a.StagingDB.Conn)
		if err != nil {
			slog.Error("delete of processed report failed", slog.String("error", err.Error()))
		}
	}
}

func (a *App) ProcessStagedReport(reportRow *ReportStagingTableRow) (err error) {
	slog.Info("Processing", slog.String("report", reportRow.ReportIdentifer()))

	var report telemetrylib.TelemetryReport

	err = json.Unmarshal(reportRow.Data.([]byte), &report)
	if err != nil {
		slog.Error("data unmarshal failed", slog.String("report", reportRow.ReportIdentifer()), slog.String("error", err.Error()))
		return
	}

	// process available bundles, extracting the data items and
	// storing them in the telemetry DB
	for _, bundle := range report.TelemetryBundles {
		bKey := bundle.Header.BundleId
		slog.Info("Processing", slog.String("bundleId", bKey))

		// for each data item in the bundle, process it
		for _, item := range bundle.TelemetryDataItems {
			if err := a.StoreTelemetry(&item, &bundle.Header); err != nil {
				slog.Error("failed to store telemetry data from bundle %q: %s", bKey, err.Error())
				return err
			}
		}
	}

	return
}

const reportsTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	clientId INTEGER NOT NULL,
	reportId VARCHAR(64) NOT NULL,
	data BLOB NOT NULL,
	receivedAt VARCHAR(32) NOT NULL,
	allocated BOOLEAN DEFAULT false NOT NULL,
	allocatedAt VARCHAR(32) NULL
)`

type ReportStagingTableRow struct {
	Id          int64  `json:"id"`
	ClientId    string `json:"clientId"`
	ReportId    string `json:"reportId"`
	Data        any    `json:"data"`
	ReceivedAt  string `json:"receivedAt"`
	Allocated   bool   `json:"allocated"`
	AllocatedAt string `json:"allocatedAt"`
}

func (r *ReportStagingTableRow) ReportIdentifer() string {
	return fmt.Sprintf("reportId: %v, clientId: %v, receivedAt: %v", r.ReportId, r.ClientId, r.ReceivedAt)
}

func (r *ReportStagingTableRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(
		`SELECT id FROM reports WHERE clientId = ? AND reportId = ?`,
		r.ClientId,
		r.ReportId,
	)
	if err := row.Scan(&r.Id); err != nil {
		if err != sql.ErrNoRows {
			slog.Error("row.Scan() failed for existence of report id %q: %s", r.ReportId, err.Error())
		}
		return false
	}
	return true
}

func (r *ReportStagingTableRow) FirstUnallocated(DB *sql.DB) bool {
	// begin a transaction
	TX, err := DB.Begin()
	if err != nil {
		slog.Error("transaction begin failed", slog.String("error", err.Error()))
		return false
	}

	// retrieve the first unallocated report from the table, returning false if none was found
	row := TX.QueryRow(
		`SELECT id, clientId, reportId, data, receivedAt FROM reports WHERE allocated = false LIMIT 1`,
	)
	if err := row.Scan(&r.Id, &r.ClientId, &r.ReportId, &r.Data, &r.ReceivedAt); err != nil {
		if err == sql.ErrNoRows {
			slog.Info("no unallocated staged report rows found")
		} else {
			slog.Error("unallocated staged report retrieval failed", slog.String("error", err.Error()))
		}

		if err := TX.Rollback(); err != nil {
			slog.Error("empty transaction rollback failed", slog.String("error", err.Error()))
		}

		return false
	}

	slog.Info("unallocated report found", slog.Int64("id", r.Id), slog.String("report", r.ReportIdentifer()))

	// set AllocatedAt to Now, allows for detection of report processing that got lost
	r.Allocated = true
	r.AllocatedAt = types.Now().String()

	_, err = TX.Exec(`UPDATE reports SET allocated = ?, allocatedAt = ? WHERE id = ?`, r.Allocated, r.AllocatedAt, r.Id)
	if err != nil {
		slog.Error("staged report update failed", slog.Int64("id", r.Id), slog.String("error", err.Error()))

		if err := TX.Rollback(); err != nil {
			slog.Error("update rollback failed", slog.String("error", err.Error()))
		}

		return false
	}

	if err := TX.Commit(); err != nil {
		slog.Error("staged report transaction commit failed", slog.Int64("id", r.Id), slog.String("error", err.Error()))

		if err := TX.Rollback(); err != nil {
			slog.Error("failed to rollback update transaction: %s", slog.String("error", err.Error()))
		}

		return false
	}

	return true
}

func (r *ReportStagingTableRow) Insert(DB *sql.DB) (err error) {
	res, err := DB.Exec(
		`INSERT INTO reports(clientId, reportId, data, receivedAt) VALUES(?, ?, ?, ?)`,
		r.ClientId, r.ReportId, r.Data, r.ReceivedAt,
	)
	if err != nil {
		slog.Error("report insert failed", slog.String("report", r.ReportIdentifer()), slog.String("error", err.Error()))
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("report insertion id retrieval failed", slog.String("report", r.ReportIdentifer()), slog.String("error", err.Error()))
		return
	}
	r.Id = id

	return
}

func (r *ReportStagingTableRow) Delete(DB *sql.DB) (err error) {
	_, err = DB.Exec("DELETE FROM reports WHERE id = ?", r.Id)
	if err != nil {
		slog.Error("report delete failed", slog.String("report", r.ReportIdentifer()), slog.String("error", err.Error()))
		return err
	}

	return
}
