package dbmanager

import (
	_ "github.com/mattn/go-sqlite3"
)

// Sqlite3 specific variant of SqlDbManager
type Sqlite3Manager struct {
	SqlDbManager
}

func NewSqlite3Manager(dbType DbType, dataSource string) DbManager {
	m := new(Sqlite3Manager)
	m.Init(dbType, dataSource, "sqlite3")

	return m
}

// verify that Sqlite3Manager conforms to the DbManager interface
var _ DbManager = (*Sqlite3Manager)(nil)
