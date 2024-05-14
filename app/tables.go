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
	AuthToken        string `json:"AuthToken"`
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

const tagElementsTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	tag VARCHAR(256) NOT NULL
)`

type TagElementRow struct {
	Id  int64  `json:"id"`
	Tag string `json:"tag"`
}

func (t *TagElementRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *TagElementRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT id FROM tagElements WHERE tag = ?`, t.Tag)
	if err := row.Scan(&t.Id); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of tag %q: %s", t.Tag, err.Error())
		}
		return false
	}
	return true
}

func (t *TagElementRow) Insert(DB *sql.DB) (err error) {
	res, err := DB.Exec(`INSERT INTO tagElements(tag) VALUES(?)`, t.Tag)
	if err != nil {
		log.Printf("ERR: failed to add tag %q to tagElements table: %s", t.Tag, err.Error())
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("ERR: failed to retrieve id for inserted tag %q: %s", t.Tag, err.Error())
		return err
	}
	t.Id = id

	return
}

const tagListsTableColumns = `(
	telemetryDataId INTEGER NOT NULL,
	tagId INTEGER NOT NULL,
	FOREIGN KEY (telemetryDataId) REFERENCES telemetryData (id)
	FOREIGN KEY (tagId) REFERENCES tagElements (id)
	PRIMARY KEY (telemetryDataId, tagId)
)`

type TagListRow struct {
	TelemetryDataId int64 `json:"telemetryDataId"`
	TagId           int64 `json:"tagId"`
}

func (t *TagListRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *TagListRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT telemetryDataId FROM tagLists WHERE telemetryDataId = ? AND tagId = ?`, t.TelemetryDataId, t.TagId)
	var tid int64
	if err := row.Scan(&tid); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of telemetryDataId %q tag %q: %s", t.TelemetryDataId, t.TagId, err.Error())
		}
		return false
	}
	return true
}

func (t *TagListRow) Insert(DB *sql.DB) (err error) {
	_, err = DB.Exec(
		`INSERT INTO tagLists(telemetryDataId, tagId) VALUES(?, ?)`,
		t.TelemetryDataId, t.TagId,
	)
	if err != nil {
		log.Printf("ERR: failed to add tagList (%d, %d): %s", t.TelemetryDataId, t.TagId, err.Error())
		return err
	}

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
	timestamp VARCHAR(32) NOT NULL,
	staged BOOLEAN DEFAULT false,
	dataItem BLOB NOT NULL
)`

type TelemetryDataRow struct {
	Id            int64       `json:"id"`
	ClientId      int64       `json:"clientId"`
	CustomerId    string      `json:"customerId"`
	TelemetryId   string      `json:"telemetryId"`
	TelemetryType string      `json:"telemetryType"`
	Timestamp     string      `json:"timestamp"`
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
	"tagElements":   tagElementsTableColumns,
	"tagLists":      tagListsTableColumns,
	"telemetryData": telemetryDataTableColumns,
}
