package errors

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
)

// HTTPStatus — маппинг gRPC codes → HTTP.
func HTTPStatus(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
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

// ToHTTP — пишет JSON-тело совместимое с ErrorResponse.
func (e ErrorResponse) ToHTTP(w http.ResponseWriter) {
	status := HTTPStatus(e.Code)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code       string            `json:"code"`
		Reason     Reason            `json:"reason,omitempty"`
		Message    string            `json:"message"`
		Details    map[string]string `json:"details,omitempty"`
		Violations []FieldViolation  `json:"violations,omitempty"`
	}{
		Code:       e.Code.String(),
		Reason:     e.Reason,
		Message:    e.Message,
		Details:    e.Details,
		Violations: e.Violations,
	})
}
