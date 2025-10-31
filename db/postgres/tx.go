package postgres

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// TxConfig — дополнительные настройки транзакции.
type TxConfig struct {
	Iso        pgx.TxIsoLevel // default: ReadCommitted
	ReadOnly   bool           // default: false
	Deferrable bool           // работает только с SERIALIZABLE (в PG имеет смысл для read-only)

	// Локальные таймауты (SET LOCAL ...), если нужны на время TX
	StatementTimeout         time.Duration // общий timeout каждого стейтмента
	IdleInTransactionTimeout time.Duration // idle_in_transaction_session_timeout
}

// WithTx — безопасная (panic-safe) транзакция с дефолтными опциями (ReadCommitted, RW).
func (c *Client) WithTx(ctx context.Context, fn func(run Runner) error) (err error) {
	return c.WithTxOpts(ctx, TxConfig{}, fn)
}

// WithTxRO — Read-Only транзакция (для консистентных чтений в несколько запросов).
func (c *Client) WithTxRO(ctx context.Context, fn func(run Runner) error) error {
	return c.WithTxOpts(ctx, TxConfig{ReadOnly: true}, fn)
}

// WithSerializable — SERIALIZABLE + авто-ретраи на 40001 (serialization_failure), учитывает ctx.
func (c *Client) WithSerializable(ctx context.Context, maxRetries int, fn func(run Runner) error) error {
	if maxRetries < 1 {
		maxRetries = 3
	}
	var last error
	for attempt := 0; attempt < maxRetries; attempt++ {
		last = c.WithTxOpts(ctx, TxConfig{Iso: pgx.Serializable}, fn)
		if !isSerializationFailure(last) {
			return last
		}
		// Небольшой джиттер между повторами, прерываемся по ctx
		d := time.Duration(25+rand.Intn(50)) * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
	return last
}

// Удобный хелпер: SERIALIZABLE + ReadOnly + (опц.) DEFERRABLE
func (c *Client) WithSerializableRO(ctx context.Context, deferrable bool, fn func(run Runner) error) error {
	return c.WithTxOpts(ctx, TxConfig{
		Iso:        pgx.Serializable,
		ReadOnly:   true,
		Deferrable: deferrable,
	}, fn)
}

func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "40001"
}

// WithTxOpts — транзакция с опциями + panic-safe commit/rollback + локальные SET LOCAL таймауты.
func (c *Client) WithTxOpts(ctx context.Context, cfg TxConfig, fn func(run Runner) error) (err error) {
	opts := pgx.TxOptions{
		IsoLevel:   cfg.Iso,
		AccessMode: pgx.ReadWrite,
	}
	if cfg.ReadOnly {
		opts.AccessMode = pgx.ReadOnly
	}

	tx, err := c.Pool.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	// panic-safe; корректно закрываем транзакцию
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return
		}
		err = tx.Commit(ctx)
	}()

	// DEFERRABLE — в pgx/v5 задаётся отдельной командой.
	if cfg.Deferrable {
		if cfg.Iso != pgx.Serializable {
			return fmt.Errorf("DEFERRABLE допустим только при SERIALIZABLE")
		}
		if !cfg.ReadOnly {
			return fmt.Errorf("DEFERRABLE имеет смысл только в read-only транзакции")
		}
		if _, e := tx.Exec(ctx, "SET TRANSACTION DEFERRABLE"); e != nil {
			return e
		}
	}

	// Локальные таймауты на время текущей TX
	if cfg.StatementTimeout > 0 {
		ms := cfg.StatementTimeout.Milliseconds()
		if _, e := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", ms)); e != nil {
			return e
		}
	}
	if cfg.IdleInTransactionTimeout > 0 {
		ms := cfg.IdleInTransactionTimeout.Milliseconds()
		if _, e := tx.Exec(ctx, fmt.Sprintf("SET LOCAL idle_in_transaction_session_timeout = %d", ms)); e != nil {
			return e
		}
	}

	run := txRunner{tx: tx}
	err = fn(run)
	return err
}

// WithSavepoint — если уже внутри транзакции, создаёт SAVEPOINT; иначе откроет обычную TX.
func (c *Client) WithSavepoint(ctx context.Context, run Runner, fn func(run Runner) error) error {
	// Уже в транзакции?
	if tx, ok := asTx(run); ok {
		sp := fmt.Sprintf("sp_%d", time.Now().UnixNano())
		if _, err := tx.Exec(ctx, "SAVEPOINT "+sp); err != nil {
			return err
		}
		if err := fn(txRunner{tx: tx}); err != nil {
			_, _ = tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+sp)
			_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT "+sp) // можно опустить
			return err
		}
		_, _ = tx.Exec(ctx, "RELEASE SAVEPOINT "+sp)
		return nil
	}
	// Вне транзакции — обычная WithTx.
	return c.WithTx(ctx, fn)
}

// asTx — вытащить pgx.Tx из Runner, если это txRunner.
func asTx(run Runner) (pgx.Tx, bool) {
	if t, ok := run.(txRunner); ok {
		return t.tx, true
	}
	return nil, false
}
