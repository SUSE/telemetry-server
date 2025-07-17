package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry-server/app/config"
	"github.com/SUSE/telemetry-server/app/database/dbmanager"
	"github.com/SUSE/telemetry-server/app/database/dialect"
)

// DbConnection is a struct tracking a DB connection and associated DB settings
type DbConnection struct {
	name        string
	dbMgr       dbmanager.DbManager
	Placeholder dialect.PlaceholderGenerator
}

func (d DbConnection) Name() string {
	return d.name
}

func (d DbConnection) DbMgr() dbmanager.DbManager {
	return d.dbMgr
}

func (d DbConnection) String() string {
	return fmt.Sprintf("%s:%s", d.name, d.dbMgr.String())
}

func (d DbConnection) Close() (err error) {
	// close the DB
	err = d.dbMgr.Close()
	if err != nil {
		slog.Debug(
			"Failed to close DB connection",
			slog.String("name", d.name),
			slog.String("dbMgr", d.dbMgr.String()),
			slog.String("error", err.Error()),
		)
		return
	}

	slog.Info(
		"DB closed",
		slog.String("name", d.name),
		slog.String("dbMgr", d.dbMgr.String()),
	)

	return
}

func (d *DbConnection) Setup(name string, dbcfg *config.DBConfig) error {
	dbMgr, err := dbmanager.New(dbcfg.Driver, dbcfg.Params)
	if err != nil {
		return err
	}

	d.name = name
	d.dbMgr = dbMgr

	switch {
	case d.dbMgr.Type().IsPostgres():
		// postgres uses `$1`, `$2`, ... as placeholders
		d.Placeholder = dialect.DollarCounter
	case d.dbMgr.Type().IsSqlite3():
		// sqlite3 uses `?` as placeholder
		d.Placeholder = dialect.QuestionMarker
	}

	return err
}

func (d *DbConnection) Connect() (err error) {
	slog.Debug("Connecting to DB", slog.String("name", d.name))

	// connect to the specified DB
	err = d.dbMgr.Connect()
	if err != nil {
		slog.Error(
			"db manager connect failed",
			slog.String("name", d.name),
			slog.String("dbMgr", d.dbMgr.String()),
			slog.String("Error", err.Error()),
		)
		return
	}

	err = d.dbMgr.Ping()
	if err != nil {
		slog.Error(
			"db ping after connect failed",
			slog.String("name", d.name),
			slog.String("dbMgr", d.dbMgr.String()),
			slog.String("Error", err.Error()),
		)
		return
	}

	slog.Info("Connected to DB", slog.String("name", d.name))

	return
}

func (d *DbConnection) Ping() (err error) {
	if err = d.dbMgr.Ping(); err != nil {
		slog.Error(
			"db ping failed",
			slog.String("db", d.name),
			slog.String("dbMgr", d.dbMgr.String()),
			slog.String("Error", err.Error()),
		)
	}

	return
}
func (d *DbConnection) DB() *sql.DB {
	if d.dbMgr != nil {
		return d.dbMgr.DB()
	}
	panic(fmt.Errorf("attempted to access uninitialised %q DB", d.name))
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
		_, err = d.DB().Exec(lockStmt)
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
	switch {
	case d.dbMgr.Type().IsPostgres():
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
	_, err = d.DB().Exec(unlockStmt)
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
	switch {
	case d.dbMgr.Type().IsPostgres():
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
	row := d.DB().QueryRow(existsStmt, table.Name)
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
	tx, err := d.DB().Begin()
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

	// defer releasing the advisory lock
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

func (d *DbConnection) EnsureTableSpecsExist(tables []*TableSpec) (err error) {
	slog.Debug("Updating schemas", slog.String("database", d.name))

	for _, table := range tables {
		err = d.CreateTableFromSpec(table)
	}
	slog.Info("Updated schemas", slog.String("database", d.name))

	return
}
