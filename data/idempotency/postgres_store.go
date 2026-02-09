package idempotency

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	if rec.UpdatedAt.IsZero() {
		rec.UpdatedAt = now
	}
	if rec.Status == "" {
		rec.Status = StatusInProgress
	}
	if rec.ExpiresAt.IsZero() {
		return ReserveResult{}, fmt.Errorf("expires_at is required for idempotency record")
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
		rec.CreatedAt.UTC(),
		rec.UpdatedAt.UTC(),
		rec.ExpiresAt.UTC(),
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
	return ReserveResult{Reserved: false, Record: existing}, nil
}

func (s *PostgresStore) Get(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey string) (*Record, error) {
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
	return &rec, nil
}

func (s *PostgresStore) ReacquireRetryable(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey, requestHash string, updatedAt time.Time) (bool, error) {
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
	`, updatedAt.UTC(), principal, grpcMethod, idemKey, requestHash)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

func (s *PostgresStore) Complete(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey string, done Completion) (bool, error) {
	updatedAt := done.UpdatedAt.UTC()
	if done.UpdatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

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
	`, done.Status, done.ResponseCode, done.ResponsePayload, nullIfEmpty(done.ErrorMessage), updatedAt, principal, grpcMethod, idemKey)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

func (s *PostgresStore) DeleteExpired(ctx context.Context, run pg.Runner, before time.Time) (int64, error) {
	res, err := run.Exec(ctx, `
		DELETE FROM idempotency_keys
		 WHERE expires_at <= $1
	`, before.UTC())
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
