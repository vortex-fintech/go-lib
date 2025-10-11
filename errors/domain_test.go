package errors

import (
	"testing"

	"google.golang.org/grpc/codes"
)

func TestDomainInvariantToValidation(t *testing.T) {
	de := DomainInvariant("user.id", "empty")
	if de.Error() != "user.id: empty" {
		t.Fatalf("unexpected DomainError string: %s", de.Error())
	}
	er := ConvertDomainToValidation(de)
	if er.Code != codes.InvalidArgument || er.Reason != Reason("validation_failed") {
		t.Fatalf("invalid mapping: %+v", er)
	}
	if er.Details["user.id"] != "empty" || len(er.Violations) == 0 {
		t.Fatalf("details/violations missing")
	}
}

func TestDomainErrorsBatchToValidation(t *testing.T) {
	es := DomainErrors{
		{Field: "email", Reason: "invalid_email"},
		{Field: "password", Reason: "too_short"},
	}
	er := ConvertDomainErrorsToValidation(es)
	if er.Code != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", er.Code)
	}
	if er.Details["email"] != "invalid_email" || er.Details["password"] != "too_short" {
		t.Fatalf("details missing")
	}
	if len(er.Violations) != 2 {
		t.Fatalf("violations count mismatch")
	}
}

func TestToErrorResponsePassThrough(t *testing.T) {
	in := InvalidArgument().WithReason("bad").WithDetail("x", "y")
	out := ToErrorResponse(in)
	if out.Reason != "bad" || out.Details["x"] != "y" {
		t.Fatalf("passthrough mismatch")
	}
}

func TestToErrorResponseFromDomain(t *testing.T) {
	out := ToErrorResponse(DomainInvariant("email", "invalid_email"))
	if out.Code != codes.InvalidArgument || out.Details["email"] != "invalid_email" {
		t.Fatalf("domain adaptation mismatch: %+v", out)
	}
}
