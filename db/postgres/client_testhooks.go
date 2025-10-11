//go:build testhooks
// +build testhooks

package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Возвращает предыдущий хук, чтобы можно было восстановить в t.Cleanup.
func TestHookSetNewPool(fn func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error)) func(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
	old := newPool
	newPool = fn
	return old
}

func TestHookSetPingPool(fn func(ctx context.Context, p *pgxpool.Pool) error) func(ctx context.Context, p *pgxpool.Pool) error {
	old := pingPool
	pingPool = fn
	return old
}
