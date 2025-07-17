package database

import (
	"fmt"
	"log/slog"

	"github.com/SUSE/telemetry-server/app/config"
)

type AppDb struct {
	name         string
	dbConn       *DbConnection
	dbTables     DbTables
	dbMigrations DbMigrations
}

func NewAppDb(name string, tables DbTables, migrations DbMigrations) (adb *AppDb) {
	adb = new(AppDb)
	adb.Init(name, tables, migrations)
	return
}

func (adb *AppDb) Init(name string, tables DbTables, migrations DbMigrations) {
	adb.name = name
	adb.dbConn = new(DbConnection)
	adb.dbTables = tables
	adb.dbMigrations = migrations
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

func (adb *AppDb) PerformDbMigration() (err error) {
	// TODO: this work should be done within a transaction holding an advisory lock

	dms := NewDbMigrationState(adb.dbMigrations)
	dbVerRow := new(DbVersionRow)

	if err = dbVerRow.SetupDB(adb); err != nil {
		return fmt.Errorf("failed to setup db %q for dbversions query: %w", adb.Name(), err)
	}

	if err = dbVerRow.LastRow(); err != nil {
		slog.Error(
			"Failed to retrieve the last row from dbVersions table, assuming empty",
			slog.String("db", adb.Name()),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to retrieve dbversion last row for db %q: %w", adb.Name(), err)
	}

	if dbVerRow.Version == "" {
		// this is a fresh database so tables will be created using latest
		// schema definitions so just add an entry for the target version
		migration := dms.Migrations[dms.VersionIndex[dms.TargetVersion]]
		slog.Info(
			"Initialising dbversions with latest version",
			slog.String("db", adb.Name()),
			slog.String("version", migration.Version),
			slog.String("date", migration.Date),
		)
		dbVerRow.Version = migration.Version
		dbVerRow.Date = migration.Date
		return dbVerRow.Insert()
	}

	// version on last row should exist in migrations list for this DB
	verInd, found := dms.VersionIndex[dbVerRow.Version]
	if !found {
		return fmt.Errorf("retrieved migration version %q not found in migrations table for db %q", dbVerRow.Version, adb.Name())
	}

	for i := verInd + 1; i < len(dms.Migrations); i++ {
		migration := dms.Migrations[i]
		if err = migration.Migrator(adb); err != nil {
			return fmt.Errorf("failed to perform db %q migration %q: %w", adb.Name(), dbVerRow.Version, err)
		}

		dbVerRow.Version = migration.Version
		dbVerRow.Date = migration.Date
		if err = dbVerRow.Insert(); err != nil {
			return fmt.Errorf("failed to insert %q db dbversions row for version %q: %w", adb.Name(), dbVerRow.Version, err)
		}
	}

	return
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

	if err = adb.PerformDbMigration(); err != nil {
		slog.Error(
			"DB Migration",
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

func GetDb(name string, cfg *config.DBConfig, tables DbTables, migrations DbMigrations) (*AppDb, error) {
	// create a new AppDb for the names application database using the
	// specified config and tables
	adb := NewAppDb(
		name,
		tables,
		migrations,
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
