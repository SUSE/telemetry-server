package app

import (
	"database/sql"
	"encoding/json"
	"log/slog"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

var defaultTelemetryTableSpec = TableSpec{
	Name: "telemetryData",
	Columns: []TableSpecColumn{
		// common fields
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "clientId", Type: "INTEGER"},
		{Name: "customerId", Type: "INTEGER"},
		{Name: "telemetryId", Type: "VARCHAR"},
		{Name: "telemetryType", Type: "VARCHAR"},
		{Name: "tagSetId", Type: "INTEGER", Nullable: true},
		{Name: "timestamp", Type: "VARCHAR"},

		// table specific fields
		{Name: "dataItem", Type: "TEXT"},
	},
}

type DefaultTelemetryDataRow struct {
	// Embed the common rows
	TelemetryDataCommon

	DataItem []byte `json:"dataItem"`
}

func (t *DefaultTelemetryDataRow) Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) (err error) {
	t.TelemetryDataCommon.Init(dItm, bHdr, tagSetId)
	t.DataItem = []byte(dItm.TelemetryData)

	return
}

func (t *DefaultTelemetryDataRow) SetupDB(db *DbConnection) error {
	t.tableSpec = &defaultTelemetryTableSpec
	return t.TelemetryDataCommon.SetupDB(db)
}

func (t *DefaultTelemetryDataRow) TableName() string {
	return t.TableRowCommon.TableName()
}

func (t *DefaultTelemetryDataRow) RowId() int64 {
	return t.Id
}

func (t *DefaultTelemetryDataRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *DefaultTelemetryDataRow) Exists() bool {

	stmt, err := t.SelectStmt(
		// select columns
		[]string{
			"id",
			"customerId",
			"telemetryType",
			"tagSetId",
			"dataItem",
		},
		// match columns
		[]string{
			"clientId",
			"telemetryId",
			"timestamp",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"exists statement generation failed",
			slog.String("table", t.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := t.DB().QueryRow(
		stmt,
		t.ClientId,
		t.TelemetryId,
		t.Timestamp,
	)
	// if the entry was found, all fields not used to find the entry will have
	// been updated to match what is in the DB
	if err := row.Scan(
		&t.Id,
		&t.CustomerId,
		&t.TelemetryType,
		&t.TagSetId,
		&t.DataItem,
	); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"check for matching entry failed",
				slog.String("table", t.TableName()),
				slog.Int64("clientId", t.ClientId),
				slog.String("telemetryId", t.TelemetryId),
				slog.String("timestamp", t.Timestamp),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (t *DefaultTelemetryDataRow) Insert() (err error) {
	stmt, err := t.InsertStmt(
		[]string{
			"clientId",
			"customerId",
			"telemetryId",
			"telemetryType",
			"timestamp",
			"tagSetId",
			"dataItem",
		},
		"id",
	)
	if err != nil {
		slog.Error(
			"insert statement generation failed",
			slog.String("table", t.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	row := t.DB().QueryRow(
		stmt,
		t.ClientId,
		t.CustomerId,
		t.TelemetryId,
		t.TelemetryType,
		t.Timestamp,
		t.TagSetId,
		t.DataItem,
	)
	if err = row.Scan(
		&t.Id,
	); err != nil {
		slog.Error(
			"insert failed",
			slog.String("table", t.TableName()),
			slog.Int64("clientId", t.ClientId),
			slog.String("telemetryId", t.TelemetryId),
			slog.String("timestamp", t.Timestamp),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (t *DefaultTelemetryDataRow) Update() (err error) {
	stmt, err := t.UpdateStmt(
		[]string{
			"clientId",
			"customerId",
			"telemetryId",
			"telemetryType",
			"timestamp",
			"tagSetId",
			"dataItem",
		},
		[]string{
			"Id",
		},
	)
	if err != nil {
		slog.Error(
			"update statement generation failed",
			slog.String("table", t.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	_, err = t.DB().Exec(
		stmt,
		t.ClientId,
		t.CustomerId,
		t.TelemetryId,
		t.TelemetryType,
		t.Timestamp,
		t.TagSetId,
		t.DataItem,
		t.Id,
	)
	if err != nil {
		slog.Error(
			"update failed",
			slog.String("table", t.TableName()),
			slog.Int64("id", t.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

func (t *DefaultTelemetryDataRow) Delete() (err error) {
	stmt, err := t.DeleteStmt(
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"delete statement generation failed",
			slog.String("table", t.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	_, err = t.DB().Exec(
		stmt,
		t.Id,
	)
	if err != nil {
		slog.Error(
			"delete failed",
			slog.String("table", t.TableName()),
			slog.Int64("id", t.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

// validate that DefaultTelemetryDataRow implements TelemetryDataRow interface
var _ TelemetryDataRowHandler = (*DefaultTelemetryDataRow)(nil)
