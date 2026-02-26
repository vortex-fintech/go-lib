# gRPC Interceptor Chain

Composes gRPC interceptors in the correct order with sensible defaults.

## Where to use it

- gRPC servers needing multiple interceptors
- Ensuring correct interceptor execution order
- Production-ready server setup with auth, errors, circuit breaker

## Execution order

```
Pre → ContextCancel → Authz → CircuitBreaker → Errors → Post
```

**Why this order:**
1. **Pre** (e.g., metrics) - first to measure full request time
2. **ContextCancel** - early rejection if context already cancelled
3. **Authz** - reject unauthorized requests before business logic
4. **CircuitBreaker** - protect downstream services
5. **Errors** - normalize all errors before returning
6. **Post** - custom post-processing

## Basic usage

```go
verifier, _ := jwt.NewJWKSVerifier(jwt.JWKSConfig{
    URL: "https://sso.internal/.well-known/jwks.json",
})

authzInterceptor := authz.UnaryServerInterceptor(authz.Config{
    Verifier:  verifier,
    Audience:  "wallet",
})

cb := circuitbreaker.New(
    circuitbreaker.WithFailureThreshold(5),
    circuitbreaker.WithRecoveryTimeout(10*time.Second),
)

server := grpc.NewServer(
    chain.Default(chain.Options{
        AuthzInterceptor: authzInterceptor,
        CircuitBreaker:   cb,
    }),
)
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `Pre` | empty | Interceptors to run before built-in ones |
| `Post` | empty | Interceptors to run after built-in ones |
| `AuthzInterceptor` | nil | Authorization interceptor (skipped if nil) |
| `CircuitBreaker` | nil | Circuit breaker instance (skipped if nil) |
| `DisableCtxCancel` | false | Disable context cancellation check |
| `DisableErrors` | false | Disable error normalization |

## With metrics

```go
server := grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            metricsmw.UnaryServerInterceptor(metricsmw.Config{...}),
        },
        AuthzInterceptor: authzInterceptor,
    }),
)
```

## With recovery

```go
server := grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            recoverymw.Unary(),
        },
        AuthzInterceptor: authzInterceptor,
    }),
)
```

## Minimal setup

```go
server := grpc.NewServer(
    chain.Default(chain.Options{
        DisableCtxCancel: true, // if you don't need early cancellation check
        DisableErrors:    true, // if you handle errors manually
    }),
)
```

## Stream support

```go
server := grpc.NewServer(
    chain.Default(chain.Options{
        AuthzInterceptor: authzUnaryInterceptor,
    }),
    chain.DefaultStream(chain.StreamOptions{
        AuthzInterceptor: authz.StreamServerInterceptor(authzConfig),
    }),
)
```

### StreamOptions

| Option | Default | Description |
|--------|---------|-------------|
| `Pre` | empty | Stream interceptors to run first |
| `Post` | empty | Stream interceptors to run last |
| `AuthzInterceptor` | nil | Stream authorization interceptor |
| `DisableCtxCancel` | false | Disable context cancellation check |
| `DisableErrors` | false | Disable error normalization |

Note: CircuitBreaker is not included in stream chain (streams are long-lived).

## Full example

```go
func NewGRPCServer(verifier jwt.Verifier) *grpc.Server {
    authzInterceptor := authz.UnaryServerInterceptor(authz.Config{
        Verifier:   verifier,
        Audience:   "wallet",
        RequirePoP: true,
    })

    streamAuthzInterceptor := authz.StreamServerInterceptor(authz.Config{
        Verifier:   verifier,
        Audience:   "wallet",
        RequirePoP: true,
    })

    cb := circuitbreaker.New(
        circuitbreaker.WithFailureThreshold(5),
        circuitbreaker.WithRecoveryTimeout(15*time.Second),
    )

    return grpc.NewServer(
        chain.Default(chain.Options{
            Pre: []grpc.UnaryServerInterceptor{
                recoverymw.Unary(),
                metricsmw.Unary(metricsmw.Config{Reporter: promReporter}),
            },
            AuthzInterceptor: authzInterceptor,
            CircuitBreaker:   cb,
        }),
        chain.DefaultStream(chain.StreamOptions{
            Pre: []grpc.StreamServerInterceptor{
                recoverymw.Stream(),
            },
            AuthzInterceptor: streamAuthzInterceptor,
        }),
    )
}
```

## Production notes

- Always include `recoverymw` in `Pre` to prevent panics from crashing the server
- Put metrics in `Pre` to measure full request lifecycle
- `CircuitBreaker` protects downstream services, not the current one
- `Errors` normalizes domain errors to gRPC status codes
- Use `DefaultStream` for bi-directional streaming endpoints
