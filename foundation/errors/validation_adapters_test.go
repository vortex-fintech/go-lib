package errors

import (
	"strings"
	"testing"

	play "github.com/go-playground/validator/v10"
)

type nestedReq struct {
	User struct {
		Email string `validate:"required,email"`
	} `validate:"required"`
}

func TestFromPlaygroundStructNamespace(t *testing.T) {
	v := play.New()

	req := nestedReq{}
	err := v.Struct(req)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	ves := err.(play.ValidationErrors)

	tagToReason := map[string]string{
		"required": "required",
		"email":    "invalid_email",
	}

	resp := FromPlayground(ves, tagToReason)
	if resp.Code.String() != "InvalidArgument" {
		t.Fatalf("expected InvalidArgument, got %v", resp.Code)
	}

	// Ensure at least one violation path contains "User.Email".
	hasNested := false
	for _, v := range resp.Violations {
		if strings.Contains(v.Field, "User.Email") {
			hasNested = true
			break
		}
	}
	if !hasNested {
		t.Fatalf("expected nested field path containing 'User.Email', got: %+v", resp.Violations)
	}
}
