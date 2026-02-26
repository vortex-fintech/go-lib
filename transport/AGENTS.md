# AGENTS.md

Operational guide for agentic coding assistants working in this repository.

## Scope
- Module: `github.com/vortex-fintech/go-lib/transport`
- Language: Go
- Go version: `go 1.25`
- Toolchain in `go.mod`: `go1.25.7`
- Main packages: `grpc/middleware/*`, `grpc/dial`, `grpc/creds`, `grpc/metadata`
- This repository is a shared library module for gRPC transport utilities

## Source-of-truth docs
- `README.md` (root)
- `grpc/README.md`
- Package READMEs in each middleware subdirectory

## Cursor / Copilot rules
- `.cursorrules`: not present
- `.cursor/rules/`: not present
- `.github/copilot-instructions.md`: not present
- If any are added later, treat them as higher-priority constraints than this file.

## Build / format / lint / test

### Build
- Build all packages: `go build ./...`
- Build one package: `go build ./grpc/middleware/errorsmw`

### Format
- Format all code: `go fmt ./...`
- Optional formatting check: `gofmt -l .`

### Lint / static analysis
- Run vet: `go vet ./...`
- No `golangci-lint` config exists in this repo currently.

### Default tests (unit-style)
- Run all tests: `go test ./...`
- Run all tests with verbose: `go test -v ./...`
- Run single package: `go test ./grpc/middleware/errorsmw`
- Run with race detector: `go test -race ./...`
- If local `-race` fails due to missing CGO/C compiler (common on Windows), run in Docker:
  `docker run --rm -v "C:/Vortex Services/go-lib/transport:/work" golang:1.25.7 sh -c 'cd /work && go mod download && go test -race ./...'`

### Running a single test (important)
- Single interceptor test: `go test ./grpc/middleware/errorsmw -run '^TestUnary_PassesStatusAsIs$'`
- Single circuit breaker test: `go test ./grpc/middleware/circuitbreaker -run '^Test_CLOSED_to_OPEN_after_threshold$'`
- Single dial test: `go test ./grpc/dial -run '^TestDial_InvalidTarget$'`
- Prefix match: `go test ./grpc/middleware/circuitbreaker -run '^Test_HALF_OPEN'`

### Root Makefile commands
From parent directory `C:\Vortex Services\go-lib`:
- Run unit tests: `make test`
- Run with race detector: `make test-race`
- Run integration tests: `make test-integration`
- Run all: `make test-all`
- Generate coverage: `make cover`

## Code style guidelines

### General style
- Follow idiomatic Go and keep code `gofmt`-clean.
- Prefer small focused functions and early returns.
- Keep behavior deterministic; avoid hidden side effects.
- Preserve backward-compatible public APIs unless explicitly changing contracts.
- **DO NOT ADD COMMENTS** unless explicitly asked.

### Imports
- Use standard `gofmt` grouping: stdlib, blank line, external modules.
- Alias internal packages for clarity: `gliberrors "github.com/vortex-fintech/go-lib/foundation/errors"`
- Alias local middleware packages when needed: `cb "github.com/vortex-fintech/go-lib/transport/grpc/middleware/circuitbreaker"`
- Avoid dot imports.
- Group grpc-related imports together.

### Formatting and layout
- Do not manually align spacing; let `gofmt` decide.
- Keep struct field tags on one line with spaces.

### Types and interfaces
- Prefer `Options` struct with functional options pattern for configuration.
- Use `Option func(*Options)` for functional options.
- Keep interfaces narrow: `Logger` interface with `Info`, `Warn`, `Error` methods.
- Use unexported interfaces for internal abstractions (e.g., `grpcConvertible`).
- Exported options structs should have sensible defaults in constructors.

### Naming conventions
- Exported identifiers: `PascalCase`.
- Unexported identifiers: `camelCase`.
- Exported sentinel errors: `ErrXxx` (e.g., `ErrNilTLSConfig`, `ErrMissingCert`).
- Options structs: `Options`, `ServerOptions`, `ClientOptions`.
- Config structs: `Config` for complex configuration.
- Interceptor constructors: `Unary`, `Stream` returning `grpc.UnaryServerInterceptor` or `grpc.StreamServerInterceptor`.
- Test names: `Test<Subject>_<Scenario>` (e.g., `Test_CLOSED_to_OPEN_after_threshold`).

### Functional options pattern
- Use functional options for configurable behavior:
  ```go
  type Option func(*Options)
  
  func WithFallback(f func(error) error) Option {
      return func(o *Options) { o.Fallback = f }
  }
  ```
- Apply defaults in constructor before applying options:
  ```go
  o := Options{Fallback: defaultFallback}
  for _, f := range opts { f(&o) }
  ```

