# AGENTS.md

## Purpose
- Guide for autonomous coding agents in `github.com/vortex-fintech/go-lib`.
- Prefer minimal, targeted changes and keep package conventions intact.
- Avoid broad refactors unless explicitly requested.

## Repository Snapshot
- Multi-module workspace (`go.work`) with modules:
  - `github.com/vortex-fintech/go-lib/foundation`
  - `github.com/vortex-fintech/go-lib/security`
  - `github.com/vortex-fintech/go-lib/transport`
  - `github.com/vortex-fintech/go-lib/data`
  - `github.com/vortex-fintech/go-lib/runtime`
- Go: `go 1.25`, toolchain `go1.25.1`
- Main entrypoints: `Makefile`, `README.md`, `go.work`
- Integration infra: `data/postgres/docker-compose.test.yml`
- Build tags used: `unit`, `testhooks`, `integration`, `unix`

## Local Rule Files (must check)
- Cursor rules: not found (`.cursor/rules/`, `.cursorrules`)
- Copilot rules: not found (`.github/copilot-instructions.md`)

## Build / Lint / Test Commands

### Build
```bash
go build -v ./...
make build
```

### Dependency hygiene
```bash
for m in foundation security transport data runtime; do (cd "$m" && go mod tidy); done
make tidy
```

### Lint / static checks
```bash
go vet ./...
```

### Unit tests (default)
```bash
go test -count=1 -tags=unit -v ./...
go test -count=1 -tags="unit testhooks" -v ./data/postgres
make test
```

### Integration tests
```bash
make test-integration
```
Manual flow:
```bash
docker compose -f data/postgres/docker-compose.test.yml up -d --wait --wait-timeout 60
go test -count=1 -tags=integration -v ./...
docker compose -f data/postgres/docker-compose.test.yml down -v
```

### Full checks
```bash
make test-all
make test-race
make cover
```

## Running a Single Test (Important)

### Single test by exact name
```bash
go test -count=1 -run '^TestName$' ./path/to/package
```

### Single subtest
```bash
go test -count=1 -run '^TestParent$/^subtest name$' ./path/to/package
```

### Single unit-tagged test
```bash
go test -count=1 -tags=unit -run '^TestName$' ./path/to/package
```

### Single `unit+testhooks` test (Postgres hooks)
```bash
go test -count=1 -tags="unit testhooks" -run '^TestOpen_Success$' ./data/postgres
```

### Single integration test
```bash
go test -count=1 -tags=integration -run '^TestOpen_Integration$' ./data/postgres
```
- Ensure Postgres container is up before DB integration tests.

### Unix-only tests
- Files with `//go:build unix` run only on Unix-like systems.

## Build Tag Matrix
- `unit`: opt-in unit tests
- `testhooks`: enables testing hooks (non-production support)
- `integration`: integration tests with external/runtime behavior
- `unix`: OS-specific behavior

## Code Style Guidelines

### Imports
- Use standard grouping: stdlib, blank line, external/internal modules.
- Keep imports `gofmt` sorted.
- Alias imports only for collisions or clarity.
- Avoid dot imports.

### Formatting
- Follow `gofmt` strictly.
- Prefer small focused functions and early returns.
- Avoid deep nesting when guard clauses are clearer.
- Add comments only for non-obvious invariants/logic.

### Types and API design
- Prefer explicit config structs (`Config`, `Options`) for constructors.
- Use functional options where package already follows this style.
- Use `any` (not `interface{}`) in new code.
- Preserve exported API signatures unless change is requested.

### Naming
- Package names: lowercase, short, no underscores.
- Exported: `PascalCase`; unexported: `camelCase`.
- Sentinel errors: `Err...`.
- Constants should be domain-descriptive.
- Tests: `TestFeature_Scenario` when practical.

### Error handling
- Return errors; do not panic in normal runtime paths.
- Panic is acceptable only in explicit `Must*` helpers or invalid required config.
- Wrap root errors with context via `%w`.
- Prefer `errors.Is` / `errors.As` over string matching.
- Reuse `errors` package adapters (`ToGRPC`, `ToHTTP`, etc.).

### Context, timeouts, concurrency
- Pass `context.Context` as the first arg for I/O/long-running ops.
- Use `context.WithTimeout` around external calls and health checks.
- Always call `cancel()` for derived contexts.
- Protect shared mutable state with `sync` / `atomic`.
- Avoid goroutine leaks; stop timers where needed.

### Security and logging
- Never log secrets, tokens, OTPs, passwords, private keys, or raw cert/key material.
- Follow `logutil` redaction/sanitization patterns.
- Preserve strict TLS defaults and mTLS/PoP/replay checks in auth flows.

## Testing Guidelines
- Prefer table-driven tests for behavior matrices.
- Use `t.Run` for subcases.
- Use `t.Parallel()` where tests are isolated.
- Match local assertion style (`testing`, `assert`, `require`).
- If tests mutate globals/hooks, restore state in `t.Cleanup`.

## Agent Workflow
1. Read target package and nearby tests first.
2. Implement the smallest viable change.
3. Run narrow tests first (single test / package).
4. Run broader checks relevant to modified surface.
5. Avoid unrelated opportunistic refactors.