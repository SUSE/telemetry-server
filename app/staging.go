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

	reportStagingRow := new(ReportStagingTableRow)
	if err = reportStagingRow.SetupDB(&a.StagingDB); err != nil {
		slog.Error("ReportStagingTableRow.SetupDB failed", slog.String("error", err.Error()))
		return
	}

	reportStagingRow.Init(
		fmt.Sprintf("%d", rHeader.ReportClientId),
		rHeader.ReportId,
		reqBody,
	)

	if err = reportStagingRow.Insert(); err != nil {
		slog.Error("staged report insert failed", slog.String("report", reportStagingRow.ReportIdentifer()), slog.String("error", err.Error()))
	}

	return
}

func (a *App) ProcessStagedReports() {
	reportRow := new(ReportStagingTableRow)
	reportRow.SetupDB(&a.StagingDB)

	for reportRow.FirstUnallocated() {
		err := a.ProcessStagedReport(reportRow)
		if err != nil {
			slog.Error("report processing failed", slog.String("error", err.Error()))
			continue
		}
		err = reportRow.Delete()
		if err != nil {
			slog.Error("delete of processed report failed", slog.String("error", err.Error()))
		}
	}
}

func (a *App) ProcessStagedReport(reportRow *ReportStagingTableRow) (err error) {
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

var reportsStagingTableSpec = TableSpec{
	Name: "reports",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "clientId", Type: "INTEGER"},
		{Name: "reportId", Type: "VARCHAR"},
		{Name: "data", Type: "TEXT"},
		{Name: "receivedAt", Type: "VARCHAR"},
		{Name: "allocated", Type: "BOOLEAN", Default: "false"},
		{Name: "allocatedAt", Type: "VARCHAR", Nullable: true},
	},
}

type ReportStagingTableRow struct {
	TableRowCommon

	Id          int64  `json:"id"`
	ClientId    string `json:"clientId"`
	ReportId    string `json:"reportId"`
	Data        any    `json:"data"`
	ReceivedAt  string `json:"receivedAt"`
	Allocated   bool   `json:"allocated"`
	AllocatedAt string `json:"allocatedAt"`
}

func (r *ReportStagingTableRow) Init(clientId, reportId string, data any) {
	r.ClientId = clientId
	r.ReportId = reportId
	r.Data = data
	r.ReceivedAt = types.Now().String()
}

func (r *ReportStagingTableRow) SetupDB(db *DbConnection) error {
	r.tableSpec = &reportsStagingTableSpec
	return r.TableRowCommon.SetupDB(db)
}

func (r *ReportStagingTableRow) ReportIdentifer() string {
	return fmt.Sprintf("reportId: %v, clientId: %v, receivedAt: %v", r.ReportId, r.ClientId, r.ReceivedAt)
}

func (r *ReportStagingTableRow) Exists() bool {
	stmt, err := r.SelectStmt(
		[]string{
			"id",
		},
		[]string{
			"clientId",
			"reportId",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"exists statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := r.DB().QueryRow(
		stmt,
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

func (r *ReportStagingTableRow) FirstUnallocated() bool {
	queryStmt, err := r.SelectStmt(
		[]string{
			"id",
			"clientId",
			"reportId",
			"data",
			"receivedAt",
		},
		[]string{
			"allocated",
		},
		SelectOpts{
			Limit: 1,
		},
	)
	if err != nil {
		slog.Error(
			"query statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	updateStmt, err := r.UpdateStmt(
		[]string{
			"allocated",
			"allocatedAt",
		},
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"update statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	// begin a transaction
	TX, err := r.DB().Begin()
	if err != nil {
		slog.Error("transaction begin failed", slog.String("error", err.Error()))
		return false
	}

	// retrieve the first unallocated report from the table, returning false if none was found
	row := TX.QueryRow(
		queryStmt,
		false,
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

	_, err = TX.Exec(
		updateStmt,
		r.Allocated,
		r.AllocatedAt,
		r.Id,
	)
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

func (r *ReportStagingTableRow) Insert() (err error) {
	stmt, err := r.InsertStmt(
		[]string{
			"clientId",
			"reportId",
			"data",
			"receivedAt",
		},
		"id",
	)
	if err != nil {
		slog.Error(
			"insert statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	row := r.DB().QueryRow(
		stmt,
		r.ClientId,
		r.ReportId,
		r.Data,
		r.ReceivedAt,
	)
	if err = row.Scan(
		&r.Id,
	); err != nil {
		slog.Error(
			"report insert failed",
			slog.String("report", r.ReportIdentifer()),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (r *ReportStagingTableRow) Delete() (err error) {
	stmt, err := r.DeleteStmt(
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"delete statement generation failed",
			slog.String("table", r.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	_, err = r.DB().Exec(stmt, r.Id)
	if err != nil {
		slog.Error("report delete failed", slog.String("report", r.ReportIdentifer()), slog.String("error", err.Error()))
		return err
	}

	return
}
