# gRPC Errors Middleware

Converts domain errors to gRPC status errors.

## Where to use it

- Any gRPC server that uses go-lib error types
- Ensuring consistent error responses across services
- Hiding implementation details from clients

## How it works

Conversion priority:
1. Already gRPC status → pass through
2. `ToGRPC()` interface → call it
3. InvariantError → use `ToErrorResponse().ToGRPC()`
4. `context.Canceled` → `Canceled`
5. `context.DeadlineExceeded` → `DeadlineExceeded`
6. Unknown → fallback (default: `Internal`)

## Basic usage

```go
import "github.com/vortex-fintech/go-lib/transport/grpc/middleware/errorsmw"

server := grpc.NewServer(
    grpc.UnaryInterceptor(errorsmw.Unary()),
    grpc.StreamInterceptor(errorsmw.Stream()),
)
```

## With chain

```go
return grpc.NewServer(
    chain.Default(chain.Options{
        Post: []grpc.UnaryServerInterceptor{
            errorsmw.Unary(),
        },
    }),
)
```

## Custom fallback

For unknown errors, customize the response:

```go
errorsmw.Unary(
    errorsmw.WithFallback(func(err error) error {
        log.Printf("unmapped error: %v", err)
        return status.Error(codes.Internal, "internal error")
    }),
)
```

## Error mapping

| Error type | gRPC code |
|------------|-----------|
| `*status.Status` | unchanged |
| `ErrorResponse` (go-lib) | via `ToGRPC()` |
| `InvariantError` (DomainInvariant) | `InvalidArgument` |
| `InvariantError` (StateInvariant) | `FailedPrecondition` |
| `InvariantError` (TransitionInvariant) | `FailedPrecondition` |
| `context.Canceled` | `Canceled` |
| `context.DeadlineExceeded` | `DeadlineExceeded` |
| Unknown | `Internal` (or custom fallback) |

## Supported error types

The middleware automatically handles:

- **go-lib errors**: `ValidationFields`, `NotFound`, `AlreadyExists`, etc.
- **Standard errors**: `context.Canceled`, `context.DeadlineExceeded`
- **Custom types**: implement `ToGRPC() error`

## Production notes

- Place at the end of middleware chain (Post, not Pre)
- Use `WithFallback` for logging unmapped errors
- All gRPC status errors pass through unchanged
- Thread-safe for concurrent requests
