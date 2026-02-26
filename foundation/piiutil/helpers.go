package piiutil

import "unicode"

const (
	shortDigitCountThreshold = 4
	keepShortDigits          = 1
	keepLongDigits           = 4
)

// maskDigitsKeepLast4Or1 masks digits in place and keeps:
//   - 1 last digit when total digits <= 4
//   - 4 last digits when total digits > 4
//
// It returns false when there are no digits at all.
func maskDigitsKeepLast4Or1(runes []rune) bool {
	totalDigits := 0
	for _, r := range runes {
		if unicode.IsDigit(r) {
			totalDigits++
		}
	}
	if totalDigits == 0 {
		return false
	}

	keepDigits := keepLongDigits
	if totalDigits <= shortDigitCountThreshold {
		keepDigits = keepShortDigits
	}

	digitsSeen := 0
	for i := len(runes) - 1; i >= 0; i-- {
		if unicode.IsDigit(runes[i]) {
			digitsSeen++
			if digitsSeen > keepDigits {
				runes[i] = '*'
			}
		}
	}

	return true
}

// maskLettersAndDigitsKeepLast masks all letters/digits except last keep significant ones.
func maskLettersAndDigitsKeepLast(runes []rune, keep int) string {
	n := len(runes)
	if n == 0 {
		return ""
	}
	if keep < 1 {
		keep = 1
	}

	total := 0
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			total++
		}
	}
	if total == 0 {
		return string(runes)
	}
	if keep > total {
		keep = total
	}

	seen := 0
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			seen++
			if seen > keep {
				runes[i] = '*'
			}
		}
	}
	return string(runes)
}
