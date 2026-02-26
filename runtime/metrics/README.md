# Metrics Handler

HTTP handler for Prometheus metrics and health/ready probes.

## Where to use it

- Expose `/metrics` for Prometheus scraping.
- Provide `/health` (liveness) and `/ready` (readiness) endpoints for Kubernetes probes.
- Register business metrics alongside standard process and Go runtime metrics.

## Endpoints

| Path | Purpose | Auth |
|------|---------|------|
| `/metrics` | Prometheus exposition format | Optional |
| `/health` | Liveness probe (is process alive?) | No |
| `/ready` | Readiness probe (can handle traffic?) | No |

## Basic usage

```go
handler, _ := metrics.New(metrics.Options{
    Register: func(reg prometheus.Registerer) error {
        reg.MustRegister(myBusinessCounter)
        return nil
    },
})

http.ListenAndServe(":8080", handler)
```

## Full example with all features

```go
handler, _ := metrics.New(metrics.Options{
    HealthPath: "/healthz",
    ReadyPath:  "/readyz",
    HealthTimeout: 200 * time.Millisecond,
    ReadyTimeout:  500 * time.Millisecond,

    Health: func(ctx context.Context, r *http.Request) error {
        return nil
    },

    Ready: func(ctx context.Context, r *http.Request) error {
        return db.PingContext(ctx)
    },

    Register: func(reg prometheus.Registerer) error {
        reg.MustRegister(ordersTotal)
        reg.MustRegister(requestDuration)
        return nil
    },

    MetricsAuth: func(r *http.Request) bool {
        return r.Header.Get("Authorization") == "Bearer "+metricsToken
    },

    Log: func(level metrics.LogLevel, path, method string, status int, d time.Duration) {
        slog.Info("probe", "level", level, "path", path, "status", status, "duration", d)
    },
})
```

## Kubernetes probes

```yaml
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

## Options reference

| Option | Default | Description |
|--------|---------|-------------|
| `Registry` | New registry | Shared Prometheus registry |
| `Register` | None | Callback to register business metrics |
| `Health` | None | Liveness check function (must respect `ctx.Done()`) |
| `Ready` | None | Readiness check function (must respect `ctx.Done()`) |
| `HealthPath` | `/health` | Path for liveness endpoint |
| `ReadyPath` | `/ready` | Path for readiness endpoint |
| `MetricsPath` | `/metrics` | Path for metrics endpoint |
| `HealthTimeout` | 500ms | Timeout for health check |
| `ReadyTimeout` | 500ms | Timeout for ready check |
| `MetricsAuth` | None | Auth function for /metrics |
| `Log` | None | Logging callback |
| `StrictRegister` | false | Return `(nil, nil)` if registration fails (silent if `Log=nil`) |
| `DisableBuildInfo` | false | Disable `go_build_info` metric |

## Strict mode

```go
handler, reg := metrics.New(metrics.Options{
    StrictRegister: true,
    Register: func(reg prometheus.Registerer) error {
        return reg.Register(myMetric)
    },
})
if handler == nil {
    log.Fatal("metrics registration failed")
}
```

## Concurrency and safety

- Health/ready checks are concurrency-limited to 64 simultaneous checks.
- Timeouts prevent slow dependencies from blocking probes.
- Health/ready callbacks must respect context cancellation to avoid exhausting the limiter.
- Standard metrics (`process_*`, `go_*`, `go_build_info`) registered automatically with `AlreadyRegistered` safety.
- `HEAD` requests return no body (only status code).
- `Cache-Control: no-store` on all endpoints.

## Production notes

- Keep health checks lightweight (no DB writes, no external API calls).
- Use ready checks for dependency validation (DB, cache, upstream services).
- Protect `/metrics` with auth if exposed publicly.
- Set appropriate timeouts: health should be fast (<200ms), ready can be slower (<1s).
