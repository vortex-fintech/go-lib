# gRPC Metrics Middleware

Collects metrics for gRPC requests - latency, status codes, error counters.

## Where to use it

- Production gRPC services
- Observability and monitoring
- SLO/SLA tracking

## Basic usage

```go
import (
    "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metrics"
    promreporter "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metricsmw/promreporter"
)

reporter := promreporter.Reporter{
    M: yourMetricsInstance,
}

server := grpc.NewServer(
    grpc.UnaryInterceptor(metrics.UnaryFull(reporter)),
    grpc.StreamInterceptor(metrics.StreamFull(reporter)),
)
```

## FullReporter interface

```go
type FullReporter interface {
    ObserveRPCFull(ctx context.Context, fullMethod string, code codes.Code, secs float64)
}
```

## Prometheus integration

The package includes a Prometheus reporter:

```go
import (
    prometheus "github.com/prometheus/client_golang/prometheus"
)

type promMetrics struct {
    rpcDuration prometheus.Histogram
    rpcErrors  *prometheus.CounterVec
}

func (m *promMetrics) ObserveRPC(service, method, code string, seconds float64) {
    m.rpcDuration.WithLabelValues(service, method, code).Observe(seconds)
}

func (m *promMetrics) IncError(typ, service, method string) {
    m.rpcErrors.WithLabelValues(typ, service, method).Inc()
}
```

## Method splitting

The reporter splits gRPC method names:
- `/pkg.Service/Method` → service=`pkg.Service`, method=`Method`
- `pkg.Service/Method` → service=`pkg.Service`, method=`Method`
- `pkg.Service.Method` → service=`pkg.Service`, method=`Method`
- `Method` → service=`unknown`, method=`Method`

## With chain

```go
return grpc.NewServer(
    chain.Default(chain.Options{
        Post: []grpc.UnaryServerInterceptor{
            metrics.UnaryFull(promReporter),
        },
    }),
)
```

## What is tracked

| Metric | Labels | Description |
|--------|---------|-------------|
| RPC duration | service, method, code | Histogram of request latency |
| RPC errors | type, service, method | Counter of failed requests |

Error types:
- `grpc` - any non-OK gRPC status

## Production notes

- Place at the end of middleware chain (Post, not Pre)
- Use histogram with appropriate buckets for your latency distribution
- Separate services/methods in labels for high cardinality
- Track `OK` responses too for success rate
- Thread-safe for concurrent requests
