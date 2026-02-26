# gRPC Deadline Middleware

Enforces timeout limits on incoming gRPC requests.

## Where to use it

- Services that need guaranteed request timeouts
- Protecting against clients with excessive deadlines
- Ensuring consistent timeout behavior across all methods

## How it works

1. If client sent no deadline → applies `DefaultTimeout`
2. If client deadline exceeds `MaxTimeout` → caps to `MaxTimeout`
3. If client deadline is within limits → keeps client deadline
4. Method-specific timeouts override `DefaultTimeout`

## Basic usage

```go
import "github.com/vortex-fintech/go-lib/transport/grpc/middleware/deadlinemw"

server := grpc.NewServer(
    grpc.UnaryInterceptor(deadlinemw.Unary(deadlinemw.Config{
        DefaultTimeout: 30 * time.Second,
        MaxTimeout:     2 * time.Minute,
    })),
)
```

## Config

| Field | Description |
|-------|-------------|
| `DefaultTimeout` | Applied when client sends no deadline |
| `MaxTimeout` | Maximum allowed deadline (caps both client and default) |
| `MethodTimeouts` | Per-method overrides, e.g. `{"/svc/SlowOp": 5*time.Minute}` |

## Method-specific timeouts

```go
deadlinemw.Unary(deadlinemw.Config{
    DefaultTimeout: 30 * time.Second,
    MaxTimeout:     2 * time.Minute,
    MethodTimeouts: map[string]time.Duration{
        "/svc/Export":     10 * time.Minute,
        "/svc/Health":     5 * time.Second,
        "/svc/StreamData": 30 * time.Minute,
    },
})
```

## Behavior matrix

| Client deadline | DefaultTimeout | MaxTimeout | Applied deadline |
|-----------------|----------------|------------|------------------|
| none | 30s | 2m | 30s |
| none | none | 2m | 2m |
| none | none | none | none |
| 1m | 30s | 2m | 30s (client > default) |
| 10s | 30s | 2m | 10s (client < default) |
| 5m | 30s | 2m | 2m (client > max) |
| 5m | none | 2m | 2m (client > max) |

## With chain

```go
return grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            deadlinemw.Unary(deadlinemw.Config{
                DefaultTimeout: 30 * time.Second,
                MaxTimeout:     2 * time.Minute,
            }),
        },
    }),
)
```

## Production notes

- Place early in middleware chain (before expensive operations)
- `MaxTimeout` protects against misbehaving clients
- Set `DefaultTimeout` to your p99 latency goal
- Use `MethodTimeouts` for known slow operations (exports, batch jobs)
- Only Unary interceptor (stream deadlines managed by gRPC runtime)
