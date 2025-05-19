package dbmanager

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/mattn/go-sqlite3"
)

// general database/sql DB Manager
type SqlDbManager struct {
	dbType     DbType
	dataSource string
	manager    string
	db         *sql.DB
}

func NewSqlDbManager(dbType DbType, dataSource string, manager string) DbManager {
	m := new(SqlDbManager)
	m.Init(dbType, dataSource, manager)

	return m
}

func (m *SqlDbManager) String() string {
	return fmt.Sprintf("<%s:%s:%s>", m.manager, m.dbType.String(), m.dataSource)
}

func (m *SqlDbManager) Init(dbType DbType, dataSource string, manager string) {
	m.dbType = dbType
	m.dataSource = dataSource
	m.manager = manager
}

func (m *SqlDbManager) Type() DbType {
	return m.dbType
}

func (m *SqlDbManager) DB() *sql.DB {
	return m.db
}

func (m *SqlDbManager) Connect() (err error) {
	// determine which DB Driver to use
	dbDriver, err := m.dbType.DbDriver()
	if err != nil {
		slog.Error(
			"dbDriver lookup failed",
			slog.String("manager", m.manager),
			slog.String("dbType", m.dbType.String()),
			slog.String("dataSource", m.dataSource),
			slog.String("Error", err.Error()),
		)
		return
	}

	// connect to SQL DB using the provided driver and dataSource
	db, err := sql.Open(dbDriver, m.dataSource)
	if err != nil {
		slog.Error(
			"sql DB open failed",
			slog.String("manager", m.manager),
			slog.String("driver", m.dbType.String()),
			slog.String("dataSource", m.dataSource),
			slog.String("Error", err.Error()),
		)
		return
	}

	m.db = db

	return
}

func (m *SqlDbManager) Ping() (err error) {
	// ping the DB to ensure connection is valid
	err = m.db.Ping()
	if err != nil {
		slog.Error(
			"sql DB ping failed",
			slog.String("manager", m.manager),
			slog.String("dbType", m.dbType.String()),
			slog.String("dataSource", m.dataSource),
			slog.String("Error", err.Error()),
		)
		return
	}

	return
}

func (m *SqlDbManager) Close() (err error) {
	err = m.db.Close()
	if err != nil {
		slog.Error(
			"Failed to close sql DB",
			slog.String("manager", m.manager),
			slog.String("dbType", m.dbType.String()),
			slog.String("dataSource", m.dataSource),
			slog.String("Error", err.Error()),
		)
	}
	return
}

// verify that SqlDbManager conforms to the DbManager interface
var _ DbManager = (*SqlDbManager)(nil)
