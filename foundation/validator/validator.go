package validator

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var v *validator.Validate

var asciiEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func init() {
	v = validator.New()
	v.RegisterValidation("ascii_email", validateASCIIEmail)
}

func Instance() *validator.Validate {
	return v
}

func Validate(i any) map[string]string {
	if err := v.Struct(i); err != nil {
		if errs, ok := err.(validator.ValidationErrors); ok {
			out := make(map[string]string)
			for _, e := range errs {
				out[fieldPath(e)] = mapTagToCode(e.Tag())
			}
			return out
		}
		return map[string]string{"_error": "validation_failed"}
	}
	return nil
}

func fieldPath(e validator.FieldError) string {
	field := e.StructNamespace()
	if field == "" || field == e.Field() {
		field = e.Namespace()
	}
	if i := strings.Index(field, "."); i >= 0 && i+1 < len(field) {
		field = field[i+1:]
	}
	if field == "" {
		field = e.Field()
	}
	return field
}

func validateASCIIEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	if email == "" {
		return true
	}
	for i := 0; i < len(email); i++ {
		if email[i] > 127 {
			return false
		}
	}
	return asciiEmailRegex.MatchString(email)
}
