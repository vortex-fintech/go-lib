//go:build integration

package idempotency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vortex-fintech/go-lib/data/idempotency"
	"github.com/vortex-fintech/go-lib/data/postgres"
)

func TestPostgresStore_RequestHashMismatch_Integration(t *testing.T) {
	c := openIntegrationClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	run := c.RunnerFromPool()
	require.NoError(t, ensureIdempotencySchema(ctx, run))
	require.NoError(t, truncateIdempotencyKeys(ctx, run))

	s := idempotency.NewPostgresStore()
	expiresAt := time.Now().UTC().Add(30 * time.Minute)

	res, err := s.Reserve(ctx, run, idempotency.Record{
		Principal:      "merchant-1",
		GRPCMethod:     "/payments.v1.Payments/Authorize",
		IdempotencyKey: "idem-hash-mismatch",
		RequestHash:    "hash-v1",
		ExpiresAt:      expiresAt,
	})
	require.NoError(t, err)
	require.True(t, res.Reserved)

	_, err = s.Reserve(ctx, run, idempotency.Record{
		Principal:      "merchant-1",
		GRPCMethod:     "/payments.v1.Payments/Authorize",
		IdempotencyKey: "idem-hash-mismatch",
		RequestHash:    "hash-v2",
		ExpiresAt:      expiresAt,
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, idempotency.ErrRequestHashMismatch), "expected ErrRequestHashMismatch, got %v", err)
}

func TestPostgresStore_StaleCompletionRejectedAfterReacquire_Integration(t *testing.T) {
	c := openIntegrationClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	run := c.RunnerFromPool()
	require.NoError(t, ensureIdempotencySchema(ctx, run))
	require.NoError(t, truncateIdempotencyKeys(ctx, run))

	s := idempotency.NewPostgresStore()
	expiresAt := time.Now().UTC().Add(30 * time.Minute)

	reserved, err := s.Reserve(ctx, run, idempotency.Record{
		Principal:      "merchant-2",
		GRPCMethod:     "/payments.v1.Payments/Capture",
		IdempotencyKey: "idem-stale-complete",
		RequestHash:    "hash-capture",
		ExpiresAt:      expiresAt,
	})
	require.NoError(t, err)
	require.True(t, reserved.Reserved)
	require.NotNil(t, reserved.Record)

	firstLease := reserved.Record.UpdatedAt

	ok, err := s.Complete(ctx, run, "merchant-2", "/payments.v1.Payments/Capture", "idem-stale-complete", idempotency.Completion{
		Status:    idempotency.StatusFailedRetry,
		UpdatedAt: firstLease,
	})
	require.NoError(t, err)
	require.True(t, ok)

	secondLease := firstLease.Add(2 * time.Second)
	ok, err = s.ReacquireRetryable(ctx, run, "merchant-2", "/payments.v1.Payments/Capture", "idem-stale-complete", "hash-capture", secondLease)
	require.NoError(t, err)
	require.True(t, ok)

	staleOK, err := s.Complete(ctx, run, "merchant-2", "/payments.v1.Payments/Capture", "idem-stale-complete", idempotency.Completion{
		Status:    idempotency.StatusSucceeded,
		UpdatedAt: firstLease,
	})
	require.NoError(t, err)
	require.False(t, staleOK, "stale worker must not complete newer attempt")

	freshOK, err := s.Complete(ctx, run, "merchant-2", "/payments.v1.Payments/Capture", "idem-stale-complete", idempotency.Completion{
		Status:    idempotency.StatusSucceeded,
		UpdatedAt: secondLease,
	})
	require.NoError(t, err)
	require.True(t, freshOK)

	finalRec, err := s.Get(ctx, run, "merchant-2", "/payments.v1.Payments/Capture", "idem-stale-complete")
	require.NoError(t, err)
	require.NotNil(t, finalRec)
	require.Equal(t, idempotency.StatusSucceeded, finalRec.Status)
}

func TestPostgresStore_DeleteExpiredOnlyTerminal_Integration(t *testing.T) {
	c := openIntegrationClient(t)
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	run := c.RunnerFromPool()
	require.NoError(t, ensureIdempotencySchema(ctx, run))
	require.NoError(t, truncateIdempotencyKeys(ctx, run))

	now := time.Now().UTC().Truncate(time.Microsecond)
	createdAt := now.Add(-10 * time.Minute)
	expiredAt := now.Add(-5 * time.Minute)

	_, err := run.Exec(ctx, `
		INSERT INTO idempotency_keys (
			principal, grpc_method, idempotency_key, request_hash,
			status, response_code, response_payload, error_message,
			created_at, updated_at, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, "merchant-3", "/payments.v1.Payments/Void", "idem-in-progress", "hash-void", "IN_PROGRESS", 0, nil, nil, createdAt, createdAt, expiredAt)
	require.NoError(t, err)

	_, err = run.Exec(ctx, `
		INSERT INTO idempotency_keys (
			principal, grpc_method, idempotency_key, request_hash,
			status, response_code, response_payload, error_message,
			created_at, updated_at, expires_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`, "merchant-3", "/payments.v1.Payments/Void", "idem-succeeded", "hash-void-2", "SUCCEEDED", 0, nil, nil, createdAt, createdAt, expiredAt)
	require.NoError(t, err)

	s := idempotency.NewPostgresStore()
	deleted, err := s.DeleteExpired(ctx, run, now)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)

	inProgress, err := s.Get(ctx, run, "merchant-3", "/payments.v1.Payments/Void", "idem-in-progress")
	require.NoError(t, err)
	require.NotNil(t, inProgress, "in-progress row must stay")

	completed, err := s.Get(ctx, run, "merchant-3", "/payments.v1.Payments/Void", "idem-succeeded")
	require.NoError(t, err)
	require.Nil(t, completed, "terminal row should be removed")
}

func openIntegrationClient(t *testing.T) *postgres.Client {
	t.Helper()

	cfg := postgres.Config{URL: "postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable", MaxConns: 5, MinConns: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := postgres.Open(ctx, cfg)
	require.NoError(t, err)
	return c
}

func ensureIdempotencySchema(ctx context.Context, run postgres.Runner) error {
	if _, err := run.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS idempotency_keys (
			principal TEXT NOT NULL,
			grpc_method TEXT NOT NULL,
			idempotency_key TEXT NOT NULL,
			request_hash TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('IN_PROGRESS', 'SUCCEEDED', 'FAILED_RETRYABLE', 'FAILED_FINAL')),
			response_code INTEGER NOT NULL DEFAULT 0,
			response_payload BYTEA,
			error_message TEXT,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			CONSTRAINT idempotency_keys_pkey PRIMARY KEY (principal, grpc_method, idempotency_key),
			CONSTRAINT idempotency_keys_expiry_chk CHECK (expires_at > created_at)
		)
	`); err != nil {
		return err
	}
	if _, err := run.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_terminal
			ON idempotency_keys (expires_at)
			WHERE status IN ('SUCCEEDED', 'FAILED_RETRYABLE', 'FAILED_FINAL')
	`); err != nil {
		return err
	}
	return nil
}

func truncateIdempotencyKeys(ctx context.Context, run postgres.Runner) error {
	_, err := run.Exec(ctx, "TRUNCATE TABLE idempotency_keys")
	return err
}
