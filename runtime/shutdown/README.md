# Shutdown Manager

Graceful shutdown manager for multiple servers (HTTP, gRPC) with timeouts, signal handling, and Prometheus metrics.

## Where to use it

- Coordinate graceful shutdown of multiple servers.
- Handle SIGINT/SIGTERM signals in containerized environments (Kubernetes, Docker).
- Collect metrics on shutdown behavior (success vs force stops).
- Ensure in-flight requests complete before termination.

## Components

| Package | Purpose |
|---------|---------|
| `shutdown` | Core manager that coordinates server lifecycle |
| `shutdown/adapters` | HTTP and gRPC adapters implementing `shutdown.Server` |
| `shutdown/prommetrics` | Prometheus metrics for shutdown statistics |

## Basic usage

```go
mgr := shutdown.New(shutdown.Config{
    ShutdownTimeout: 30 * time.Second,
    HandleSignals:   true,
})

mgr.Add(&adapters.HTTP{Srv: httpSrv, Lis: httpLis, NameStr: "http-api"})
mgr.Add(&adapters.GRPC{Srv: grpcSrv, Lis: grpcLis, NameStr: "grpc-api"})

if err := mgr.Run(ctx); err != nil {
    log.Fatal(err)
}
```

## Full example with metrics

```go
reg := prometheus.NewRegistry()

shutdownMetrics, err := prommetrics.New(reg, "myapp", "shutdown")
if err != nil {
    log.Fatal(err)
}

handler, _ := metrics.New(metrics.Options{
    Registry: reg,
})

httpLis, _ := net.Listen("tcp", ":8080")
grpcLis, _ := net.Listen("tcp", ":9090")

httpSrv := &http.Server{Handler: handler}
grpcSrv := grpc.NewServer()

mgr := shutdown.New(shutdown.Config{
    ShutdownTimeout: 30 * time.Second,
    HandleSignals:   true,
    Metrics:         shutdownMetrics,
    Logger: func(level, msg string, kv ...any) {
        slog.Info(msg, "level", level, "kv", kv)
    },
})

mgr.Add(&adapters.HTTP{Srv: httpSrv, Lis: httpLis, NameStr: "http-metrics"})
mgr.Add(&adapters.GRPC{Srv: grpcSrv, Lis: grpcLis, NameStr: "grpc-api"})

if err := mgr.Run(context.Background()); err != nil {
    log.Fatal(err)
}
```

## Config reference

| Option | Default | Description |
|--------|---------|-------------|
| `ShutdownTimeout` | 0 (immediate) | Maximum time for graceful shutdown |
| `HandleSignals` | `false` | Enable SIGINT/SIGTERM handling |
| `IsNormalError` | `DefaultIsNormalErr` | Function to classify expected errors |
| `Logger` | `log.Printf` | Logging callback |
| `Metrics` | `nil` | Metrics collector (implement `shutdown.Metrics`) |

## Adapters

### HTTP

```go
ad := &adapters.HTTP{
    Srv:     &http.Server{Handler: mux},  // required
    Lis:     listener,                      // optional, uses ListenAndServe if nil
    NameStr: "http-api",                    // optional, defaults to "http"
}
```

### gRPC

```go
ad := &adapters.GRPC{
    Srv:     grpcServer,    // required
    Lis:     listener,      // required
    NameStr: "grpc-api",    // optional, defaults to "grpc"
}
```

## Metrics

When using `prommetrics`, the following metrics are exposed:

| Metric | Labels | Description |
|--------|--------|-------------|
| `{ns}_{sub}_graceful_stop_total` | `result` | Total shutdowns by result (success/force) |
| `{ns}_{sub}_server_serve_errors_total` | `name` | Non-normal serve errors per server |
| `{ns}_{sub}_server_stop_result_total` | `name`, `result` | Per-server stop result |
| `{ns}_{sub}_graceful_duration_seconds` | - | Histogram of shutdown duration |

## Shutdown behavior

1. **Trigger**: Context cancellation, signal (SIGINT/SIGTERM), or server error
2. **Graceful phase**: Each server gets `ShutdownTimeout` to complete in-flight requests
3. **Force phase**: If timeout exceeded, `ForceStop()` is called
4. **Metrics**: Results recorded (success/force per server, total, duration)

## Concurrency and safety

- `Stop()` is idempotent and safe to call multiple times.
- `Add()` should be called before `Run()` (not thread-safe).
- All adapters handle nil checks gracefully.
- Signal handling uses `signal.NotifyContext` for proper cleanup.

## Production notes

- Set `ShutdownTimeout` based on your slowest request (e.g., 30s for long-polling).
- Enable `HandleSignals` in containers; Kubernetes sends SIGTERM before killing pods.
- Monitor `graceful_stop_total{result="force"}` — high values indicate timeout issues.
- Use `server_stop_result_total` to identify which server causes force stops.
- Combine with `runtime/metrics` for a complete observability stack.
