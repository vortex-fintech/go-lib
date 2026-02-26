# AGENTS.md

Operational guide for agentic coding assistants working in this repository.

## Scope
- Module: `github.com/vortex-fintech/go-lib/foundation`
- Language: Go
- Go version: `go 1.25`
- Toolchain in `go.mod`: `go1.25.7`
- Main packages: `errors`, `logger`, `validator`, `domain`, `domainutil`, `retry`, `timeutil`, `piiutil`, `contactutil`, `geo`, `hash`, `logutil`, `netutil`, `textutil`
- This repository is a shared library module (not an app entrypoint)

## Source-of-truth docs
- `README.md` (root)
- Package READMEs: `errors/README.md`, `logger/README.md`, etc.

## Cursor / Copilot rules
- `.cursorrules`: not present
- `.cursor/rules/`: not present
- `.github/copilot-instructions.md`: not present
- If any are added later, treat them as higher-priority constraints than this file.

## Build / format / lint / test

### Build
- Build all packages: `go build ./...`
- Build one package: `go build ./errors`

### Format
- Format all code: `go fmt ./...`
- Optional formatting check: `gofmt -l .`

### Lint / static analysis
- Run vet: `go vet ./...`
- No `golangci-lint` config exists in this repo currently.

### Default tests (unit-style)
- Run all unit tests: `go test -tags unit ./...`
- Run a single package: `go test -tags unit ./errors`
- Run with race detector: `go test -tags unit -race ./...`
- If local `-race` fails due to missing CGO/C compiler (common on Windows), run in Docker:
  `docker run --rm -v "C:/Vortex Services/go-lib/foundation:/work" golang:1.25.7 sh -c 'cd /work && go mod download && go test -race -tags unit ./...'`

### Running a single test (important)
- Single errors test: `go test ./errors -run '^TestErrorResponseToString$'`
- Single validator test: `go test -tags unit ./validator -run '^TestValidate_Valid$'`
- Single retry test: `go test -tags unit ./retry -run '^TestRetryInit_Success$'`
- Single timeutil test: `go test ./timeutil -run '^TestFrozenClock'`
- Prefix match: `go test -tags unit ./retry -run '^TestRetryFast_'`

### Tests without build tags
- Some packages have tests without `unit` tags: `contactutil`, `domain`, `domainutil`, `errors`, `geo`, `logutil`, `netutil`, `piiutil`, `textutil`
- Run those directly: `go test ./errors`, `go test ./piiutil`, etc.

### Fuzz tests
- Run fuzz tests: `go test ./domain -fuzz=FuzzBaseEvent -fuzztime=10s`

## Code style guidelines

### General style
- Follow idiomatic Go and keep code `gofmt`-clean.
- Prefer small focused functions and early returns.
- Keep behavior deterministic; avoid hidden side effects.
- Preserve backward-compatible public APIs unless explicitly changing contracts.
- **DO NOT ADD COMMENTS** unless explicitly asked.

### Imports
- Use standard `gofmt` grouping: stdlib, blank line, external modules.
- Alias imports for clarity or collisions (e.g., `ferrors` for foundation/errors).
- Avoid dot imports.

### Formatting and layout
- Do not manually align spacing; let `gofmt` decide.
- Keep JSON struct tags on one line with spaces: `json:"field,omitempty"`.

### Types and interfaces
- Prefer concrete structs for domain/config payloads (`ErrorResponse`, `BaseEvent`, `Logger`).
- Keep interfaces narrow and capability-focused (`Clock`, `Event`).
- Use typed string constants for finite states (`Reason`).
- Keep zero-value semantics intentional and validated.
- Use copy-on-write pattern for builder-style methods that mutate maps/slices.

### Naming conventions
- Exported identifiers: `PascalCase`.
- Unexported identifiers: `camelCase`.
- Exported sentinel errors: `ErrXxx` (e.g., `ErrInvalidEvent`, `ErrInvalidEventName`).
- Test names: `Test<Subject>_<Scenario>` (e.g., `TestRetryInit_Success`, `TestMaskEmail`).
- Package-level constants: `PascalCase` for exported, `camelCase` for internal.

### Error handling
- Validate inputs up front and fail fast.
- Return sentinel errors for expected invalid-input paths.
- Wrap propagated errors with `%w` when adding context: `fmt.Errorf("%w: %w", ErrInvalidEvent, ErrInvalidEventName)`.
- Use `errors.Is` and `errors.As` instead of string matching.
- Provide detailed sentinel errors for diagnostics (e.g., `ErrInvalidEventNameTooLong`).

### Context, time, and cancellation
- Accept `context.Context` as the first argument of operations.
- Use `context.WithTimeout` around external I/O and cleanup paths.
- Always `defer cancel()` for derived contexts.
- Normalize timestamps to UTC; use `timeutil.Now()` for testability.
- Use `timeutil.FrozenClock` for deterministic time in tests.

### Builder pattern
- Use builder-style methods that return a new value (immutable pattern).
- Implement copy-on-write for map/slice fields to avoid hidden sharing.
- Example: `e.WithDetail("key", "value").WithReason("failed")`.

### Logging patterns
- Use `logger.New(serviceName, env)` for initialization.
- Always `defer log.SafeSync()` after creating logger.
- Use contextual logging with `*w` methods: `log.Infow("message", "key", value)`.
- Inject trace/request IDs via `logger.ContextWithTraceID` and `logger.ContextWithRequestID`.
- Use `*wCtx` methods to auto-extract context fields: `log.InfowCtx(ctx, "message", "key", value)`.

### Retry patterns
- Use `retry.RetryInit` for startup/initialization flows with exponential backoff.
- Use `retry.RetryFast` for quick transient failures with fixed attempts.
- Mark non-retryable errors with `retry.Permanent(err)`.
- Check `retry.IsPermanent(err)` to detect permanent errors.

### Validation patterns
- Use `validator.Validate(struct)` to get field-to-error map.
- Convert to `ErrorResponse` via `errors.ValidationFields(errs)`.
- Register custom validators via `validator.Instance().RegisterValidation()`.

### Error response patterns
- Use presets: `BadRequest()`, `NotFound()`, `Conflict()`, etc.
- Add machine-readable reason: `.WithReason("validation_failed")`.
- Add context via details: `.WithDetail("field", "email")`.
- Use `NotFoundID(resource, id)` for not-found with ID context.
- Use `Forbidden(action, resource)` for RBAC failures.
- Use `RateLimited(retryAfter)` with retry delay.

### Testing conventions
- Prefer table-driven tests for validation matrices.
- Use `t.Parallel()` for independent unit tests (optional).
- Use same-package tests for unexported behavior.
- Use external package tests (`<pkg>_test`) for public API validation.
- Use `github.com/stretchr/testify/assert` in external tests for readability.
- Use `t.Fatalf()` in same-package tests.
- Use `timeutil.NewFrozenClock()` and `timeutil.WithDefault()` for time-dependent tests.
- Use `domain.BaseEvent` for event test fixtures when `Event` interface is needed.

## Agent workflow expectations
- Before completing substantial changes, run `go test -tags unit ./...`.
- If formatting-sensitive files change, run `go fmt ./...`.
- If exported behavior changes, update the relevant package README.
- Avoid adding new dependencies unless necessary.
- Run `go vet ./...` after making changes.
