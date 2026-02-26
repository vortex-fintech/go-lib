# gRPC Context Cancel

Early detection of cancelled contexts to avoid unnecessary work.

## Where to use it

- High-traffic services where clients frequently disconnect
- Expensive operations that should be aborted early
- Preventing wasted CPU on cancelled requests

## How it works

Checks context state at two points:
1. **Before handler** - returns `Canceled`/`DeadlineExceeded` immediately if context is already done
2. **After handler** - if handler succeeded but context was cancelled during execution, returns error instead of response

## Basic usage

```go
import "github.com/vortex-fintech/go-lib/transport/grpc/middleware/contextcancel"

server := grpc.NewServer(
    grpc.UnaryInterceptor(contextcancel.Unary()),
    grpc.StreamInterceptor(contextcancel.Stream()),
)
```

## With chain

```go
return grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            recoverymw.Unary(),
            contextcancel.Unary(),
        },
    }),
)
```

## Behavior

| Scenario | Unary Result | Stream Result |
|----------|--------------|---------------|
| Context cancelled before handler | `Canceled` | `Canceled` |
| Context deadline exceeded before handler | `DeadlineExceeded` | `DeadlineExceeded` |
| Handler succeeds, context cancelled during | `Canceled` (no response) | `Canceled` |
| Handler returns error | Handler error | Handler error |
| Normal flow | Response | nil |

## Why check after handler?

If client disconnects during handler execution:
- Without check: server sends response to dead connection (wasted bandwidth)
- With check: server detects cancellation and returns error (clean shutdown)

## Production notes

- Place early in middleware chain (after recovery)
- Low overhead - only checks `ctx.Err()`
- Works with both `context.Cancel` and deadline timeouts
- Thread-safe for concurrent requests
