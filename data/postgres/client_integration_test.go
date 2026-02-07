//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/data/postgres"
)

func TestOpen_Integration(t *testing.T) {
	// docker-compose: порт 5433
	cfg := postgres.Config{
		URL: "postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable",
		// можно задать лимиты пула
		MaxConns: 5, MinConns: 1,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := postgres.Open(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, c)
	defer c.Close()

	// простой sanity check
	run := c.RunnerFromPool()
	row := run.QueryRow(ctx, "SELECT 1")
	var x int
	require.NoError(t, row.Scan(&x))
	require.Equal(t, 1, x)
}
