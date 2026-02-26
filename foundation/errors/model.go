package errors

import (
	"encoding/json"

	"google.golang.org/grpc/codes"
)

// Reason is a stable machine-readable code.
type Reason string

type FieldViolation struct {
	Field       string `json:"field"`
	Reason      string `json:"reason,omitempty"`
	Description string `json:"description,omitempty"`
}

type ErrorResponse struct {
	Code       codes.Code        `json:"code"`
	Reason     Reason            `json:"reason,omitempty"`
	Domain     string            `json:"domain,omitempty"` // optional service domain (for example "auth-service")
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
	Violations []FieldViolation  `json:"violations,omitempty"`
}

func New(message string, code codes.Code, details map[string]string) ErrorResponse {
	return ErrorResponse{Code: code, Message: message, Details: cloneDetails(details)}
}

func (e ErrorResponse) WithReason(r string) ErrorResponse { e.Reason = Reason(r); return e }
func (e ErrorResponse) WithDomain(d string) ErrorResponse { e.Domain = d; return e }

func (e ErrorResponse) WithDetail(k, v string) ErrorResponse {
	if e.Details == nil {
		e.Details = map[string]string{k: v}
		return e
	}

	// Copy-on-write to keep builder-style methods immutable.
	details := cloneDetails(e.Details)
	if details == nil {
		details = map[string]string{}
	}
	details[k] = v
	e.Details = details
	return e
}

func (e ErrorResponse) WithDetails(m map[string]string) ErrorResponse {
	if len(m) == 0 {
		return e
	}
	if e.Details == nil {
		e.Details = cloneDetails(m)
		return e
	}

	// Copy-on-write to keep builder-style methods immutable.
	details := cloneDetails(e.Details)
	if details == nil {
		details = map[string]string{}
	}
	for k, v := range m {
		details[k] = v
	}
	e.Details = details
	return e
}

func (e ErrorResponse) WithViolations(v []FieldViolation) ErrorResponse {
	if len(v) == 0 {
		return e
	}
	e.Violations = append([]FieldViolation(nil), v...)
	return e
}

func (e ErrorResponse) ToString() string {
	type out struct {
		Code       string            `json:"code"`
		Reason     Reason            `json:"reason,omitempty"`
		Domain     string            `json:"domain,omitempty"`
		Message    string            `json:"message"`
		Details    map[string]string `json:"details,omitempty"`
		Violations []FieldViolation  `json:"violations,omitempty"`
	}
	b, _ := json.Marshal(out{
		Code:       e.Code.String(),
		Reason:     e.Reason,
		Domain:     e.Domain,
		Message:    e.Message,
		Details:    e.Details,
		Violations: e.Violations,
	})
	return string(b)
}

func (e ErrorResponse) Error() string { return e.ToString() }

func ViolationsFromMap(m map[string]string) []FieldViolation {
	if len(m) == 0 {
		return nil
	}
	out := make([]FieldViolation, 0, len(m))
	for f, r := range m {
		out = append(out, FieldViolation{Field: f, Reason: r})
	}
	return out
}

func cloneDetails(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
