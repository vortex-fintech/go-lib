package contactutil

import "strings"

// NormalizeEmail lowercases and trims an e-mail string.
// It normalizes only and does not validate format.
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// NormalizeE164 trims a phone value expected to be in E.164 format.
// Validation/formatting must be handled by upper layers.
func NormalizeE164(s string) string {
	return strings.TrimSpace(s)
}
