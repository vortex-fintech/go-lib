package idempotency

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	pg "github.com/vortex-fintech/go-lib/data/postgres"
)

type PostgresStore struct{}

func NewPostgresStore() *PostgresStore {
	return &PostgresStore{}
}

var _ Store = (*PostgresStore)(nil)

func (s *PostgresStore) Reserve(ctx context.Context, run pg.Runner, rec Record) (ReserveResult, error) {
	ctx = ensureContext(ctx)

	if err := validateRunner(run); err != nil {
		return ReserveResult{}, err
	}
	if err := validateIdentity(rec.Principal, rec.GRPCMethod, rec.IdempotencyKey); err != nil {
		return ReserveResult{}, err
	}
	if strings.TrimSpace(rec.RequestHash) == "" {
		return ReserveResult{}, ErrRequestHashRequired
	}

	now := nowUTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	} else {
		rec.CreatedAt = normalizeUTC(rec.CreatedAt)
	}
	if rec.UpdatedAt.IsZero() {
		rec.UpdatedAt = now
	} else {
		rec.UpdatedAt = normalizeUTC(rec.UpdatedAt)
	}
	if rec.Status == "" {
		rec.Status = StatusInProgress
	}
	if !rec.Status.IsValid() {
		return ReserveResult{}, fmt.Errorf("%w: %q", ErrInvalidStatus, rec.Status)
	}
	if rec.ExpiresAt.IsZero() {
		return ReserveResult{}, ErrExpiresAtRequired
	}
	rec.ExpiresAt = normalizeUTC(rec.ExpiresAt)
	if !rec.ExpiresAt.After(rec.CreatedAt) {
		return ReserveResult{}, ErrExpiresAtInvalid
	}

	err := run.QueryRow(ctx, `
		INSERT INTO idempotency_keys (
			principal, grpc_method, idempotency_key, request_hash,
			status, response_code, response_payload, error_message,
			created_at, updated_at, expires_at
		) VALUES (
			$1,$2,$3,$4,
			$5,$6,$7,$8,
			$9,$10,$11
		)
		ON CONFLICT (principal, grpc_method, idempotency_key) DO NOTHING
		RETURNING
			principal, grpc_method, idempotency_key, request_hash,
			status, response_code, response_payload, COALESCE(error_message, ''),
			created_at, updated_at, expires_at
	`,
		rec.Principal,
		rec.GRPCMethod,
		rec.IdempotencyKey,
		rec.RequestHash,
		rec.Status,
		rec.ResponseCode,
		rec.ResponsePayload,
		nullIfEmpty(rec.ErrorMessage),
		rec.CreatedAt,
		rec.UpdatedAt,
		rec.ExpiresAt,
	).Scan(
		&rec.Principal,
		&rec.GRPCMethod,
		&rec.IdempotencyKey,
		&rec.RequestHash,
		&rec.Status,
		&rec.ResponseCode,
		&rec.ResponsePayload,
		&rec.ErrorMessage,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&rec.ExpiresAt,
	)
	if err == nil {
		return ReserveResult{Reserved: true, Record: &rec}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ReserveResult{}, err
	}

	existing, getErr := s.Get(ctx, run, rec.Principal, rec.GRPCMethod, rec.IdempotencyKey)
	if getErr != nil {
		return ReserveResult{}, getErr
	}
	if existing == nil {
		return ReserveResult{}, ErrInconsistentState
	}
	if existing.RequestHash != rec.RequestHash {
		return ReserveResult{}, fmt.Errorf(
			"%w: principal=%q grpc_method=%q idempotency_key=%q",
			ErrRequestHashMismatch,
			rec.Principal,
			rec.GRPCMethod,
			rec.IdempotencyKey,
		)
	}
	return ReserveResult{Reserved: false, Record: existing}, nil
}

