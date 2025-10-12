//go:build unit && testhooks
// +build unit,testhooks

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/db/postgres"
)

func TestOpen_Success(t *testing.T) {
	origNewPool := postgres.TestHookSetNewPool(func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
		// вернём nil — ping заменён хуком ниже
		return (*pgxpool.Pool)(nil), nil
	})
	origPing := postgres.TestHookSetPingPool(func(ctx context.Context, p *pgxpool.Pool) error { return nil })
	t.Cleanup(func() {
		postgres.TestHookSetNewPool(origNewPool)
		postgres.TestHookSetPingPool(origPing)
	})

	cfg := postgres.Config{URL: "postgres://u:p@h:5432/d?sslmode=disable"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	c, err := postgres.Open(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, c)
	c.Close()
}

func TestOpen_NewPoolError(t *testing.T) {
	origNewPool := postgres.TestHookSetNewPool(func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
		return nil, errors.New("newpool failed")
	})
	origPing := postgres.TestHookSetPingPool(func(ctx context.Context, p *pgxpool.Pool) error { return nil })
	t.Cleanup(func() {
		postgres.TestHookSetNewPool(origNewPool)
		postgres.TestHookSetPingPool(origPing)
	})

	cfg := postgres.Config{URL: "postgres://u:p@h:5432/d?sslmode=disable"}
	_, err := postgres.Open(context.Background(), cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "newpool failed")
}

func TestOpen_PingError(t *testing.T) {
	origNewPool := postgres.TestHookSetNewPool(func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
		return (*pgxpool.Pool)(nil), nil
	})
	origPing := postgres.TestHookSetPingPool(func(ctx context.Context, p *pgxpool.Pool) error { return errors.New("ping failed") })
	t.Cleanup(func() {
		postgres.TestHookSetNewPool(origNewPool)
		postgres.TestHookSetPingPool(origPing)
	})

	cfg := postgres.Config{URL: "postgres://u:p@h:5432/d?sslmode=disable"}
	_, err := postgres.Open(context.Background(), cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ping failed")
}

func TestConstraint_And_Unique(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:           postgres.SQLStateUniqueViolation,
		ConstraintName: "users_credentials_email_key",
	}

	info, ok := postgres.Constraint(pgErr)
	require.True(t, ok)
	require.Equal(t, postgres.SQLStateUniqueViolation, info.Code)
	require.Equal(t, "users_credentials_email_key", info.Name)
	require.True(t, info.IsUnique)

	// заодно проверим шорткат-хелпер
	require.True(t, postgres.IsUniqueViolation(pgErr))
}
