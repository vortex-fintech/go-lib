package idempotency

import (
	"context"
	"errors"
	"time"

	pg "github.com/vortex-fintech/go-lib/data/postgres"
)

type Status string

const (
	StatusInProgress  Status = "IN_PROGRESS"
	StatusSucceeded   Status = "SUCCEEDED"
	StatusFailedRetry Status = "FAILED_RETRYABLE"
	StatusFailedFinal Status = "FAILED_FINAL"
)

var (
	ErrNilStore               = errors.New("idempotency: store is required")
	ErrNilRunner              = errors.New("idempotency: runner is required")
	ErrPrincipalRequired      = errors.New("idempotency: principal is required")
	ErrGRPCMethodRequired     = errors.New("idempotency: grpc method is required")
	ErrIdempotencyKeyRequired = errors.New("idempotency: idempotency key is required")
	ErrRequestHashRequired    = errors.New("idempotency: request hash is required")
	ErrUpdatedAtRequired      = errors.New("idempotency: updated_at is required")
	ErrExpiresAtRequired      = errors.New("idempotency: expires_at is required")
	ErrExpiresAtInvalid       = errors.New("idempotency: expires_at must be after created_at")
	ErrInvalidStatus          = errors.New("idempotency: invalid status")
	ErrCompletionNotTerminal  = errors.New("idempotency: completion status must be terminal")
	ErrRequestHashMismatch    = errors.New("idempotency: idempotency key reused with different request hash")
	ErrInconsistentState      = errors.New("idempotency: inconsistent state")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusInProgress, StatusSucceeded, StatusFailedRetry, StatusFailedFinal:
		return true
	default:
		return false
	}
}

func (s Status) IsTerminal() bool {
	switch s {
	case StatusSucceeded, StatusFailedRetry, StatusFailedFinal:
		return true
	default:
		return false
	}
}

type Record struct {
	Principal       string
	GRPCMethod      string
	IdempotencyKey  string
	RequestHash     string
	Status          Status
	ResponseCode    int32
	ResponsePayload []byte
	ErrorMessage    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ExpiresAt       time.Time
}

type ReserveResult struct {
	Reserved bool
	Record   *Record
}

type Completion struct {
	Status          Status
	ResponseCode    int32
	ResponsePayload []byte
	ErrorMessage    string
	UpdatedAt       time.Time
}

type Store interface {
	Reserve(ctx context.Context, run pg.Runner, rec Record) (ReserveResult, error)
	Get(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey string) (*Record, error)
	ReacquireRetryable(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey, requestHash string, updatedAt time.Time) (bool, error)
	Complete(ctx context.Context, run pg.Runner, principal, grpcMethod, idemKey string, done Completion) (bool, error)
	DeleteExpired(ctx context.Context, run pg.Runner, before time.Time) (int64, error)
}

func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
