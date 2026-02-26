# gRPC Middleware

Collection of gRPC interceptors for production-ready services.

## Available middleware

| Middleware | Purpose | Unary | Stream |
|-------------|---------|--------|--------|
| `recoverymw` | Panic recovery | ✅ | ✅ |
| `contextcancel` | Early context cancel detection | ✅ | ✅ |
| `errorsmw` | Convert domain errors to gRPC status | ✅ | ✅ |
| `metricsmw` | Prometheus metrics (latency, codes) | ✅ | ✅ |
| `deadlinemw` | Enforce timeout limits | ✅ | ❌ |
| `drainmw` | Graceful server drain | ✅ | ✅ |
| `idempotencymw` | Idempotency key extraction | ✅ | ❌ |
| `circuitbreaker` | Protect against cascading failures | ✅ | ✅ |
| `authz` | Authorization checks | ✅ | ❌ |

## Quick start with chain

```go
import (
    "github.com/vortex-fintech/go-lib/transport/grpc/middleware/chain"
    "github.com/vortex-fintech/go-lib/transport/grpc/middleware/recoverymw"
    "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metricsmw"
    promreporter "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metricsmw/promreporter"
)

server := grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            recoverymw.Unary(recoverymw.Options{OnPanic: panicLogger}),
            deadlinemw.Unary(deadlinemw.Config{
                DefaultTimeout: 30 * time.Second,
                MaxTimeout:     2 * time.Minute,
            }),
            contextcancel.Unary(),
            idempotencymw.Unary(idempotencymw.Config{
                ResolvePrincipal: resolveUser,
            }),
        },
        AuthzInterceptor: authz.UnaryServerInterceptor(authzConfig),
        CircuitBreaker:   circuitbreaker.New(),
        Post: []grpc.UnaryServerInterceptor{
            metricsmw.UnaryFull(promReporter),
            errorsmw.Unary(),
        },
    }),
)
```

## Middleware order matters

### Pre middleware (run before handler)

1. **recoverymw** - MUST be first, catches panics
2. **drainmw** - early reject during shutdown
3. **deadlinemw** - enforce server-side timeouts
4. **contextcancel** - check for canceled context
5. **idempotencymw** - extract idempotency key
6. **authz** - authorization checks

### Post middleware (run after handler)

7. **metricsmw** - record latency/status codes
8. **errorsmw** - convert domain errors to gRPC status

### Middlewares with special placement

- **circuitbreaker** - placed in chain.Config, runs in middle
- **authz** - placed in chain.Config, runs in middle

## Individual packages

### recoverymw
Panic recovery - always use in production.

[README](./recoverymw/README.md)

### contextcancel
Early detection of canceled contexts to avoid wasted work.

[README](./contextcancel/README.md)

### errorsmw
Converts domain errors to gRPC status errors with proper codes.

[README](./errorsmw/README.md)

### metricsmw
Collects Prometheus metrics for latency and status codes.

[README](./metricsmw/README.md)

### deadlinemw
Enforces timeout limits with method-specific overrides.

[README](./deadlinemw/README.md)

### drainmw
Graceful server draining for zero-downtime deployments.

[README](./drainmw/README.md)

### idempotencymw
Extracts idempotency key for idempotent operations.

[README](./idempotencymw/README.md)

### circuitbreaker
Circuit breaker pattern for fault tolerance.

[README](./circuitbreaker/README.md)

### authz
Authorization middleware with role-based access control.

[README](./authz/README.md)

## Production checklist

- [ ] Add `recoverymw` FIRST in chain
- [ ] Configure `deadlinemw` with appropriate timeouts
- [ ] Add `metricsmw` for observability
- [ ] Use `errorsmw` to convert domain errors
- [ ] Configure `circuitbreaker` for external dependencies
- [ ] Add `drainmw` for graceful shutdown
- [ ] Use `idempotencymw` for mutating operations
- [ ] Configure `authz` for authorization
- [ ] Add `contextcancel` to avoid wasted work

## Testing middleware

```go
import "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metricsmw"

fakeReporter := &fakeFullReporter{}
interceptor := metricsmw.UnaryFull(fakeReporter)

resp, err := interceptor(ctx, req, info, handler)
```

Each middleware package includes unit tests as examples.
