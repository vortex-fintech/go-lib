package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsSerializationFailure(t *testing.T) {
	t.Parallel()

	if !isSerializationFailure(&pgconn.PgError{Code: sqlStateSerializationFailure}) {
		t.Fatalf("expected true for SQLSTATE 40001")
	}
	if isSerializationFailure(&pgconn.PgError{Code: "23505"}) {
		t.Fatalf("expected false for non-serialization SQLSTATE")
	}
	wrapped := errors.New("wrapper: " + (&pgconn.PgError{Code: sqlStateSerializationFailure}).Error())
	if isSerializationFailure(wrapped) {
		t.Fatalf("expected false for non-wrapped plain error")
	}
}

func TestIsRetriableTxError(t *testing.T) {
	t.Parallel()

	if !isRetriableTxError(&pgconn.PgError{Code: sqlStateSerializationFailure}) {
		t.Fatalf("expected retriable for serialization failure")
	}
	if !isRetriableTxError(&pgconn.PgError{Code: sqlStateDeadlockDetected}) {
		t.Fatalf("expected retriable for deadlock")
	}
	if isRetriableTxError(&pgconn.PgError{Code: "23505"}) {
		t.Fatalf("expected non-retriable for unique violation")
	}
}

func TestWithSavepoint_Success(t *testing.T) {
	t.Parallel()

	tx := &txStub{}
	c := &Client{}
	run := txRunner{tx: tx}
	txCtx := ContextWithRunner(context.Background(), run)

	err := c.WithSavepoint(txCtx, func(ctx context.Context) error {
		_, e := MustRunnerFromContext(ctx).Exec(ctx, "SELECT 1")
		return e
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tx.execs) < 3 {
		t.Fatalf("expected savepoint, work, release; got %v", tx.execs)
	}
	if !strings.HasPrefix(tx.execs[0], "SAVEPOINT") {
		t.Fatalf("first statement must create savepoint, got %q", tx.execs[0])
	}
	if !strings.HasPrefix(tx.execs[len(tx.execs)-1], "RELEASE") {
		t.Fatalf("last statement must release savepoint, got %q", tx.execs[len(tx.execs)-1])
	}
}

func TestWithSavepoint_RollbackOnError(t *testing.T) {
	t.Parallel()

	tx := &txStub{}
	c := &Client{}
	run := txRunner{tx: tx}
	txCtx := ContextWithRunner(context.Background(), run)

	expected := errors.New("boom")
	err := c.WithSavepoint(txCtx, func(context.Context) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}

	if len(tx.execs) < 3 {
		t.Fatalf("expected savepoint, rollback to savepoint, release; got %v", tx.execs)
	}
	if !strings.HasPrefix(tx.execs[1], "ROLLBACK TO SAVEPOINT") {
		t.Fatalf("second statement must rollback savepoint, got %q", tx.execs[1])
	}
}

func TestWithSavepoint_RollbackCleanupErrorsAreJoined(t *testing.T) {
	t.Parallel()

	rbErr := errors.New("rollback failed")
	relErr := errors.New("release failed")
	tx := &txStub{errByPrefix: map[string]error{
		"ROLLBACK TO SAVEPOINT": rbErr,
		"RELEASE SAVEPOINT":     relErr,
	}}
	c := &Client{}
	run := txRunner{tx: tx}
	txCtx := ContextWithRunner(context.Background(), run)

	original := errors.New("boom")
	err := c.WithSavepoint(txCtx, func(context.Context) error { return original })
	if !errors.Is(err, original) {
		t.Fatalf("expected original error in chain, got %v", err)
	}
	if !strings.Contains(err.Error(), "rollback to savepoint failed") {
		t.Fatalf("expected rollback cleanup error, got %v", err)
	}
	if !strings.Contains(err.Error(), "release savepoint failed") {
		t.Fatalf("expected release cleanup error, got %v", err)
	}
}

func TestWithSavepoint_NilCallback(t *testing.T) {
	t.Parallel()

	c := &Client{}
	err := c.WithSavepoint(context.Background(), nil)
	if !errors.Is(err, errNilTxCallback) {
		t.Fatalf("expected errNilTxCallback, got %v", err)
	}
}

func TestWithTx_NilClientPool(t *testing.T) {
	t.Parallel()

	var c *Client
	err := c.WithTx(context.Background(), func(context.Context) error { return nil })
	if !errors.Is(err, errNilClientPool) {
		t.Fatalf("expected errNilClientPool, got %v", err)
	}
}

func TestWithTxOpts_NilCallback(t *testing.T) {
	t.Parallel()

	err := (&Client{}).WithTxOpts(context.Background(), TxConfig{}, nil)
	if !errors.Is(err, errNilTxCallback) {
		t.Fatalf("expected errNilTxCallback, got %v", err)
	}
}

func TestAsTx_FromRawTxProvider(t *testing.T) {
	t.Parallel()

	tx := &txStub{}
	run := rawRunnerStub{tx: tx}

	got, ok := asTx(run)
	if !ok {
		t.Fatalf("expected asTx to resolve tx from RawTx provider")
	}
	if got != tx {
		t.Fatalf("expected same tx instance")
	}
}

type txStub struct {
	execs       []string
	errByPrefix map[string]error
}

func (t *txStub) Begin(context.Context) (pgx.Tx, error) { return nil, errors.New("not implemented") }
func (t *txStub) Commit(context.Context) error          { return nil }
func (t *txStub) Rollback(context.Context) error        { return nil }
func (t *txStub) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.execs = append(t.execs, sql)
	for prefix, err := range t.errByPrefix {
		if strings.HasPrefix(sql, prefix) {
			return pgconn.CommandTag{}, err
		}
	}
	return pgconn.CommandTag{}, nil
}
func (t *txStub) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (t *txStub) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (t *txStub) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (t *txStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (t *txStub) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (t *txStub) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (t *txStub) Conn() *pgx.Conn { return nil }

type rawRunnerStub struct{ tx pgx.Tx }

func (r rawRunnerStub) RawTx() pgx.Tx { return r.tx }

func (r rawRunnerStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (r rawRunnerStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}
func (r rawRunnerStub) QueryRow(context.Context, string, ...any) pgx.Row { return nil }
