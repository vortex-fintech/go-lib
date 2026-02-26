# logger

Zap-based logger wrapper used across services.

## Entry Point

- `New(serviceName, env) (*Logger, error)`
- `Init(serviceName, env) *Logger` (must-style helper, exits process on init failure)

Supported environments:

- `development`
- `debug`
- `production`

## Context-Aware Logging

Use `*Ctx` methods to automatically extract `trace_id` and `request_id` from context:

```go
func (l *Logger) InfowCtx(ctx context.Context, msg string, kv ...any)
func (l *Logger) WarnwCtx(ctx context.Context, msg string, kv ...any)
func (l *Logger) ErrorwCtx(ctx context.Context, msg string, kv ...any)
func (l *Logger) DebugwCtx(ctx context.Context, msg string, kv ...any)
func (l *Logger) FatalwCtx(ctx context.Context, msg string, kv ...any)
```

### Adding IDs to Context

```go
import "github.com/vortex-fintech/go-lib/foundation/logger"

func handler(ctx context.Context) {
    ctx = logger.ContextWithTraceID(ctx, traceID)
    ctx = logger.ContextWithRequestID(ctx, requestID)
    
    log.InfowCtx(ctx, "processing request", "user_id", userID)
    // Output: {"msg": "processing request", "trace_id": "...", "request_id": "...", "user_id": "..."}
}
```

## Important Behavior

- `New` returns an error and is recommended for service bootstrap code.
- `Init` exits process on configuration/build failure.
- `debug` mode enables caller field in logs for faster incident triage.
- `SafeSync` suppresses common stdout/stderr sync noise (`invalid argument`, `inappropriate ioctl for device`).
- `Fatal` methods call `os.Exit(1)` and cannot be tested directly.

## Recommended Service Integration (Production)

```go
log, err := logger.New("payments-service", env)
if err != nil {
    return fmt.Errorf("bootstrap logger: %w", err)
}
defer log.SafeSync()

log.Infow("service started", "env", env, "version", buildVersion)
```

### In HTTP Handlers

```go
func (h *Handler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    ctx = logger.ContextWithTraceID(ctx, extractTraceID(r))
    ctx = logger.ContextWithRequestID(ctx, generateRequestID())
    
    h.log.InfowCtx(ctx, "payment started", "amount", amount)
    
    if err := h.service.Process(ctx, payment); err != nil {
        h.log.ErrorwCtx(ctx, "payment failed", "error", err)
        // handle error
    }
    
    h.log.InfowCtx(ctx, "payment completed")
}
```

## Business Examples

- **Kubernetes startup**: `New` lets you fail fast with clear bootstrap error and return it to main instead of hard process exit in shared library code.
- **Incident debugging**: `debug` mode includes caller so on-call can jump directly to the source line that produced noisy/repeated errors.
- **Cross-service analytics**: structured `Infow/Warnw/Errorw` logs keep fields machine-parseable for dashboards and alerting.
- **Distributed tracing**: `*Ctx` methods automatically include trace_id for correlating logs across services.

## Recommendation

- Use structured methods (`Infow`, `Warnw`, `Errorw`) for machine-parseable logs.
- Use `*Ctx` methods when you have request context to automatically include trace_id and request_id.

## Tests

Run logger tests with unit tag:

`go test -tags unit ./foundation/logger/...`
