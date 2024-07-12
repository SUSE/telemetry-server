package app

import (
	"database/sql"
	"encoding/json"
	"log/slog"

	"github.com/SUSE/telemetry/pkg/restapi"
)

var clientsTableSpec = TableSpec{
	Name: "clients",
	Columns: []TableSpecColumn{
		{Name: "id", Type: "INTEGER", PrimaryKey: true, Identity: true},
		{Name: "clientInstanceId", Type: "VARCHAR"},
		{Name: "registrationDate", Type: "VARCHAR"},
		{Name: "authToken", Type: "VARCHAR"},
	},
}

type ClientsRow struct {
	// include common table row fields
	TableRowCommon

	Id               int64  `json:"id"`
	ClientInstanceId string `json:"clientInstanceId"`
	RegistrationDate string `json:"registrationDate"`
	AuthToken        string `json:"authToken"`
}

func (c *ClientsRow) Init(crReq *restapi.ClientRegistrationRequest) {
	c.ClientInstanceId = crReq.ClientInstanceId
}

func (c *ClientsRow) TableName() string {
	return c.TableRowCommon.TableName()
}

func (c *ClientsRow) RowId() int64 {
	return c.Id
}

func (c *ClientsRow) String() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

func (c *ClientsRow) SetupDB(db *DbConnection) (err error) {
	c.tableSpec = &clientsTableSpec
	err = c.TableRowCommon.SetupDB(db)
	if err != nil {
		slog.Error("SetupDB() failed", slog.String("table", c.tableSpec.Name))
		return err
	}

	var ph Placeholder
	var stmt string

	// prepare exists check statement
	ph = c.db.Placeholder(1)
	stmt = `SELECT ` +
		c.tableSpec.Columns[0].Name + `, ` +
		c.tableSpec.Columns[2].Name + `, ` +
		c.tableSpec.Columns[3].Name +
		` FROM ` + c.TableName() +
		` WHERE ` +
		c.tableSpec.Columns[1].Name + ` = ` + ph.Next()

	c.exists, err = c.db.Conn.Prepare(stmt)
	if err != nil {
		slog.Error(
			"exists statement prep failed",
			slog.String("table", c.TableName()),
			slog.String("statement", stmt),
			slog.String("error", err.Error()),
		)
		return
	}

	// prepare insert statement
	ph = c.db.Placeholder(3)
	stmt = `INSERT INTO ` + c.TableName() +
		`(` + c.tableSpec.Columns[1].Name +
		`, ` + c.tableSpec.Columns[2].Name +
		`, ` + c.tableSpec.Columns[3].Name + `) ` +
		`VALUES(` +
		ph.Next() + `, ` +
		ph.Next() + `, ` +
		ph.Next() + `) ` +
		`RETURNING ` + c.tableSpec.Columns[0].Name
	c.insert, err = c.db.Conn.Prepare(stmt)
	if err != nil {
		slog.Error(
			"insert statement prep failed",
			slog.String("table", c.TableName()),
			slog.String("statement", stmt),
			slog.String("error", err.Error()),
		)
		return
	}

	// prepare update statement
	ph = c.db.Placeholder(4)
	stmt = `UPDATE ` + c.TableName() +
		` SET ` +
		c.tableSpec.Columns[1].Name + ` = ` + ph.Next() + ", " +
		c.tableSpec.Columns[2].Name + ` = ` + ph.Next() + ", " +
		c.tableSpec.Columns[3].Name + ` = ` + ph.Next() +
		` WHERE ` +
		c.tableSpec.Columns[0].Name + ` = ` + ph.Next()
	c.update, err = c.db.Conn.Prepare(stmt)
	if err != nil {
		slog.Error(
			"update statement prep failed",
			slog.String("table", c.TableName()),
			slog.String("statement", stmt),
			slog.String("error", err.Error()),
		)
		return
	}

	// prepare delete statement
	ph = c.db.Placeholder(1)
	stmt = `DELETE FROM ` + c.TableName() + ` WHERE ` +
		c.tableSpec.Columns[0].Name + ` = ` + ph.Next()
	c.delete, err = c.db.Conn.Prepare(stmt)
	if err != nil {
		slog.Error(
			"delete statement prep failed",
			slog.String("table", c.TableName()),
			slog.String("statement", stmt),
			slog.String("error", err.Error()),
		)
		return
	}

	return
}

func (c *ClientsRow) Exists() bool {
	row := c.exists.QueryRow(c.ClientInstanceId)
	// if the entry was found, all fields not used to find the entry will have
	// been updated to match what is in the DB
	if err := row.Scan(
		&c.Id,
		&c.RegistrationDate,
		&c.AuthToken,
	); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"check for matching entry failed",
				slog.String("table", c.TableName()),
				slog.String("clientInstanceId", c.ClientInstanceId),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (c *ClientsRow) Insert() (err error) {
	row := c.insert.QueryRow(
		c.ClientInstanceId, c.RegistrationDate, c.AuthToken,
	)
	if err = row.Scan(
		&c.Id,
	); err != nil {
		slog.Error(
			"insert failed",
			slog.String("table", c.TableName()),
			slog.Any("clientInstanceId", c.ClientInstanceId),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (c *ClientsRow) Update() (err error) {
	_, err = c.update.Exec(
		c.ClientInstanceId,
		c.RegistrationDate,
		c.AuthToken,
		c.Id,
	)
	if err != nil {
		slog.Error(
			"update failed",
			slog.String("table", c.table),
			slog.Int64("id", c.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

func (c *ClientsRow) Delete() (err error) {
	_, err = c.delete.Exec(
		c.Id,
	)
	if err != nil {
		slog.Error(
			"delete failed",
			slog.String("table", c.table),
			slog.Int64("id", c.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

// verify that ClientsRow conforms to the TableRowHandler interface
var _ TableRowHandler = (*ClientsRow)(nil)
