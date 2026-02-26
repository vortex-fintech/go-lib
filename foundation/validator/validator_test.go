//go:build unit
// +build unit

package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vortex-fintech/go-lib/foundation/validator"
)

type testStruct struct {
	Email string `validate:"required,email"`
	Age   int    `validate:"min=18"`
}

type nestedStruct struct {
	User struct {
		Email string `validate:"required,email"`
	} `validate:"required"`
}

type asciiEmailStruct struct {
	Email string `validate:"required,ascii_email"`
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
	res := validator.Validate(123)
	assert.NotNil(t, res)
	assert.Equal(t, "validation_failed", res["_error"])
}

func TestInstance(t *testing.T) {
	assert.NotNil(t, validator.Instance())
}

func TestValidate_NestedFieldPath(t *testing.T) {
	var s nestedStruct
	res := validator.Validate(s)
	assert.NotNil(t, res)
	assert.Equal(t, "required", res["User.Email"])
}

func TestASCIIEmail_Valid(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{"standard", "user@example.com", true},
		{"with dots", "user.name@example.com", true},
		{"with plus", "user+tag@example.com", true},
		{"subdomain", "user@mail.example.com", true},
		{"empty", "", false}, // required fails first
		{"unicode local", "юзер@example.com", false},
		{"unicode domain", "user@пример.рф", false},
		{"mixed unicode", "user@exаmple.com", false}, // Cyrillic 'а'
		{"no domain", "user@", false},
		{"no local", "@example.com", false},
		{"no TLD", "user@example", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := asciiEmailStruct{Email: tt.email}
			res := validator.Validate(s)
			if tt.valid {
				assert.Nil(t, res, "expected valid for %q", tt.email)
			} else {
				assert.NotNil(t, res, "expected invalid for %q", tt.email)
			}
		})
	}
}
