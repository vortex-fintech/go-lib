package piiutil

import "strings"

// MaskEmail masks the local-part of an e-mail while keeping minimal visibility.
// It shows the first and last character of the local-part.
// Examples:
//
//	"user@example.com"    -> "u**r@example.com"
//	"ab@example.com"      -> "a*@example.com"
//	"u@example.com"       -> "u@example.com"  (single char, nothing to hide)
//	"weird"               -> "w***d"
//	"x"                   -> "x"
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}

	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return maskGenericToken(email)
	}

	local := email[:at]
	domain := email[at:] // includes '@'
	localRunes := []rune(local)
	if len(localRunes) < 2 {
		return local + domain
	}

	// For 2 chars, show first and mask second
	if len(localRunes) == 2 {
		return string(localRunes[0]) + "*" + domain
	}

	// Keep first and last char, mask the rest
	var b strings.Builder
	b.Grow(len(local) + len(domain))
	b.WriteRune(localRunes[0])
	for i := 1; i < len(localRunes)-1; i++ {
		b.WriteRune('*')
	}
	b.WriteRune(localRunes[len(localRunes)-1])
	b.WriteString(domain)
	return b.String()
}

// maskGenericToken masks a non-email token keeping first and last rune.
func maskGenericToken(s string) string {
	runes := []rune(s)
	n := len(runes)
	if n == 1 {
		return string(runes)
	}
	if n == 2 {
		return string(runes[0]) + "*"
	}

	var b strings.Builder
	b.Grow(len(s))
	b.WriteRune(runes[0])
	for i := 1; i < n-1; i++ {
		b.WriteByte('*')
	}
	b.WriteRune(runes[n-1])
	return b.String()
}
