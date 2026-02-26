# domain

Domain primitives for event-driven flows.

## Main Types

- `BaseEvent` - transport-agnostic event metadata
- `EventBuffer` - in-memory event collector

## Example

```go
package main

import (
    "fmt"
    
    "github.com/vortex-fintech/go-lib/foundation/domain"
)

func main() {
    // Create event via constructor (recommended)
    e, err := domain.NewBaseEvent("payment.completed", "payment-service")
    if err != nil {
        panic(err)
    }
    
    // Enrich with tracing context
    e = e.WithTrace("trace-123", "corr-456")
    
    // Add metadata (copy-on-write, safe)
    e = e.WithMeta("transaction_id", "tx-789")
    e = e.WithMeta("amount", "1000.00")
    
    // Buffer with strict validation
    var buf domain.EventBuffer
    if err := buf.RecordStrict(e); err != nil {
        panic(err) // invalid events rejected
    }
    
    // Pull events for publishing (clears buffer)
    events := buf.Pull()
    fmt.Printf("events: %d\n", len(events))
}
```

## Strict Validation APIs

- `BaseEvent.Validate()` - core invariants (name/producer/time/id/schema)
- `BaseEvent.ValidateWithLimits(EventLimits)` - core invariants + size/cardinality limits
- `EventBuffer.RecordStrict(event)` - validates before recording and rejects invalid/non-validatable events
  - uses `ValidateWithLimits(DefaultEventLimits)` when available

## Key Invariants

- event `At` must use strict `time.UTC` location
- event `ID` must be non-nil UUID
- schema version must be positive

## EventBuffer Behavior

- `Record(nil)` is ignored
- `Peek()` returns a cloned snapshot
- `Pull()` returns buffered events and clears the buffer
- implementation is thread-safe for concurrent method calls

## Fintech Recommendation

Use `ValidateWithLimits` and `RecordStrict` on write paths that feed outbox,
ledger, or regulatory audit streams to fail closed on malformed events.

### Minimal Integration Checklist

- create events via `NewBaseEvent`, not manual struct literals
- enrich trace/correlation/causation before buffering
- record with `RecordStrict` and reject on error
- persist outbox events in the same transaction as business state
- emit metrics for every strict-validation rejection reason
