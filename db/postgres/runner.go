package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Runner — единый интерфейс для пула и транзакции.
type Runner interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// poolRunner — реализация Runner для пула.
type poolRunner struct{ p *pgxpool.Pool }

func (r poolRunner) Exec(ctx context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	return r.p.Exec(ctx, q, args...)
}
func (r poolRunner) Query(ctx context.Context, q string, args ...any) (pgx.Rows, error) {
	return r.p.Query(ctx, q, args...)
}
func (r poolRunner) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	return r.p.QueryRow(ctx, q, args...)
}

// txRunner — реализация Runner для транзакции.
type txRunner struct{ tx pgx.Tx }

func (r txRunner) Exec(ctx context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	return r.tx.Exec(ctx, q, args...)
}
func (r txRunner) Query(ctx context.Context, q string, args ...any) (pgx.Rows, error) {
	return r.tx.Query(ctx, q, args...)
}
func (r txRunner) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	return r.tx.QueryRow(ctx, q, args...)
}

// RunnerFromPool — получить Runner для пула (вне транзакции).
func (c *Client) RunnerFromPool() Runner { return poolRunner{p: c.Pool} }
