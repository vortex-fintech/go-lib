package errors

import (
	"fmt"
	"strings"
)

// одиночная доменная ошибка (инвариант)
type DomainError struct {
	Field  string
	Reason string
}

func (e DomainError) Error() string {
	if e.Field == "" {
		return e.Reason
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

func DomainInvariant(field, reason string) DomainError {
	return DomainError{Field: field, Reason: reason}
}

func IsDomainError(err error) bool {
	_, ok := err.(DomainError)
	return ok
}

// батч доменных ошибок (несколько инвариантов за раз)
type DomainErrors []DomainError

func (es DomainErrors) Error() string {
	if len(es) == 0 {
		return "domain_errors: empty"
	}
	parts := make([]string, 0, len(es))
	for _, e := range es {
		parts = append(parts, e.Error())
	}
	return "domain_errors: " + strings.Join(parts, "; ")
}

func ConvertDomainToValidation(err error) ErrorResponse {
	if e, ok := err.(DomainError); ok {
		return ValidationFields(map[string]string{e.Field: e.Reason})
	}
	return Internal().WithReason("unexpected_domain_error")
}

func ConvertDomainErrorsToValidation(errs DomainErrors) ErrorResponse {
	if len(errs) == 0 {
		return InvalidArgument().WithReason("validation_failed")
	}
	fields := make(map[string]string, len(errs))
	viol := make([]FieldViolation, 0, len(errs))
	for _, e := range errs {
		fields[e.Field] = e.Reason
		viol = append(viol, FieldViolation{Field: e.Field, Reason: e.Reason})
	}
	return ValidationViolations(viol).WithDetails(fields)
}
