package app

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

const (
	CREATE_TABLE_ADVISORY = 3141592653589793
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
	slog.Debug("Connecting", slog.String("database", d.name))

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

	slog.Info("Connected", slog.String("database", d.name))

	return
}

func (d *DbConnection) AcquireAdvisoryLockPostgres(lockId int64, tx *sql.Tx, shared bool) (err error) {
	var lockStmt, typePfx, sharedSfx string

	// set an appropriate lock type prefix if in a transaction
	if tx != nil {
		typePfx = "_xact"
	}

	// set the appropriate suffix if a shared lock is requested
	if shared {
		sharedSfx = "_shared"
	}

	// construct the advisory lock statement
	lockStmt = fmt.Sprintf("SELECT pg_advisory%s_lock%s(%d);", typePfx, sharedSfx, lockId)

	slog.Debug(
		"attempting to acquire advisory lock",
		slog.String("db", d.name),
		slog.Int64("lockId", lockId),
		slog.Bool("inTx", tx != nil),
		slog.Bool("shared", shared),
	)

	// use appropriate Exec() method to acquire the lock
	if tx != nil {
		_, err = tx.Exec(lockStmt)
	} else {
		_, err = d.Conn.Exec(lockStmt)
	}
	if err != nil {
		slog.Debug(
			"acquire failed for advisory lock",
			slog.String("db", d.name),
			slog.Int64("lockId", lockId),
			slog.Bool("inTx", tx != nil),
			slog.Bool("shared", shared),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (d *DbConnection) AcquireAdvisoryLock(lockId int64, tx *sql.Tx, shared bool) (err error) {
	switch d.Driver {
	case "pgx":
		err = d.AcquireAdvisoryLockPostgres(lockId, tx, shared)
	}

	return
}

func (d *DbConnection) ReleaseAdvisoryLockPostgres(lockId int64, tx *sql.Tx, shared bool) (err error) {
	// transactional advisory locks are dropped as part of a transaction's
	// commit or rollback so only need to unlock for non-transactions
	if tx != nil {
		slog.Debug(
			"unlock of transactional advisory locks is automatic",
			slog.String("db", d.name),
			slog.Int64("lockId", lockId),
			slog.Bool("inTx", tx != nil),
			slog.Bool("shared", shared),
		)
		return
	}

	var unlockStmt, sharedSfx string

	// set the appropriate suffix if a shared lock is being released
	if shared {
		sharedSfx = "_shared"
	}

	// construct the advisory unlock statement
	unlockStmt = fmt.Sprintf("SELECT pg_advisory_unlock%s(%d);", sharedSfx, lockId)

	slog.Debug(
		"attempting to release advisory lock",
		slog.String("db", d.name),
		slog.Int64("lockId", lockId),
		slog.Bool("inTx", tx != nil),
		slog.Bool("shared", shared),
	)
	_, err = d.Conn.Exec(unlockStmt)
	if err != nil {
		slog.Debug(
			"release failed for advisory lock",
			slog.String("db", d.name),
			slog.Int64("lockId", lockId),
			slog.Bool("inTx", tx != nil),
			slog.Bool("shared", shared),
			slog.String("error", err.Error()),
		)
	}

	return
}

func (d *DbConnection) ReleaseAdvisdoryLock(lockId int64, tx *sql.Tx, shared bool) (err error) {
	switch d.Driver {
	case "pgx":
		err = d.ReleaseAdvisoryLockPostgres(lockId, tx, shared)
	}

	return
}

func (d *DbConnection) CheckTableExists(table *TableSpec) (bool, error) {
	var name string
	// exists query statement
	existsStmt := `
	SELECT table_name
	FROM information_schema.tables
	WHERE table_schema = current_schema()
	  AND table_name = $1;
	`
	// query if table exists
	row := d.Conn.QueryRow(existsStmt, table.Name)
	if err := row.Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug(
				"table does not exist",
				slog.String("db", d.name),
				slog.String("table", table.Name),
			)
			return false, nil
		}
		slog.Error(
			"query if table exists failed",
			slog.String("db", d.name),
			slog.String("table", table.Name),
			slog.String("error", err.Error()),
		)
		return false, fmt.Errorf(
			"failed to query if table %q exists: %w",
			table.Name,
			err,
		)
	}

	slog.Debug(
		"table exists",
		slog.String("db", d.name),
		slog.String("table", table.Name),
	)

	return true, nil
}

func (d *DbConnection) CreateTableFromSpec(table *TableSpec) (err error) {
	// generate the create table command
	createCmd, err := table.CreateCmd(d)
	if err != nil {
		slog.Error(
			"sql create table statement generation failed",
			slog.String("db", d.name),
			slog.String("table", table.Name),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("generation of create table statement failed: %w", err)
	}

	slog.Debug(
		"generated sql create table command",
		slog.String("db", d.name),
		slog.String("table", table.Name),
		slog.String("createCmd", createCmd),
	)

	// begin a transaction
	tx, err := d.Conn.Begin()
	if err != nil {
		return fmt.Errorf(
			"failed to begin a transaction to create the %q table: %w",
			table.Name,
			err,
		)
	}

	// defer performing a rollback, which can be safely called even if
	// the transaction was safely committed
	defer func() {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) && !errors.Is(err, sql.ErrConnDone) {
			slog.Warn(
				"failed to rollback table creation transaction",
				slog.String("db", d.name),
				slog.String("table", table.Name),
				slog.String("createCmd", createCmd),
				slog.String("error", err.Error()),
			)
		}
	}()

	// acquire an advisory lock for this transaction
	err = d.AcquireAdvisoryLock(CREATE_TABLE_ADVISORY, tx, false)
	if err != nil {
		return fmt.Errorf(
			"failed to acquire create table advisory lock for db %q: %w",
			d.name,
			err,
		)
	}

	// defer releaseing the advisory lock
	defer func() {
		d.ReleaseAdvisdoryLock(CREATE_TABLE_ADVISORY, tx, false)
	}()

	// attempt to execute the create table command
	_, err = tx.Exec(createCmd)
	if err == nil {
		slog.Debug(
			"create table succeeded, committing",
			slog.String("db", d.name),
			slog.String("table", table.Name),
		)

		// exec succeeded so attempt to commit the change
		if commitErr := tx.Commit(); commitErr != nil {
			slog.Warn(
				"failed to commit create table updates",
				slog.String("db", d.name),
				slog.String("table", table.Name),
				slog.String("error", commitErr.Error()),
			)
			err = fmt.Errorf(
				"commit of create table %q in db %q command failed: %w",
				table.Name,
				d.name,
				commitErr,
			)
		}
	} else {
		// exec failed so a roll back will be needed
		slog.Warn(
			"create table failed, rollback will be attempted",
			slog.String("db", d.name),
			slog.String("table", table.Name),
			slog.String("createCmd", createCmd),
			slog.String("error", err.Error()),
		)

		// record that the exec failed
		err = fmt.Errorf(
			"exec of create table %q in db %q command failed: %w",
			table.Name,
			d.name,
			err,
		)
	}

	// if either the exec failed, or it succeeded but the commit failed
	// then check if the table exists, and if so then the failures may
	// have been related to another instance racing to create the same
	// table.
	if err != nil {
		tableExists, checkErr := d.CheckTableExists(table)
		if checkErr != nil {
			slog.Error(
				"table existence check failed after create failed",
				slog.String("db", d.name),
				slog.String("table", table.Name),
				slog.String("error", checkErr.Error()),
			)
			err = fmt.Errorf(
				"failed to create table %q in db %q and unable to check if table exists: %w",
				table.Name,
				d.name,
				checkErr,
			)
		} else if tableExists {
			// table exists, so failure to commit or exec can be ignored
			slog.Info(
				"table exists even though attempt to create it failed, proceeding",
				slog.String("db", d.name),
				slog.String("table", table.Name),
			)
			err = nil
		}
	} else {
		slog.Info(
			"sucessfully created table",
			slog.String("db", d.name),
			slog.String("table", table.Name),
		)
	}

	return
}

func (d *DbConnection) EnsureTableSpecsExist(tables []TableSpec) (err error) {
	slog.Debug("Updating schemas", slog.String("database", d.name))

	for _, table := range tables {
		err = d.CreateTableFromSpec(&table)
	}
	slog.Info("Updated schemas", slog.String("database", d.name))

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

type TableSpecForeignKey struct {
	Name             string
	Column           string
	ReferencedTable  string
	ReferencedColumn string
}

func (fk *TableSpecForeignKey) Create(db *DbConnection) string {
	elements := []string{}

	if fk.Name != "" {
		elements = append(elements, "CONSTRAINT", fk.Name)
	}

	elements = append(
		elements,
		"FOREIGN",
		"KEY",
		"("+fk.Column+")",
		"REFERENCES",
		fk.ReferencedTable,
		"("+fk.ReferencedColumn+")",
	)

	return strings.Join(elements, " ")
}

type TableSpec struct {
	Name        string
	Columns     []TableSpecColumn
	ForeignKeys []TableSpecForeignKey
	Extras      []string

	// TODO: add a sync.Map to hold sql.Prepare()'d statements to
	// eliminate repeated generation of the same SQL statements
}

func (ts *TableSpec) CreateCmd(db *DbConnection) (string, error) {
	table := "CREATE TABLE IF NOT EXISTS " + ts.Name + " ("

	for i, column := range ts.Columns {
		if i > 0 {
			table += ", "
		}
		table += column.Create(db)
	}

	if len(ts.ForeignKeys) > 0 {
		table += ", "
	}

	for i, fk := range ts.ForeignKeys {
		if err := ts.CheckColumnNames([]string{fk.Column}); err != nil {
			return "", fmt.Errorf(
				"foreign key column %q not found in table %q",
				fk.Column,
				ts.Name,
			)
		}

		if i > 0 {
			table += ", "
		}
		table += fk.Create(db)
	}

	if len(ts.Extras) > 0 {
		table += ", "
	}

	for _, extra := range ts.Extras {
		table += " " + extra
	}

	table += ")"

	return table, nil
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
			err = fmt.Errorf("column %q not part of table %q", colName, ts.Name)
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

type TableRowHandler interface {
	//TableRowCommonHandler
	// Setup DB access
	SetupDB(*DbConnection) error

	// Retrieve the TableName
	TableName() string

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
