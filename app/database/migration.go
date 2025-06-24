package database

import (
	"fmt"
)

type DbMigrator func(adb *AppDb) error

type DbVersionMigration struct {
	Version  string
	Date     string
	Migrator DbMigrator
}

type DbMigrationMap map[string]*DbVersionMigration

type DbMigrations []*DbVersionMigration

type DbMigrationState struct {
	Migrations     DbMigrations
	TargetVersion  string
	CurrentVersion string
	VersionIndex   map[string]int
}

func NewDbMigrationState(migrations DbMigrations) *DbMigrationState {
	dms := new(DbMigrationState)
	dms.Init(migrations)
	return dms
}

func (dms *DbMigrationState) Init(migrations DbMigrations) {
	// this shouldn't happen
	if len(migrations) < 1 {
		panic(fmt.Errorf("empty migrations list provided to DbMigrationState.Init()"))
	}

	dms.Migrations = migrations

	// for now we assume the migrations are ordered from oldest to newest
	dms.TargetVersion = migrations[len(migrations)-1].Version

	// we won't know what the current version is until we query the DB
	// to determine the current version
	dms.CurrentVersion = ""

	// initialise the VersionIndex
	dms.VersionIndex = make(map[string]int)
	for i, migration := range migrations {
		dms.VersionIndex[migration.Version] = i
	}
}
