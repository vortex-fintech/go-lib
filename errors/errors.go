package errors

import "google.golang.org/grpc/codes"

var (
	UnknownError = ErrorResponse{
		Code:    codes.Unknown,
		Message: "Unknown error occurred",
	}

	InvalidArgumentError = ErrorResponse{
		Code:    codes.InvalidArgument,
		Message: "Invalid argument",
	}

	DeadlineExceededError = ErrorResponse{
		Code:    codes.DeadlineExceeded,
		Message: "Deadline exceeded",
	}

	NotFoundError = ErrorResponse{
		Code:    codes.NotFound,
		Message: "Resource not found",
	}

	AlreadyExistsError = ErrorResponse{
		Code:    codes.AlreadyExists,
		Message: "Resource already exists",
	}

	PermissionDeniedError = ErrorResponse{
		Code:    codes.PermissionDenied,
		Message: "Access denied",
	}

	ResourceExhaustedError = ErrorResponse{
		Code:    codes.ResourceExhausted,
		Message: "Quota or limit exceeded",
	}

	FailedPreconditionError = ErrorResponse{
		Code:    codes.FailedPrecondition,
		Message: "Operation cannot be performed in the current state",
	}

	AbortedError = ErrorResponse{
		Code:    codes.Aborted,
		Message: "Request aborted",
	}

	OutOfRangeError = ErrorResponse{
		Code:    codes.OutOfRange,
		Message: "Value out of range",
	}

	UnimplementedError = ErrorResponse{
		Code:    codes.Unimplemented,
		Message: "Not implemented",
	}

	InternalError = ErrorResponse{
		Code:    codes.Internal,
		Message: "Internal error",
	}

	UnavailableError = ErrorResponse{
		Code:    codes.Unavailable,
		Message: "Service unavailable",
	}

	DataLossError = ErrorResponse{
		Code:    codes.DataLoss,
		Message: "Data loss occurred",
	}

	UnauthenticatedError = ErrorResponse{
		Code:    codes.Unauthenticated,
		Message: "Unauthenticated",
	}
)

func ValidationError(fields map[string]string) ErrorResponse {
	return ErrorResponse{
		Code:    codes.InvalidArgument,
		Message: "Validation failed",
		Details: fields,
	}
}

func UnsupportedError(name, value string) ErrorResponse {
	return ErrorResponse{
		Code:    codes.InvalidArgument,
		Message: "Unsupported " + name,
		Details: map[string]string{
			name: value,
		},
	}
}

func NotFound(resource string, value string) ErrorResponse {
	return ErrorResponse{
		Code:    codes.NotFound,
		Message: resource + " not found",
		Details: map[string]string{
			resource: value,
		},
	}
}

func NewError(message string, code codes.Code, details map[string]string) ErrorResponse {
	return ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}
}
