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

	if !isSerializationFailure(&pgconn.PgError{Code: "40001"}) {
		t.Fatalf("expected true for SQLSTATE 40001")
	}
	if isSerializationFailure(&pgconn.PgError{Code: "23505"}) {
		t.Fatalf("expected false for non-serialization SQLSTATE")
	}
	wrapped := errors.New("wrapper: " + (&pgconn.PgError{Code: "40001"}).Error())
	if isSerializationFailure(wrapped) {
		t.Fatalf("expected false for non-wrapped plain error")
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

type txStub struct {
	execs []string
}

func (t *txStub) Begin(context.Context) (pgx.Tx, error) { return nil, errors.New("not implemented") }
func (t *txStub) Commit(context.Context) error          { return nil }
func (t *txStub) Rollback(context.Context) error        { return nil }
func (t *txStub) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.execs = append(t.execs, sql)
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
