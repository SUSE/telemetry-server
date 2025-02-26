package app

import (
	"database/sql"
	"encoding/json"
	"log/slog"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

func (a *App) StoreTelemetry(dItm *telemetrylib.TelemetryDataItem, bHeader *telemetrylib.TelemetryBundleHeader) (err error) {
	// generate a tagSet from the bundle and data item tags
	tagSet := createTagSet(append(dItm.Header.TelemetryAnnotations, bHeader.BundleAnnotations...))

	tsRow := new(TagSetRow)
	if err = tsRow.SetupDB(&a.TelemetryDB); err != nil {
		slog.Error("TagSetRow.SetupDB failed", slog.String("error", err.Error()))
		return
	}

	tsRow.Init(tagSet)

	// add the tagSet to the tagSets table, if not already present
	if !tsRow.Exists() {
		if err := tsRow.Insert(); err != nil {
			slog.Error("tagSet insert failed", slog.String("tagSet", tsRow.TagSet), slog.String("error", err.Error()))
			return err
		}

		slog.Info("tagSet added successfully", slog.String("tagSet", tsRow.TagSet), slog.Int64("id", tsRow.Id))
	}

	// store the telemetry
	err = a.StoreTelemetryData(dItm, bHeader, tsRow.Id)
	if err != nil {
		slog.Error("telemetry store failed", slog.String("telemetryId", dItm.Header.TelemetryId), slog.String("error", err.Error()))
	}
	return
}

func (a *App) StoreTelemetryData(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) (err error) {
	tdRow := new(TelemetryDataRow)

	tdRow.SetupDB(&a.TelemetryDB)

	err = tdRow.Init(dItm, bHdr, tagSetId)
	if err != nil {
		slog.Error(
			"unstructured tdRow init failed",
			slog.String("telemetryId", dItm.Header.TelemetryId),
			slog.String("error", err.Error()),
		)
		return
	}

	if !tdRow.Exists() {
		if err := tdRow.Insert(); err != nil {
			slog.Error(
				"unstructured tdRow insert failed",
				slog.String("tableName", tdRow.TableName()),
				slog.String("telemetryId", dItm.Header.TelemetryId),
				slog.String("error", err.Error()),
			)
			return err
		}

		slog.Info(
			"unstructured tdRow insert success",
			slog.String("tableName", tdRow.TableName()),
			slog.String("telemetryId", dItm.Header.TelemetryId),
		)
	}

	return
}

var telemetryTableSpec = TableSpec{
	Name: "telemetryData",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "clientId", Type: "VARCHAR"},
		{Name: "customerId", Type: "INTEGER"},
		{Name: "telemetryId", Type: "VARCHAR"},
		{Name: "telemetryType", Type: "VARCHAR"},
		{Name: "tagSetId", Type: "INTEGER", Nullable: true},
		{Name: "timestamp", Type: "VARCHAR"},
		{Name: "dataItem", Type: "TEXT"},
	},
}

type TelemetryDataRow struct {
	TableRowCommon

	// public table fields
	Id            int64  `json:"id"`
	ClientId      string `json:"clientId"`
	CustomerId    string `json:"customerId"`
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
	Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) error
}

func (t *TelemetryDataRow) Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) (err error) {
	// init common telemetry data fields
	t.ClientId = bHdr.BundleClientId
	t.CustomerId = bHdr.BundleCustomerId
	t.TelemetryId = dItm.Header.TelemetryId
	t.TelemetryType = dItm.Header.TelemetryType
	t.Timestamp = dItm.Header.TelemetryTimeStamp
	t.TagSetId = tagSetId
	t.DataItem = []byte(dItm.TelemetryData)

	return
}

func (t *TelemetryDataRow) SetupDB(db *DbConnection) (err error) {
	// save DB reference
	t.tableSpec = &telemetryTableSpec
	return t.TableRowCommon.SetupDB(db)
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

// validate that TelemetryDataRow implements TelemetryDataRowHandler interface
var _ TelemetryDataRowHandler = (*TelemetryDataRow)(nil)
