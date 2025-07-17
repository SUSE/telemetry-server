package database

import (
	"database/sql"
	"encoding/json"
	"log/slog"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

var telemetryTableSpec = TableSpec{
	Name: "telemetryData",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "clientId", Type: "VARCHAR"},
		{Name: "customerRefId", Type: "INTEGER"},
		{Name: "telemetryId", Type: "VARCHAR"},
		{Name: "telemetryType", Type: "VARCHAR"},
		{Name: "tagSetId", Type: "INTEGER", Nullable: true},
		{Name: "timestamp", Type: "VARCHAR"},
		{Name: "dataItem", Type: "TEXT"},
	},
	ForeignKeys: []TableSpecForeignKey{
		{Column: "tagSetId", ReferencedTable: "tagSets", ReferencedColumn: "id"},
		{Column: "customerRefId", ReferencedTable: "customers", ReferencedColumn: "id"},
	},
}

func GetTelemetryTableSpec() *TableSpec {
	return &telemetryTableSpec
}

type TelemetryDataRow struct {
	TableRowCommon

	// public table fields
	Id            int64  `json:"id"`
	ClientId      string `json:"clientId"`
	CustomerRefId int64  `json:"customerRefId"`
	TelemetryId   string `json:"telemetryId"`
	TelemetryType string `json:"telemetryType"`
	Timestamp     string `json:"timestamp"`
	TagSetId      int64  `json:"tagSetId"`
	DataItem      []byte `json:"dataItem"`
}

type TelemetryDataRowHandler interface {
	// TelemetryDataRow is a superset of TableRow
	TableRowHandler

	// Initialise the row fields
	Init(
		dItm *telemetrylib.TelemetryDataItem,
		bHdr *telemetrylib.TelemetryBundleHeader,
		tagSetId int64,
		customerRefId int64,
	) error
}

func (t *TelemetryDataRow) Init(
	dItm *telemetrylib.TelemetryDataItem,
	bHdr *telemetrylib.TelemetryBundleHeader,
	tagSetId int64,
	customerRefId int64,
) (err error) {
	// init common telemetry data fields
	t.ClientId = bHdr.BundleClientId
	t.CustomerRefId = customerRefId
	t.TelemetryId = dItm.Header.TelemetryId
	t.TelemetryType = dItm.Header.TelemetryType
	t.Timestamp = dItm.Header.TelemetryTimeStamp
	t.TagSetId = tagSetId
	t.DataItem = []byte(dItm.TelemetryData)

	return
}

func (t *TelemetryDataRow) SetupDB(adb *AppDb, tx *sql.Tx) {
	// save DB reference
	t.SetTableSpec(GetTelemetryTableSpec())
	t.TableRowCommon.SetupDB(adb, tx)
}

func (t *TelemetryDataRow) TableName() string {
	return t.TableRowCommon.TableName()
}

func (t *TelemetryDataRow) RowId() int64 {
	return t.Id
}

func (t *TelemetryDataRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *TelemetryDataRow) Exists() bool {

	stmt, err := t.SelectStmt(
		// select columns
		[]string{
			"id",
			"customerRefId",
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

	row := t.Tx().QueryRow(
		stmt,
		t.ClientId,
		t.TelemetryId,
		t.Timestamp,
	)
	// if the entry was found, all fields not used to find the entry will have
	// been updated to match what is in the DB
	if err := row.Scan(
		&t.Id,
		&t.CustomerRefId,
		&t.TelemetryType,
		&t.TagSetId,
		&t.DataItem,
	); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"check for matching entry failed",
				slog.String("table", t.TableName()),
				slog.String("clientId", t.ClientId),
				slog.String("telemetryId", t.TelemetryId),
				slog.String("timestamp", t.Timestamp),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (t *TelemetryDataRow) Insert() (err error) {
	stmt, err := t.InsertStmt(
		[]string{
			"clientId",
			"customerRefId",
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

	row := t.Tx().QueryRow(
		stmt,
		t.ClientId,
		t.CustomerRefId,
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
			slog.String("clientId", t.ClientId),
			slog.String("telemetryId", t.TelemetryId),
			slog.String("timestamp", t.Timestamp),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (t *TelemetryDataRow) Update() (err error) {
	stmt, err := t.UpdateStmt(
		[]string{
			"clientId",
			"customerRefId",
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

	_, err = t.Tx().Exec(
		stmt,
		t.ClientId,
		t.CustomerRefId,
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

func (t *TelemetryDataRow) Delete() (err error) {
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

	_, err = t.Tx().Exec(
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

// validate that TelemetryDataRow implements TelemetryDataRowHandler interface
var _ TelemetryDataRowHandler = (*TelemetryDataRow)(nil)
