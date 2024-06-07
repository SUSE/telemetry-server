package app

import (
	"database/sql"
	"encoding/json"
	"log"
)

const clientsTableColumns = `(
	id               INTEGER     NOT NULL PRIMARY KEY,
	clientInstanceId VARCHAR(64) NOT NULL,
	registrationDate VARCHAR(32) NOT NULL,
	authToken        VARCHAR(32) NOT NULL
)`

type ClientsRow struct {
	Id               int64  `json:"id"`
	ClientInstanceId string `json:"clientInstanceId"`
	RegistrationDate string `json:"registrationDate"`
	AuthToken        string `json:"authToken"`
}

func (c *ClientsRow) String() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

func (c *ClientsRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT id FROM clients WHERE clientInstanceId = ?`, c.ClientInstanceId)
	if err := row.Scan(&c.Id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of client with clientInstanceId = %q: %s", c.ClientInstanceId, err.Error())
		}
		return false
	}
	return true
}

func (c *ClientsRow) Insert(DB *sql.DB) (err error) {
	res, err := DB.Exec(
		`INSERT INTO clients(clientInstanceId, registrationDate, authToken) VALUES(?, ?, ?)`,
		c.ClientInstanceId, c.RegistrationDate, c.AuthToken,
	)
	if err != nil {
		log.Printf("ERR: failed to add client %q: %s", c.ClientInstanceId, err.Error())
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("ERR: failed to retrieve id for inserted client %q: %s", c.ClientInstanceId, err.Error())
		return err
	}
	c.Id = id

	return
}

const tagSetsTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	tagSet VARCHAR NOT NULL UNIQUE
)`

type TagSetRow struct {
	Id     int64  `json:"id"`
	TagSet string `json:"tagSet"`
}

func (t *TagSetRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *TagSetRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT id FROM tagSets WHERE tagSet = ?`, t.TagSet)
	if err := row.Scan(&t.Id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of tagSet %q: %s", t.TagSet, err.Error())
		}
		return false
	}
	return true
}

func (t *TagSetRow) Insert(DB *sql.DB) (err error) {
	res, err := DB.Exec(
		`INSERT INTO tagSets(tagSet) VALUES(?)`,
		t.TagSet,
	)
	if err != nil {
		log.Printf("ERR: failed to add tagSet %q: %s", t.TagSet, err.Error())
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("ERR: failed to retrieve id for inserted tagSet %q: %s", t.TagSet, err.Error())
		return err
	}
	t.Id = id

	return
}

// NOTE: clientID is technically a foreign key reference to the client.id
// only so long as we are not dealing with relayed bundles; once relays
// are part of the picture then the clientId in the Bundle header can be
// different then the clientId in a received Report header.
const telemetryDataTableColumns = `(
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

type TelemetryDataRow struct {
	Id            int64       `json:"id"`
	ClientId      int64       `json:"clientId"`
	CustomerId    string      `json:"customerId"`
	TelemetryId   string      `json:"telemetryId"`
	TelemetryType string      `json:"telemetryType"`
	Timestamp     string      `json:"timestamp"`
	TagSetId      int64       `json:"tagSetId"`
	Staged        bool        `json:"staged"`
	DataItem      interface{} `json:"dataItem"`
}

func (t *TelemetryDataRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *TelemetryDataRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT id, staged FROM telemetryData WHERE telemetryId = ? AND telemetryType = ?`, t.TelemetryId, t.TelemetryType)
	if err := row.Scan(&t.Id, &t.Staged); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of telemetry data id %q, type %q: %s", t.Id, t.TelemetryType, err.Error())
		}
		return false
	}
	return true
}

func (t *TelemetryDataRow) Insert(DB *sql.DB) (err error) {
	res, err := DB.Exec(
		`INSERT INTO telemetryData(clientId, customerId, telemetryId, telemetryType, timestamp, dataItem) VALUES(?, ?, ?, ?, ?, ?)`,
		t.ClientId, t.CustomerId, t.TelemetryId, t.TelemetryType, t.Timestamp, t.DataItem,
	)
	if err != nil {
		log.Printf("failed to add telemetryData entry for customerId %q telemetryId %q: %s", t.CustomerId, t.TelemetryId, err.Error())
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("ERR: failed to retrieve id for inserted telemetryData %q: %s", t.TelemetryId, err.Error())
		return err
	}
	t.Id = id

	return
}

// list of predefined tables
var dbTables = map[string]string{
	"clients":       clientsTableColumns,
	"tagSets":       tagSetsTableColumns,
	"telemetryData": telemetryDataTableColumns,
}

const reportsTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	key VARCHAR(64) NOT NULL,
	data BLOB NOT NULL,
	processed BOOLEAN DEFAULT false,
	receivedTimestamp VARCHAR(32) NOT NULL
)`

var dbTablesStaging = map[string]string{
	"reports": reportsTableColumns,
}

type ReportStagingTableRow struct {
	Key               string
	Data              interface{}
	Processed         bool
	ReceivedTimestamp string
}

func (r *ReportStagingTableRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT key FROM reports WHERE key = ?`, r.Key)
	if err := row.Scan(&r.Key); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of report id %q: %s", r.Key, err.Error())
		}
		return false
	}
	return true
}

func (r *ReportStagingTableRow) Insert(DB *sql.DB) (err error) {
	_, err = DB.Exec(
		`INSERT INTO reports(key, data, processed, receivedTimestamp) VALUES(?, ?, ?, ?)`,
		r.Key, r.Data, false, r.ReceivedTimestamp,
	)
	if err != nil {
		log.Printf("failed to insert Report entry with ReportId %q: %s", r.Key, err.Error())
		return err
	}

	return
}

func (r *ReportStagingTableRow) Delete(DB *sql.DB) (err error) {
	_, err = DB.Exec("DELETE FROM reports WHERE key = ?", r.Key)
	if err != nil {
		log.Printf("failed to delete Report entry with ReportId %q: %s", r.Key, err.Error())
		return err
	}

	return
}
