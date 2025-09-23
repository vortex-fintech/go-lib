package netutil

import "time"

func SanitizeTimeout(d, min, fallback time.Duration) time.Duration {
	if d < 0 {
		return fallback
	}
	if min > 0 && d < min {
		return min
	}
	return d
}
