package app

import (
	"database/sql"
	"encoding/json"
	"log/slog"
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
			slog.Error("client existence check failed", slog.Any("clientInstanceId", c.ClientInstanceId), slog.String("error", err.Error()))
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
		slog.Error("client insert failed", slog.Any("clientInstanceId", c.ClientInstanceId), slog.String("error", err.Error()))
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		slog.Error("client LastInsertId() failed", slog.Any("clientInstanceId", c.ClientInstanceId), slog.String("error", err.Error()))
		return err
	}
	c.Id = id

	return
}
