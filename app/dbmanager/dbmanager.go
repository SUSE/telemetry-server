package dbmanager

import (
	"database/sql"
	"fmt"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

// Supported DB Types
type DbType int

const (
	DB_TYPE_UNKNOWN DbType = iota

	// valid DB types
	DB_TYPE_PGX
	DB_TYPE_POSTGRES
	DB_TYPE_SQLITE3
)

// DB Type ==> Name
var dbType2Name = map[DbType]string{
	DB_TYPE_PGX:      "pgx",
	DB_TYPE_POSTGRES: "postgres",
	DB_TYPE_SQLITE3:  "sqlite3",
}

func (t DbType) String() string {
	typeName, found := dbType2Name[t]
	if !found {
		typeName = "UNKNOWN"
	}

	return typeName
}

// DB Type ==> DB Driver to use
var dbType2DbDriver = map[DbType]string{
	DB_TYPE_PGX:      "pgx",
	DB_TYPE_POSTGRES: "pgx",
	DB_TYPE_SQLITE3:  "sqlite3",
}

func (t DbType) DbDriver() (dbDriver string, err error) {
	dbDriver, found := dbType2DbDriver[t]
	if !found {
		err = fmt.Errorf("failed to determine DbDriver for %q", t)
	}
	return
}

// DB Type flavour checks
func (t DbType) IsPostgres() (isPostgres bool) {
	switch t {
	case DB_TYPE_PGX:
		fallthrough
	case DB_TYPE_POSTGRES:
		isPostgres = true
	}
	return
}

func (t DbType) IsSqlite3() (isSqlite3 bool) {
	return t == DB_TYPE_SQLITE3
}

var dbDriver2Type = map[string]DbType{
	"pgx":      DB_TYPE_PGX,
	"postgres": DB_TYPE_POSTGRES,
	"sqlite":   DB_TYPE_SQLITE3,
	"sqlite3":  DB_TYPE_SQLITE3,
}

type newDbManager func(dbType DbType, dataSource string) DbManager

var dbDriver2NewMgr = map[DbType]newDbManager{
	DB_TYPE_PGX:      NewPgxPoolManager,
	DB_TYPE_POSTGRES: NewPostgresManager,
	DB_TYPE_SQLITE3:  NewSqlite3Manager,
}

func New(driver, dataSource string) (dbMgr DbManager, err error) {
	// attempt to lookup driver in map
	dbType, found := dbDriver2Type[driver]
	if !found {
		err = fmt.Errorf("failed to determine a DbType for %q", driver)
		return
	}

	newDbMgr, found := dbDriver2NewMgr[dbType]
	if !found {
		err = fmt.Errorf("failed to determine a DbManager for %q", driver)
		return
	}

	// allocate a DbManager for the driver
	dbMgr = newDbMgr(dbType, dataSource)

	return
}

const (
	POSTGRES_CONN_SCALE int32 = 4
	POSTGRES_CONN_MAX   int32 = 30
	POSTGRES_CONN_IDLE  int32 = 2
	POSTGRES_CONN_MIN   int32 = 2
)

// max open connections, scaled down by system size
func postgresMaxOpenConns() int32 {
	return min(
		POSTGRES_CONN_MAX,
		int32(runtime.NumCPU())*POSTGRES_CONN_SCALE,
	)
}

// Interface for managing DB specific resouces
type DbManager interface {
	String() string
	Init(dbType DbType, dataSource string, manager string)
	Type() DbType
	DB() *sql.DB
	Connect() error
	Ping() error
	Close() error
}
