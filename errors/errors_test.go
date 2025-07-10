//go:build unit
// +build unit

package errors_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vortex-fintech/go-lib/errors"
	"google.golang.org/grpc/codes"
)

func TestValidationError(t *testing.T) {
	fields := map[string]string{"field1": "is required", "field2": "invalid"}
	errResp := errors.ValidationError(fields)

	assert.Equal(t, codes.InvalidArgument, errResp.Code)
	assert.Equal(t, "Validation failed", errResp.Message)
	assert.Equal(t, fields, errResp.Details)
}

func TestUnsupportedError(t *testing.T) {
	errResp := errors.UnsupportedError("type", "xml")

	assert.Equal(t, codes.InvalidArgument, errResp.Code)
	assert.Equal(t, "Unsupported type", errResp.Message)
	assert.Equal(t, map[string]string{"type": "xml"}, errResp.Details)
}

func TestNotFound(t *testing.T) {
	errResp := errors.NotFound("User", "12345")

	assert.Equal(t, codes.NotFound, errResp.Code)
	assert.Equal(t, "User not found", errResp.Message)
	assert.Equal(t, map[string]string{"User": "12345"}, errResp.Details)
}

func TestNewError(t *testing.T) {
	details := map[string]string{"foo": "bar"}
	errResp := errors.NewError("custom error", codes.Aborted, details)

	assert.Equal(t, codes.Aborted, errResp.Code)
	assert.Equal(t, "custom error", errResp.Message)
	assert.Equal(t, details, errResp.Details)
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name    string
		errResp errors.ErrorResponse
		code    codes.Code
		message string
	}{
		{"UnknownError", errors.UnknownError, codes.Unknown, "Unknown error occurred"},
		{"InvalidArgumentError", errors.InvalidArgumentError, codes.InvalidArgument, "Invalid argument"},
		{"DeadlineExceededError", errors.DeadlineExceededError, codes.DeadlineExceeded, "Deadline exceeded"},
		{"NotFoundError", errors.NotFoundError, codes.NotFound, "Resource not found"},
		{"AlreadyExistsError", errors.AlreadyExistsError, codes.AlreadyExists, "Resource already exists"},
		{"PermissionDeniedError", errors.PermissionDeniedError, codes.PermissionDenied, "Access denied"},
		{"ResourceExhaustedError", errors.ResourceExhaustedError, codes.ResourceExhausted, "Quota or limit exceeded"},
		{"FailedPreconditionError", errors.FailedPreconditionError, codes.FailedPrecondition, "Operation cannot be performed in the current state"},
		{"AbortedError", errors.AbortedError, codes.Aborted, "Request aborted"},
		{"OutOfRangeError", errors.OutOfRangeError, codes.OutOfRange, "Value out of range"},
		{"UnimplementedError", errors.UnimplementedError, codes.Unimplemented, "Not implemented"},
		{"InternalError", errors.InternalError, codes.Internal, "Internal error"},
		{"UnavailableError", errors.UnavailableError, codes.Unavailable, "Service unavailable"},
		{"DataLossError", errors.DataLossError, codes.DataLoss, "Data loss occurred"},
		{"UnauthenticatedError", errors.UnauthenticatedError, codes.Unauthenticated, "Unauthenticated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.code, tt.errResp.Code)
			assert.Equal(t, tt.message, tt.errResp.Message)
			assert.Nil(t, tt.errResp.Details)
		})
	}
}

func TestValidationError_EmptyFields(t *testing.T) {
	errResp := errors.ValidationError(nil)
	assert.Equal(t, codes.InvalidArgument, errResp.Code)
	assert.Equal(t, "Validation failed", errResp.Message)
	assert.Nil(t, errResp.Details)
}

func TestUnsupportedError_EmptyNameValue(t *testing.T) {
	errResp := errors.UnsupportedError("", "")
	assert.Equal(t, codes.InvalidArgument, errResp.Code)
	assert.Equal(t, "Unsupported ", errResp.Message)
	assert.True(t, reflect.DeepEqual(map[string]string{"": ""}, errResp.Details))
}

func TestNotFound_EmptyResourceValue(t *testing.T) {
	errResp := errors.NotFound("", "")
	assert.Equal(t, codes.NotFound, errResp.Code)
	assert.Equal(t, " not found", errResp.Message)
	assert.True(t, reflect.DeepEqual(map[string]string{"": ""}, errResp.Details))
}
