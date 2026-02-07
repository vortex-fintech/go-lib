package errors

import (
	"fmt"
	"strings"

	play "github.com/go-playground/validator/v10"
)

// FromPlayground — адаптер go-playground/validator -> InvalidArgument + Violations.
// Пытаемся получить вложенный путь поля: StructNamespace() -> (fallback) Namespace() без корня.
func FromPlayground(err play.ValidationErrors, tagToReason map[string]string) ErrorResponse {
	violations := make([]FieldViolation, 0, len(err))
	for _, fe := range err {
		tag := fe.Tag()
		reason := tagToReason[tag]
		if reason == "" {
			reason = "invalid"
		}

		field := fe.StructNamespace()
		if field == "" || field == fe.Field() {
			// fallback: Namespace() может быть "Type.User.Email" — отрежем "Type."
			ns := fe.Namespace()
			if i := strings.Index(ns, "."); i >= 0 && i+1 < len(ns) {
				ns = ns[i+1:]
			}
			if ns != "" {
				field = ns
			} else {
				field = fe.Field()
			}
		}

		violations = append(violations, FieldViolation{
			Field:       field,
			Reason:      reason,
			Description: fmt.Sprintf("%s validation failed (%s)", field, tag),
		})
	}
	return ValidationViolations(violations)
}
