package database

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry/pkg/types"
)

var reportsStagingTableSpec = TableSpec{
	Name: "reports",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "clientId", Type: "VARCHAR"},
		{Name: "reportId", Type: "VARCHAR"},
		{Name: "data", Type: "TEXT"},
		{Name: "receivedAt", Type: "VARCHAR"},
		{Name: "allocated", Type: "BOOLEAN", Default: "false"},
		{Name: "allocatedAt", Type: "VARCHAR", Nullable: true},
	},
}

func GetReportsStagingTableSpec() *TableSpec {
	return &reportsStagingTableSpec
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

func (r *ReportStagingTableRow) SetupDB(adb *AppDb, tx *sql.Tx) {
	r.SetTableSpec(GetReportsStagingTableSpec())
	r.TableRowCommon.SetupDB(adb, tx)
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

	row := r.Tx().QueryRow(
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

	// retrieve the first unallocated report from the table, returning false if none was found
	row := r.Tx().QueryRow(
		queryStmt,
		false,
	)
	if err := row.Scan(&r.Id, &r.ClientId, &r.ReportId, &r.Data, &r.ReceivedAt); err != nil {
		if err == sql.ErrNoRows {
			slog.Info("no unallocated staged report rows found")
		} else {
			slog.Error("unallocated staged report retrieval failed", slog.String("error", err.Error()))
		}

		return false
	}

	slog.Info("unallocated report found", slog.Int64("id", r.Id), slog.String("report", r.ReportIdentifer()))

	// set AllocatedAt to Now, allows for detection of report processing that got lost
	r.Allocated = true
	r.AllocatedAt = types.Now().String()

	_, err = r.Tx().Exec(
		updateStmt,
		r.Allocated,
		r.AllocatedAt,
		r.Id,
	)
	if err != nil {
		slog.Error("staged report update failed", slog.Int64("id", r.Id), slog.String("error", err.Error()))

		return false
	}

	return true
}

func (r *ReportStagingTableRow) Insert() (stagingId int64, err error) {
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

	row := r.Tx().QueryRow(
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
		return
	}

	stagingId = r.Id

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

	_, err = r.Tx().Exec(stmt, r.Id)
	if err != nil {
		slog.Error("report delete failed", slog.String("report", r.ReportIdentifer()), slog.String("error", err.Error()))
		return err
	}

	return
}
