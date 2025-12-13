package timeutil

import (
	"context"
	"sync"
	"time"
)

// Clock — абстракция источника времени.
type Clock interface {
	// Now возвращает текущее время (ожидаем UTC).
	Now() time.Time
	// Since — сахар для измерения интервалов относительно Now().
	Since(t time.Time) time.Duration
	// Sleep — "спать" d, с поддержкой отмены через ctx.
	Sleep(ctx context.Context, d time.Duration) error
}

// ===== Реализации =====

// UTCClock — системные часы в UTC.
type UTCClock struct{}

func (UTCClock) Now() time.Time { return time.Now().UTC() }

// Важно: считаем разницу относительно Clock.Now(), а не time.Since.
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

// OffsetClock — часы с постоянным смещением относительно Base.
// ВАЖНО: Base=nil НЕ должен ссылаться на DefaultClock(), иначе можно словить рекурсию,
// если defaultClock == OffsetClock{Base:nil,...}.
type OffsetClock struct {
	Base   Clock
	Offset time.Duration
}

func (c OffsetClock) base() Clock {
	if c.Base != nil {
		return c.Base
	}
	// безопасный fallback, не зависящий от глобального DefaultClock()
	return UTCClock{}
}

func (c OffsetClock) Now() time.Time { return c.base().Now().Add(c.Offset) }

func (c OffsetClock) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

func (c OffsetClock) Sleep(ctx context.Context, d time.Duration) error {
	return c.base().Sleep(ctx, d)
}

// FrozenClock — фиксированное время с возможностью ручного сдвига.
type FrozenClock struct {
	mu sync.RWMutex
	t  time.Time // всегда UTC
}

func NewFrozenClock(t time.Time) *FrozenClock { return &FrozenClock{t: t.UTC()} }

func (c *FrozenClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.t
}

func (c *FrozenClock) Since(t time.Time) time.Duration { return c.Now().Sub(t) }

// Sleep — для тестов НЕ делает реального ожидания: просто двигает время.
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
	c.t = c.t.Add(d) // уже UTC
	c.mu.Unlock()
}

// ===== Глобальные помощники (потокобезопасно) =====

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

// SetDefault ставит глобальный clock и возвращает предыдущий (удобно для тестов).
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

// WithDefault ставит clock и возвращает restore-функцию.
func WithDefault(c Clock) (restore func()) {
	prev := SetDefault(c)
	return func() { SetDefault(prev) }
}

// Now — алиас для DefaultClock().Now().
// Ожидаем UTC по контракту Clock.
func Now() time.Time { return DefaultClock().Now() }

// Since — сахар для DefaultClock().Since(t).
func Since(t time.Time) time.Duration { return DefaultClock().Since(t) }

// Sleep — сахар для DefaultClock().Sleep(ctx, d).
func Sleep(ctx context.Context, d time.Duration) error { return DefaultClock().Sleep(ctx, d) }

func PtrNow() *time.Time {
	t := Now()
	return &t
}

// StartOfDay — начало суток для заданной локали (возвращает в UTC).
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := t.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return start.UTC()
}
