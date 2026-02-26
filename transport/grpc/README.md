# gRPC Transport

Production-ready gRPC utilities and middleware.

## Packages

| Package | Purpose |
|----------|---------|
| [middleware](./middleware/) | Interceptors (recovery, authz, circuit breaker, etc.) |
| [dial](./dial/) | Connection dialing with retries and TLS |
| [creds](./creds/) | TLS credential loading and validation |
| [metadata](./metadata/) | gRPC metadata utilities (user, trace ID, etc.) |

## Quick start

```go
import (
    "github.com/vortex-fintech/go-lib/transport/grpc/dial"
    "github.com/vortex-fintech/go-lib/transport/grpc/middleware/chain"
    "github.com/vortex-fintech/go-lib/transport/grpc/middleware/recoverymw"
    "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metricsmw"
    promreporter "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metricsmw/promreporter"
)

conn, err := dial.Default(dial.Options{
    Address: "localhost:8080",
    TLS: &dial.TLSOptions{
        Insecure: false,
        CACert:   caCert,
    },
})
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

client := pb.NewUserServiceClient(conn)
```

## Middleware chain

```go
server := grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            recoverymw.Unary(recoverymw.Options{}),
        },
        Post: []grpc.UnaryServerInterceptor{
            metricsmw.UnaryFull(promReporter),
        },
    }),
)
```

## See also

- [Middleware guide](./middleware/README.md) - full middleware list and usage
- [Dial options](./dial/README.md) - connection configuration
- [TLS credentials](./creds/README.md) - certificate handling
- [Metadata helpers](./metadata/README.md) - extract user/tenant from context
