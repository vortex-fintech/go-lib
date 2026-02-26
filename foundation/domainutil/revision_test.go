package domainutil

import (
	"errors"
	"math"
	"strings"
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
	if IsUTC(time.Now().In(time.FixedZone("UTC0", 0))) {
		t.Fatalf("expected non-time.UTC location to be rejected")
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

func TestUTCOrZero(t *testing.T) {
	t.Parallel()

	zero := time.Time{}
	if got := UTCOrZero(zero); !got.IsZero() {
		t.Fatalf("zero time must stay zero, got %v", got)
	}

	local := time.Date(2026, 2, 9, 10, 0, 0, 0, time.FixedZone("UTC+5", 5*60*60))
	got := UTCOrZero(local)
	if !got.Equal(local) {
		t.Fatalf("expected same instant, got %v and %v", got, local)
	}
	if !IsUTC(got) {
		t.Fatalf("expected UTC output")
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

	t.Run("revision max saturates", func(t *testing.T) {
		nextAt, nextRev := NextRevisionState(time.Now().UTC(), math.MaxInt64, time.Now().UTC())
		if nextRev != math.MaxInt64 {
			t.Fatalf("expected saturated revision %d, got %d", int64(math.MaxInt64), nextRev)
		}
		if !IsUTC(nextAt) {
			t.Fatalf("expected UTC output")
		}
	})
}

func TestNextRevisionStateWithCeiling(t *testing.T) {
	t.Parallel()

	t.Run("clamps client future timestamp", func(t *testing.T) {
		updatedAt := time.Date(2026, 2, 8, 10, 0, 0, 0, time.UTC)
		at := time.Date(2026, 2, 8, 13, 0, 0, 0, time.UTC)
		ceiling := time.Date(2026, 2, 8, 11, 0, 0, 0, time.UTC)

		nextAt, nextRev := NextRevisionStateWithCeiling(updatedAt, 7, at, ceiling)
		if !nextAt.Equal(ceiling) {
			t.Fatalf("expected ceiling clamp, got %v", nextAt)
		}
		if nextRev != 8 {
			t.Fatalf("expected revision 8, got %d", nextRev)
		}
		if !IsUTC(nextAt) {
			t.Fatalf("expected UTC output")
		}
	})

	t.Run("keeps monotonic timestamp when updatedAt already above ceiling", func(t *testing.T) {
		updatedAt := time.Date(2026, 2, 8, 13, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
		at := time.Date(2026, 2, 8, 8, 0, 0, 0, time.UTC)
		ceiling := time.Date(2026, 2, 8, 9, 0, 0, 0, time.UTC)

		nextAt, nextRev := NextRevisionStateWithCeiling(updatedAt, 10, at, ceiling)
		if !nextAt.Equal(updatedAt.UTC()) {
			t.Fatalf("expected updatedAt.UTC to preserve monotonicity, got %v", nextAt)
		}
		if nextRev != 11 {
			t.Fatalf("expected revision 11, got %d", nextRev)
		}
		if !IsUTC(nextAt) {
			t.Fatalf("expected UTC output")
		}
	})
}

func TestRequireRevision(t *testing.T) {
	t.Parallel()

	t.Run("invalid expected revision", func(t *testing.T) {
		err := RequireRevision(5, 0)
		if !errors.Is(err, ErrInvalidExpectedRevision) {
			t.Fatalf("got=%v, want ErrInvalidExpectedRevision", err)
		}

		var typedErr *InvalidExpectedRevisionError
		if !errors.As(err, &typedErr) {
			t.Fatalf("expected InvalidExpectedRevisionError, got %T", err)
		}
		if typedErr.Expected != 0 {
			t.Fatalf("expected Expected=0, got %d", typedErr.Expected)
		}
		if got := typedErr.Error(); !strings.Contains(got, "expected=0") {
			t.Fatalf("expected error details in message, got %q", got)
		}
	})

	t.Run("negative expected revision", func(t *testing.T) {
		err := RequireRevision(5, -2)
		if !errors.Is(err, ErrInvalidExpectedRevision) {
			t.Fatalf("got=%v, want ErrInvalidExpectedRevision", err)
		}

		var typedErr *InvalidExpectedRevisionError
		if !errors.As(err, &typedErr) {
			t.Fatalf("expected InvalidExpectedRevisionError, got %T", err)
		}
		if typedErr.Expected != -2 {
			t.Fatalf("expected Expected=-2, got %d", typedErr.Expected)
		}
		if got := typedErr.Error(); !strings.Contains(got, "expected=-2") {
			t.Fatalf("expected error details in message, got %q", got)
		}
	})

	t.Run("revision conflict", func(t *testing.T) {
		err := RequireRevision(5, 4)
		if !errors.Is(err, ErrRevisionConflict) {
			t.Fatalf("got=%v, want ErrRevisionConflict", err)
		}

		var typedErr *RevisionConflictError
		if !errors.As(err, &typedErr) {
			t.Fatalf("expected RevisionConflictError, got %T", err)
		}
		if typedErr.Current != 5 || typedErr.Expected != 4 {
			t.Fatalf("expected current=5 expected=4, got current=%d expected=%d", typedErr.Current, typedErr.Expected)
		}
		if got := typedErr.Error(); !strings.Contains(got, "current=5") || !strings.Contains(got, "expected=4") {
			t.Fatalf("expected error details in message, got %q", got)
		}
	})

	t.Run("match", func(t *testing.T) {
		if err := RequireRevision(5, 5); err != nil {
			t.Fatalf("expected nil, got=%v", err)
		}
	})
}
