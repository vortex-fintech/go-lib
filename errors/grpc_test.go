package errors

import (
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCAndFromGRPC_ErrorInfo(t *testing.T) {
	e := InvalidArgument().
		WithReason("validation_failed").
		WithDetails(map[string]string{"email": "invalid_email"}).
		WithViolations([]FieldViolation{{Field: "email", Reason: "invalid_email"}})

	err := e.ToGRPC()
	st, _ := status.FromError(err)

	if st.Code() != codes.InvalidArgument {
		t.Fatalf("code mismatch: got %v", st.Code())
	}

	var foundInfo, foundBR bool
	for _, d := range st.Details() {
		switch x := d.(type) {
		case *errdetails.ErrorInfo:
			foundInfo = true
			if x.GetReason() != "validation_failed" {
				t.Fatalf("reason mismatch: %s", x.GetReason())
			}
			if x.Metadata["email"] != "invalid_email" {
				t.Fatalf("metadata mismatch")
			}
		case *errdetails.BadRequest:
			foundBR = true
			if len(x.FieldViolations) == 0 || x.FieldViolations[0].GetField() != "email" {
				t.Fatalf("badrequest violations missing")
			}
		}
	}
	if !foundInfo || !foundBR {
		t.Fatalf("missing details: ErrorInfo=%v BadRequest=%v", foundInfo, foundBR)
	}

	back := FromGRPC(err)
	if back.Code != codes.InvalidArgument || back.Reason != Reason("validation_failed") {
		t.Fatalf("roundtrip mismatch: %+v", back)
	}
	if back.Details["email"] != "invalid_email" {
		t.Fatalf("details mismatch after roundtrip")
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
