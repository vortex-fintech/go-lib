package errors

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
)

const statusClientClosedRequest = 499

func HTTPStatus(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Canceled:
		return statusClientClosedRequest
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func (e ErrorResponse) ToHTTP(w http.ResponseWriter) {
	status := HTTPStatus(e.Code)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code       string            `json:"code"`
		Reason     Reason            `json:"reason,omitempty"`
		Domain     string            `json:"domain,omitempty"`
		Message    string            `json:"message"`
		Details    map[string]string `json:"details,omitempty"`
		Violations []FieldViolation  `json:"violations,omitempty"`
	}{
		Code:       e.Code.String(),
		Reason:     e.Reason,
		Domain:     e.Domain,
		Message:    e.Message,
		Details:    e.Details,
		Violations: e.Violations,
	})
}

// ToHTTPWithRetry sets Retry-After (seconds) and writes the error body.
func (e ErrorResponse) ToHTTPWithRetry(w http.ResponseWriter, retryAfter time.Duration) {
	sec := int(math.Ceil(retryAfter.Seconds()))
	if sec < 0 {
		sec = 0
	}
	w.Header().Set("Retry-After", strconv.Itoa(sec))
	e.ToHTTP(w)
}
