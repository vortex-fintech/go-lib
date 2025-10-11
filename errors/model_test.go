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
