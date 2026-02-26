package errors

import (
	"testing"
	"time"

	"google.golang.org/grpc/codes"
)

func TestNewf(t *testing.T) {
	e := Newf(codes.PermissionDenied, "forbidden", "role %s cannot %s", "viewer", "delete")
	if e.Code != codes.PermissionDenied {
		t.Fatalf("code mismatch: %v", e.Code)
	}
	if e.Reason != "forbidden" {
		t.Fatalf("reason mismatch: %v", e.Reason)
	}
	if e.Message != "role viewer cannot delete" {
		t.Fatalf("message mismatch: %q", e.Message)
	}
}

func TestDomainConstructors(t *testing.T) {
	if got := Conflict("email", "taken"); got.Code != codes.AlreadyExists || got.Reason != "conflict" || got.Details["email"] != "taken" {
		t.Fatalf("Conflict mismatch: %+v", got)
	}

	if got := Precondition("kyc_required", map[string]string{"account_id": "acc-1"}); got.Code != codes.FailedPrecondition || got.Reason != "kyc_required" || got.Details["account_id"] != "acc-1" {
		t.Fatalf("Precondition mismatch: %+v", got)
	}

	if got := Forbidden("withdraw", "wallet"); got.Code != codes.PermissionDenied || got.Reason != "forbidden" || got.Details["action"] != "withdraw" || got.Details["resource"] != "wallet" {
		t.Fatalf("Forbidden mismatch: %+v", got)
	}

	if got := NotFoundID("invoice", "inv-1"); got.Code != codes.NotFound || got.Details["invoice_id"] != "inv-1" {
		t.Fatalf("NotFoundID mismatch: %+v", got)
	}
}

func TestUnauthorizedConstructor(t *testing.T) {
	noHints := Unauthorized("", "")
	if noHints.Code != codes.Unauthenticated || noHints.Reason != "unauthorized" {
		t.Fatalf("Unauthorized mismatch: %+v", noHints)
	}
	if len(noHints.Details) != 0 {
		t.Fatalf("expected no details without hints, got %+v", noHints.Details)
	}

	withHints := Unauthorized("Bearer", "payments")
	if withHints.Details["auth_scheme"] != "Bearer" || withHints.Details["realm"] != "payments" {
		t.Fatalf("Unauthorized hints mismatch: %+v", withHints.Details)
	}
}

func TestRateLimitedConstructor(t *testing.T) {
	if got := RateLimited(1500 * time.Millisecond); got.Code != codes.ResourceExhausted || got.Details["retry_after_ms"] != "1500" {
		t.Fatalf("RateLimited positive mismatch: %+v", got)
	}

	if got := RateLimited(-1500 * time.Millisecond); got.Details["retry_after_ms"] != "0" {
		t.Fatalf("RateLimited negative must clamp to 0, got %+v", got)
	}
}
