package app

import (
	"database/sql"
	"encoding/json"
	"log/slog"

	telemetrylib "github.com/SUSE/telemetry/pkg/lib"
)

const defaultTelemetryTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	clientId INTEGER NOT NULL,
	customerId INTEGER NOT NULL,
	telemetryId VARCHAR(64) NOT NULL,
	telemetryType VARCHAR(64) NOT NULL,
	tagSetId INTEGER NULL,
	timestamp VARCHAR(32) NOT NULL,
	staged BOOLEAN DEFAULT false,
	dataItem BLOB NOT NULL,
	FOREIGN KEY (tagSetId) REFERENCES tagSets (id)
)`

type DefaultTelemetryDataRow struct {
	// Embed the common rows
	TelemetryDataCommon

	DataItem any `json:"dataItem"`
}

func (t *DefaultTelemetryDataRow) Init(dItm *telemetrylib.TelemetryDataItem, bHdr *telemetrylib.TelemetryBundleHeader, tagSetId int64) (err error) {
	t.TelemetryDataCommon.Init(dItm, bHdr, tagSetId)

	// marshal telemetry data as JSON
	jsonData, err := json.Marshal(dItm.TelemetryData)
	if err != nil {
		slog.Error(
			"JSON marshal failed",
			slog.Int64("clientId", t.ClientId),
			slog.String("telemetryId", t.TelemetryId),
			slog.String("timestamp", t.Timestamp),
			slog.String("error", err.Error()),
		)
		return
	}
	t.DataItem = jsonData

	return
}

func (t *DefaultTelemetryDataRow) SetupDB(db *sql.DB) (err error) {
	t.TelemetryDataCommon.SetupDB(db)

	t.table = `telemetryData`

	// prepare exists check statement
	t.exists, err = t.db.Prepare(
		`SELECT id, customerId, telemetryType, tagSetId, dataItem FROM telemetryData WHERE clientId = ? AND telemetryId = ? AND timestamp = ?`,
	)
	if err != nil {
		slog.Error("exists statement prep failed", slog.String("table", t.table), slog.String("error", err.Error()))
		return
	}

	// prepare insert statement
	t.insert, err = t.db.Prepare(
		`INSERT INTO telemetryData(clientId, customerId, telemetryId, telemetryType, timestamp, tagSetId, dataItem) VALUES(?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		slog.Error("insert statement prep failed", slog.String("table", t.table), slog.String("error", err.Error()))
		return
	}

	// prepare update statement
	t.update, err = t.db.Prepare(
		`UPDATE telemetryData SET clientId = ?, customerId = ?, telemetryId = ?, telemetryType = ?, timestamp = ?, tagSetId = ?, dataItem = ? WHERE id = ?`,
	)
	if err != nil {
		slog.Error("update statement prep failed", slog.String("table", t.table), slog.String("error", err.Error()))
		return
	}

	// prepare delete statement
	t.delete, err = t.db.Prepare(
		`DELETE FROM telemetryData WHERE id = ?`,
	)
	if err != nil {
		slog.Error("delete statement prep failed", slog.String("table", t.table), slog.String("error", err.Error()))
		return
	}

	return
}

func (t *DefaultTelemetryDataRow) TableName() string {
	return t.table
}

func (t *DefaultTelemetryDataRow) RowId() int64 {
	return t.Id
}

func (t *DefaultTelemetryDataRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *DefaultTelemetryDataRow) Exists() bool {
	row := t.exists.QueryRow(
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
				slog.String("table", t.table),
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
	res, err := t.insert.Exec(
		t.ClientId,
		t.CustomerId,
		t.TelemetryId,
		t.TelemetryType,
		t.Timestamp,
		t.TagSetId,
		t.DataItem,
	)
	if err != nil {
		slog.Error(
			"insert failed",
			slog.String("table", t.table),
			slog.Int64("clientId", t.ClientId),
			slog.String("telemetryId", t.TelemetryId),
			slog.String("timestamp", t.Timestamp),
			slog.String("error", err.Error()),
		)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error(
			"LastInsertId() failed",
			slog.String("table", t.table),
			slog.Int64("clientId", t.ClientId),
			slog.String("telemetryId", t.TelemetryId),
			slog.String("timestamp", t.Timestamp),
			slog.String("error", err.Error()),
		)
		return
	}
	t.Id = id

	return
}

func (t *DefaultTelemetryDataRow) Update() (err error) {
	_, err = t.update.Exec(
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
			slog.String("table", t.table),
			slog.Int64("id", t.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

func (t *DefaultTelemetryDataRow) Delete() (err error) {
	_, err = t.delete.Exec(
		t.Id,
	)
	if err != nil {
		slog.Error(
			"delete failed",
			slog.String("table", t.table),
			slog.Int64("id", t.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

// validate that DefaultTelemetryDataRow implements TelemetryDataRow interface
var _ TelemetryDataRow = (*DefaultTelemetryDataRow)(nil)
