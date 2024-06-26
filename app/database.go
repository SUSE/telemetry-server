package app

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
)

// DbConnection is a struct tracking a DB connection and associated DB settings
type DbConnection struct {
	Conn       *sql.DB
	Driver     string
	DataSource string
}

func (d DbConnection) String() string {
	return fmt.Sprintf("%p:%s:%s", d.Conn, d.Driver, d.DataSource)
}

func (d *DbConnection) Setup(dbcfg DBConfig) {
	d.Driver, d.DataSource = dbcfg.Driver, dbcfg.Params
}

func (d *DbConnection) Connect() (err error) {
	d.Conn, err = sql.Open(d.Driver, d.DataSource)
	if err != nil {
		slog.Error(
			"db connect failed",
			slog.String("driver", d.Driver),
			slog.String("dataSource", d.DataSource),
			slog.String("Error", err.Error()),
		)
	}

	return
}

func (d *DbConnection) EnsureTablesExist(tables map[string]string) (err error) {
	for name, columns := range tables {
		createCmd := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s %s", name, columns)
		slog.Debug("sql", slog.String("createCmd", createCmd))
		_, err = d.Conn.Exec(createCmd)
		if err != nil {
			slog.Error("create table failed", slog.String("table", name), slog.String("error", err.Error()))
			return
		}
	}

	return
}
