package timeutil

import "time"

// FirstDayOfNextMonthUTC returns 00:00:00 UTC at the first day
// of the month immediately following t.
func FirstDayOfNextMonthUTC(t time.Time) time.Time {
	ut := t.UTC()
	y, m, _ := ut.Date()
	if m == time.December {
		return time.Date(y+1, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
}

// IsNotFutureUTC returns true when at is not in the future
// compared to now after normalizing both values to UTC.
// Zero values are treated as invalid and return false.
func IsNotFutureUTC(now, at time.Time) bool {
	if now.IsZero() || at.IsZero() {
		return false
	}
	return !at.UTC().After(now.UTC())
}