### Error handling
- Return sentinel errors for expected invalid-input paths (e.g., `ErrNilTLSConfig`).
- Use `google.golang.org/grpc/status` for gRPC errors: `status.Error(codes.NotFound, "message")`.
- Check gRPC status with `status.FromError(err)`.
- Map domain errors to gRPC codes in errorsmw interceptor.
- Use `errors.Is` and `errors.As` for error checking.

### gRPC interceptor patterns
- Unary interceptors: `func(ctx, req, info, handler) (resp, err)`
- Stream interceptors: `func(srv, stream, info, handler) error`
- Always return `grpc.UnaryServerInterceptor` or `grpc.StreamServerInterceptor` types.
- Provide both `Unary` and `Stream` variants when applicable.
- Use `grpc.ChainUnaryInterceptor` for composing multiple interceptors.

### Context and metadata
- Accept `context.Context` as the first argument of operations.
- Use `google.golang.org/grpc/metadata` for gRPC metadata operations.
- Use `grpc/metadata` package utilities: `WithBearer`, `WithPoP`, `WithAZP`.

### TLS and credentials
- Use `grpc/creds` for transport credentials.
- Validate TLS config early: check for nil, missing certificates, missing CAs.
- Use `SkipRootCAValidation` option for development/test scenarios.

### Circuit breaker patterns
- Use state machine: CLOSED → OPEN → HALF-OPEN → CLOSED.
- Use `TripFunc func(codes.Code) bool` for customizable failure detection.
- Provide `Reset()` for manual recovery.
- Provide `State()` for monitoring/metrics.

### Testing conventions
- Prefer table-driven tests for validation matrices.
- Use helper functions to reduce boilerplate:
  ```go
  func call(t *testing.T, itc grpc.UnaryServerInterceptor, h grpc.UnaryHandler) (any, error) {
      t.Helper()
      return itc(nil, nil, &grpc.UnaryServerInfo{FullMethod: "/svc/Method"}, h)
      }
  ```
- Use `t.Helper()` for test helper functions.
- Use `t.Fatalf()` for failures in same-package tests.
- Use fake clock injection for time-dependent tests (circuit breaker).
- Use `sync` and `atomic` for concurrent test scenarios.

## Middleware integration patterns

### Chain middleware usage
Use `chain.Default()` to compose multiple interceptors:
```go
import "github.com/vortex-fintech/go-lib/transport/grpc/middleware/chain"

server := grpc.NewServer(
    chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{
            recoverymw.Unary(recoverymw.Options{}),
            deadlinemw.Unary(deadlinemw.Config{DefaultTimeout: 30*time.Second}),
        },
        CircuitBreaker: circuitbreaker.New(),
        AuthzInterceptor: authz.UnaryServerInterceptor(config),
        Post: []grpc.UnaryServerInterceptor{
            metricsmw.UnaryFull(reporter),
            errorsmw.Unary(),
        },
    }),
)
```

### Middleware integration with other packages

#### errorsmw ↔ foundation/errors
```go
import gliberrors "github.com/vortex-fintech/go-lib/foundation/errors"

err := gliberrors.NotFound()
// errorsmw converts to gRPC codes.NotFound automatically
```

#### idempotencymw ↔ data/idempotency
```go
import idempotency "github.com/vortex-fintech/go-lib/data/idempotency"

meta, ok := idempotencymw.FromContext(ctx)
begin, err := idempotency.Begin(ctx, store, db, idempotency.BeginInput{
    Principal:      meta.Principal,
    GRPCMethod:     meta.GRPCMethod,
    IdempotencyKey: meta.IdempotencyKey,
    RequestHash:    meta.RequestHash,
    ExpiresAt:      time.Now().UTC().Add(24*time.Hour),
})
```

#### metricsmw ↔ runtime/metrics
```go
import "github.com/vortex-fintech/go-lib/runtime/metrics"

handler, reg := metrics.New(metrics.Options{
    Register: func(r prometheus.Registerer) error {
        r.MustRegister(myRPCMetrics)
        return nil
    },
})

reporter := promreporter.Reporter{M: myRPCMetrics}
grpc.NewServer(
    grpc.UnaryInterceptor(metricsmw.UnaryFull(reporter)),
)
```

## Agent workflow expectations
- Before completing substantial changes, run `go test ./...`.
- If formatting-sensitive files change, run `go fmt ./...`.
- If exported behavior changes, update the relevant package README.
- Avoid adding new dependencies unless necessary.
- Run `go vet ./...` after making changes.
- For middleware changes, ensure both `Unary` and `Stream` variants are consistent.
