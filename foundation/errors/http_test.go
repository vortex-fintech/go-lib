package errors

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
)

func TestHTTPStatusMappingTable(t *testing.T) {
	cases := map[codes.Code]int{
		codes.Canceled:           499,
		codes.InvalidArgument:    400,
		codes.DeadlineExceeded:   504,
		codes.NotFound:           404,
		codes.AlreadyExists:      409,
		codes.PermissionDenied:   403,
		codes.ResourceExhausted:  429,
		codes.FailedPrecondition: 412,
		codes.Aborted:            409,
		codes.OutOfRange:         400,
		codes.Unimplemented:      501,
		codes.Internal:           500,
		codes.Unavailable:        503,
		codes.DataLoss:           500,
		codes.Unauthenticated:    401,
	}

	for code, want := range cases {
		if got := HTTPStatus(code); got != want {
			t.Fatalf("HTTPStatus(%v) = %d, want %d", code, got, want)
		}
	}

	if got := HTTPStatus(codes.OK); got != 500 {
		t.Fatalf("HTTPStatus(OK) must fallback to 500, got %d", got)
	}
}

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

	if got := rec.Header().Get("Retry-After"); got != "2" {
		t.Fatalf("unexpected Retry-After: %s", got)
	}
	if rec.Code != HTTPStatus(e.Code) {
		t.Fatalf("status mismatch")
	}
}

func TestHTTPWithRetryHeader_CeilSmallPositive(t *testing.T) {
	e := RateLimited(100 * time.Millisecond)
	rec := httptest.NewRecorder()
	e.ToHTTPWithRetry(rec, 100*time.Millisecond)

	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("unexpected Retry-After: %s", got)
	}
}

func TestHTTPWithRetryHeader_NegativeClampedToZero(t *testing.T) {
	e := RateLimited(-1 * time.Second)
	rec := httptest.NewRecorder()
	e.ToHTTPWithRetry(rec, -1*time.Second)

	if got := rec.Header().Get("Retry-After"); got != "0" {
		t.Fatalf("unexpected Retry-After: %s", got)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"retry_after_ms":"0"`)) {
		t.Fatalf("body missing clamped retry_after_ms")
	}
}
