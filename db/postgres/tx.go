package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// WithTx — обёртка для атомарных операций в транзакции.
func (c *Client) WithTx(ctx context.Context, fn func(run Runner) error) error {
	tx, err := c.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	run := txRunner{tx: tx}
	if err := fn(run); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}
