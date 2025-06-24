package database

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
		{Name: "clientId", Type: "VARCHAR"},
		{Name: "systemUUID", Type: "VARCHAR", Nullable: true},
		{Name: "clientTimestamp", Type: "VARCHAR"},
		{Name: "registrationDate", Type: "VARCHAR"},
		{Name: "authToken", Type: "VARCHAR"},
	},
}

func GetClientsTableSpec() *TableSpec {
	return &clientsTableSpec
}

type ClientsRow struct {
	// include common table row fields
	TableRowCommon

	Id               int64  `json:"id"`
	ClientId         string `json:"clientId"`
	SystemUUID       string `json:"systemUUID"`
	ClientTimestamp  string `json:"clientTimestamp"`
	RegistrationDate string `json:"registrationDate"`
	AuthToken        string `json:"authToken"`
}

func (c *ClientsRow) InitAuthentication(caReq *restapi.ClientAuthenticationRequest) {
	c.InitRegistrationId(caReq.RegistrationId)
}

func (c *ClientsRow) InitRegistrationId(registrationId int64) {
	c.Id = registrationId
}

func (c *ClientsRow) InitRegistration(crReq *restapi.ClientRegistrationRequest) {
	c.InitClientRegistration(&crReq.ClientRegistration)
}

func (c *ClientsRow) InitClientRegistration(reg *types.ClientRegistration) {
	c.ClientId = reg.ClientId
	c.SystemUUID = reg.SystemUUID
	c.ClientTimestamp = reg.Timestamp
}

func (c *ClientsRow) GetClientRegistration() *types.ClientRegistration {
	return &types.ClientRegistration{
		ClientId:   c.ClientId,
		SystemUUID: c.SystemUUID,
		Timestamp:  c.ClientTimestamp,
	}
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

func (c *ClientsRow) SetupDB(adb *AppDb) (err error) {
	c.SetTableSpec(GetClientsTableSpec())
	return c.TableRowCommon.SetupDB(adb)
}

func (c *ClientsRow) Exists() bool {
	stmt, err := c.SelectStmt(
		// select columns
		[]string{
			"clientId",
			"systemUUID",
			"clientTimestamp",
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
		&c.ClientId,
		&c.SystemUUID,
		&c.ClientTimestamp,
		&c.RegistrationDate,
		&c.AuthToken,
	); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"check for matching entry failed",
				slog.String("table", c.TableName()),
				slog.Int64("id", c.Id),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (c *ClientsRow) RegistrationExists() bool {
	stmt, err := c.SelectStmt(
		// select columns
		[]string{
			"id",
			"registrationDate",
			"authToken",
		},
		// match columns
		[]string{
			"clientId",
			"systemUUID",
			"clientTimestamp",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"registrationExists statement generation failed",
			slog.String("table", c.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := c.DB().QueryRow(
		stmt,
		c.ClientId,
		c.SystemUUID,
		c.ClientTimestamp,
	)
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
				slog.String("clientId", c.ClientId),
				slog.String("systemUUID", c.SystemUUID),
				slog.String("clientTimestamp", c.ClientTimestamp),
				slog.String("error", err.Error()),
			)
		}
		return false
	}
	return true
}

func (c *ClientsRow) ClientIdExists() bool {
	stmt, err := c.SelectStmt(
		// select columns
		[]string{
			"id",
			"systemUUID",
			"clientTimestamp",
			"registrationDate",
			"authToken",
		},
		// match columns
		[]string{
			"clientId",
		},
		SelectOpts{}, // no special options
	)
	if err != nil {
		slog.Error(
			"registrationExists statement generation failed",
			slog.String("table", c.TableName()),
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	row := c.DB().QueryRow(
		stmt,
		c.ClientId,
	)
	// if the entry was found, all fields not used to find the entry will have
	// been updated to match what is in the DB
	if err := row.Scan(
		&c.Id,
		&c.SystemUUID,
		&c.ClientTimestamp,
		&c.RegistrationDate,
		&c.AuthToken,
	); err != nil {
		if err != sql.ErrNoRows {
			slog.Error(
				"check for matching entry failed",
				slog.String("table", c.TableName()),
				slog.String("clientId", c.ClientId),
				slog.String("systemUUID", c.SystemUUID),
				slog.String("clientTimestamp", c.ClientTimestamp),
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
			"clientId",
			"systemUUID",
			"clientTimestamp",
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
		c.ClientId,
		c.SystemUUID,
		c.ClientTimestamp,
		c.RegistrationDate,
		c.AuthToken,
	)
	if err = row.Scan(
		&c.Id,
	); err != nil {
		slog.Error(
			"insert failed",
			slog.String("table", c.TableName()),
			slog.String("clientId", c.ClientId),
			slog.String("systemUUID", c.SystemUUID),
			slog.String("clientTimestamp", c.ClientTimestamp),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (c *ClientsRow) Update() (err error) {
	stmt, err := c.UpdateStmt(
		[]string{
			"clientId",
			"systemUUID",
			"clientTimestamp",
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
		c.ClientId,
		c.SystemUUID,
		c.ClientTimestamp,
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
