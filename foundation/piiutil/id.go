package piiutil

import "strings"

// MaskIDLast4 masks identifiers (TaxID, SSN, NRIC, etc.).
// It preserves separators and keeps the last 1 or 4 digits:
//   - if total digits <= 4 -> keep 1 last digit
//   - if total digits > 4  -> keep 4 last digits
//
// If there are no digits, it masks letters/digits except the last significant one.
//
// Examples:
//
//	"123-45-6789"    -> "***-**-6789"
//	"S1234567D"      -> "S***4567D"
//	"AB-1234-CD"     -> "AB-***4-CD"   (short digit count, keep 1)
//	"12-AB"          -> "*2-AB"
//	"ABCD"           -> "***D"
//	"X"              -> "X"
func MaskIDLast4(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	runes := []rune(s)
	if !maskDigitsKeepLast4Or1(runes) {
		return maskLettersAndDigitsKeepLast(runes, 1)
	}
	return string(runes)
}
