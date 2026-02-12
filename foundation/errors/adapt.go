package errors

import (
	"errors"

	"google.golang.org/grpc/codes"
)

// ToErrorResponse — унифицирует любую ошибку до ErrorResponse (без привязки к транспорту).
// Поддерживает:
// - ErrorResponse (прямой passthrough)
// - InvariantError (DomainInvariant/StateInvariant/TransitionInvariant)
func ToErrorResponse(err error) ErrorResponse {
	if e, ok := err.(ErrorResponse); ok {
		return e
	}

	var ie InvariantError
	if !errors.As(err, &ie) {
		return Internal().WithReason("unexpected_error")
	}

	switch ie.Kind {
	case KindState, KindTransition:
		resp := FailedPrecondition().
			WithReason("invariant_violation").
			WithDetail("invariant_kind", string(ie.Kind))
		resp = resp.WithDetail("field", ie.Field)
		resp = resp.WithDetail("reason", ie.Reason)
		return resp

	case KindDomain:
		if ie.Field == "" {
			return InvalidArgument().WithReason(ie.Reason)
		}
		return ValidationFields(map[string]string{ie.Field: ie.Reason})

	default:
		return InvalidArgument().WithReason("unknown_invariant")
	}
}

// Хелперы, если нужно точно.
func ToValidation(field, reason string) ErrorResponse {
	return ValidationFields(map[string]string{field: reason})
}

func To(code codes.Code, reason, msg string) ErrorResponse {
	return New(msg, code, nil).WithReason(reason)
}
