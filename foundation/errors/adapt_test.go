package errors

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestToErrorResponseContextCanceled(t *testing.T) {
	err := fmt.Errorf("request aborted: %w", context.Canceled)
	out := ToErrorResponse(err)

	if out.Code != codes.Canceled {
		t.Fatalf("expected Canceled, got %v", out.Code)
	}
	if out.Reason != Reason("canceled") {
		t.Fatalf("expected reason=canceled, got %v", out.Reason)
	}
}

func TestToErrorResponseContextDeadlineExceeded(t *testing.T) {
	err := fmt.Errorf("request timeout: %w", context.DeadlineExceeded)
	out := ToErrorResponse(err)

	if out.Code != codes.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", out.Code)
	}
	if out.Reason != Reason("deadline_exceeded") {
		t.Fatalf("expected reason=deadline_exceeded, got %v", out.Reason)
	}
}

func TestToErrorResponseNil(t *testing.T) {
	out := ToErrorResponse(nil)
	if out.Code != codes.Internal || out.Reason != Reason("unexpected_error") {
		t.Fatalf("unexpected nil adaptation: %+v", out)
	}
}

func TestToValidationAndTo(t *testing.T) {
	v := ToValidation("email", "invalid_email")
	if v.Code != codes.InvalidArgument || v.Details["email"] != "invalid_email" {
		t.Fatalf("unexpected validation adaptation: %+v", v)
	}

	e := To(codes.PermissionDenied, "forbidden", "no access")
	if e.Code != codes.PermissionDenied || e.Reason != "forbidden" || e.Message != "no access" {
		t.Fatalf("unexpected To(...) result: %+v", e)
	}
}
