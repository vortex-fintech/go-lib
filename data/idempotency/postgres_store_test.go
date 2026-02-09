package idempotency

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestReserve_RequiresExpiresAt(t *testing.T) {
	t.Parallel()

	s := NewPostgresStore()
	r := &runnerStub{}

	_, err := s.Reserve(context.Background(), r, Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
	})
	if err == nil {
		t.Fatalf("expected error when expires_at is zero")
	}
}

func TestReserve_InsertSuccess(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	recFromDB := Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
		Status:         StatusInProgress,
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(5 * time.Minute),
	}

	r := &runnerStub{rows: []pgx.Row{rowStub{scanFn: scanRecord(recFromDB)}}}
	s := NewPostgresStore()

	res, err := s.Reserve(context.Background(), r, Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
		ExpiresAt:      now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Reserved || res.Record == nil {
		t.Fatalf("expected reserved record")
	}
	if res.Record.Status != StatusInProgress {
		t.Fatalf("expected default status IN_PROGRESS, got %s", res.Record.Status)
	}
	if len(r.queryRowArgs) == 0 {
		t.Fatalf("expected insert query args to be captured")
	}
	createdAt, ok := r.queryRowArgs[0][8].(time.Time)
	if !ok || createdAt.IsZero() || createdAt.Location() != time.UTC {
		t.Fatalf("expected created_at argument in UTC")
	}
	updatedAt, ok := r.queryRowArgs[0][9].(time.Time)
	if !ok || updatedAt.IsZero() || updatedAt.Location() != time.UTC {
		t.Fatalf("expected updated_at argument in UTC")
	}
}

func TestReserve_OnConflictReturnsExisting(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	existing := Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
		Status:         StatusSucceeded,
		ResponseCode:   0,
		CreatedAt:      now,
		UpdatedAt:      now,
		ExpiresAt:      now.Add(5 * time.Minute),
	}

	r := &runnerStub{rows: []pgx.Row{
		rowStub{err: sql.ErrNoRows},
		rowStub{scanFn: scanRecord(existing)},
	}}
	s := NewPostgresStore()

	res, err := s.Reserve(context.Background(), r, Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
		ExpiresAt:      now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Reserved {
		t.Fatalf("expected Reserved=false on conflict")
	}
	if res.Record == nil || res.Record.Status != StatusSucceeded {
		t.Fatalf("expected existing successful record")
	}
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()

	r := &runnerStub{rows: []pgx.Row{rowStub{err: sql.ErrNoRows}}}
	s := NewPostgresStore()

	rec, err := s.Get(context.Background(), r, "u1", "/svc.Method", "k1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec != nil {
		t.Fatalf("expected nil record when not found")
	}
}

func TestReacquireRetryable_And_Complete(t *testing.T) {
	t.Parallel()

	r := &runnerStub{execResults: []execResult{{tag: mustTag("UPDATE 1")}, {tag: mustTag("UPDATE 0")}, {tag: mustTag("UPDATE 1")}}}
	s := NewPostgresStore()

	ok, err := s.ReacquireRetryable(context.Background(), r, "u1", "/svc.Method", "k1", "h1", time.Now().UTC())
	if err != nil || !ok {
		t.Fatalf("expected reacquire true, err=%v", err)
	}

	ok, err = s.Complete(context.Background(), r, "u1", "/svc.Method", "k1", Completion{Status: StatusSucceeded})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected false when no rows updated")
	}

	ok, err = s.Complete(context.Background(), r, "u1", "/svc.Method", "k1", Completion{Status: StatusFailedFinal, UpdatedAt: time.Now()})
	if err != nil || !ok {
		t.Fatalf("expected complete true, err=%v", err)
	}
}

func TestDeleteExpired(t *testing.T) {
	t.Parallel()

	r := &runnerStub{execResults: []execResult{{tag: mustTag("DELETE 3")}}}
	s := NewPostgresStore()

	n, err := s.DeleteExpired(context.Background(), r, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 3 {
		t.Fatalf("expected 3 deleted rows, got %d", n)
	}
}

func TestNullIfEmpty(t *testing.T) {
	t.Parallel()

	if v := nullIfEmpty("  "); v != nil {
		t.Fatalf("expected nil for blank value")
	}
	if v := nullIfEmpty("x"); v == nil {
		t.Fatalf("expected non-nil for non-empty value")
	}
}

type execResult struct {
	tag pgconn.CommandTag
	err error
}

type runnerStub struct {
	rows         []pgx.Row
	queryRowArgs [][]any
	execResults  []execResult
	execCalls    int
}

func (r *runnerStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	if r.execCalls >= len(r.execResults) {
		return mustTag("UPDATE 0"), nil
	}
	res := r.execResults[r.execCalls]
	r.execCalls++
	return res.tag, res.err
}

func (r *runnerStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func (r *runnerStub) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	r.queryRowArgs = append(r.queryRowArgs, args)
	if len(r.rows) == 0 {
		return rowStub{err: sql.ErrNoRows}
	}
	out := r.rows[0]
	r.rows = r.rows[1:]
	return out
}

type rowStub struct {
	err    error
	scanFn func(dest ...any) error
}

func (r rowStub) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if r.scanFn != nil {
		return r.scanFn(dest...)
	}
	return nil
}

func scanRecord(rec Record) func(dest ...any) error {
	return func(dest ...any) error {
		*(dest[0].(*string)) = rec.Principal
		*(dest[1].(*string)) = rec.GRPCMethod
		*(dest[2].(*string)) = rec.IdempotencyKey
		*(dest[3].(*string)) = rec.RequestHash
		*(dest[4].(*Status)) = rec.Status
		*(dest[5].(*int32)) = rec.ResponseCode
		*(dest[6].(*[]byte)) = rec.ResponsePayload
		*(dest[7].(*string)) = rec.ErrorMessage
		*(dest[8].(*time.Time)) = rec.CreatedAt
		*(dest[9].(*time.Time)) = rec.UpdatedAt
		*(dest[10].(*time.Time)) = rec.ExpiresAt
		return nil
	}
}

func mustTag(v string) pgconn.CommandTag {
	return pgconn.NewCommandTag(v)
}
