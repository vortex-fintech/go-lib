package errors

import (
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCAndFromGRPC_ErrorInfoAndBadRequest(t *testing.T) {
	e := InvalidArgument().
		WithReason("validation_failed").
		WithDomain("auth-service").
		WithDetails(map[string]string{"email": "invalid_email"}).
		WithViolations([]FieldViolation{{Field: "email", Reason: "invalid_email"}})

	err := e.ToGRPC()
	st, _ := status.FromError(err)

	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code mismatch: got %v", st.Code())
	}

	var foundInfo, foundBR, domainOK bool
	for _, d := range st.Details() {
		switch x := d.(type) {
		case *errdetails.ErrorInfo:
			foundInfo = true
			if x.GetReason() != "validation_failed" {
				t.Fatalf("reason mismatch: %s", x.GetReason())
			}
			if x.GetMetadata()["email"] != "invalid_email" {
				t.Fatalf("metadata mismatch")
			}
			if x.GetDomain() == "auth-service" {
				domainOK = true
			}
		case *errdetails.BadRequest:
			foundBR = true
			if len(x.FieldViolations) == 0 || x.FieldViolations[0].GetField() != "email" {
				t.Fatalf("badrequest violations missing")
			}
		}
	}
	if !foundInfo || !foundBR || !domainOK {
		t.Fatalf("missing details: ErrorInfo=%v BadRequest=%v Domain=%v", foundInfo, foundBR, domainOK)
	}

	back := FromGRPC(err)
	if back.Domain != "auth-service" {
		t.Fatalf("domain didn't roundtrip")
	}
	if back.Details["email"] != "invalid_email" {
		t.Fatalf("details didn't roundtrip: %+v", back.Details)
	}
}

func TestGRPCRateLimited(t *testing.T) {
	err := GRPCRateLimited(1500 * time.Millisecond)
	st, _ := status.FromError(err)
	if st.Code() != codes.ResourceExhausted {
		t.Fatalf("wrong code: %v", st.Code())
	}
	hasRetry := false
	for _, d := range st.Details() {
		if _, ok := d.(*errdetails.RetryInfo); ok {
			hasRetry = true
		}
	}
	if !hasRetry {
		t.Fatalf("missing RetryInfo details")
	}
}

func TestGRPCRateLimited_NegativeClampedToZero(t *testing.T) {
	err := GRPCRateLimited(-1500 * time.Millisecond)
	st, _ := status.FromError(err)

	if st.Code() != codes.ResourceExhausted {
		t.Fatalf("wrong code: %v", st.Code())
	}

	var hasZeroRetry, hasZeroMetadata bool
	for _, d := range st.Details() {
		switch x := d.(type) {
		case *errdetails.RetryInfo:
			if x.GetRetryDelay().AsDuration() == 0 {
				hasZeroRetry = true
			}
		case *errdetails.ErrorInfo:
			if x.GetMetadata()["retry_after_ms"] == "0" {
				hasZeroMetadata = true
			}
		}
	}

	if !hasZeroRetry || !hasZeroMetadata {
		t.Fatalf("expected zero retry details, got RetryInfo=%v Metadata=%v", hasZeroRetry, hasZeroMetadata)
	}
}

func TestToGRPCAndFromGRPC_PreservesViolationReason(t *testing.T) {
	e := ValidationViolations([]FieldViolation{{
		Field:       "email",
		Reason:      "invalid_email",
		Description: "Email format is invalid",
	}})

	back := FromGRPC(e.ToGRPC())
	if len(back.Violations) != 1 {
		t.Fatalf("expected one violation, got %+v", back.Violations)
	}

	v := back.Violations[0]
	if v.Field != "email" {
		t.Fatalf("field mismatch: %+v", v)
	}
	if v.Reason != "invalid_email" {
		t.Fatalf("reason mismatch: %+v", v)
	}
	if v.Description != "Email format is invalid" {
		t.Fatalf("description mismatch: %+v", v)
	}
	if len(back.Details) != 0 {
		t.Fatalf("internal violation metadata must not leak to details: %+v", back.Details)
	}
}
