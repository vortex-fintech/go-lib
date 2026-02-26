package timeutil

import "time"

// InPeriod returns true when t is inside half-open interval [from, to).
// Boundaries can be nil:
//   - from == nil -> lower bound is -infinity
//   - to == nil   -> upper bound is +infinity
func InPeriod(from, to *time.Time, t time.Time) bool {
	if from != nil && t.Before(*from) {
		return false
	}
	// to is exclusive boundary: t >= to => false
	if to != nil && !t.Before(*to) {
		return false
	}
	return true
}
