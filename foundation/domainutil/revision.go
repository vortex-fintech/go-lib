package domainutil

import (
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrInvalidExpectedRevision = errors.New("invalid expected revision")
	ErrRevisionConflict        = errors.New("revision conflict")
)

type InvalidExpectedRevisionError struct {
	Expected int64
}

func (e *InvalidExpectedRevisionError) Error() string {
	return fmt.Sprintf("%s: expected=%d", ErrInvalidExpectedRevision, e.Expected)
}

func (e *InvalidExpectedRevisionError) Is(target error) bool {
	return target == ErrInvalidExpectedRevision
}

type RevisionConflictError struct {
	Current  int64
	Expected int64
}

func (e *RevisionConflictError) Error() string {
	return fmt.Sprintf("%s: current=%d expected=%d", ErrRevisionConflict, e.Current, e.Expected)
}

func (e *RevisionConflictError) Is(target error) bool {
	return target == ErrRevisionConflict
}

func IsUTC(t time.Time) bool {
	return t.Location() == time.UTC
}

func CloneTimePtrUTC(p *time.Time) *time.Time {
	if p == nil {
		return nil
	}
	t := p.UTC()
	return &t
}

func UTCOrZero(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC()
}

func NextRevisionState(updatedAt time.Time, revision int64, at time.Time) (time.Time, int64) {
	return NextRevisionStateWithCeiling(updatedAt, revision, at, time.Now().UTC())
}

func NextRevisionStateWithCeiling(updatedAt time.Time, revision int64, at, ceiling time.Time) (time.Time, int64) {
	updated := updatedAt.UTC()
	t := at.UTC()
	maxAt := ceiling.UTC()

	if maxAt.Before(updated) {
		maxAt = updated
	}
	if t.Before(updated) {
		t = updated
	}
	if t.After(maxAt) {
		t = maxAt
	}

	var rev int64
	switch {
	case revision < 0:
		rev = 1
	case revision == math.MaxInt64:
		rev = math.MaxInt64
	default:
		rev = revision + 1
	}

	return t, rev
}

func RequireRevision(current, expected int64) error {
	if expected <= 0 {
		return &InvalidExpectedRevisionError{Expected: expected}
	}
	if current != expected {
		return &RevisionConflictError{Current: current, Expected: expected}
	}
	return nil
}
