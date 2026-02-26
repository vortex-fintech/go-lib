# AGENTS.md

Operational guide for agentic coding assistants working in `go-lib` workspace.

## Scope

- Workspace root: `C:\Vortex Services\go-lib`
- Language: Go
- Go version: `go 1.25`
- Toolchain: `go1.25.7`
- Modules (from `go.work`):
  - `./foundation`
  - `./security`
  - `./transport`
  - `./data`
  - `./runtime`
  - `./messaging/kafka/franzgo`
  - `./messaging/kafka/schemaregistry`

## Rule precedence

1. Root `AGENTS.md` (this file) defines cross-workspace defaults.
2. Module-level `AGENTS.md` files define package-specific rules and override root defaults for that module:
   - `foundation/AGENTS.md`
   - `security/AGENTS.md`
   - `transport/AGENTS.md`
   - `data/AGENTS.md`
   - `runtime/AGENTS.md`
   - `messaging/AGENTS.md`

When editing files inside a module, always follow that module's `AGENTS.md` as the primary source.

## Workspace build / test policy

Run checks per module (not `go test ./...` from workspace root).

### Build

```bash
for m in foundation security transport data runtime messaging/kafka/franzgo messaging/kafka/schemaregistry; do
  (cd "$m" && go build ./...)
done
```

### Tests

```bash
for m in foundation security transport data runtime messaging/kafka/franzgo messaging/kafka/schemaregistry; do
  (cd "$m" && go test -count=1 ./...)
done
```

### Unit-tagged suites

```bash
(cd foundation && go test -count=1 -tags unit ./...)
(cd runtime && go test -count=1 -tags unit ./...)
(cd data && go test -count=1 -tags "unit testhooks" ./postgres)
```

### Vet

```bash
for m in foundation security transport data runtime messaging/kafka/franzgo messaging/kafka/schemaregistry; do
  (cd "$m" && go vet ./...)
done
```

### Integration and race

- Integration tests use Docker infrastructure for Postgres, Redis, and Redpanda.
- Race checks should run in Docker using `golang:1.25.7` when local CGO toolchain is unavailable.

## Code change policy

- Keep public API backward-compatible unless task explicitly requires breaking changes.
- Keep functions nil-safe at module boundaries where contexts/pointers may be nil.
- Prefer deterministic behavior and explicit validation errors.
- Do not add new external dependencies unless strictly required.
- If exported behavior changes, update relevant README files.

## CI/CD files

GitHub Actions workflows are stored in `.github/workflows`:

- `ci.yml`
- `integration-race.yml`
- `release.yml`

Any change to build/test process should update both this file and workflow definitions.
