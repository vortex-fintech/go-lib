package errors

import (
	"fmt"

	play "github.com/go-playground/validator/v10"
)

// FromPlayground — адаптер go-playground/validator -> InvalidArgument + Violations.
func FromPlayground(err play.ValidationErrors, tagToReason map[string]string) ErrorResponse {
	violations := make([]FieldViolation, 0, len(err))
	for _, fe := range err {
		tag := fe.Tag()
		reason := tagToReason[tag]
		if reason == "" {
			reason = "invalid"
		}
		violations = append(violations, FieldViolation{
			Field:       fe.Field(),
			Reason:      reason,
			Description: fmt.Sprintf("%s validation failed (%s)", fe.Field(), tag),
		})
	}
	return ValidationViolations(violations)
}
