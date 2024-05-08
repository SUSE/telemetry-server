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

const tagElementTableColumns = `(
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

const tagListTableColumns = `(
	telemetryId INTEGER NOT NULL,
	tagId INTEGER NOT NULL,
	FOREIGN KEY (telemetryId) REFERENCES telemetryData (id)
	FOREIGN KEY (tagId) REFERENCES tagElement (id)
	PRIMARY KEY (telemetryId, tagId)
)`

type TagListRow struct {
	TelemetryId int64 `json:"telemetryId"`
	TagId       int64 `json:"tagId"`
}

func (t *TagListRow) String() string {
	bytes, _ := json.Marshal(t)
	return string(bytes)
}

func (t *TagListRow) Exists(DB *sql.DB) bool {
	row := DB.QueryRow(`SELECT telemetryId FROM tagList WHERE telemetryId = ? AND tagId = ?`, t.TelemetryId, t.TagId)
	var tid int64
	if err := row.Scan(&tid); err != nil {
		if err != sql.ErrNoRows {
			log.Printf("ERR: failed when checking for existence of telemetryId %q tag %q: %s", t.TelemetryId, t.TagId, err.Error())
		}
		return false
	}
	return true
}

const telemetryDataTableColumns = `(
	id INTEGER NOT NULL PRIMARY KEY,
	clientID INTEGER NOT NULL,
	telemetryId VARCHAR(64) NOT NULL,
	telemetryType VARCHAR(64) NOT NULL,
	timestamp VARCHAR(32) NOT NULL,
	staged BOOLEAN DEFAULT false,
	dataItem BLOB NOT NULL,
	FOREIGN KEY (clientId) REFERENCES clients (id)
)`

type TelemetryDataRow struct {
	Id            int64       `json:"id"`
	ClientId      string      `json:"clientId"`
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

// list of predefined tables
var dbTables = map[string]string{
	"clients":       clientsTableColumns,
	"tagElement":    tagElementTableColumns,
	"tagList":       tagListTableColumns,
	"telemetryData": telemetryDataTableColumns,
}
