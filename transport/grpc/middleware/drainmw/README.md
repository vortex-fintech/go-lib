# gRPC Drain Middleware

Graceful server draining - reject new mutating requests while allowing in-flight and read-only operations.

## Where to use it

- Graceful shutdown in Kubernetes/Docker
- Blue-green deployments
- Traffic migration between instances

## How it works

1. Call `StartDraining()` before shutdown
2. New mutating requests get `Unavailable` error
3. Read-only requests continue
4. In-flight requests complete
5. Safe to shutdown when requests drop to zero

## Basic usage

```go
import "github.com/vortex-fintech/go-lib/transport/grpc/middleware/drainmw"

drainCtrl := drainmw.NewController()

server := grpc.NewServer(
    grpc.UnaryInterceptor(drainmw.Unary(drainCtrl, isMutating)),
    grpc.StreamInterceptor(drainmw.Stream(drainCtrl, isMutating)),
)

// On SIGTERM:
drainCtrl.StartDraining()
time.Sleep(30 * time.Second) // wait for in-flight
server.GracefulStop()
```

## isMutating function

Define which methods mutate state:

```go
isMutating := func(method string) bool {
    reads := []string{"/svc/Get", "/svc/List", "/svc/Health"}
    for _, r := range reads {
        if method == r {
            return false
        }
    }
    return true
}
```

Or use naming convention:

```go
isMutating := func(method string) bool {
    return !strings.Contains(method, "/Get") &&
           !strings.Contains(method, "/List")
}
```

## With chain

```go
drainCtrl := drainmw.NewController()

return grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            drainmw.Unary(drainCtrl, isMutating),
        },
    }),
)
```

## Kubernetes example

```go
func main() {
    drainCtrl := drainmw.NewController()

    server := startGRPCServer(drainCtrl)

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM)

    <-sigCh
    log.Println("Received SIGTERM, starting drain")
    drainCtrl.StartDraining()

    // Wait for connections to drop or timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    server.GracefulStop()
}
```

## Nil safety

- `Unary(nil, nil)` - allows all requests (no draining)
- `StartDraining()` on nil controller is safe

## Production notes

- Call `StartDraining()` before `GracefulStop()`
- Use health check to signal readiness
- Set drain timeout (30s typical)
- Client should retry on `Unavailable` during drain
- Thread-safe for concurrent requests
