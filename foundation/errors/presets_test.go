package errors

import (
	"testing"

	"google.golang.org/grpc/codes"
)

func TestPresetFactories(t *testing.T) {
	cases := []struct {
		name   string
		err    ErrorResponse
		code   codes.Code
		reason Reason
	}{
		{name: "Unknown", err: Unknown(), code: codes.Unknown, reason: "unknown"},
		{name: "InvalidArgument", err: InvalidArgument(), code: codes.InvalidArgument, reason: "invalid_argument"},
		{name: "DeadlineExceeded", err: DeadlineExceeded(), code: codes.DeadlineExceeded, reason: "deadline_exceeded"},
		{name: "NotFound", err: NotFound(), code: codes.NotFound, reason: "not_found"},
		{name: "AlreadyExists", err: AlreadyExists(), code: codes.AlreadyExists, reason: "already_exists"},
		{name: "PermissionDenied", err: PermissionDenied(), code: codes.PermissionDenied, reason: "permission_denied"},
		{name: "ResourceExhausted", err: ResourceExhausted(), code: codes.ResourceExhausted, reason: "resource_exhausted"},
		{name: "FailedPrecondition", err: FailedPrecondition(), code: codes.FailedPrecondition, reason: "failed_precondition"},
		{name: "Aborted", err: Aborted(), code: codes.Aborted, reason: "aborted"},
		{name: "OutOfRange", err: OutOfRange(), code: codes.OutOfRange, reason: "out_of_range"},
		{name: "Unimplemented", err: Unimplemented(), code: codes.Unimplemented, reason: "unimplemented"},
		{name: "Internal", err: Internal(), code: codes.Internal, reason: "internal"},
		{name: "Unavailable", err: Unavailable(), code: codes.Unavailable, reason: "unavailable"},
		{name: "DataLoss", err: DataLoss(), code: codes.DataLoss, reason: "data_loss"},
		{name: "Unauthenticated", err: Unauthenticated(), code: codes.Unauthenticated, reason: "unauthenticated"},
		{name: "Canceled", err: Canceled(), code: codes.Canceled, reason: "canceled"},
	}

	for _, tc := range cases {
		if tc.err.Code != tc.code || tc.err.Reason != tc.reason {
			t.Fatalf("%s mismatch: %+v", tc.name, tc.err)
		}
		if tc.err.Message == "" {
			t.Fatalf("%s must provide default message", tc.name)
		}
	}
}

func TestValidationAndHelpers(t *testing.T) {
	fields := map[string]string{"email": "invalid_email"}
	vf := ValidationFields(fields)
	if vf.Code != codes.InvalidArgument || vf.Reason != "validation_failed" {
		t.Fatalf("ValidationFields mismatch: %+v", vf)
	}
	if vf.Details["email"] != "invalid_email" || len(vf.Violations) != 1 {
		t.Fatalf("ValidationFields details/violations mismatch: %+v", vf)
	}

	vv := ValidationViolations([]FieldViolation{{Field: "amount", Reason: "too_small"}})
	if vv.Code != codes.InvalidArgument || vv.Reason != "validation_failed" || len(vv.Violations) != 1 {
		t.Fatalf("ValidationViolations mismatch: %+v", vv)
	}

	if got := Unsupported("currency", "BTC"); got.Details["currency"] != "BTC" {
		t.Fatalf("Unsupported mismatch: %+v", got)
	}

	if got := NotFoundWith("invoice_id", "inv-1"); got.Details["invoice_id"] != "inv-1" {
		t.Fatalf("NotFoundWith mismatch: %+v", got)
	}

	if got := ViolationsFromMap(map[string]string{"email": "invalid_email", "amount": "too_small"}); len(got) != 2 {
		t.Fatalf("ViolationsFromMap length mismatch: %+v", got)
	}
	if got := ViolationsFromMap(nil); got != nil {
		t.Fatalf("expected nil violations for nil input")
	}
}
