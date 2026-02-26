# AGENTS.md

Operational guide for agentic coding assistants working in this repository.

## Scope
- Module: `github.com/vortex-fintech/go-lib/runtime`
- Language: Go
- Go version: `go 1.25`
- Toolchain in `go.mod`: `go1.25.7`
- Main packages: `metrics`, `shutdown`, `shutdown/adapters`, `shutdown/prommetrics`
- This repository is a shared library module (not an app entrypoint)

## Source-of-truth docs
- `README.md` (root)
- `metrics/README.md`
- `shutdown/README.md`

## Cursor / Copilot rules
- `.cursorrules`: not present
- `.cursor/rules/`: not present
- `.github/copilot-instructions.md`: not present
- If any are added later, treat them as higher-priority constraints than this file.

## Build / format / lint / test

### Build
- Build all packages: `go build ./...`
- Build one package: `go build ./shutdown`, `go build ./metrics`

### Format
- Format all code: `go fmt ./...`
- Optional formatting check: `gofmt -l .`

### Lint / static analysis
- Run vet: `go vet ./...`
- No `golangci-lint` config exists in this repo currently.

### Default tests (unit-style)
- Run all default tests: `go test ./...`
- Run a single package: `go test ./shutdown`, `go test ./metrics`
- Run with race detector: `go test -race ./...`
- If local `-race` fails due to missing CGO/C compiler (common on Windows), run in Docker:
  `docker run --rm -v "C:/Vortex Services/go-lib/runtime:/work" golang:1.25.7 sh -c 'cd /work && go mod download && go test -race ./...'`

### Running a single test (important)
- Single shutdown test: `go test ./shutdown -run '^Test_Run_NoServers_OK$'`
- Single adapter test: `go test ./shutdown/adapters -run '^TestHTTPAdapter_ServeAndGracefulShutdown_WithListener$'`
- Single prommetrics test: `go test ./shutdown/prommetrics -run '^TestPromMetrics_CountersAndHistogram$'`
- Single metrics test: `go test ./metrics -run '^TestMetricsHandler_Defaults$'`
- Prefix match: `go test ./shutdown -run '^Test_Run_'`
- Subtest match: `go test ./shutdown -run '^Test_Stop_PerServerTimeouts_'`

### Tagged tests (unit and integration)
- Unit tests with build tag: `go test -tags=unit ./...`
- Integration tests: `go test -tags=integration ./...`
- Unix-only signal tests: `go test -tags=unix ./...`
- Single integration test:
  `go test -tags=integration ./shutdown -run '^Test_Manager_With_GRPCAdapter_GracefulCancel_OK$'`

### Test file patterns
- `*_test.go`: Default unit tests (no build tag required)
- `//go:build unit`: Explicit unit test tag
- `//go:build integration`: Integration tests requiring external resources
- `//go:build unix`: Unix-only tests (signal handling)

## Code style guidelines

### General style
- Follow idiomatic Go and keep code `gofmt`-clean.
- Prefer small focused functions and early returns.
- Keep behavior deterministic; avoid hidden side effects.
- Preserve backward-compatible public APIs unless explicitly changing contracts.
- **DO NOT ADD COMMENTS** unless explicitly asked.

### Imports
- Use standard `gofmt` grouping: stdlib, blank line, external modules.
- Alias imports for clarity or collisions (e.g., `promhttp` for prometheus handler).
- Avoid dot imports.

### Formatting and layout
- Do not manually align spacing; let `gofmt` decide.
- Keep struct field ordering logical (config fields first, internal state last).
- Keep JSON struct tags on one line with spaces: `json:"field,omitempty"`.

### Types and interfaces
- Prefer concrete structs for domain/config payloads (`Config`, `Options`, `Manager`).
- Keep interfaces narrow and capability-focused (`Server`, `Metrics`, `LogFunc`).
- Use typed string constants for finite states (`LogLevel`: `LogDebug`, `LogInfo`, etc.).
- Keep zero-value semantics intentional and validated.

