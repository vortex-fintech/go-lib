package idempotency

import (
	"context"
	"errors"
	"testing"
	"time"

	pg "github.com/vortex-fintech/go-lib/data/postgres"
)

func TestBegin_ExecuteDecision(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	st := &workflowStoreStub{
		reserveResult: ReserveResult{Reserved: true, Record: &Record{
			Principal:      "u1",
			GRPCMethod:     "/svc.Method",
			IdempotencyKey: "k1",
			RequestHash:    "h1",
			Status:         StatusInProgress,
			UpdatedAt:      now,
		}},
	}

	out, err := Begin(context.Background(), st, nil, BeginInput{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
		ExpiresAt:      now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Decision != BeginDecisionExecute {
		t.Fatalf("expected execute decision, got %s", out.Decision)
	}
	if out.Lease == nil || out.Lease.UpdatedAt.IsZero() {
		t.Fatalf("expected lease with non-zero updated_at")
	}
	if out.Existing != nil {
		t.Fatalf("existing must be nil for execute decision")
	}
	if st.reserveRec.Principal != "u1" || st.reserveRec.IdempotencyKey != "k1" {
		t.Fatalf("unexpected reserve input: %+v", st.reserveRec)
	}
}

func TestBegin_DuplicateDecisions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   Status
		expected BeginDecision
	}{
		{name: "in progress", status: StatusInProgress, expected: BeginDecisionInProgress},
		{name: "succeeded", status: StatusSucceeded, expected: BeginDecisionReplay},
		{name: "failed final", status: StatusFailedFinal, expected: BeginDecisionReplay},
		{name: "failed retry", status: StatusFailedRetry, expected: BeginDecisionRetryable},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			st := &workflowStoreStub{
				reserveResult: ReserveResult{Reserved: false, Record: &Record{Status: tc.status}},
			}

			out, err := Begin(context.Background(), st, nil, BeginInput{
				Principal:      "u1",
				GRPCMethod:     "/svc.Method",
				IdempotencyKey: "k1",
				RequestHash:    "h1",
				ExpiresAt:      time.Now().UTC().Add(time.Minute),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if out.Decision != tc.expected {
				t.Fatalf("expected %s, got %s", tc.expected, out.Decision)
			}
			if out.Existing == nil {
				t.Fatalf("expected existing record for duplicate path")
			}
		})
	}
}

func TestBegin_RejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	st := &workflowStoreStub{reserveResult: ReserveResult{Reserved: false, Record: &Record{Status: Status("BROKEN")}}}

	_, err := Begin(context.Background(), st, nil, BeginInput{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
		ExpiresAt:      time.Now().UTC().Add(time.Minute),
	})
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestBegin_RequiresStore(t *testing.T) {
	t.Parallel()

	var st *workflowStoreStub
	_, err := Begin(context.Background(), st, nil, BeginInput{})
	if !errors.Is(err, ErrNilStore) {
		t.Fatalf("expected ErrNilStore, got %v", err)
	}
}

func TestFinish_UsesLeaseUpdatedAtWhenMissing(t *testing.T) {
	t.Parallel()

	st := &workflowStoreStub{completeOK: true}
	lease := Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		UpdatedAt:      time.Now().UTC(),
	}

	ok, err := Finish(context.Background(), st, nil, lease, Completion{Status: StatusSucceeded})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected completion success")
	}
	if st.completeCall.done.UpdatedAt.IsZero() {
		t.Fatalf("expected UpdatedAt to be propagated from lease")
	}
	if !st.completeCall.done.UpdatedAt.Equal(lease.UpdatedAt) {
		t.Fatalf("expected UpdatedAt %v, got %v", lease.UpdatedAt, st.completeCall.done.UpdatedAt)
	}
}

func TestFinish_RequiresUpdatedAtWhenLeaseMissing(t *testing.T) {
	t.Parallel()

	st := &workflowStoreStub{}
	_, err := Finish(context.Background(), st, nil, Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
	}, Completion{Status: StatusSucceeded})
	if !errors.Is(err, ErrUpdatedAtRequired) {
		t.Fatalf("expected ErrUpdatedAtRequired, got %v", err)
	}
}

