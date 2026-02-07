package domainutil

import (
	"testing"
	"time"
)

func TestIsUTC(t *testing.T) {
	t.Parallel()

	if !IsUTC(time.Now().UTC()) {
		t.Fatalf("expected UTC time to be detected")
	}
	if IsUTC(time.Now().In(time.FixedZone("UTC+3", 3*60*60))) {
		t.Fatalf("expected non-UTC offset to be rejected")
	}
}

func TestCloneTimePtrUTC(t *testing.T) {
	t.Parallel()

	if CloneTimePtrUTC(nil) != nil {
		t.Fatalf("nil input must return nil")
	}

	orig := time.Date(2026, 2, 8, 12, 30, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	cloned := CloneTimePtrUTC(&orig)
	if cloned == nil {
		t.Fatalf("expected cloned pointer")
	}
	if cloned == &orig {
		t.Fatalf("expected a cloned pointer, got same address")
	}
	if !cloned.Equal(orig) {
		t.Fatalf("expected same instant, got %v and %v", *cloned, orig)
	}
	if !IsUTC(*cloned) {
		t.Fatalf("expected cloned time to be in UTC")
	}
}

func TestNextRevisionState(t *testing.T) {
	t.Parallel()

	t.Run("uses at when newer", func(t *testing.T) {
		updatedAt := time.Date(2026, 2, 8, 9, 0, 0, 0, time.UTC)
		at := time.Date(2026, 2, 8, 13, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))

		nextAt, nextRev := NextRevisionState(updatedAt, 41, at)
		if !nextAt.Equal(at.UTC()) {
			t.Fatalf("expected at.UTC, got %v", nextAt)
		}
		if !IsUTC(nextAt) {
			t.Fatalf("expected UTC output")
		}
		if nextRev != 42 {
			t.Fatalf("expected revision 42, got %d", nextRev)
		}
	})

	t.Run("clamps to updatedAt and keeps utc", func(t *testing.T) {
		updatedAt := time.Date(2026, 2, 8, 12, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
		at := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)

		nextAt, nextRev := NextRevisionState(updatedAt, 0, at)
		if !nextAt.Equal(updatedAt.UTC()) {
			t.Fatalf("expected updatedAt.UTC, got %v", nextAt)
		}
		if !IsUTC(nextAt) {
			t.Fatalf("expected UTC output")
		}
		if nextRev != 1 {
			t.Fatalf("expected revision 1, got %d", nextRev)
		}
	})

	t.Run("revision floor", func(t *testing.T) {
		nextAt, nextRev := NextRevisionState(time.Now().UTC(), -5, time.Now().UTC())
		if nextRev != 1 {
			t.Fatalf("expected revision floor at 1, got %d", nextRev)
		}
		if !IsUTC(nextAt) {
			t.Fatalf("expected UTC output")
		}
	})
}
