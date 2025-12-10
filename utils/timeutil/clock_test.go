package timeutil

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- вспомогательный fake clock для OffsetClock ---

type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time                  { return f.now }
func (f fakeClock) Since(t time.Time) time.Duration { return f.now.Sub(t) }
func (f fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	// в тестах нам не важно реальное ожидание
	return nil
}

// --- UTCClock ---

func TestUTCClockNowIsUTC(t *testing.T) {
	c := UTCClock{}
	now := c.Now()

	if now.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %v", now.Location())
	}
}

func TestUTCClockSinceNonNegative(t *testing.T) {
	c := UTCClock{}
	start := c.Now()

	// чуть подождём
	time.Sleep(5 * time.Millisecond)

	d := c.Since(start)
	if d < 0 {
		t.Fatalf("expected non-negative duration, got %v", d)
	}
}

func TestUTCClockSleepImmediateZeroDuration(t *testing.T) {
	c := UTCClock{}
	ctx := context.Background()

	start := time.Now()
	if err := c.Sleep(ctx, 0); err != nil {
		t.Fatalf("expected nil error for zero duration, got %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 20*time.Millisecond {
		t.Fatalf("Sleep(0) should return immediately, elapsed=%v", elapsed)
	}
}

func TestUTCClockSleepCancelable(t *testing.T) {
	c := UTCClock{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // сразу отменяем

	err := c.Sleep(ctx, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// --- OffsetClock ---

func TestOffsetClockNowUsesBasePlusOffset(t *testing.T) {
	baseNow := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	offset := 2 * time.Hour

	c := OffsetClock{
		Base:   fakeClock{now: baseNow},
		Offset: offset,
	}

	got := c.Now()
	want := baseNow.Add(offset)

	if !got.Equal(want) {
		t.Fatalf("Now() mismatch: got %v, want %v", got, want)
	}
}

func TestOffsetClockSinceUsesOffset(t *testing.T) {
	baseNow := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	offset := 30 * time.Minute

	c := OffsetClock{
		Base:   fakeClock{now: baseNow},
		Offset: offset,
	}

	// измеряем разницу между "теперь" смещённого часа и базовой точкой
	t0 := baseNow
	d := c.Since(t0)

	if d != offset {
		t.Fatalf("expected Since(...)=%v, got %v", offset, d)
	}
}

// --- FrozenClock ---

func TestFrozenClockNowAndAdvance(t *testing.T) {
	start := time.Date(2025, 1, 2, 3, 4, 5, 0, time.FixedZone("Asia/Bangkok", 7*3600))
	c := NewFrozenClock(start)

	if got := c.Now(); !got.Equal(start.UTC()) {
		t.Fatalf("expected initial time %v, got %v", start.UTC(), got)
	}

	advance := 15 * time.Minute
	c.Advance(advance)

	if got := c.Now(); !got.Equal(start.UTC().Add(advance)) {
		t.Fatalf("expected advanced time %v, got %v", start.UTC().Add(advance), got)
	}
}

func TestFrozenClockSet(t *testing.T) {
	start := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	c := NewFrozenClock(start)

	newTime := time.Date(2030, 5, 6, 7, 8, 9, 0, time.FixedZone("Asia/Singapore", 8*3600))
	c.Set(newTime)

	if got := c.Now(); !got.Equal(newTime.UTC()) {
		t.Fatalf("expected %v, got %v", newTime.UTC(), got)
	}
}

func TestFrozenClockSince(t *testing.T) {
	start := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	c := NewFrozenClock(start)

	past := start.Add(-10 * time.Minute)
	d := c.Since(past)

	if d != 10*time.Minute {
		t.Fatalf("expected 10m, got %v", d)
	}
}

// --- Default / Now / PtrNow ---

func TestNowAndPtrNowUseDefaultClock(t *testing.T) {
	// сохраняем старое значение и восстановим после теста
	oldDefault := Default
	defer func() { Default = oldDefault }()

	frozen := NewFrozenClock(time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC))
	Default = frozen

	n := Now()
	if !n.Equal(frozen.Now()) {
		t.Fatalf("Now() should use Default clock: got %v, want %v", n, frozen.Now())
	}

	p := PtrNow()
	if p == nil {
		t.Fatalf("PtrNow() returned nil pointer")
	}
	if !p.Equal(frozen.Now()) {
		t.Fatalf("PtrNow() should point to Default.Now(): got %v, want %v", *p, frozen.Now())
	}
}

// --- StartOfDay ---

func TestStartOfDayUTC(t *testing.T) {
	// 15:04 того же дня -> начало суток в UTC
	tm := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)
	got := StartOfDay(tm, nil)

	want := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("StartOfDay UTC mismatch: got %v, want %v", got, want)
	}
}

func TestStartOfDayWithLocation(t *testing.T) {
	// Время в Бангкоке, но создадим в UTC и потом интерпретируем локалью

	loc := time.FixedZone("Asia/Bangkok", 7*3600)

	// 2025-01-02 15:00 UTC == 2025-01-02 22:00 Asia/Bangkok
	tm := time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC)

	got := StartOfDay(tm, loc)

	// Начало суток в Бангкоке:
	// локально: 2025-01-02 00:00 Asia/Bangkok
	// в UTC: минус 7 часов
	want := time.Date(2025, 1, 1, 17, 0, 0, 0, time.UTC)

	if !got.Equal(want) {
		t.Fatalf("StartOfDay Bangkok mismatch: got %v, want %v", got, want)
	}
}
