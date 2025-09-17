//go:build unit
// +build unit

package dbsql_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/db/dbsql"
)

type stubExec struct{}

func (s *stubExec) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}
func (s *stubExec) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}
func (s *stubExec) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return &sql.Row{}
}

func TestUseExecutor_ReturnsExec_WhenProvided(t *testing.T) {
	exec := &stubExec{}
	db, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	got := dbsql.UseExecutor(db, exec)
	require.Same(t, exec, got)
}

func TestUseExecutor_ReturnsDB_WhenExecNil(t *testing.T) {
	var exec dbsql.Executor
	db, _, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	got := dbsql.UseExecutor(db, exec)
	require.Same(t, db, got)
}
