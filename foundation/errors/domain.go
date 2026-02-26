package errors

import (
	"errors"
	"fmt"
)

// InvariantKind classifies domain invariant failures.
type InvariantKind string

const (
	KindDomain     InvariantKind = "domain"
	KindState      InvariantKind = "state"
	KindTransition InvariantKind = "transition"
)

// InvariantError is a unified type for field/state/transition invariant failures.
type InvariantError struct {
	Kind   InvariantKind
	Base   error
	Field  string
	Reason string
}

func (e InvariantError) Error() string {
	switch e.Kind {
	case KindState:
		if e.Reason == "" {
			if e.Base == nil {
				return "state: invalid"
			}
			return fmt.Sprintf("state: %v", e.Base)
		}
		if e.Base == nil {
			return fmt.Sprintf("state: %s", e.Reason)
		}
		return fmt.Sprintf("state: %v: %s", e.Base, e.Reason)
	case KindTransition:
		if e.Reason == "" {
			if e.Base == nil {
				return "transition: invalid"
			}
			return fmt.Sprintf("transition: %v", e.Base)
		}
		if e.Base == nil {
			return fmt.Sprintf("transition: %s", e.Reason)
		}
		return fmt.Sprintf("transition: %v: %s", e.Base, e.Reason)
	default:
		if e.Field == "" {
			return e.Reason
		}
		return fmt.Sprintf("%s: %s", e.Field, e.Reason)
	}
}

// Unwrap enables errors.Is / errors.As.
func (e InvariantError) Unwrap() error {
	return e.Base
}

func DomainInvariant(field, reason string) error {
	return InvariantError{Kind: KindDomain, Field: field, Reason: reason}
}

func StateInvariant(base error, field, reason string) error {
	return InvariantError{Kind: KindState, Base: base, Field: field, Reason: reason}
}

func TransitionInvariant(base error, field, reason string) error {
	return InvariantError{Kind: KindTransition, Base: base, Field: field, Reason: reason}
}

func IsInvariant(err error) bool {
	var ie InvariantError
	return errors.As(err, &ie)
}
