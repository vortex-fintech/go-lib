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
	"github.com/vortex-fintech/go-lib/db/postgres" // ваш путь к пакету dbpgx (у вас он называется postgres)
)

func TestOpen_Success(t *testing.T) {
	origNewPool := postgres.TestHookSetNewPool(func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
		// создаём real config -> создаём in-memory пул нельзя, поэтому вернём nil + подмена ping вернёт успех
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
		// вернём nil, ping будет вызван с nil — наш хук не использует p
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
	// Сгенерировать *pgconn.PgError вручную
	pgErr := &pgconn.PgError{
		Code:           "23505",
		ConstraintName: "users_credentials_email_key",
	}
	code, constr, ok := postgres.Constraint(pgErr)
	require.True(t, ok)
	require.Equal(t, "23505", code)
	require.Equal(t, "users_credentials_email_key", constr)
	require.True(t, postgres.IsUniqueViolation(pgErr))
}
