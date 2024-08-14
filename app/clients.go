package app

import (
	"database/sql"
	"encoding/json"
	"log/slog"

	"github.com/SUSE/telemetry/pkg/restapi"
	"github.com/SUSE/telemetry/pkg/types"
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

	Id               int64                  `json:"id"`
	ClientInstanceId types.ClientInstanceId `json:"clientInstanceId"`
	RegistrationDate string                 `json:"registrationDate"`
	AuthToken        string                 `json:"authToken"`
}

func (c *ClientsRow) InitAuthentication(caReq *restapi.ClientAuthenticationRequest) {
	c.InitClientId(caReq.ClientId)
}

func (c *ClientsRow) InitClientId(clientId int64) {
	c.Id = clientId
}

func (c *ClientsRow) InitRegistration(crReq *restapi.ClientRegistrationRequest) {
	c.InitClientInstanceId(&crReq.ClientInstanceId)
}

func (c *ClientsRow) InitClientInstanceId(instId *types.ClientInstanceId) {
	c.ClientInstanceId = *instId
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
	return c.TableRowCommon.SetupDB(db)
}

func (c *ClientsRow) Exists() bool {
	stmt, err := c.SelectStmt(
		// select columns
		[]string{
			"clientInstanceId",
			"registrationDate",
			"authToken",
		},
		// match columns
		[]string{
			"id",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"exists statement generation failed",
			slog.String("table", c.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := c.DB().QueryRow(stmt, c.Id)
	// if the entry was found, all fields not used to find the entry will have
	// been updated to match what is in the DB
	if err := row.Scan(
		&c.ClientInstanceId,
		&c.RegistrationDate,
		&c.AuthToken,
	); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"check for matching entry failed",
				slog.String("table", c.TableName()),
				slog.String("clientInstanceId", c.ClientInstanceId.String()),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (c *ClientsRow) InstIdExists() bool {
	stmt, err := c.SelectStmt(
		// select columns
		[]string{
			"id",
			"registrationDate",
			"authToken",
		},
		// match columns
		[]string{
			"clientInstanceId",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"instIdExists statement generation failed",
			slog.String("table", c.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := c.DB().QueryRow(stmt, c.ClientInstanceId)
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
				slog.String("clientInstanceId", c.ClientInstanceId.String()),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (c *ClientsRow) Insert() (err error) {
	stmt, err := c.InsertStmt(
		[]string{
			"clientInstanceId",
			"registrationDate",
			"authToken",
		},
		"id",
	)
	if err != nil {
		slog.Error(
			"insert statement generation failed",
			slog.String("table", c.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}
	row := c.DB().QueryRow(
		stmt,
		c.ClientInstanceId,
		c.RegistrationDate,
		c.AuthToken,
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
	stmt, err := c.UpdateStmt(
		[]string{
			"clientInstanceId",
			"registrationDate",
			"authToken",
		},
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"update statement generation failed",
			slog.String("table", c.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}
	_, err = c.DB().Exec(
		stmt,
		c.ClientInstanceId,
		c.RegistrationDate,
		c.AuthToken,
		c.Id,
	)
	if err != nil {
		slog.Error(
			"update failed",
			slog.String("table", c.TableName()),
			slog.Int64("id", c.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

func (c *ClientsRow) Delete() (err error) {
	stmt, err := c.DeleteStmt(
		[]string{
			"id",
		},
	)
	if err != nil {
		slog.Error(
			"delete statement generation failed",
			slog.String("table", c.TableName()),
			slog.String("error", err.Error()),
		)
		return
	}

	_, err = c.DB().Exec(
		stmt,
		c.Id,
	)
	if err != nil {
		slog.Error(
			"delete failed",
			slog.String("table", c.TableName()),
			slog.Int64("id", c.Id),
			slog.String("error", err.Error()),
		)
	}
	return
}

// verify that ClientsRow conforms to the TableRowHandler interface
var _ TableRowHandler = (*ClientsRow)(nil)
