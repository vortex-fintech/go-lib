# Runtime

Production-ready utilities for Go services: metrics, health probes, and graceful shutdown.

## Packages

| Package | Description |
|---------|-------------|
| `metrics` | HTTP handler for Prometheus metrics and health/ready probes |
| `shutdown` | Graceful shutdown manager for multiple servers |
| `shutdown/adapters` | HTTP and gRPC adapters for shutdown manager |
| `shutdown/prommetrics` | Prometheus metrics for shutdown statistics |

## Quick start

```go
package main

import (
    "context"
    "log"
    "net"
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/vortex-fintech/go-lib/runtime/metrics"
    "github.com/vortex-fintech/go-lib/runtime/shutdown"
    "github.com/vortex-fintech/go-lib/runtime/shutdown/adapters"
    "github.com/vortex-fintech/go-lib/runtime/shutdown/prommetrics"
    "google.golang.org/grpc"
)

func main() {
    reg := prometheus.NewRegistry()

    shutdownMetrics, err := prommetrics.New(reg, "myapp", "shutdown")
    if err != nil {
        log.Fatal(err)
    }

    handler, _ := metrics.New(metrics.Options{
        Registry: reg,
        Ready: func(ctx context.Context, r *http.Request) error {
            return db.PingContext(ctx)
        },
    })

    httpLis, _ := net.Listen("tcp", ":8080")
    grpcLis, _ := net.Listen("tcp", ":9090")

    httpSrv := &http.Server{Handler: handler}
    grpcSrv := grpc.NewServer()

    mgr := shutdown.New(shutdown.Config{
        ShutdownTimeout: 30 * time.Second,
        HandleSignals:   true,
        Metrics:         shutdownMetrics,
    })

    mgr.Add(&adapters.HTTP{Srv: httpSrv, Lis: httpLis, NameStr: "http-api"})
    mgr.Add(&adapters.GRPC{Srv: grpcSrv, Lis: grpcLis, NameStr: "grpc-api"})

    if err := mgr.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Architecture

```
                    ┌─────────────────┐
                    │   Prometheus    │
                    │   /metrics      │
                    └────────┬────────┘
                             │
┌────────────────────────────┼────────────────────────────┐
│                            │                            │
│  ┌─────────────┐    ┌──────┴──────┐    ┌─────────────┐  │
│  │ HTTP Server │    │   Metrics   │    │ gRPC Server │  │
│  │  :8080      │    │   Handler   │    │   :9090     │  │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘  │
│         │                  │                  │         │
│         └──────────────────┼──────────────────┘         │
│                            │                            │
│                    ┌───────┴───────┐                    │
│                    │ Shutdown      │                    │
│                    │ Manager       │◄─── SIGTERM        │
│                    └───────────────┘                    │
│                            │                            │
│         Graceful shutdown with timeout                  │
│         Metrics: success/force, duration                │
└─────────────────────────────────────────────────────────┘
```

## Features

### metrics
- `/metrics` — Prometheus exposition format
- `/health` — Liveness probe (process alive?)
- `/ready` — Readiness probe (can handle traffic?)
- Optional auth for metrics endpoint
- Automatic Go and process metrics

### shutdown
- Graceful shutdown with configurable timeout
- Signal handling (SIGINT, SIGTERM)
- Per-server success/force metrics
- HTTP and gRPC adapters
- Idempotent `Stop()` calls

## Kubernetes deployment

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: app
          ports:
            - containerPort: 8080
              name: http
            - containerPort: 9090
              name: grpc
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
```

## Production checklist

- [ ] Set `ShutdownTimeout` based on slowest request
- [ ] Enable `HandleSignals: true` for containers
- [ ] Implement `/ready` check for dependencies
- [ ] Monitor `graceful_stop_total{result="force"}`
- [ ] Set `HealthTimeout` < 200ms
- [ ] Set `ReadyTimeout` < 1s
