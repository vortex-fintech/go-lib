package timeutil

import "time"

// InPeriod возвращает true, если момент t попадает в полуинтервал [from, to),
// где from/to могут быть nil:
//   - from == nil -> нижняя граница -∞
//   - to == nil   -> верхняя граница +∞
func InPeriod(from, to *time.Time, t time.Time) bool {
	if from != nil && t.Before(*from) {
		return false
	}
	// to — исключая границу: t >= to => false
	if to != nil && !t.Before(*to) {
		return false
	}
	return true
}
