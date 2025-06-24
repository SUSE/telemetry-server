package database

import (
	"database/sql"
	"fmt"
	"log/slog"
)

type TableRowCommon struct {
	// private db settings
	db        *AppDb
	tableSpec *TableSpec
}

func (t *TableRowCommon) SetTableSpec(ts *TableSpec) {
	t.tableSpec = ts
}

func (t *TableRowCommon) GetTableSpec() *TableSpec {
	if t.tableSpec == nil {
		err := fmt.Errorf("attempt to access unitialised TableRowCommon.tableSpec")
		slog.Error(
			"TableRowCommon.tableSpec not yet setup",
			slog.String("error", err.Error()),
		)
		panic(err)
	}

	return t.tableSpec
}

func (t *TableRowCommon) SetupDB(adb *AppDb) (err error) {
	// tableSpec should have already been initialised
	if t.tableSpec == nil {
		return fmt.Errorf("tableSpec should be initialised before calling SetupDB")
	}

	t.db = adb

	return
}

func (t *TableRowCommon) DB() *sql.DB {
	// SetupDB should have already been called
	if t.db == nil {
		err := fmt.Errorf("%q row db connection not setup", t.TableName())
		slog.Error("TableRowCommon.db not setup", slog.String("error", err.Error()))
		panic(err)
	}

	return t.db.Conn().DB()
}

func (t *TableRowCommon) TableName() string {
	return t.GetTableSpec().Name
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
	Count      bool
	Distinct   bool
	Limit      uint
	OrderBy    string
	Descending bool
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

	if len(opts.OrderBy) > 0 {
		// ensure whereCols are valid
		if err = t.tableSpec.CheckColumnNames([]string{opts.OrderBy}); err != nil {
			return "", fmt.Errorf("invalid orderby column: %w", err)
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
		ph := t.db.Conn().Placeholder(len(whereCols))

		// add where conditions
		for i, whereCol := range whereCols {
			if i > 0 {
				stmt += " AND "
			}

			// add where clause with appropriate placeholder
			stmt += whereCol + " = " + ph.Next()
		}
	}

	// add an order by directive if specified
	if opts.OrderBy != "" {
		stmt += fmt.Sprintf(" ORDER BY %s", opts.OrderBy)
		if opts.Descending {
			stmt += " DESC"
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
	ph := t.db.Conn().Placeholder(len(insertCols))

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
	ph := t.db.Conn().Placeholder(len(updateCols) + len(whereCols))

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
	ph := t.db.Conn().Placeholder(len(whereCols))

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
	// Set the associated TableSpec
	SetTableSpec(ts *TableSpec)

	// Get the associated TableSpec
	GetTableSpec() *TableSpec

	// Setup DB access
	SetupDB(*AppDb) error

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
