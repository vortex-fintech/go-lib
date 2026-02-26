package timeutil_test

import (
	"context"
	"testing"
	"time"

	"github.com/vortex-fintech/go-lib/foundation/timeutil"
)

func TestUTCClock_NowIsUTC(t *testing.T) {
	var c timeutil.UTCClock
	now := c.Now()
	if now.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", now.Location())
	}
}

func TestUTCClock_Sleep_Cancelled(t *testing.T) {
	var c timeutil.UTCClock

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.Sleep(ctx, time.Hour)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestUTCClock_Sleep_NonPositive(t *testing.T) {
	var c timeutil.UTCClock

	ctx := context.Background()
	if err := c.Sleep(ctx, 0); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := c.Sleep(ctx, -time.Second); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestFrozenClock_SleepAdvancesTime(t *testing.T) {
	restore := timeutil.WithDefault(timeutil.NewFrozenClock(time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC)))
	t.Cleanup(restore)

	start := timeutil.Now()
	if start.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", start.Location())
	}

	err := timeutil.Sleep(context.Background(), 10*time.Second)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	got := timeutil.Now()
	if got.Sub(start) != 10*time.Second {
		t.Fatalf("expected +10s, got %v", got.Sub(start))
	}
}

func TestFrozenClock_SleepCancelledDoesNotAdvance(t *testing.T) {
	restore := timeutil.WithDefault(timeutil.NewFrozenClock(time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC)))
	t.Cleanup(restore)

	start := timeutil.Now()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := timeutil.Sleep(ctx, 10*time.Second)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	got := timeutil.Now()
	if !got.Equal(start) {
		t.Fatalf("expected no advance, start=%v got=%v", start, got)
	}
}

func TestOffsetClock_BaseNil_NoRecursionAndUTC(t *testing.T) {
	// OffsetClock{Base:nil} must not depend on DefaultClock().
	restore := timeutil.WithDefault(timeutil.OffsetClock{Base: nil, Offset: time.Hour})
	t.Cleanup(restore)

	now := timeutil.Now()
	if now.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", now.Location())
	}
}

func TestDefaultClock_SetDefaultAndRestore(t *testing.T) {
	orig := timeutil.SetDefault(timeutil.UTCClock{})
	t.Cleanup(func() { timeutil.SetDefault(orig) })

	fc := timeutil.NewFrozenClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	prev := timeutil.SetDefault(fc)
	t.Cleanup(func() { timeutil.SetDefault(prev) })

	if got := timeutil.Now(); !got.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected Now(): %v", got)
	}
}

func TestSince_UsesDefaultClockSince(t *testing.T) {
	restore := timeutil.WithDefault(timeutil.NewFrozenClock(time.Date(2025, 12, 13, 0, 0, 10, 0, time.UTC)))
	t.Cleanup(restore)

	t0 := time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC)
	d := timeutil.Since(t0)
	if d != 10*time.Second {
		t.Fatalf("expected 10s, got %v", d)
	}
}

func TestStartOfDay_ReturnsUTCStartOfLocalDay(t *testing.T) {
	loc := time.FixedZone("BKK", 7*3600)                 // UTC+7
	tm := time.Date(2025, 12, 13, 15, 0, 0, 0, time.UTC) // 22:00 local on Dec 13
	got := timeutil.StartOfDay(tm, loc)

	// Local day = Dec 13, start local = Dec 13 00:00 +07 => Dec 12 17:00 UTC
	want := time.Date(2025, 12, 12, 17, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	if got.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", got.Location())
	}
}

func TestMonotonic_ResolvesBackwardsTime(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(-time.Second) // "now" is before "prev"

	// Should return prev
	got := timeutil.Monotonic(t2, t1)
	if !got.Equal(t1) {
		t.Fatalf("expected prev (t1), got %v", got)
	}

	// Should return now if now > prev
	t3 := t1.Add(time.Second)
	got = timeutil.Monotonic(t3, t1)
	if !got.Equal(t3) {
		t.Fatalf("expected now (t3), got %v", got)
	}

	// Should return prev when equal
	got = timeutil.Monotonic(t1, t1)
	if !got.Equal(t1) {
		t.Fatalf("expected prev when equal, got %v", got)
	}

	// Should return now if prev is zero
	got = timeutil.Monotonic(t2, time.Time{})
	if !got.Equal(t2) {
		t.Fatalf("expected now (t2) when prev is zero, got %v", got)
	}
}
