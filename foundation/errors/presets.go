package errors

import "google.golang.org/grpc/codes"

// Фабричные функции (неизменяемые пресеты).
func Unknown() ErrorResponse {
	return New("Unknown error occurred", codes.Unknown, nil).WithReason("unknown")
}
func InvalidArgument() ErrorResponse {
	return New("Invalid argument", codes.InvalidArgument, nil).WithReason("invalid_argument")
}
func DeadlineExceeded() ErrorResponse {
	return New("Deadline exceeded", codes.DeadlineExceeded, nil).WithReason("deadline_exceeded")
}
func NotFound() ErrorResponse {
	return New("Resource not found", codes.NotFound, nil).WithReason("not_found")
}
func AlreadyExists() ErrorResponse {
	return New("Resource already exists", codes.AlreadyExists, nil).WithReason("already_exists")
}
func PermissionDenied() ErrorResponse {
	return New("Access denied", codes.PermissionDenied, nil).WithReason("permission_denied")
}
func ResourceExhausted() ErrorResponse {
	return New("Quota or limit exceeded", codes.ResourceExhausted, nil).WithReason("resource_exhausted")
}
func FailedPrecondition() ErrorResponse {
	return New("Operation cannot be performed in the current state", codes.FailedPrecondition, nil).WithReason("failed_precondition")
}
func Aborted() ErrorResponse { return New("Request aborted", codes.Aborted, nil).WithReason("aborted") }
func OutOfRange() ErrorResponse {
	return New("Value out of range", codes.OutOfRange, nil).WithReason("out_of_range")
}
func Unimplemented() ErrorResponse {
	return New("Not implemented", codes.Unimplemented, nil).WithReason("unimplemented")
}
func Internal() ErrorResponse {
	return New("Internal error", codes.Internal, nil).WithReason("internal")
}
func Unavailable() ErrorResponse {
	return New("Service unavailable", codes.Unavailable, nil).WithReason("unavailable")
}
func DataLoss() ErrorResponse {
	return New("Data loss occurred", codes.DataLoss, nil).WithReason("data_loss")
}
func Unauthenticated() ErrorResponse {
	return New("Unauthenticated", codes.Unauthenticated, nil).WithReason("unauthenticated")
}

// Быстрые конструкторы частых кейсов
func ValidationFields(fields map[string]string) ErrorResponse {
	return InvalidArgument().WithReason("validation_failed").WithDetails(fields).WithViolations(ViolationsFromMap(fields))
}

func ValidationViolations(v []FieldViolation) ErrorResponse {
	return InvalidArgument().WithReason("validation_failed").WithViolations(v)
}

func Unsupported(name, value string) ErrorResponse {
	return InvalidArgument().WithReason("unsupported").WithDetail(name, value)
}

func NotFoundWith(resourceKey, value string) ErrorResponse {
	return NotFound().WithDetail(resourceKey, value)
}
