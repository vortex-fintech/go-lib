package errors

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestHTTPMappingAndBody(t *testing.T) {
	e := NotFound().WithReason("not_found").WithDetail("user_id", "123")
	rec := httptest.NewRecorder()
	e.ToHTTP(rec)

	if rec.Code != HTTPStatus(codes.NotFound) {
		t.Fatalf("status mismatch: got %d", rec.Code)
	}
	body := rec.Body.Bytes()
	if !bytes.Contains(body, []byte(`"code":"NotFound"`)) {
		t.Fatalf("body missing code")
	}
	if !bytes.Contains(body, []byte(`"reason":"not_found"`)) {
		t.Fatalf("body missing reason")
	}
	if !bytes.Contains(body, []byte(`"user_id":"123"`)) {
		t.Fatalf("body missing details")
	}
}
