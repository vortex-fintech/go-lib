package logutil

import (
	"maps"
	"strings"
	"unicode"
)

var defaultSensitiveTokens = map[string]struct{}{
	"password": {},
	"pass":     {},
	"passcode": {},
	"secret":   {},
	"token":    {},
	"otp":      {},
	"pin":      {},
	"cvv":      {},
	"cvc":      {},
	"pan":      {},
	"iban":     {},
	"account":  {},
	"routing":  {},
	"swift":    {},
}

func SanitizeValidationErrors(
	fields map[string]string,
	env string,
	replacement string,
	sensitiveKeys ...string,
) map[string]string {
	return sanitizeValidationErrors(fields, env, replacement, false, sensitiveKeys...)
}

// SanitizeValidationErrorsStrict always redacts sensitive values regardless of environment.
// Use this variant in production fintech write and audit flows.
func SanitizeValidationErrorsStrict(
	fields map[string]string,
	replacement string,
	sensitiveKeys ...string,
) map[string]string {
	return sanitizeValidationErrors(fields, "", replacement, true, sensitiveKeys...)
}

func sanitizeValidationErrors(
	fields map[string]string,
	env string,
	replacement string,
	forceRedact bool,
	sensitiveKeys ...string,
) map[string]string {
	if fields == nil {
		return nil
	}

	e := strings.ToLower(strings.TrimSpace(env))
	if !forceRedact && (e == "development" || e == "debug") {
		out := make(map[string]string, len(fields))
		maps.Copy(out, fields)
		return out
	}

	if replacement == "" {
		replacement = "[REDACTED]"
	}

	sanitized := make(map[string]string, len(fields))

	sensExact := map[string]struct{}{}
	sensTokens := map[string]struct{}{}
	for _, k := range sensitiveKeys {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" {
			continue
		}
		sensExact[k] = struct{}{}
		for _, tok := range tokenizeKey(k) {
			sensTokens[tok] = struct{}{}
		}
	}

	for field, msg := range fields {
		if isSensitiveField(field, sensExact, sensTokens) {
			sanitized[field] = replacement
		} else {
			sanitized[field] = msg
		}
	}

	return sanitized
}

func isSensitiveField(field string, sensExact, sensTokens map[string]struct{}) bool {
	fieldNorm := strings.ToLower(strings.TrimSpace(field))
	if fieldNorm == "" {
		return false
	}
	if _, ok := sensExact[fieldNorm]; ok {
		return true
	}
	if _, ok := defaultSensitiveTokens[fieldNorm]; ok {
		return true
	}

	for _, tok := range tokenizeKey(strings.TrimSpace(field)) {
		if _, ok := defaultSensitiveTokens[tok]; ok {
			return true
		}
		if _, ok := sensTokens[tok]; ok {
			return true
		}
	}

	return false
}

func tokenizeKey(s string) []string {
	if s == "" {
		return nil
	}

	var b strings.Builder
	b.Grow(len(s))

	var prevLowerOrDigit bool
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			isUpper := unicode.IsUpper(r)
			if isUpper && prevLowerOrDigit {
				b.WriteByte(' ')
			}
			b.WriteRune(unicode.ToLower(r))
			prevLowerOrDigit = unicode.IsLower(r) || unicode.IsDigit(r)
		default:
			b.WriteByte(' ')
			prevLowerOrDigit = false
		}
	}

	return strings.Fields(b.String())
}
