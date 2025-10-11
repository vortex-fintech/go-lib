package errors

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
)

func TestHTTPMappingAndBody(t *testing.T) {
	e := NotFound().WithReason("not_found").WithDetail("user_id", "123").WithDomain("auth-service")
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
	if !bytes.Contains(body, []byte(`"domain":"auth-service"`)) {
		t.Fatalf("body missing domain")
	}
	if !bytes.Contains(body, []byte(`"user_id":"123"`)) {
		t.Fatalf("body missing details")
	}
}

func TestHTTPWithRetryHeader(t *testing.T) {
	e := RateLimited(1500 * time.Millisecond)
	rec := httptest.NewRecorder()
	e.ToHTTPWithRetry(rec, 1500*time.Millisecond)

	if got := rec.Header().Get("Retry-After"); got != "2" && got != "1" {
		// округление до сек, допускаем 1..2 в зависимости от платформы округления
		t.Fatalf("unexpected Retry-After: %s", got)
	}
	if rec.Code != HTTPStatus(e.Code) {
		t.Fatalf("status mismatch")
	}
}
