//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/data/postgres"
)

func TestWithTx_RollbackOnError_Integration(t *testing.T) {
	c := openIntegrationClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.RunnerFromPool().Exec(ctx, "CREATE TABLE IF NOT EXISTS tx_test (id BIGINT PRIMARY KEY)")
	require.NoError(t, err)

	id := time.Now().UnixNano()
	err = c.WithTx(ctx, func(txCtx context.Context) error {
		run := postgres.MustRunnerFromContext(txCtx)
		_, e := run.Exec(txCtx, "INSERT INTO tx_test(id) VALUES($1)", id)
		require.NoError(t, e)
		return fmt.Errorf("force rollback")
	})
	require.Error(t, err)

	row := c.RunnerFromPool().QueryRow(ctx, "SELECT count(*) FROM tx_test WHERE id=$1", id)
	var cnt int
	require.NoError(t, row.Scan(&cnt))
	require.Equal(t, 0, cnt)
}

func TestWithSerializable_RetriesOnSerializationFailure_Integration(t *testing.T) {
	c := openIntegrationClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	attempts := 0
	err := c.WithSerializable(ctx, 3, func(txCtx context.Context) error {
		attempts++
		if attempts < 3 {
			return &pgconn.PgError{Code: "40001", Message: "serialization_failure"}
		}
		run := postgres.MustRunnerFromContext(txCtx)
		_, e := run.Exec(txCtx, "SELECT 1")
		return e
	})
	require.NoError(t, err)
	require.Equal(t, 3, attempts)
}

func openIntegrationClient(t *testing.T) *postgres.Client {
	t.Helper()

	cfg := postgres.Config{URL: "postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable", MaxConns: 5, MinConns: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := postgres.Open(ctx, cfg)
	require.NoError(t, err)
	return c
}
