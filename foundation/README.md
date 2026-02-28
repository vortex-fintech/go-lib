# foundation

Shared Go libraries for Vortex fintech services.

## Packages

| Package | Description |
|---------|-------------|
| [contactutil](./contactutil) | Contact normalization (email, phone E.164) |
| [domain](./domain) | Domain primitives for event-driven flows |
| [domainutil](./domainutil) | Revision-based consistency, UTC helpers |
| [errors](./errors) | Transport-agnostic error model (HTTP/gRPC) |
| [geo](./geo) | Country code normalization (ISO 3166-1 alpha-2) |
| [hash](./hash) | Deterministic hashing for idempotency keys |
| [idutil](./idutil) | Type-safe UUID v7 identifiers with generics |
| [logger](./logger) | Zap-based structured logger with tracing |
| [logutil](./logutil) | Log-safe output sanitization |
| [netutil](./netutil) | Timeout sanitization for network clients |
| [piiutil](./piiutil) | PII masking for logs and responses |
| [retry](./retry) | Retry helpers for transient failures |
| [textutil](./textutil) | Text canonicalization and validation |
| [timeutil](./timeutil) | Time abstraction for deterministic tests |
| [validator](./validator) | Request validation with fintech rules |

## Quick Start

```go
import (
    "github.com/vortex-fintech/go-lib/foundation/logger"
    "github.com/vortex-fintech/go-lib/foundation/errors"
    "github.com/vortex-fintech/go-lib/foundation/validator"
    "github.com/vortex-fintech/go-lib/foundation/piiutil"
)

func main() {
    // Initialize logger
    log, err := logger.New("payment-service", "production")
    if err != nil {
        panic(err)
    }
    defer log.SafeSync()

    // Validate request
    if errs := validator.Validate(req); errs != nil {
        log.Warnw("validation failed", "errors", errs)
        errors.ValidationFields(errs).ToHTTP(w)
        return
    }

    // Log with masked PII
    log.Infow("payment processed",
        "email", piiutil.MaskEmail(user.Email),
        "phone", piiutil.MaskPhone(user.Phone),
    )
}
```

## By Category

### Core Infrastructure

| Package | Purpose |
|---------|---------|
| [logger](./logger) | Structured logging with trace_id/request_id |
| [errors](./errors) | Unified error model across HTTP/gRPC |
| [validator](./validator) | Request validation |

### Domain & Events

| Package | Purpose |
|---------|---------|
| [domain](./domain) | Event primitives, EventBuffer |
| [domainutil](./domainutil) | Revision CAS, UTC time helpers |
| [idutil](./idutil) | Typed identifiers for domain entities |

### Data Handling

| Package | Purpose |
|---------|---------|
| [contactutil](./contactutil) | Normalize email/phone |
| [textutil](./textutil) | Text canonicalization |
| [geo](./geo) | Country code normalization |
| [hash](./hash) | Idempotency key hashing |

### Security & Privacy

| Package | Purpose |
|---------|---------|
| [piiutil](./piiutil) | Mask PII in logs/responses |
| [logutil](./logutil) | Sanitize sensitive fields |

### Networking & Reliability

| Package | Purpose |
|---------|---------|
| [retry](./retry) | Exponential backoff, permanent errors |
| [netutil](./netutil) | Timeout sanitization |

### Testing Utilities

| Package | Purpose |
|---------|---------|
| [timeutil](./timeutil) | FrozenClock, deterministic time |

## Common Patterns

### Service Bootstrap

```go
log, err := logger.New("service-name", env)
if err != nil {
    return fmt.Errorf("bootstrap logger: %w", err)
}
defer log.SafeSync()
```

### Request Validation

```go
if errs := validator.Validate(req); errs != nil {
    return errors.ValidationFields(errs)
}
```

### Error Handling

```go
account, err := repo.GetAccount(ctx, id)
if err != nil {
    return errors.NotFoundID("account", id)
}
```

### PII Logging

```go
log.Infow("user action",
    "email", piiutil.MaskEmail(user.Email),
    "phone", piiutil.MaskPhone(user.Phone),
)
```

### Deterministic Testing

```go
frozen := timeutil.NewFrozenClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
restore := timeutil.WithDefault(frozen)
defer restore()
```

## Testing

```bash
# Run all tests
go test -tags unit ./foundation/...

# Run with race detector
go test -race -tags unit ./foundation/...
```
