package dbsql

import (
	"context"
	"database/sql"
)

// Executor abstracts *sql.DB or *sql.Tx
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// UseExecutor returns exec if provided, otherwise returns db.
// Helps unify usage of *sql.DB and *sql.Tx.
func UseExecutor(db *sql.DB, exec Executor) Executor {
	if exec != nil {
		return exec
	}
	return db
}
