package errors

import (
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
)

// Newf — форматированный конструктор с reason.
func Newf(code codes.Code, reason, format string, a ...any) ErrorResponse {
	return New(fmt.Sprintf(format, a...), code, nil).WithReason(reason)
}

// Conflict(field,value) -> 409/AlreadyExists.
func Conflict(field, value string) ErrorResponse {
	return AlreadyExists().WithReason("conflict").WithDetail(field, value)
}

// Precondition(reason, details) -> 412/FailedPrecondition.
func Precondition(reason string, details map[string]string) ErrorResponse {
	return FailedPrecondition().WithReason(reason).WithDetails(details)
}

// Unauthorized с подсказками для клиента.
func Unauthorized(scheme, realm string) ErrorResponse {
	e := Unauthenticated().WithReason("unauthorized")
	if scheme != "" {
		e = e.WithDetail("auth_scheme", scheme)
	}
	if realm != "" {
		e = e.WithDetail("realm", realm)
	}
	return e
}

// Forbidden для RBAC.
func Forbidden(action, resource string) ErrorResponse {
	return PermissionDenied().
		WithReason("forbidden").
		WithDetail("action", action).
		WithDetail("resource", resource)
}

// NotFoundID(resource,id).
func NotFoundID(resource, id string) ErrorResponse {
	return NotFound().
		WithDetail(resource+"_id", id)
}

// RateLimited — с machine-friendly задержкой в мс.
func RateLimited(retryAfter time.Duration) ErrorResponse {
	ms := int(retryAfter / time.Millisecond)
	return ResourceExhausted().
		WithReason("rate_limited").
		WithDetail("retry_after_ms", fmt.Sprintf("%d", ms))
}
