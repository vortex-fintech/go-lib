# timeutil

Time abstraction helpers for deterministic tests and UTC-safe behavior.

## Clock Types

- `UTCClock` - real system UTC clock
- `OffsetClock` - base clock + fixed offset
- `FrozenClock` - controllable test clock

## Global Helpers

- `Now()`, `Since(t)`, `Sleep(ctx, d)`
- `SetDefault(clock)`, `WithDefault(clock)`

## Additional Helpers

- `StartOfDay(t, loc)` -> UTC timestamp of local day start
- `Monotonic(now, prev)` -> non-decreasing timeline guard
- `FirstDayOfNextMonthUTC(t)` -> first day of next month
- `IsNotFutureUTC(now, at)` -> check if time is not in future
- `InPeriod(from, to, t)` -> check if t is in interval

## Example

### Service with Time Dependency

```go
package payment

import (
    "context"
    "time"
    
    "github.com/vortex-fintech/go-lib/foundation/timeutil"
)

type Service struct {
    clock timeutil.Clock
}

func NewService() *Service {
    return &Service{
        clock: timeutil.UTCClock{}, // production
    }
}

func (s *Service) ProcessPayment(ctx context.Context, req *Request) error {
    now := s.clock.Now()
    
    // Check payment deadline
    if !timeutil.IsNotFutureUTC(now, req.Deadline) {
        return ErrExpired
    }
    
    // Sleep with context cancellation
    if err := s.clock.Sleep(ctx, 100*time.Millisecond); err != nil {
        return err // context cancelled
    }
    
    return nil
}
```

### Testing with Frozen Clock

```go
package payment_test

import (
    "testing"
    "time"
    
    "github.com/vortex-fintech/go-lib/foundation/timeutil"
)

func TestPaymentExpiresAfter24Hours(t *testing.T) {
    // Freeze time at specific moment
    frozen := timeutil.NewFrozenClock(time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC))
    restore := timeutil.WithDefault(frozen)
    defer restore()
    
    // Create payment at frozen time
    payment := CreatePayment()
    if !payment.IsActive() {
        t.Fatal("expected active payment")
    }
    
    // Advance time by 23 hours - still active
    frozen.Advance(23 * time.Hour)
    if !payment.IsActive() {
        t.Fatal("expected active after 23h")
    }
    
    // Advance time by 1 more hour - expired
    frozen.Advance(time.Hour)
    if payment.IsActive() {
        t.Fatal("expected expired after 24h")
    }
}

func TestMonotonicTime_PreventsBackwards(t *testing.T) {
    prev := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
    
    // Simulate NTP sync causing backwards time
    now := prev.Add(-5 * time.Second) // 5 seconds before prev
    
    // Monotonic should return prev to prevent backwards time
    got := timeutil.Monotonic(now, prev)
    if !got.Equal(prev) {
        t.Fatalf("expected prev, got %v", got)
    }
    
    // Normal forward time
    now2 := prev.Add(5 * time.Second)
    got2 := timeutil.Monotonic(now2, prev)
    if !got2.Equal(now2) {
        t.Fatalf("expected now2, got %v", got2)
    }
}
```

### Subscription Billing Cycle

```go
package subscription

import (
    "time"
    
    "github.com/vortex-fintech/go-lib/foundation/timeutil"
)

type Subscription struct {
    StartDate time.Time
}

func (s *Subscription) NextBillingDate() time.Time {
    return timeutil.FirstDayOfNextMonthUTC(s.StartDate)
}

func (s *Subscription) IsActive(now time.Time) bool {
    // Active if within billing period
    nextBilling := s.NextBillingDate()
    return timeutil.InPeriod(&s.StartDate, &nextBilling, now)
}
```

### Daily Report Generation

```go
package report

import (
    "time"
    
    "github.com/vortex-fintech/go-lib/foundation/timeutil"
)

func GenerateDailyReport(userTimezone *time.Location) {
    now := timeutil.Now()
    
    // Get start of user's local day in UTC
    dayStart := timeutil.StartOfDay(now, userTimezone)
    
    // Generate report for that day
    // ...
    _ = dayStart
}
```

### Sleep with Context Cancellation

```go
package worker

import (
    "context"
    "time"
    
    "github.com/vortex-fintech/go-lib/foundation/timeutil"
)

func (w *Worker) Run(ctx context.Context) error {
    for {
        if err := w.process(ctx); err != nil {
            return err
        }
        
        // Wait before next iteration, but respect context
        if err := timeutil.Sleep(ctx, 5*time.Second); err != nil {
            return err // context cancelled
        }
    }
}
```

### Time Period Validation

```go
package validation

import (
    "time"
    
    "github.com/vortex-fintech/go-lib/foundation/timeutil"
)

func ValidatePromoPeriod(start, end *time.Time, now time.Time) bool {
    // Check if current time is within promo period
    if !timeutil.InPeriod(start, end, now) {
        return false
    }
    
    // Both boundaries must not be in future
    if start != nil && !timeutil.IsNotFutureUTC(now, *start) {
        return false
    }
    
    return true
}
```

## Business Examples

- **Payment expiration**: Use `FrozenClock` to test time-based expiry without waiting
- **Subscription billing**: Use `FirstDayOfNextMonthUTC` for billing cycles
- **Daily reports**: Use `StartOfDay` for timezone-aware day boundaries
- **Distributed systems**: Use `Monotonic` to prevent backwards timestamps from NTP sync
- **Long-running jobs**: Use `Sleep(ctx, d)` for cancellable waits
- **Promo validation**: Use `InPeriod` and `IsNotFutureUTC` for date range checks

## Testing Pattern

```go
func TestSomething(t *testing.T) {
    // Setup frozen clock
    frozen := timeutil.NewFrozenClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
    restore := timeutil.WithDefault(frozen)
    defer restore()
    
    // Test code using timeutil.Now() is now deterministic
    
    // Advance time as needed
    frozen.Advance(time.Hour)
}
```
