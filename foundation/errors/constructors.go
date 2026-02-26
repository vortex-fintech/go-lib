package errors

import (
	"fmt"
	"strconv"
	"time"

	"google.golang.org/grpc/codes"
)

// Newf is a formatted constructor with reason.
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

// Unauthorized with optional client hints.
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

// Forbidden for RBAC checks.
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

// RateLimited with machine-friendly retry delay in milliseconds.
func RateLimited(retryAfter time.Duration) ErrorResponse {
	ms := retryAfter.Milliseconds()
	if ms < 0 {
		ms = 0
	}
	return ResourceExhausted().
		WithReason("rate_limited").
		WithDetail("retry_after_ms", strconv.FormatInt(ms, 10))
}