func TestReacquire_UsesRecordIdentityAndHash(t *testing.T) {
	t.Parallel()

	st := &workflowStoreStub{reacquireOK: true}
	rec := Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
	}
	newLease := time.Now().UTC()

	ok, err := Reacquire(context.Background(), st, nil, rec, newLease)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected reacquire true")
	}
	if st.reacquireCall.principal != rec.Principal || st.reacquireCall.idemKey != rec.IdempotencyKey {
		t.Fatalf("unexpected reacquire identity: %+v", st.reacquireCall)
	}
	if st.reacquireCall.requestHash != rec.RequestHash {
		t.Fatalf("unexpected request hash: %q", st.reacquireCall.requestHash)
	}
}

func TestReacquire_RequiresUpdatedAt(t *testing.T) {
	t.Parallel()

	st := &workflowStoreStub{}
	_, err := Reacquire(context.Background(), st, nil, Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
	}, time.Time{})
	if !errors.Is(err, ErrUpdatedAtRequired) {
		t.Fatalf("expected ErrUpdatedAtRequired, got %v", err)
	}
}

func TestWorkflow_TODOContext_IsPropagated(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	ctx := context.TODO()
	st := &workflowStoreStub{
		reserveResult: ReserveResult{Reserved: true, Record: &Record{
			Principal:      "u1",
			GRPCMethod:     "/svc.Method",
			IdempotencyKey: "k1",
			RequestHash:    "h1",
			Status:         StatusInProgress,
			UpdatedAt:      now,
		}},
		completeOK:  true,
		reacquireOK: true,
	}

	beginOut, err := Begin(ctx, st, nil, BeginInput{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
		ExpiresAt:      now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Begin(ctx, ...): %v", err)
	}
	if beginOut.Lease == nil {
		t.Fatalf("expected lease from Begin")
	}

	if _, err := Finish(ctx, st, nil, *beginOut.Lease, Completion{Status: StatusSucceeded}); err != nil {
		t.Fatalf("Finish(ctx, ...): %v", err)
	}

	if _, err := Reacquire(ctx, st, nil, Record{
		Principal:      "u1",
		GRPCMethod:     "/svc.Method",
		IdempotencyKey: "k1",
		RequestHash:    "h1",
	}, now.Add(time.Second)); err != nil {
		t.Fatalf("Reacquire(ctx, ...): %v", err)
	}

	if st.reserveCtx != ctx {
		t.Fatalf("expected Reserve to receive the same context")
	}
	if st.completeCtx != ctx {
		t.Fatalf("expected Complete to receive the same context")
	}
	if st.reacquireCtx != ctx {
		t.Fatalf("expected ReacquireRetryable to receive the same context")
	}
}

type workflowStoreStub struct {
	reserveCtx    context.Context
	reserveRec    Record
	reserveResult ReserveResult
	reserveErr    error

	completeCtx  context.Context
	completeCall completeCall
	completeOK   bool
	completeErr  error

	reacquireCtx  context.Context
	reacquireCall reacquireCall
	reacquireOK   bool
	reacquireErr  error
}

func (s *workflowStoreStub) Reserve(ctx context.Context, _ pg.Runner, rec Record) (ReserveResult, error) {
	s.reserveCtx = ctx
	s.reserveRec = rec
	return s.reserveResult, s.reserveErr
}

func (s *workflowStoreStub) Get(context.Context, pg.Runner, string, string, string) (*Record, error) {
	return nil, nil
}

func (s *workflowStoreStub) ReacquireRetryable(ctx context.Context, _ pg.Runner, principal, grpcMethod, idemKey, requestHash string, updatedAt time.Time) (bool, error) {
	s.reacquireCtx = ctx
	s.reacquireCall = reacquireCall{
		principal:   principal,
		grpcMethod:  grpcMethod,
		idemKey:     idemKey,
		requestHash: requestHash,
		updatedAt:   updatedAt,
	}
	return s.reacquireOK, s.reacquireErr
}

func (s *workflowStoreStub) Complete(ctx context.Context, _ pg.Runner, principal, grpcMethod, idemKey string, done Completion) (bool, error) {
	s.completeCtx = ctx
	s.completeCall = completeCall{principal: principal, grpcMethod: grpcMethod, idemKey: idemKey, done: done}
	return s.completeOK, s.completeErr
}

func (s *workflowStoreStub) DeleteExpired(context.Context, pg.Runner, time.Time) (int64, error) {
	return 0, nil
}

type completeCall struct {
	principal  string
	grpcMethod string
	idemKey    string
	done       Completion
}

type reacquireCall struct {
	principal   string
	grpcMethod  string
	idemKey     string
	requestHash string
	updatedAt   time.Time
}
