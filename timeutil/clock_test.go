package timeutil

import (
	"context"
	"testing"
	"time"
)

func TestUTCClockNowIsUTC(t *testing.T) {
	var c UTCClock
	now := c.Now()
	if now.Location() != time.UTC {
		t.Fatalf("expected UTC, got %v", now.Location())
	}
}

func TestNowHelpers(t *testing.T) {
	t1 := Now()
	if t1.Location() != time.UTC {
		t.Fatalf("Now() must be UTC")
	}
	p := PtrNow()
	if p == nil || p.Location() != time.UTC {
		t.Fatalf("PtrNow() must return non-nil UTC time")
	}
}

func TestOffsetClockNow(t *testing.T) {
	base := &FrozenClock{t: time.Date(2025, 10, 11, 10, 0, 0, 0, time.UTC)}
	c := OffsetClock{Base: base, Offset: 30 * time.Minute}
	got := c.Now()
	want := base.Now().Add(30 * time.Minute)
	if !got.Equal(want) {
		t.Fatalf("OffsetClock.Now mismatch: got %v want %v", got, want)
	}
}

func TestSleepCancel(t *testing.T) {
	var c UTCClock
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем сразу
	start := time.Now()
	err := c.Sleep(ctx, 200*time.Millisecond)
	if err == nil {
		t.Fatalf("expected error on canceled context")
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatalf("sleep should return quickly on cancel")
	}
}

func TestFrozenClockAdvance(t *testing.T) {
	start := time.Date(2025, 10, 11, 11, 0, 0, 0, time.UTC)
	c := NewFrozenClock(start)
	if !c.Now().Equal(start) {
		t.Fatalf("frozen now mismatch")
	}
	c.Advance(2 * time.Hour)
	want := start.Add(2 * time.Hour)
	if !c.Now().Equal(want) {
		t.Fatalf("frozen advance mismatch: got %v want %v", c.Now(), want)
	}
}

func TestStartOfDay(t *testing.T) {
	// Bangkok (UTC+7)
	loc := time.FixedZone("Asia/Bangkok", 7*3600)
	// 15:30 BKK -> start of same day BKK -> back to UTC (which is 17:00 previous day + 7? nope: convert carefully)
	// Let's pick a fixed moment:
	// 2025-10-11 09:45:00Z -> in BKK it's 2025-10-11 16:45:00+07
	tUTC := time.Date(2025, 10, 11, 9, 45, 0, 0, time.UTC)
	sod := StartOfDay(tUTC, loc)
	// Start of day in BKK: 2025-10-11 00:00:00+07 -> in UTC it's 2025-10-10 17:00:00Z
	want := time.Date(2025, 10, 10, 17, 0, 0, 0, time.UTC)
	if !sod.Equal(want) {
		t.Fatalf("StartOfDay mismatch: got %v want %v", sod, want)
	}
}
