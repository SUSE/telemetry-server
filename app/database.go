package app

import (
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
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

	// TODO: add a sync.Map to hold sql.Prepare()'d statements to
	// eliminate repeated generation of the same SQL statements
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

func (ts *TableSpec) ColumnName(ind int) (name string, err error) {
	switch {
	case ind < 0:
		fallthrough
	case ind >= len(ts.Columns):
		err = fmt.Errorf("invalid column index %d for table %q", ind, ts.Name)
	default:
		name = ts.Columns[ind].Name
	}
	return
}

func (ts *TableSpec) CheckColumnNames(colNames []string) (err error) {
	for _, colName := range colNames {
		matchInd := slices.IndexFunc(ts.Columns, func(cs TableSpecColumn) bool {
			return cs.Name == colName
		})
		if matchInd == -1 {
			err = fmt.Errorf("column %q not part of table %s", colName, ts.Name)
			break
		}
	}

	// log a debug message only if an error occurred
	if err != nil {
		slog.Debug("Invalid columns", slog.String("error", err.Error()))
	}
	return err
}

type TableRowCommon struct {
	// private db settings
	db        *DbConnection
	tableSpec *TableSpec
}

func (t *TableRowCommon) SetupDB(db *DbConnection) (err error) {
	t.db = db

	// tableSpec should have already been initialised
	if t.tableSpec == nil {
		return fmt.Errorf("tableSpec should be initialised before calling SetupDB")
	}

	return
}

func (t *TableRowCommon) DB() *sql.DB {
	if t.db == nil {
		err := fmt.Errorf("%q row db connection not setup", t.TableName())
		slog.Error("TableRowCommon.db not setup", slog.String("error", err.Error()))
		panic(err)
	}

	return t.db.Conn
}

func (t *TableRowCommon) TableName() string {
	if t.tableSpec == nil {
		err := fmt.Errorf("row table spec not setup")
		slog.Error("TableRowCommon.tableSpec not setup", slog.String("error", err.Error()))
		panic(err)
	}

	return t.tableSpec.Name
}

func (t *TableRowCommon) ColumnName(ind int) string {
	name, err := t.tableSpec.ColumnName(ind)
	if err != nil {
		slog.Error(
			"Invalid column index",
			slog.String("error", err.Error()),
		)
		panic(err)
	}
	return name
}

func (t *TableRowCommon) Columns(inputColumns ...string) (columns []string, err error) {
	// ensure specified columns are valid for this table
	err = t.tableSpec.CheckColumnNames(inputColumns)

	// setup columns to be returned if no error occured
	if err == nil {
		columns = inputColumns
	}

	return
}

type SelectOpts struct {
	Count    bool
	Distinct bool
	Limit    uint
}

func (t *TableRowCommon) SelectStmt(selectCols, whereCols []string, opts SelectOpts) (stmt string, err error) {
	switch len(selectCols) {
	case 0:
		return "", fmt.Errorf("no select columns specified")
	default:
		// ensure selectCols are valid
		if err = t.tableSpec.CheckColumnNames(selectCols); err != nil {
			return "", fmt.Errorf("invalid select column: %w", err)
		}
	}

	if len(whereCols) > 0 {
		// ensure whereCols are valid
		if err = t.tableSpec.CheckColumnNames(whereCols); err != nil {
			return "", fmt.Errorf("invalid where column: %w", err)
		}
	}

	stmt = "SELECT "

	// start wrap of the selection with COUNT( if requested
	if opts.Count {
		stmt += "COUNT("
	}

	// start wrap of the selection with DISTINCT( if requested
	if opts.Distinct {
		stmt += "DISTINCT("
	}

	// add selection column(s)
	for i, selectCol := range selectCols {
		if i > 0 {
			stmt += ", "
		}
		stmt += selectCol
	}

	// end DISTINCT( with a )
	if opts.Distinct {
		stmt += ")"
	}

	// end COUNT( with a )
	if opts.Count {
		stmt += ")"
	}

	// add the table
	stmt += " FROM " + t.TableName()

	// if where columns were specified
	if len(whereCols) > 0 {
		stmt += " WHERE "

		// instantiate placeholder generator for required column count
		ph := t.db.Placeholder(len(whereCols))

		// add where conditions
		for i, whereCol := range whereCols {
			if i > 0 {
				stmt += " AND "
			}

			// add where clause with appropriate placeholder
			stmt += whereCol + " = " + ph.Next()
		}
	}

	// add a limit count if specified
	if opts.Limit > 0 {
		stmt += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	slog.Debug("Generated SELECT statement", slog.String("stmt", stmt))

	return
}

func (t *TableRowCommon) InsertStmt(insertCols []string, returning string) (stmt string, err error) {
	switch len(insertCols) {
	case 0:
		return "", fmt.Errorf("no insert columns specified")
	default:
		// ensure insertCols are valid
		if err = t.tableSpec.CheckColumnNames(insertCols); err != nil {
			return "", fmt.Errorf("invalid insert column: %w", err)
		}
	}

	if returning != "" {
		// ensure returning is a valid column
		if err = t.tableSpec.CheckColumnNames([]string{returning}); err != nil {
			return "", fmt.Errorf("invalid returning column: %w", err)
		}
	}

	// start an insert statement
	stmt = "INSERT INTO " + t.TableName() + "("

	// add the insert columns
	for i, insertCol := range insertCols {
		if i > 0 {
			stmt += ", "
		}
		stmt += insertCol
	}

	// end insert columns list and start value placeholders list
	stmt += ") VALUES("

	// instantiate placeholder generator for required column count
	ph := t.db.Placeholder(len(insertCols))

	// add the value placeholders
	for i := 0; i < len(insertCols); i += 1 {
		if i > 0 {
			stmt += ", "
		}

		// add appropriate placeholder
		stmt += ph.Next()
	}

	// end the value placeholders list
	stmt += ")"

	// if requested add the returning column directive
	if returning != "" {
		stmt += " RETURNING " + returning
	}

	slog.Debug("Generated INSERT statement", slog.String("stmt", stmt))

	return
}

func (t *TableRowCommon) UpdateStmt(updateCols, whereCols []string) (stmt string, err error) {
	switch len(updateCols) {
	case 0:
		return "", fmt.Errorf("no update columns specified")
	default:
		// ensure updateCols are valid
		if err = t.tableSpec.CheckColumnNames(updateCols); err != nil {
			return "", fmt.Errorf("invalid update column: %w", err)
		}
	}

	if len(whereCols) > 0 {
		// ensure whereCols are valid
		if err = t.tableSpec.CheckColumnNames(whereCols); err != nil {
			return "", fmt.Errorf("invalid where column: %w", err)
		}
	}

	// start an update statement
	stmt = "UPDATE " + t.TableName() + " SET "

	// instantiate placeholder generator for required column count
	ph := t.db.Placeholder(len(updateCols) + len(whereCols))

	// add update assignments
	for i, updateCol := range updateCols {
		if i > 0 {
			stmt += ", "
		}

		// add update assignment with appropriate placeholder
		stmt += updateCol + " = " + ph.Next()
	}

	// if where columns were specified
	if len(whereCols) > 0 {
		stmt += " WHERE "

		// add where conditions
		for i, whereCol := range whereCols {
			if i > 0 {
				stmt += " AND "
			}

			// add where clause with appropriate placeholder
			stmt += whereCol + " = " + ph.Next()
		}
	}

	slog.Debug("Generated UPDATE statement", slog.String("stmt", stmt))

	return
}

func (t *TableRowCommon) DeleteStmt(whereCols []string) (stmt string, err error) {
	switch len(whereCols) {
	case 0:
		return "", fmt.Errorf("no delete where columns specified")
	default:
		// ensure whereCols are valid
		if err = t.tableSpec.CheckColumnNames(whereCols); err != nil {
			return "", fmt.Errorf("invalid delete where column: %w", err)
		}
	}

	// start a delete statement
	stmt = "DELETE FROM " + t.TableName() + " WHERE "

	// instantiate placeholder generator for required column count
	ph := t.db.Placeholder(len(whereCols))

	// add where conditions
	for i, whereCol := range whereCols {
		if i > 0 {
			stmt += " AND "
		}

		// add where specification with appropriate placeholder
		stmt += whereCol + " = " + ph.Next()
	}

	slog.Debug("Generated DELETE statement", slog.String("stmt", stmt))

	return
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
