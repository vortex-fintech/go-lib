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
	// Since — удобный сахар для измерения интервалов.
	Since(t time.Time) time.Duration
	// Sleep — "спать" d, с поддержкой отмены через ctx.
	Sleep(ctx context.Context, d time.Duration) error
}

// ===== Реализации =====

// UTCClock — системные часы в UTC.
type UTCClock struct{}

func (UTCClock) Now() time.Time                  { return time.Now().UTC() }
func (UTCClock) Since(t time.Time) time.Duration { return time.Since(t) }
func (UTCClock) Sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		// немедленный возврат (идемпотентно и удобно для тестов)
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
// Полезно для корректировок/NTP-оффсетов или симуляций.
type OffsetClock struct {
	Base   Clock
	Offset time.Duration
}

func (c OffsetClock) Now() time.Time {
	base := c.Base
	if base == nil {
		base = Default
	}
	return base.Now().Add(c.Offset)
}
func (c OffsetClock) Since(t time.Time) time.Duration { return time.Since(t) }
func (c OffsetClock) Sleep(ctx context.Context, d time.Duration) error {
	base := c.Base
	if base == nil {
		base = Default
	}
	return base.Sleep(ctx, d)
}

// FrozenClock — фиксированное время с возможностью ручного сдвига.
// Удобно для unit-тестов бизнес-логики.
type FrozenClock struct {
	mu sync.RWMutex
	t  time.Time // ожидаем UTC, но не принуждаем (оставим ответственность вызывающему)
}

func NewFrozenClock(t time.Time) *FrozenClock {
	return &FrozenClock{t: t}
}
func (c *FrozenClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.t
}
func (c *FrozenClock) Since(t time.Time) time.Duration { return time.Since(t) }
func (c *FrozenClock) Sleep(ctx context.Context, d time.Duration) error {
	// Для простоты используем реальное ожидание (обычно в unit-тестах не зовут Sleep).
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
func (c *FrozenClock) Set(t time.Time) {
	c.mu.Lock()
	c.t = t
	c.mu.Unlock()
}
func (c *FrozenClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// ===== Глобальные помощники =====

// Default — глобальные часы по умолчанию (UTC).
var Default Clock = UTCClock{}

// Now — алиас для Default.Now() (UTC).
func Now() time.Time { return Default.Now() }

// PtrNow — указатель на Now() (UTC).
func PtrNow() *time.Time { t := Default.Now(); return &t }

// StartOfDay — начало суток для заданной локали (возвращает в UTC).
// Пример: StartOfDay(Now(), time.FixedZone("Asia/Bangkok", 7*3600))
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	local := t.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return start.UTC()
}
