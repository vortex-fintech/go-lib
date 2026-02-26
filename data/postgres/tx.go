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

const (
	sqlStateSerializationFailure = "40001"
	sqlStateDeadlockDetected     = "40P01"
	txCleanupTimeout             = 5 * time.Second
)

var (
	errNilTxCallback = errors.New("postgres: tx callback is nil")
	errNilClientPool = errors.New("postgres: client pool is nil")
)

// TxConfig contains optional transaction settings.
type TxConfig struct {
	Iso        pgx.TxIsoLevel // default: ReadCommitted
	ReadOnly   bool           // default: false
	Deferrable bool           // valid only for SERIALIZABLE; meaningful for read-only

	// Local timeouts for current TX (SET LOCAL ...).
	StatementTimeout         time.Duration // statement timeout
	IdleInTransactionTimeout time.Duration // idle_in_transaction_session_timeout
}

// WithTx runs panic-safe read-write transaction with default options.
// Runner is available via RunnerFromContext(ctx) / MustRunnerFromContext(ctx).
func (c *Client) WithTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	return c.WithTxOpts(ctx, TxConfig{}, fn)
}

// WithTxRO runs read-only transaction (for consistent multi-query reads).
func (c *Client) WithTxRO(ctx context.Context, fn func(ctx context.Context) error) error {
	return c.WithTxOpts(ctx, TxConfig{ReadOnly: true}, fn)
}

// WithTxExplicit is backward-compatible variant with explicit runner callback arg.
func (c *Client) WithTxExplicit(ctx context.Context, fn func(run Runner) error) error {
	return c.WithTx(ctx, func(txCtx context.Context) error {
		return fn(MustRunnerFromContext(txCtx))
	})
}

// WithTxROExplicit is backward-compatible read-only explicit-runner variant.
func (c *Client) WithTxROExplicit(ctx context.Context, fn func(run Runner) error) error {
	return c.WithTxRO(ctx, func(txCtx context.Context) error {
		return fn(MustRunnerFromContext(txCtx))
	})
}

// WithSerializable runs SERIALIZABLE tx with retries for 40001/40P01 and ctx awareness.
func (c *Client) WithSerializable(ctx context.Context, maxRetries int, fn func(ctx context.Context) error) error {
	if maxRetries < 1 {
		maxRetries = 3
	}
	var last error
	for attempt := 0; attempt < maxRetries; attempt++ {
		last = c.WithTxOpts(ctx, TxConfig{Iso: pgx.Serializable}, fn)
		if !isRetriableTxError(last) {
			return last
		}
		// Small jitter between retries, stop if context is done.
		d := time.Duration(25+rand.Intn(50)) * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
	return last
}

// Convenience helper: SERIALIZABLE + ReadOnly + optional DEFERRABLE.
func (c *Client) WithSerializableRO(ctx context.Context, deferrable bool, fn func(ctx context.Context) error) error {
	return c.WithTxOpts(ctx, TxConfig{
		Iso:        pgx.Serializable,
		ReadOnly:   true,
		Deferrable: deferrable,
	}, fn)
}

func isSerializationFailure(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == sqlStateSerializationFailure
}

func isDeadlockDetected(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == sqlStateDeadlockDetected
}

func isRetriableTxError(err error) bool {
	return isSerializationFailure(err) || isDeadlockDetected(err)
}

// WithTxOpts runs transaction with options, panic-safe commit/rollback,
// and optional SET LOCAL timeouts.
func (c *Client) WithTxOpts(ctx context.Context, cfg TxConfig, fn func(ctx context.Context) error) (err error) {
	if fn == nil {
		return errNilTxCallback
	}
	if c == nil || c.Pool == nil {
		return errNilClientPool
	}

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

	// Panic-safe transaction closing.
	defer func() {
		if p := recover(); p != nil {
			_ = rollbackWithTimeout(tx)
			panic(p)
		}
		if err != nil {
			rbErr := rollbackWithTimeout(tx)
			if rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, fmt.Errorf("postgres: rollback failed: %w", rbErr))
			}
			return
		}
		err = tx.Commit(ctx)
	}()

	// DEFERRABLE is set via a dedicated command in pgx/v5.
	if cfg.Deferrable {
		if cfg.Iso != pgx.Serializable {
			return fmt.Errorf("DEFERRABLE is allowed only with SERIALIZABLE")
		}
		if !cfg.ReadOnly {
			return fmt.Errorf("DEFERRABLE is meaningful only for read-only transaction")
		}
		if _, e := tx.Exec(ctx, "SET TRANSACTION DEFERRABLE"); e != nil {
			return e
		}
	}

	// Local timeouts for the current transaction.
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
	txCtx := ContextWithRunner(ctx, run)
	err = fn(txCtx)
	return err
}

// WithSavepoint creates SAVEPOINT when already in tx, otherwise starts regular tx.
func (c *Client) WithSavepoint(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return errNilTxCallback
	}

	// Already in transaction?
	if r, ok := ctx.Value(ctxKeyRunner{}).(Runner); ok {
		if tx, ok := asTx(r); ok {
			sp := fmt.Sprintf("sp_%d", time.Now().UnixNano())
			if _, err := tx.Exec(ctx, "SAVEPOINT "+sp); err != nil {
				return err
			}
			spCtx := ContextWithRunner(ctx, txRunner{tx: tx})
			if err := fn(spCtx); err != nil {
				rbErr := execWithTimeout(tx, "ROLLBACK TO SAVEPOINT "+sp)
				releaseErr := execWithTimeout(tx, "RELEASE SAVEPOINT "+sp)
				var out []error
				out = append(out, err)
				if rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
					out = append(out, fmt.Errorf("postgres: rollback to savepoint failed: %w", rbErr))
				}
				if releaseErr != nil && !errors.Is(releaseErr, pgx.ErrTxClosed) {
					out = append(out, fmt.Errorf("postgres: release savepoint failed: %w", releaseErr))
				}
				if len(out) == 1 {
					return out[0]
				}
				return errors.Join(out...)
			}
			if err := execWithTimeout(tx, "RELEASE SAVEPOINT "+sp); err != nil {
				return err
			}
			return nil
		}
	}
	// Outside transaction: use regular WithTx.
	return c.WithTx(ctx, fn)
}

// asTx extracts pgx.Tx from Runner when possible.
func asTx(run Runner) (pgx.Tx, bool) {
	type rawTxProvider interface{ RawTx() pgx.Tx }
	if t, ok := run.(rawTxProvider); ok {
		tx := t.RawTx()
		if tx != nil {
			return tx, true
		}
	}
	if t, ok := run.(txRunner); ok {
		return t.tx, true
	}
	if t, ok := run.(*txRunner); ok {
		return t.tx, true
	}
	return nil, false
}

func rollbackWithTimeout(tx pgx.Tx) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), txCleanupTimeout)
	defer cancel()
	return tx.Rollback(cleanupCtx)
}

func execWithTimeout(tx pgx.Tx, sql string, args ...any) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), txCleanupTimeout)
	defer cancel()
	_, err := tx.Exec(cleanupCtx, sql, args...)
	return err
}
