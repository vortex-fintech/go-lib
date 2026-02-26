package piiutil

import "strings"

// MaskPhone masks a phone value while preserving formatting symbols.
// It keeps the last 1 or 4 digits:
//   - if total digits <= 4 -> keep 1 last digit
//   - if total digits > 4  -> keep 4 last digits
//
// Examples:
//
//	"+1234567890"       -> "+******7890"
//	"+1234"             -> "+***4"
//	"123"               -> "**3"
//	"12"                -> "*2"
//	"1"                 -> "1"
//	"AB-CD" (no digits) -> "**-*D" (mask letters except the last significant)
func MaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}

	runes := []rune(phone)
	if !maskDigitsKeepLast4Or1(runes) {
		return maskLettersAndDigitsKeepLast(runes, 1)
	}
	return string(runes)
}
