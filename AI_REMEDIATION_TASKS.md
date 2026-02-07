# AI Remediation Task Plan for `go-lib`

## Mission
Perform a full quality remediation of the repository with minimal, targeted changes. Fix correctness issues, harden security-sensitive paths, improve code quality where needed, and close test gaps.

## Scope
Modules in this workspace:
- `foundation`
- `security`
- `transport`
- `data`
- `runtime`

Do not perform broad refactors unless strictly necessary for correctness or safety.

## Operating Constraints
- Preserve existing public APIs unless an API change is explicitly required to fix a defect.
- Prefer small focused commits grouped by concern.
- Follow repository conventions from `AGENTS.md` (error wrapping with `%w`, context-first for I/O, no secret logging, strict TLS defaults).
- Keep changes gofmt-compliant.

## Phase 0: Repair Local Toolchain First (Blocking)
Current environment appears broken (`GOROOT` stdlib incomplete; packages like `strings` are missing). Before any code changes:

1. Validate Go toolchain:
   - `go version`
   - `go env GOROOT GOWORK`
   - `go list std | wc -l` (or platform equivalent)
2. Reinstall/repair Go if stdlib is missing.
3. Re-run baseline checks:
   - `go vet ./...` (run from each module dir if needed)
   - `go test -count=1 -tags=unit ./...` (per module)

Do not proceed with remediation until baseline commands run successfully.

## Phase 1: High Severity Fixes (Do First)

### 1) mTLS hot reload data-race risk
Files:
- `security/mtls/client.go`
- `security/mtls/server.go`
- `security/mtls/reload.go`

Problem:
- `tls.Config` fields are mutated in reload callback while in active use.

Tasks:
- Replace in-place mutable pattern with concurrency-safe approach:
  - Use immutable snapshots plus `atomic.Value`, and/or
  - Use `GetCertificate` / `GetConfigForClient` callbacks.
- Ensure no shared mutable TLS state is written without synchronization.
- Keep strict mTLS behavior unchanged.

Acceptance criteria:
- No direct unsynchronized runtime mutation of active `tls.Config` certificate/CA fields.
- Existing behavior preserved (reload still works).
- Add or update tests for concurrent handshake + reload safety.

### 2) Replay checker TTL precision bug
File:
- `security/replay/replay.go`

Problem:
- `int64(ttl.Seconds())` truncates sub-second durations; TTL < 1s may behave like zero.

Tasks:
- Store expiration with higher precision (`time.Time` or unix nanos).
- Keep memory implementation simple and deterministic.

Acceptance criteria:
- Sub-second TTL behaves correctly.
- Add regression tests for: `<1s`, `=0`, negative TTL fallback, and normal TTL.

## Phase 2: Medium Severity Fixes

### 3) Domain error conversion misses wrapped errors
Files:
- `foundation/errors/domain.go`
- `foundation/errors/adapt.go`

Problem:
- Type assertions only; wrapped `DomainError` is not detected.

Tasks:
- Use `errors.As` for `DomainError` detection/conversion.
- Keep existing response mapping semantics.

Acceptance criteria:
- Wrapped domain errors map to validation responses.
- Add tests for wrapped and direct domain errors.

### 4) Nil claims panic risk in OBO validation
File:
- `security/jwt/verifier.go` (claims validation path)

Problem:
- Validator assumes non-nil claims pointer.

Tasks:
- Add explicit nil guard and return a typed validation error.
- Keep existing error taxonomy coherent.

Acceptance criteria:
- No panic on nil claims input.
- Add focused test for nil claims.

### 5) JWKS unknown-kid refresh behavior
File:
- `security/jwt/jwks_verifier.go`

Problem:
- On unknown `kid`, verifier may skip immediate refresh until scheduled time.

Tasks:
- Trigger one immediate refresh on unknown `kid` miss before failing.
- Keep rate limiting/locking safe.

Acceptance criteria:
- Valid token signed by newly rotated key can pass soon after rotation.
- Add tests for unknown-kid -> refresh -> success/failure paths.

### 6) Health handler goroutine accumulation risk
File:
- `runtime/metrics/handler.go`

Problem:
- A goroutine is spawned per health request; if checker ignores context, goroutines may accumulate.

Tasks:
- Prefer synchronous check with timeout-aware contract, or bounded worker pattern.
- Preserve current HTTP behavior (`200` OK, `503` on timeout/error).

Acceptance criteria:
- No unbounded goroutine growth from repeated slow health checks.
- Add stress-style unit test for repeated timeouts.

### 7) Reloader stop idempotency
File:
- `security/mtls/reload.go`

Problem:
- Repeated `Stop()` can panic due to double-close channel.

Tasks:
- Make `Stop()` idempotent (`sync.Once` or equivalent).

Acceptance criteria:
- Multiple `Stop()` calls are safe.
- Add test covering repeated stop calls.

## Phase 3: Test Coverage Expansion

### Security tests (highest priority)
- Add comprehensive tests for `security/jwt`:
  - time claims (`exp`, `iat`, leeway),
  - audience/actor/azp rules,
  - PoP (`cnf`) binding,
  - replay handling,
  - nil claims handling,
  - key rotation behavior (JWKS).
- Add full tests for `security/replay`:
  - TTL precision,
  - max item trimming,
  - concurrent access.

### Transport/authz tests
- Add tests for `transport/grpc/middleware/authz`:
  - missing metadata,
  - skip auth paths,
  - error mapping to gRPC status,
  - replay and PoP integration hooks.

### Data layer tests
- Add tests for `data/postgres/tx.go`:
  - rollback on error/panic,
  - serializable retry path,
  - savepoint behavior,
  - timeout propagation.

## Phase 4: Verification Matrix
Run, in this order:

1. Narrow tests for touched packages.
2. Module-wide unit tests:
   - `go test -count=1 -tags=unit -v ./...` (per module)
3. Additional tagged tests where relevant:
   - `go test -count=1 -tags="unit testhooks" -v ./data/postgres`
4. Static checks:
   - `go vet ./...` (per module)

If integration behavior is changed, run integration suite with documented Docker flow.

## Commit Strategy
- Commit by concern, not by file type.
- Suggested sequence:
  1. `security/mtls` concurrency safety
  2. `security/replay` TTL fix + tests
  3. `foundation/errors` wrapped-domain handling + tests
  4. `security/jwt` nil guard + JWKS refresh + tests
  5. `runtime/metrics` health handler hardening + tests
  6. Remaining coverage expansion

## Definition of Done
- All high/medium findings above resolved.
- New/updated tests cover each fixed bug and main edge cases.
- Unit tests and vet pass across all modules.
- No API breakage unless explicitly documented.
- No unrelated refactors or formatting-only churn.
