package app

import (
	"database/sql"
	"fmt"
	"log"

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
		log.Printf("Failed to connect to DB '%s:%s': %s", d.Driver, d.DataSource, err.Error())
	}

	return
}

func (d *DbConnection) EnsureTablesExist() (err error) {
	for name, columns := range dbTables {
		createCmd := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s %s", name, columns)
		log.Printf("createCmd:\n%s", createCmd)
		_, err = d.Conn.Exec(createCmd)
		if err != nil {
			log.Printf("failed to create table %q: %s", name, err.Error())
			return
		}
	}

	return
}
