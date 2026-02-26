# gRPC Circuit Breaker

Circuit breaker interceptor for protecting services from cascading failures.

## Where to use it

- gRPC clients calling downstream services
- Protecting against cascading failures
- Preventing timeout storms on unhealthy services

## State machine

```
CLOSED (normal)
    ↓ N consecutive critical errors
OPEN (blocks all requests)
    ↓ RecoveryTimeout elapsed
HALF-OPEN (allows probe requests)
    ↓ M successful probes → CLOSED
    ↓ 1 failed probe → OPEN
```

## Basic usage

```go
cb := circuitbreaker.New(
    circuitbreaker.WithFailureThreshold(5),
    circuitbreaker.WithRecoveryTimeout(15*time.Second),
)

server := grpc.NewServer(
    chain.Default(chain.Options{
        CircuitBreaker: cb,
    }),
)
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithFailureThreshold(n)` | 5 | Consecutive failures to trip OPEN |
| `WithRecoveryTimeout(d)` | 10s | Time in OPEN before HALF-OPEN |
| `WithHalfOpenSuccess(n)` | 1 | Successful probes to close |
| `WithTripCodes(...)` | Internal, Unavailable, DeadlineExceeded | gRPC codes that count as failures |
| `WithTripFunc(fn)` | see above | Custom failure detection |
| `WithLogger(l)` | nop | Logger for state transitions |
| `WithGoLibLogger(l)` | - | Adapter for go-lib logger |

## Default trip codes

By default, circuit breaker trips on infrastructure errors:
- `codes.Internal`
- `codes.Unavailable`
- `codes.DeadlineExceeded`

Business errors (e.g., `codes.InvalidArgument`, `codes.NotFound`) are ignored.

## Custom trip function

```go
cb := circuitbreaker.New(
    circuitbreaker.WithTripFunc(func(c codes.Code) bool {
        return c == codes.Unavailable || c == codes.DeadlineExceeded
    }),
)
```

## With logging

```go
import "github.com/vortex-fintech/go-lib/foundation/logger"

cb := circuitbreaker.New(
    circuitbreaker.WithGoLibLogger(logger.Default()),
)
```

## Monitoring state

```go
state := cb.State() // "closed", "open", "half-open"

// Expose as Prometheus metric
prometheus.NewGaugeFunc(prometheus.GaugeOpts{
    Name: "circuit_breaker_state",
    Help: "Circuit breaker state: 0=closed, 1=open, 2=half-open",
}, func() float64 {
    switch cb.State() {
    case "closed": return 0
    case "open": return 1
    case "half-open": return 2
    default: return -1
    }
})
```

## Manual reset

For admin endpoints or health checks:

```go
cb.Reset() // Forces state to CLOSED
```

## HALF-OPEN behavior

In HALF-OPEN state:
- Only **one** request is allowed through (probe)
- Concurrent requests are rejected with `Unavailable`
- Successful probe → allows next probe (until HalfOpenSuccess reached)
- Failed probe → immediately back to OPEN

## Example with chain

```go
func NewGRPCServer() *grpc.Server {
    cb := circuitbreaker.New(
        circuitbreaker.WithFailureThreshold(10),
        circuitbreaker.WithRecoveryTimeout(30*time.Second),
        circuitbreaker.WithHalfOpenSuccess(3),
        circuitbreaker.WithGoLibLogger(logger.Default()),
    )

    return grpc.NewServer(
        chain.Default(chain.Options{
            Pre: []grpc.UnaryServerInterceptor{
                recoverymw.Unary(),
            },
            AuthzInterceptor: authz.UnaryServerInterceptor(authzConfig),
            CircuitBreaker:   cb,
        }),
    )
}
```

## Production notes

- Set `FailureThreshold` based on your traffic (5-10 for low, 20-50 for high)
- `RecoveryTimeout` should match downstream service recovery time
- Use `HalfOpenSuccess > 1` for gradual recovery
- Monitor state transitions in logs/metrics
- Don't use for business logic errors - only infrastructure failures
- Circuit breaker protects **downstream** services, not the current one

## Thread safety

All operations are thread-safe. Multiple goroutines can call `Unary()` interceptor simultaneously.
