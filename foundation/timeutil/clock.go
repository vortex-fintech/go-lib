package timeutil

import (
	"context"
	"sync"
	"time"
)

// Clock abstracts a time source.
type Clock interface {
	// Now returns current time (UTC expected by convention).
	Now() time.Time
	// Since is a convenience wrapper over Now().Sub(t).
	Since(t time.Time) time.Duration
	// Sleep waits for d and supports cancellation via ctx.
	Sleep(ctx context.Context, d time.Duration) error
}

// ===== Implementations =====

// UTCClock uses system time in UTC.
type UTCClock struct{}

func (UTCClock) Now() time.Time { return time.Now().UTC() }

// Important: use Clock.Now() for consistency with custom clocks.
func (c UTCClock) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

func (UTCClock) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// OffsetClock applies a fixed offset relative to Base clock.
// Important: when Base=nil, do not reference DefaultClock() to avoid recursion
// if defaultClock == OffsetClock{Base:nil,...}.
type OffsetClock struct {
	Base   Clock
	Offset time.Duration
}

func (c OffsetClock) base() Clock {
	if c.Base != nil {
		return c.Base
	}
	// Safe fallback independent from global DefaultClock().
	return UTCClock{}
}

func (c OffsetClock) Now() time.Time { return c.base().Now().Add(c.Offset) }

func (c OffsetClock) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

func (c OffsetClock) Sleep(ctx context.Context, d time.Duration) error {
	return c.base().Sleep(ctx, d)
}

// FrozenClock keeps fixed time with manual advancement.
type FrozenClock struct {
	mu sync.RWMutex
	t  time.Time // always UTC
}

func NewFrozenClock(t time.Time) *FrozenClock { return &FrozenClock{t: t.UTC()} }

func (c *FrozenClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.t
}

func (c *FrozenClock) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

// Sleep does not block in tests; it just advances frozen time.
func (c *FrozenClock) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		c.Advance(d)
		return nil
	}
}

func (c *FrozenClock) Set(t time.Time) {
	c.mu.Lock()
	c.t = t.UTC()
	c.mu.Unlock()
}

func (c *FrozenClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d) // already UTC
	c.mu.Unlock()
}

// ===== Global helpers (thread-safe) =====

var (
	defaultMu    sync.RWMutex
	defaultClock Clock = UTCClock{}
)

func DefaultClock() Clock {
	defaultMu.RLock()
	c := defaultClock
	defaultMu.RUnlock()
	return c
}

// SetDefault sets global clock and returns previous value.
func SetDefault(c Clock) (prev Clock) {
	if c == nil {
		c = UTCClock{}
	}

	defaultMu.Lock()
	prev = defaultClock
	defaultClock = c
	defaultMu.Unlock()
	return prev
}

// WithDefault sets a clock and returns restore function.
func WithDefault(c Clock) (restore func()) {
	prev := SetDefault(c)
	return func() { SetDefault(prev) }
}

// Now is an alias for DefaultClock().Now().
// It guarantees UTC even if custom clock returns non-UTC location.
func Now() time.Time { return DefaultClock().Now().UTC() }

func PtrNow() *time.Time {
	t := Now()
	return &t
}

// Since is a convenience wrapper over DefaultClock().Since(t).
func Since(t time.Time) time.Duration { return DefaultClock().Since(t) }

// Sleep is a convenience wrapper over DefaultClock().Sleep(ctx, d).
func Sleep(ctx context.Context, d time.Duration) error { return DefaultClock().Sleep(ctx, d) }

// StartOfDay returns local day start converted to UTC.
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := t.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return start.UTC()
}

// Monotonic returns now only when it is strictly after prev.
// If now <= prev (for example due to clock skew), it returns prev.
// This guarantees a non-decreasing timestamp sequence.
func Monotonic(now, prev time.Time) time.Time {
	if !prev.IsZero() && !now.After(prev) {
		return prev
	}
	return now
}
