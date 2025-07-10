//go:build unit
// +build unit

package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vortex-fintech/go-lib/validator"
)

type testStruct struct {
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18"`
}

func TestValidate_Valid(t *testing.T) {
	s := testStruct{Email: "user@example.com", Age: 20}
	res := validator.Validate(s)
	assert.Nil(t, res)
}

func TestValidate_Invalid(t *testing.T) {
	s := testStruct{Email: "", Age: 10}
	res := validator.Validate(s)
	assert.NotNil(t, res)
	assert.Equal(t, "required", res["Email"])
	assert.Equal(t, "too_short", res["Age"])
}

func TestValidate_InvalidEmail(t *testing.T) {
	s := testStruct{Email: "not-an-email", Age: 20}
	res := validator.Validate(s)
	assert.NotNil(t, res)
	assert.Equal(t, "invalid_email", res["Email"])
}

func TestValidate_ErrorType(t *testing.T) {
	// Passing a type that can't be validated (e.g., int)
	res := validator.Validate(123)
	assert.NotNil(t, res)
	assert.Equal(t, "validation_failed", res["_error"])
}

func TestInstance(t *testing.T) {
	assert.NotNil(t, validator.Instance())
}
