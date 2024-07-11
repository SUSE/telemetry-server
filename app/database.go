package app

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

// DbConnection is a struct tracking a DB connection and associated DB settings
type DbConnection struct {
	name        string
	Conn        *sql.DB
	Driver      string
	DataSource  string
	Placeholder PlaceholderGenerator
}

func (d DbConnection) String() string {
	return fmt.Sprintf("%s:%p:%s:%s", d.name, d.Conn, d.Driver, d.DataSource)
}

func (d *DbConnection) Setup(name string, dbcfg DBConfig) {
	d.name = name
	d.Driver = dbcfg.Driver
	d.DataSource = dbcfg.Params

	switch d.Driver {
	case "sqlite":
		// sqlite is an alias for sqlite3
		d.Driver = "sqlite3"
		fallthrough
	case "sqlite3":
		// sqlite3 uses `?` as placeholder
		d.Placeholder = QuestionMarker
		// TODO: Setup sqlite3 required options
	case "postgres":
		// postgres is an alias for pgx
		d.Driver = "pgx"
		fallthrough
	case "pgx":
		// postgres uses `$1`, `$2`, ... as placeholders
		d.Placeholder = DollarCounter
	}
}

func (d *DbConnection) Connect() (err error) {
	// connect to specified DB using the specified driver and dataSource
	d.Conn, err = sql.Open(d.Driver, d.DataSource)
	if err != nil {
		slog.Error(
			"db connect failed",
			slog.String("driver", d.Driver),
			slog.String("dataSource", d.DataSource),
			slog.String("Error", err.Error()),
		)
	}

	slog.Info("Database Connected", slog.String("database", d.name))

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

func (d *DbConnection) EnsureTableSpecsExist(tables []TableSpec) (err error) {
	for _, table := range tables {
		createCmd := table.CreateCmd(d)
		slog.Debug("sql", slog.String("createCmd", createCmd))
		_, err = d.Conn.Exec(createCmd)
		if err != nil {
			slog.Error("create table failed", slog.String("table", table.Name), slog.String("error", err.Error()))
			return
		}
	}

	return
}

type TableSpecColumn struct {
	Name       string
	Type       string
	Nullable   bool
	Default    string
	PrimaryKey bool
	Identity   bool
}

func (c *TableSpecColumn) Create(db *DbConnection) string {
	elements := []string{
		c.Name, c.Type,
	}
	if !c.Nullable {
		elements = append(elements, "NOT")
	}
	elements = append(elements, "NULL")
	if len(c.Default) > 0 {
		elements = append(elements, "DEFAULT", c.Default)
	}
	if c.PrimaryKey {
		elements = append(elements, "PRIMARY", "KEY")
	}
	if c.Identity {
		switch db.Driver {
		case "pgx":
			elements = append(elements, "GENERATED", "BY", "DEFAULT", "AS", "IDENTITY")
		}
	}
	return strings.Join(elements, " ")
}

type TableSpec struct {
	Name    string
	Columns []TableSpecColumn
	Extras  []string
}

func (ts *TableSpec) CreateCmd(db *DbConnection) string {
	table := "CREATE TABLE IF NOT EXISTS " + ts.Name + " ("

	for i, column := range ts.Columns {
		if i > 0 {
			table += ", "
		}
		table += column.Create(db)
	}

	if len(ts.Extras) > 0 {
		table += ","
	}

	for _, extra := range ts.Extras {
		table += " " + extra
	}

	table += ")"

	return table
}

type TableRowCommon struct {
	// private db settings
	db        *DbConnection
	table     string
	tableSpec *TableSpec

	// prepared SQL statements
	exists *sql.Stmt
	insert *sql.Stmt
	update *sql.Stmt
	delete *sql.Stmt
}

func (t *TableRowCommon) SetupDB(db *DbConnection) (err error) {
	t.db = db

	// tableSpec should have already been initialised
	if t.tableSpec == nil {
		return fmt.Errorf("tableSpec should be initialised before calling SetupDB")
	}

	return
}

func (t *TableRowCommon) TableName() string {
	if t.tableSpec != nil {
		return t.tableSpec.Name
	}
	return t.table
}

type TableRowCommonHandler interface {
	// Setup DB access
	SetupDB(*DbConnection) error

	// Retrieve the TableName
	TableName() string
}

type TableRowHandler interface {
	TableRowCommonHandler
	// Setup DB access
	//SetupDB(*DbConnection) error

	// Retrieve the TableName
	//TableName() string

	// Retrieve the RowId
	RowId() int64

	// Return string representation of the row
	String() string

	// Check if the row exists in the DB, and if so populate it
	Exists() bool

	// Insert row into the table
	Insert() error

	// Update row in the table
	Update() error

	// Delete row from the table
	Delete() error
}
