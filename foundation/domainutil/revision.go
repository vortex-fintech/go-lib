package domainutil

import (
	"errors"
	"time"
)

var (
	ErrInvalidExpectedRevision = errors.New("invalid expected revision")
	ErrRevisionConflict        = errors.New("revision conflict")
)

func IsUTC(t time.Time) bool {
	_, off := t.Zone()
	return off == 0
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
	t := at.UTC()
	if t.Before(updatedAt) {
		t = updatedAt.UTC()
	}
	rev := revision + 1
	if rev <= 0 {
		rev = 1
	}
	return t, rev
}

func RequireRevision(current, expected int64) error {
	if expected <= 0 {
		return ErrInvalidExpectedRevision
	}
	if current != expected {
		return ErrRevisionConflict
	}
	return nil
}
