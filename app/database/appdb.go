package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry-server/app/config"
)

type AppDb struct {
	name     string
	dbConn   *DbConnection
	dbTables DbTables
}

func NewAppDb(name string, tables DbTables) (adb *AppDb) {
	adb = new(AppDb)
	adb.Init(name, tables)
	return
}

func (adb *AppDb) Init(name string, tables DbTables) {
	adb.name = name
	adb.dbConn = new(DbConnection)
	adb.dbTables = tables
}

func (adb *AppDb) Name() string {
	return adb.name
}

func (adb *AppDb) String() string {
	return fmt.Sprintf("AppDatabase<%s:%s>", adb.name, adb.dbConn.String())
}

func (adb *AppDb) Setup(dbcfg *config.DBConfig) error {
	return adb.dbConn.Setup(adb.name, dbcfg)
}

func (adb *AppDb) Connect() (err error) {
	if err = adb.dbConn.Connect(); err != nil {
		slog.Error(
			"DB Connect failed",
			slog.String("db", adb.name),
			slog.String("error", err.Error()),
		)
		return
	}

	if err = adb.EnsureTablesExist(); err != nil {
		slog.Error(
			"DB Connect failed",
			slog.String("db", adb.name),
			slog.String("error", err.Error()),
		)
		return
	}

	return
}

func (adb *AppDb) EnsureTablesExist() (err error) {
	slog.Debug("Updating schemas", slog.String("database", adb.name))

	for _, ts := range adb.dbTables {
		err = adb.dbConn.CreateTableFromSpec(ts)
		if err != nil {
			slog.Error(
				"failed to create table from spec",
				slog.String("db", adb.name),
				slog.String("error", err.Error()),
			)
			return
		}
	}
	slog.Info("Updated schemas", slog.String("database", adb.name))

	return
}

func (adb *AppDb) StartTx() (*sql.Tx, error) {
	return adb.Conn().DB().Begin()
}

func (adb *AppDb) RollbackTx(tx *sql.Tx, msg string) {
	err := tx.Rollback()

	// successs
	if err == nil {
		slog.Debug(
			"RollbackTx succeeded",
			slog.String("db", adb.name),
			slog.String("message", msg),
		)
		return
	}

	// transaction or connection already completed
	if errors.Is(err, sql.ErrTxDone) || errors.Is(err, sql.ErrConnDone) {
		slog.Debug(
			"RollbackTx failed",
			slog.String("db", adb.name),
			slog.String("message", msg),
			slog.String("error", err.Error()),
		)
		return
	}

	// an error occurred
	slog.Error(
		"failed to rollback transaction",
		slog.String("db", adb.name),
		slog.String("message", msg),
		slog.String("error", err.Error()),
	)
}

func (adb *AppDb) CommitTx(tx *sql.Tx) (err error) {
	if err = tx.Commit(); err != nil {
		slog.Error(
			"CommitTx failed",
			slog.String("db", adb.name),
			slog.String("error", err.Error()),
		)
		return
	}

	slog.Debug(
		"CommitTx succeeded",
		slog.String("db", adb.name),
	)
	return
}

func (adb *AppDb) Conn() *DbConnection {
	if adb.dbConn != nil {
		return adb.dbConn
	}
	panic(fmt.Errorf("db %q dbConn not initialised", adb.name))
}

func (adb *AppDb) Close() error {
	return adb.Conn().Close()
}

func (adb *AppDb) Ping() error {
	return adb.Conn().Ping()
}

func GetDb(name string, cfg *config.DBConfig, tables DbTables) (*AppDb, error) {
	// create a new AppDb for the names application database using the
	// specified config and tables
	adb := NewAppDb(
		name,
		tables,
	)
	if err := adb.Setup(cfg); err != nil {
		slog.Error(
			"Failed to setup new AppDb",
			slog.String("name", name),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to setup new %s AppDb: %w", name, err)
	}

	return adb, nil
}
