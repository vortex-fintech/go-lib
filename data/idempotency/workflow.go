package idempotency

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	pg "github.com/vortex-fintech/go-lib/data/postgres"
)

type BeginDecision string

const (
	BeginDecisionExecute    BeginDecision = "EXECUTE"
	BeginDecisionReplay     BeginDecision = "REPLAY"
	BeginDecisionInProgress BeginDecision = "IN_PROGRESS"
	BeginDecisionRetryable  BeginDecision = "RETRYABLE"
)

type BeginInput struct {
	Principal      string
	GRPCMethod     string
	IdempotencyKey string
	RequestHash    string
	ExpiresAt      time.Time
}

type BeginResult struct {
	Decision BeginDecision
	Lease    *Record
	Existing *Record
}

func Begin(ctx context.Context, store Store, run pg.Runner, in BeginInput) (BeginResult, error) {
	ctx = ensureContext(ctx)

	if err := validateStore(store); err != nil {
		return BeginResult{}, err
	}

	reserve, err := store.Reserve(ctx, run, Record{
		Principal:      in.Principal,
		GRPCMethod:     in.GRPCMethod,
		IdempotencyKey: in.IdempotencyKey,
		RequestHash:    in.RequestHash,
		ExpiresAt:      in.ExpiresAt,
	})
	if err != nil {
		return BeginResult{}, err
	}
	if reserve.Record == nil {
		return BeginResult{}, ErrInconsistentState
	}

	if reserve.Reserved {
		return BeginResult{
			Decision: BeginDecisionExecute,
			Lease:    reserve.Record,
		}, nil
	}

	result := BeginResult{Existing: reserve.Record}
	switch reserve.Record.Status {
	case StatusInProgress:
		result.Decision = BeginDecisionInProgress
	case StatusSucceeded, StatusFailedFinal:
		result.Decision = BeginDecisionReplay
	case StatusFailedRetry:
		result.Decision = BeginDecisionRetryable
	default:
		return BeginResult{}, fmt.Errorf("%w: %q", ErrInvalidStatus, reserve.Record.Status)
	}

	return result, nil
}

func Finish(ctx context.Context, store Store, run pg.Runner, lease Record, done Completion) (bool, error) {
	ctx = ensureContext(ctx)

	if err := validateStore(store); err != nil {
		return false, err
	}
	if err := validateIdentityFields(lease.Principal, lease.GRPCMethod, lease.IdempotencyKey); err != nil {
		return false, err
	}
	if done.UpdatedAt.IsZero() {
		done.UpdatedAt = lease.UpdatedAt
	}
	if done.UpdatedAt.IsZero() {
		return false, ErrUpdatedAtRequired
	}

	return store.Complete(ctx, run, lease.Principal, lease.GRPCMethod, lease.IdempotencyKey, done)
}

func Reacquire(ctx context.Context, store Store, run pg.Runner, rec Record, newUpdatedAt time.Time) (bool, error) {
	ctx = ensureContext(ctx)

	if err := validateStore(store); err != nil {
		return false, err
	}
	if err := validateIdentityFields(rec.Principal, rec.GRPCMethod, rec.IdempotencyKey); err != nil {
		return false, err
	}
	if strings.TrimSpace(rec.RequestHash) == "" {
		return false, ErrRequestHashRequired
	}
	if newUpdatedAt.IsZero() {
		return false, ErrUpdatedAtRequired
	}

	return store.ReacquireRetryable(
		ctx,
		run,
		rec.Principal,
		rec.GRPCMethod,
		rec.IdempotencyKey,
		rec.RequestHash,
		newUpdatedAt,
	)
}

func validateStore(store Store) error {
	if store == nil {
		return ErrNilStore
	}
	rv := reflect.ValueOf(store)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if rv.IsNil() {
			return ErrNilStore
		}
	}
	return nil
}

func validateIdentityFields(principal, grpcMethod, idemKey string) error {
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
