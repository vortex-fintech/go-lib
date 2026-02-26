# gRPC Recovery Middleware

Recovers from panics in gRPC handlers, logs them, and returns Internal error.

## Where to use it

- Production gRPC servers (always)
- Prevents server crashes from unhandled panics
- Provides observability for unexpected failures

## Basic usage

```go
import "github.com/vortex-fintech/go-lib/transport/grpc/middleware/recoverymw"

server := grpc.NewServer(
    grpc.UnaryInterceptor(recoverymw.Unary(recoverymw.Options{})),
    grpc.StreamInterceptor(recoverymw.Stream(recoverymw.Options{})),
)
```

## With logging

```go
recoverymw.Unary(recoverymw.Options{
    OnPanic: func(ctx context.Context, method string, recovered any) {
        slog.ErrorCtx(ctx, "panic recovered",
            "method", method,
            "panic", recoverymw.PanicString(recovered),
            "stack", debug.Stack(),
        )
    },
})
```

## With chain

```go
return grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            recoverymw.Unary(recoverymw.Options{
                OnPanic: panicLogger,
            }),
        },
    }),
)
```

## PanicString helper

Converts any panic value to string:

```go
recoverymw.PanicString("error")         // "error"
recoverymw.PanicString(errors.New("x")) // "x"
recoverymw.PanicString(42)             // "42"
```

## Behavior

- Panic → OnPanic called (if provided) → returns `Internal` error
- No panic → normal flow
- OnPanic is optional (nil safe)

## Production notes

- Place FIRST in middleware chain (before all other middleware)
- Log stack traces for debugging
- Monitor panic frequency via logs/metrics
- Thread-safe for concurrent requests
