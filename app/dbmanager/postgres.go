package dbmanager

import (
	_ "github.com/mattn/go-sqlite3"
)

// Postgres specific variant of SqlDbManager
type PostgresManager struct {
	SqlDbManager
}

func NewPostgresManager(dbType DbType, dataSource string) DbManager {
	m := new(PostgresManager)
	m.Init(dbType, dataSource, "postgres")

	return m
}

func (m *PostgresManager) Connect() (err error) {
	err = m.SqlDbManager.Connect()
	if err != nil {
		return
	}

	// configure connection pool management
	m.db.SetMaxOpenConns(int(postgresMaxOpenConns()))
	m.db.SetMaxIdleConns(int(POSTGRES_CONN_IDLE))

	return
}

// verify that PostgresManager conforms to the DbManager interface
var _ DbManager = (*PostgresManager)(nil)