### Naming conventions
- Exported identifiers: `PascalCase`.
- Unexported identifiers: `camelCase`.
- Exported sentinel errors: none in this module (errors returned directly).
- Test names: `Test<Subject>_<Scenario>` (e.g., `Test_Run_NoServers_OK`).
- Package-level constants: `PascalCase` for exported, `camelCase` for internal.
- Interface method names: `Serve`, `GracefulStopWithTimeout`, `ForceStop`, `Name`.

### Error handling
- Validate inputs up front and fail fast.
- Return concrete errors for expected invalid-input paths (e.g., `"http adapter: Srv is nil"`).
- Wrap propagated errors with `%w` when adding context: `fmt.Errorf("register collector: %w", err)`.
- Use `errors.Is` and `errors.As` instead of string matching.
- Check for `prometheus.AlreadyRegisteredError` when registering metrics.

### Context, time, and cancellation
- Accept `context.Context` as the first argument of operations.
- Use `context.WithTimeout` around external I/O and cleanup paths.
- Always `defer cancel()` for derived contexts.
- Server `Serve` methods should respect context cancellation and return `ctx.Err()`.
- Use `context.WithDeadline` to propagate remaining time to per-server graceful stops.

### Shutdown manager patterns
- Use `shutdown.New(Config{})` with `ShutdownTimeout` for initialization.
- Register servers with `m.Add(server)`; nil servers are safely ignored.
- Use `m.Run(ctx)` to start and block until shutdown.
- `HandleSignals: true` enables automatic SIGINT/SIGTERM handling.
- `Stop()` is idempotent; safe to call multiple times.
- Use `DefaultIsNormalErr` to identify expected shutdown errors.

### Adapter patterns (HTTP/gRPC)
- HTTP adapter: `&adapters.HTTP{Srv: httpServer, Lis: listener, NameStr: "name"}`
- gRPC adapter: `&adapters.GRPC{Srv: grpcServer, Lis: listener, NameStr: "name"}`
- If `Lis` is nil, HTTP adapter uses `ListenAndServe()`.
- gRPC adapter requires both `Srv` and `Lis` to be non-nil.
- `ForceStop` should be no-op if server is nil.

### Metrics handler patterns
- Use `metrics.New(Options{})` to create handler with registry.
- Provide `Health` and `Ready` functions for probes.
- Set `HealthTimeout` and `ReadyTimeout` (defaults: 500ms).
- Use `MetricsAuth` for basic auth on `/metrics` endpoint.
- Use `Log` callback for request logging with level/status/duration.
- Set `StrictRegister: true` to silently fail on registration errors.

### Prometheus metrics patterns
- Use `prommetrics.New(registry, namespace, subsystem)` for shutdown metrics.
- Check for `AlreadyRegisteredError` to allow multiple registrations.
- Use `prometheus.CounterVec` for labeled counters.
- Use `prometheus.Histogram` for duration observations.

### Testing conventions
- Prefer table-driven tests for validation matrices.
- Use `t.Parallel()` for independent unit tests.
- Use same-package tests for unexported behavior (e.g., `fakeServer`, `fakeMetrics`).
- Use external package tests (`package prommetrics_test`) for public API validation.
- Use `t.Fatalf()` in same-package tests.
- Use `testutil.ToFloat64()` for asserting counter values.
- Use `httptest.NewRequest` and `httptest.NewRecorder` for HTTP handler tests.
- Use real network listeners (`net.Listen("tcp", "127.0.0.1:0")`) for server tests.
- Always `defer listener.Close()` and `defer cancel()` in tests.

### Concurrency patterns
- Use `errgroup.WithContext` for coordinating multiple goroutines.
- Use `sync.Once` for one-time cleanup (e.g., closing channels).
- Use `atomic.Bool` for thread-safe flags (e.g., `forced`).
- Use buffered channels for error reporting (`chan error, 1`).
- Always `defer close()` for channels owned by the function.

## Agent workflow expectations
- Before completing substantial changes, run `go test ./...`.
- If integration behavior changes, run `go test -tags=integration ./...`.
- If formatting-sensitive files change, run `go fmt ./...`.
- If exported behavior changes, update the relevant package README.
- Avoid adding new dependencies unless necessary.
- Run `go vet ./...` after making changes.
