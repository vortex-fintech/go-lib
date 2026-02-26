package errors

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestDomainInvariant(t *testing.T) {
	de := DomainInvariant("user.id", "empty")
	if de.Error() != "user.id: empty" {
		t.Fatalf("unexpected DomainError string: %s", de.Error())
	}
	if !IsInvariant(de) {
		t.Fatalf("IsInvariant should return true")
	}
}

func TestStateInvariant(t *testing.T) {
	base := errors.New("invalid state")
	se := StateInvariant(base, "address.updated_at", "before created_at")
	if se.Error() != "state: invalid state: before created_at" {
		t.Fatalf("unexpected StateInvariant string: %s", se.Error())
	}
	if !IsInvariant(se) {
		t.Fatalf("IsInvariant should return true")
	}
	if !errors.Is(se, base) {
		t.Fatalf("StateInvariant should unwrap to base error")
	}
}

func TestTransitionInvariant(t *testing.T) {
	base := errors.New("invalid transition")
	te := TransitionInvariant(base, "address.status", "cannot verify from PENDING")
	if te.Error() != "transition: invalid transition: cannot verify from PENDING" {
		t.Fatalf("unexpected TransitionInvariant string: %s", te.Error())
	}
	if !IsInvariant(te) {
		t.Fatalf("IsInvariant should return true")
	}
	if !errors.Is(te, base) {
		t.Fatalf("TransitionInvariant should unwrap to base error")
	}
}

func TestToErrorResponseFromDomain(t *testing.T) {
	de := DomainInvariant("email", "invalid_email")
	er := ToErrorResponse(de)
	if er.Code != codes.InvalidArgument || er.Details["email"] != "invalid_email" {
		t.Fatalf("domain adaptation mismatch: %+v", er)
	}
}

func TestToErrorResponseFromState(t *testing.T) {
	se := StateInvariant(errors.New("invalid state"), "address.updated_at", "before created_at")
	er := ToErrorResponse(se)
	if er.Code != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", er.Code)
	}
	if er.Reason != Reason("invariant_violation") {
		t.Fatalf("expected invariant_violation reason, got %v", er.Reason)
	}
	if er.Details["invariant_kind"] != string(KindState) {
		t.Fatalf("expected invariant_kind=state, got %v", er.Details["invariant_kind"])
	}
}

func TestToErrorResponseFromTransition(t *testing.T) {
	te := TransitionInvariant(errors.New("invalid transition"), "address.status", "cannot verify from PENDING")
	er := ToErrorResponse(te)
	if er.Code != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", er.Code)
	}
	if er.Reason != Reason("invariant_violation") {
		t.Fatalf("expected invariant_violation reason, got %v", er.Reason)
	}
	if er.Details["invariant_kind"] != string(KindTransition) {
		t.Fatalf("expected invariant_kind=transition, got %v", er.Details["invariant_kind"])
	}
}

func TestConvertInvariantToResponseUnknown(t *testing.T) {
	er := ToErrorResponse(errors.New("random error"))
	if er.Code != codes.Internal {
		t.Fatalf("expected Internal, got %v", er.Code)
	}
	if er.Reason != Reason("unexpected_error") {
		t.Fatalf("expected unexpected_error, got %v", er.Reason)
	}
}

func TestToErrorResponsePassThrough(t *testing.T) {
	in := InvalidArgument().WithReason("bad").WithDetail("x", "y")
	out := ToErrorResponse(in)
	if out.Reason != "bad" || out.Details["x"] != "y" {
		t.Fatalf("passthrough mismatch")
	}
}

func TestToErrorResponsePassThroughPointer(t *testing.T) {
	in := InvalidArgument().WithReason("bad").WithDetail("x", "y")
	err := &in
	out := ToErrorResponse(err)
	if out.Reason != "bad" || out.Details["x"] != "y" {
		t.Fatalf("pointer passthrough mismatch")
	}
}

func TestToErrorResponseFromWrappedDomain(t *testing.T) {
	err := fmt.Errorf("wrap: %w", DomainInvariant("email", "invalid_email"))
	out := ToErrorResponse(err)
	if out.Code != codes.InvalidArgument || out.Details["email"] != "invalid_email" {
		t.Fatalf("wrapped domain adaptation mismatch: %+v", out)
	}
}
