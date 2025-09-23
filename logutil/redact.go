package logutil

import (
	"regexp"
	"strings"
)

var defaultSensitiveRe = regexp.MustCompile(`(?i)(password|pass|secret|token|otp)`)

func SanitizeValidationErrors(
	fields map[string]string,
	env string,
	replacement string,
	sensitiveKeys ...string,
) map[string]string {
	if fields == nil {
		return nil
	}

	e := strings.ToLower(env)
	if e == "development" || e == "debug" {
		return fields
	}

	if replacement == "" {
		replacement = "[REDACTED]"
	}

	sanitized := make(map[string]string, len(fields))

	sens := map[string]struct{}{}
	for _, k := range sensitiveKeys {
		sens[strings.ToLower(k)] = struct{}{}
	}

	for field, msg := range fields {
		lk := strings.ToLower(field)
		if _, ok := sens[lk]; ok || defaultSensitiveRe.MatchString(lk) {
			sanitized[field] = replacement
		} else {
			sanitized[field] = msg
		}
	}

	return sanitized
}
