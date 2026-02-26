package errors

import (
	"testing"

	"google.golang.org/grpc/codes"
)

func TestErrorResponseToString(t *testing.T) {
	e := New("Invalid argument", codes.InvalidArgument, map[string]string{"email": "invalid_email"}).
		WithReason("validation_failed")
	s := e.ToString()
	if want := `"code":"InvalidArgument"`; !contains(s, want) {
		t.Fatalf("missing %s in %s", want, s)
	}
	if want := `"reason":"validation_failed"`; !contains(s, want) {
		t.Fatalf("missing %s in %s", want, s)
	}
	if want := `"email":"invalid_email"`; !contains(s, want) {
		t.Fatalf("missing %s in %s", want, s)
	}
}

func TestNew_ClonesDetailsMap(t *testing.T) {
	details := map[string]string{"email": "invalid_email"}
	e := New("Invalid argument", codes.InvalidArgument, details)
	details["email"] = "mutated"

	if e.Details["email"] != "invalid_email" {
		t.Fatalf("expected details to be cloned, got %q", e.Details["email"])
	}
}

func TestWithDetail_DoesNotMutateSource(t *testing.T) {
	base := InvalidArgument().WithDetail("email", "invalid_email")
	derived := base.WithDetail("phone", "invalid_phone")

	if _, ok := base.Details["phone"]; ok {
		t.Fatalf("base response mutated: %+v", base.Details)
	}
	if derived.Details["phone"] != "invalid_phone" {
		t.Fatalf("expected derived detail, got %+v", derived.Details)
	}
}

func TestWithDetails_DoesNotMutateSourceAndInput(t *testing.T) {
	base := InvalidArgument().WithDetails(map[string]string{"email": "invalid_email"})
	extra := map[string]string{"phone": "invalid_phone"}
	derived := base.WithDetails(extra)
	extra["phone"] = "mutated"

	if _, ok := base.Details["phone"]; ok {
		t.Fatalf("base response mutated: %+v", base.Details)
	}
	if derived.Details["phone"] != "invalid_phone" {
		t.Fatalf("expected cloned input details, got %+v", derived.Details)
	}
}

func TestError_DelegatesToToString(t *testing.T) {
	e := InvalidArgument().WithDetail("field", "email")
	if e.Error() != e.ToString() {
		t.Fatalf("Error() must match ToString()")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
