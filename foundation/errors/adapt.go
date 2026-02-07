package errors

import "google.golang.org/grpc/codes"

// ToErrorResponse — унифицирует любую ошибку до ErrorResponse (без привязки к транспорту).
func ToErrorResponse(err error) ErrorResponse {
	if e, ok := err.(ErrorResponse); ok {
		return e
	}
	if IsDomainError(err) {
		return ConvertDomainToValidation(err)
	}
	return Internal()
}

// Хелперы, если нужно точечно.
func ToValidation(field, reason string) ErrorResponse {
	return ValidationFields(map[string]string{field: reason})
}

func To(code codes.Code, reason, msg string) ErrorResponse {
	return New(msg, code, nil).WithReason(reason)
}
