package geo

import "strings"

// NormalizeISO2 trims and uppercases an ASCII ISO2-like code.
//
// Validation here is format-only (two ASCII letters) and does not check
// whether the code is an officially assigned ISO 3166-1 alpha-2 value.
func NormalizeISO2(code string) (string, bool) {
	c := strings.TrimSpace(code)
	if len(c) != 2 {
		return "", false
	}

	b0, b1 := c[0], c[1]
	if !isASCIILetter(b0) || !isASCIILetter(b1) {
		return "", false
	}

	normalized := string([]byte{toUpperASCII(b0), toUpperASCII(b1)})
	return normalized, true
}

// IsValidISO2 validates whether a value can be normalized as a two-letter
// ASCII ISO2-like code.
func IsValidISO2(code string) bool {
	_, ok := NormalizeISO2(code)
	return ok
}

func isASCIILetter(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func toUpperASCII(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - ('a' - 'A')
	}
	return b
}
