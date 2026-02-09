package idempotency

import (
	"context"
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
