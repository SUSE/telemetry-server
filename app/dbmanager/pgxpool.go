package dbmanager

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

// pgxpool specific variant of SqlDbManager
type PgxPoolManager struct {
	SqlDbManager
	pool *pgxpool.Pool
}

func NewPgxPoolManager(dbType DbType, dataSource string) DbManager {
	m := new(PgxPoolManager)
	m.Init(dbType, dataSource, "pgxpool")

	return m
}

func (m *PgxPoolManager) Type() DbType {
	return m.dbType
}

func (m *PgxPoolManager) Connect() (err error) {
	cfg, err := pgxpool.ParseConfig(m.dataSource)
	if err != nil {
		slog.Debug(
			"Failed to parse pgxpool config",
			slog.String("manager", m.manager),
			slog.String("dbType", m.dbType.String()),
			slog.String("dataSource", m.dataSource),
			slog.String("error", err.Error()),
		)
		return
	}

	// limit the number of open connection, without overloading system
	cfg.MaxConns = postgresMaxOpenConns()

	// attempt to have at least this number of connections available
	cfg.MinConns = POSTGRES_CONN_MIN

	// attempt to have at this number of idle connections available
	cfg.MinIdleConns = POSTGRES_CONN_IDLE

	// debug log connection acquisition and release
	cfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		slog.Debug(
			"BeforeAcquire() called",
			slog.String("manager", m.manager),
			slog.String("dbType", m.dbType.String()),
			slog.String("dataSource", m.dataSource),
		)
		return true
	}
	cfg.AfterRelease = func(conn *pgx.Conn) bool {
		slog.Debug(
			"AfterRelease() called",
			slog.String("manager", m.manager),
			slog.String("dbType", m.dbType.String()),
			slog.String("dataSource", m.dataSource),
		)
		return true
	}

	// create a new pool using the parsed config
	m.pool, err = pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		slog.Debug(
			"Failed to setup pgxpool",
			slog.String("manager", m.manager),
			slog.String("dbType", m.dbType.String()),
			slog.String("dataSource", m.dataSource),
			slog.String("error", err.Error()),
		)
		return
	}

	// create an sql.DB from the pool
	m.db = stdlib.OpenDBFromPool(m.pool)

	return
}

func (m *PgxPoolManager) Close() (err error) {
	// attempt to close the active DB connections
	err = m.SqlDbManager.Close()
	if err != nil {
		return
	}

	// close the pool waiting for connextions to drain
	m.pool.Close()

	return
}

// verify that PgxPoolManager conforms to the DbManager interface
var _ DbManager = (*PgxPoolManager)(nil)
