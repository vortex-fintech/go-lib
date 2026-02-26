# AGENTS.md

Operational guide for agentic coding assistants working in this repository.

## Scope
- Module: `github.com/vortex-fintech/go-lib/data`
- Language: Go
- Go version: `go 1.25`
- Toolchain in `go.mod`: `go1.25.7`
- Main packages: `postgres`, `redis`, `idempotency`
- This repository is a shared library module (not an app entrypoint)

## Source-of-truth docs
- `README.md`
- `postgres/README.md`
- `redis/README.md`
- `idempotency/README.md`
- `idempotency/schema.sql`

## Cursor / Copilot rules
- `.cursorrules`: not present
- `.cursor/rules/`: not present
- `.github/copilot-instructions.md`: not present
- If any are added later, treat them as higher-priority constraints than this file.

## Build / format / lint / test

### Build
- Build all packages: `go build ./...`
- Build one package: `go build ./postgres` (or `./redis`, `./idempotency`)

### Format
- Format package code: `go fmt ./...`
- Optional formatting check: `gofmt -l .`

### Lint / static analysis
- Run vet: `go vet ./...`
- No `golangci-lint` config exists in this repo currently.

### Default tests (unit-style)
- Run all default tests: `go test ./...`
- Run a single package: `go test ./idempotency`
- Run race detector: `go test -race ./...`
- If local `-race` fails due to missing CGO/C compiler (common on Windows), run race tests in Docker:
  `docker run --rm -v "C:/Vortex Services/go-lib/data:/work" golang:1.25.7 sh -c 'cd /work && go mod download && go test -race ./idempotency ./postgres ./redis'`

### Running a single test (important)
- Single idempotency test:
  `go test ./idempotency -run '^TestReserve_InsertSuccess$'`
- Single postgres test:
  `go test ./postgres -run '^TestConfigValidate_OK$'`
- Single redis test:
  `go test ./redis -run '^TestNewRedisClient_TLSConfigApplied$'`
- Prefix match in package:
  `go test ./idempotency -run '^TestBegin_'`

### Tagged tests
- Postgres + idempotency integration:
  `go test -tags integration ./postgres ./idempotency`
- Redis integration:
  `go test -tags integration ./redis`
- Single integration test example:
  `go test -tags integration ./postgres -run '^TestWithTx_RollbackOnError_Integration$'`
- Extra postgres tests are gated by `unit && testhooks` build tags:
  `go test -tags 'unit testhooks' ./postgres`
- Single test for that suite:
  `go test -tags 'unit testhooks' ./postgres -run '^TestOpen_Success$'`

### Integration environment setup
- Start Postgres test container:
  `docker compose -f "postgres/docker-compose.test.yml" up -d`
- Stop Postgres test container:
  `docker compose -f "postgres/docker-compose.test.yml" down -v`
- Start Redis test container:
  `docker compose -f "redis/docker-compose.test.yml" up -d`
- Stop Redis test container:
  `docker compose -f "redis/docker-compose.test.yml" down -v`

### Integration test defaults and env vars
- Postgres DSN in tests:
  `postgres://testuser:testpass@localhost:5433/testdb?sslmode=disable`
- Redis default integration address: `localhost:6380`
- Optional sentinel vars:
  `REDIS_TEST_SENTINEL_ADDRS`, `REDIS_TEST_SENTINEL_MASTER`
- Optional cluster var:
  `REDIS_TEST_CLUSTER_ADDRS`

## Code style guidelines

### General style
- Follow idiomatic Go and keep code `gofmt`-clean.
- Prefer small focused functions and early returns.
- Keep behavior deterministic; avoid hidden side effects.
- Preserve backward-compatible public APIs unless explicitly changing contracts.
- **DO NOT ADD COMMENTS** unless explicitly asked.

### Imports
- Use standard `gofmt` grouping: stdlib, blank line, external/internal modules.
- Alias imports only when useful for clarity or collisions (`pg`, `redispkg`, `goredis`).
- Avoid dot imports.

### Formatting and layout
- Do not manually align spacing; let `gofmt` decide.
- Keep SQL in raw string literals and format it for readability.
- In SQL, prefer one condition per line for long `WHERE` clauses.

### Types and interfaces
- Prefer concrete structs for domain/config payloads (`Config`, `Record`, `Completion`).
- Keep interfaces narrow and capability-focused (`Runner`, `Store`, `TxManager`).
- Use typed string constants for finite states/decisions.
- Keep zero-value semantics intentional and validated.

### Naming conventions
- Exported identifiers: `PascalCase`.
- Unexported identifiers: `camelCase`.
- Exported sentinel errors: `ErrXxx`.
- Internal package errors: `errXxx`.
- Test names: `Test<Subject>_<Scenario>`.

### Error handling
- Validate inputs up front and fail fast.
- Return sentinel errors for expected invalid-input paths.
- Wrap propagated errors with `%w` when adding context.
- Use `errors.Is` and `errors.As` instead of string matching.
- Preserve multi-step cleanup failures with `errors.Join` when applicable.

### Context, time, and cancellation
- Accept `context.Context` as the first argument of operations.
- Use `context.WithTimeout` around external I/O and cleanup paths.
- Always `defer cancel()` for derived contexts.
- Normalize DB-facing timestamps to UTC; use microsecond precision for comparisons.
- Keep transaction helpers panic-safe (rollback on panic, commit only on success).

### Database and transaction patterns
- Prefer the `Runner` abstraction in data logic.
- Inside transaction callbacks, read runner via `postgres.MustRunnerFromContext`.
- Keep optimistic-lock predicates explicit in SQL (`updated_at` guards, status guards).
- Use explicit SQLSTATE checks for retry decisions (`40001`, `40P01`).

### Redis client patterns
- Validate mode/address/master settings before creating clients.
- Centralize mode and address normalization.
- Enforce TLS minimum 1.2 when TLS is enabled.
- Keep startup ping health check and close client if ping fails.

### Testing conventions
- Prefer table-driven tests for validation matrices.
- Use `t.Parallel()` for independent unit tests.
- Rebind loop vars for parallel subtests (`tc := tc`).
- Use same-package tests for unexported behavior.
- Use external package tests (`<pkg>_test`) for public API validation.
- Use `testify/require` where it improves readability in multi-step assertions.

## Agent workflow expectations
- Before completing substantial changes, run `go test ./...`.
- If integration behavior changes, run the relevant `-tags integration` suites.
- If formatting-sensitive files change, run `go fmt ./...`.
- If exported behavior changes, update the relevant package README.
- Avoid adding new dependencies unless necessary.