func (s *PostgresStore) Get(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey string) (*Record, error) {
	ctx = ensureContext(ctx)

	if err := validateRunner(run); err != nil {
		return nil, err
	}
	if err := validateIdentity(principal, grpcMethod, idemKey); err != nil {
		return nil, err
	}

	var rec Record
	err := run.QueryRow(ctx, `
		SELECT
			principal, grpc_method, idempotency_key, request_hash,
			status, response_code, response_payload, COALESCE(error_message, ''),
			created_at, updated_at, expires_at
		FROM idempotency_keys
		WHERE principal = $1
		  AND grpc_method = $2
		  AND idempotency_key = $3
	`, principal, grpcMethod, idemKey).Scan(
		&rec.Principal,
		&rec.GRPCMethod,
		&rec.IdempotencyKey,
		&rec.RequestHash,
		&rec.Status,
		&rec.ResponseCode,
		&rec.ResponsePayload,
		&rec.ErrorMessage,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&rec.ExpiresAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rec.CreatedAt = normalizeUTC(rec.CreatedAt)
	rec.UpdatedAt = normalizeUTC(rec.UpdatedAt)
	rec.ExpiresAt = normalizeUTC(rec.ExpiresAt)
	return &rec, nil
}

func (s *PostgresStore) ReacquireRetryable(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey, requestHash string, updatedAt time.Time) (bool, error) {
	ctx = ensureContext(ctx)

	if err := validateRunner(run); err != nil {
		return false, err
	}
	if err := validateIdentity(principal, grpcMethod, idemKey); err != nil {
		return false, err
	}
	if strings.TrimSpace(requestHash) == "" {
		return false, ErrRequestHashRequired
	}
	if updatedAt.IsZero() {
		return false, ErrUpdatedAtRequired
	}
	updatedAt = normalizeUTC(updatedAt)

	res, err := run.Exec(ctx, `
		UPDATE idempotency_keys
		   SET status = 'IN_PROGRESS',
		       response_code = 0,
		       response_payload = NULL,
		       error_message = NULL,
		       updated_at = $1
		 WHERE principal = $2
		   AND grpc_method = $3
		   AND idempotency_key = $4
		   AND request_hash = $5
		   AND status = 'FAILED_RETRYABLE'
		   AND expires_at > $1
		   AND updated_at < $1
	`, updatedAt, principal, grpcMethod, idemKey, requestHash)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

func (s *PostgresStore) Complete(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey string, done Completion) (bool, error) {
	ctx = ensureContext(ctx)

	if err := validateRunner(run); err != nil {
		return false, err
	}
	if err := validateIdentity(principal, grpcMethod, idemKey); err != nil {
		return false, err
	}
	if !done.Status.IsValid() {
		return false, fmt.Errorf("%w: %q", ErrInvalidStatus, done.Status)
	}
	if !done.Status.IsTerminal() {
		return false, fmt.Errorf("%w: %q", ErrCompletionNotTerminal, done.Status)
	}

	if done.UpdatedAt.IsZero() {
		return false, ErrUpdatedAtRequired
	}
	expectedUpdatedAt := normalizeUTC(done.UpdatedAt)
	completedAt := nowUTC()

	res, err := run.Exec(ctx, `
		UPDATE idempotency_keys
		   SET status = $1,
		       response_code = $2,
		       response_payload = $3,
		       error_message = $4,
		       updated_at = $5
		 WHERE principal = $6
		   AND grpc_method = $7
		   AND idempotency_key = $8
		   AND status = 'IN_PROGRESS'
		   AND updated_at = $9
	`, done.Status, done.ResponseCode, done.ResponsePayload, nullIfEmpty(done.ErrorMessage), completedAt, principal, grpcMethod, idemKey, expectedUpdatedAt)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

func (s *PostgresStore) DeleteExpired(ctx context.Context, run pg.Runner, before time.Time) (int64, error) {
	ctx = ensureContext(ctx)

	if err := validateRunner(run); err != nil {
		return 0, err
	}
	if before.IsZero() {
		before = nowUTC()
	} else {
		before = normalizeUTC(before)
	}

	res, err := run.Exec(ctx, `
		DELETE FROM idempotency_keys
		 WHERE expires_at <= $1
		   AND status IN ('SUCCEEDED', 'FAILED_RETRYABLE', 'FAILED_FINAL')
	`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func validateIdentity(principal, grpcMethod, idemKey string) error {
	if strings.TrimSpace(principal) == "" {
		return ErrPrincipalRequired
	}
	if strings.TrimSpace(grpcMethod) == "" {
		return ErrGRPCMethodRequired
	}
	if strings.TrimSpace(idemKey) == "" {
		return ErrIdempotencyKeyRequired
	}
	return nil
}

func validateRunner(run pg.Runner) error {
	if run == nil {
		return ErrNilRunner
	}
	rv := reflect.ValueOf(run)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if rv.IsNil() {
			return ErrNilRunner
		}
	}
	return nil
}

func nowUTC() time.Time {
	return normalizeUTC(time.Now())
}

func normalizeUTC(v time.Time) time.Time {
	return v.UTC().Truncate(time.Microsecond)
}
