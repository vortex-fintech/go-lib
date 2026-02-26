package netutil

import "time"

// SanitizeTimeout applies basic timeout normalization rules:
//
//   - negative d -> fallback
//   - if min > 0 and d < min -> min
//   - otherwise -> d
//
// Note: when d == 0 and min > 0, this function returns min.
// Use SanitizeTimeoutAllowZero when zero must explicitly mean "no timeout".
func SanitizeTimeout(d, min, fallback time.Duration) time.Duration {
	if d < 0 {
		return fallback
	}
	if min > 0 && d < min {
		return min
	}
	return d
}

// SanitizeTimeoutAllowZero keeps zero timeout as-is and applies SanitizeTimeout
// for all other values.
//
// This is useful for clients where 0 means "disabled timeout".
func SanitizeTimeoutAllowZero(d, min, fallback time.Duration) time.Duration {
	if d == 0 {
		return 0
	}
	return SanitizeTimeout(d, min, fallback)
}
