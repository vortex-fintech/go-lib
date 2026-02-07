# go-lib

`go-lib` is a multi-module Go workspace with shared infrastructure code for Vortex services.

This README is intentionally detailed and documents what each tracked file does.

## Modules

- `foundation` (`github.com/vortex-fintech/go-lib/foundation`)
- `security` (`github.com/vortex-fintech/go-lib/security`)
- `transport` (`github.com/vortex-fintech/go-lib/transport`)
- `data` (`github.com/vortex-fintech/go-lib/data`)
- `runtime` (`github.com/vortex-fintech/go-lib/runtime`)

---

## Root Files

- `.gitignore` - Git ignore rules for generated files, caches, coverage artifacts, and local env noise.
- `AGENTS.md` - Instructions for autonomous coding agents (workflow, coding rules, test commands).
- `AI_REMEDIATION_TASKS.md` - Structured remediation checklist for correctness/security/testing hardening.
- `LICENSE` - MIT license text.
- `Makefile` - Common developer tasks (`build`, `test`, `test-integration`, `test-race`, `cover`, etc.).
- `README.md` - Repository documentation.

---

## data Module

### Module Manifest

- `data/go.mod` - Module definition and direct dependencies for data layer packages.

### Postgres (`data/postgres`)

- `data/postgres/client.go` - Postgres client creation via `pgxpool`, DSN handling, pool config, health ping.
- `data/postgres/client_integration_test.go` - Integration test for opening a real Postgres connection.
- `data/postgres/client_test.go` - Unit tests for open-path success/failure and error helpers.
- `data/postgres/client_testhooks.go` - Build-tagged test hooks to stub `newPool`/`pingPool` in unit tests.
- `data/postgres/config.go` - Config structs for Postgres connection and pool settings.
- `data/postgres/docker-compose.test.yml` - Local Postgres container definition for integration tests.
- `data/postgres/pgerr.go` - Helpers for PostgreSQL error classification (constraint/unique violation mapping).
- `data/postgres/runner.go` - `Runner` abstraction and pool/tx runner implementations.
- `data/postgres/tx.go` - Transaction helpers (`WithTx`, serializable retry, savepoints, timeout config).
- `data/postgres/tx_integration_test.go` - Integration tests for rollback and serialization-retry behavior.
- `data/postgres/tx_test.go` - Unit tests for savepoint and serialization helper logic.

### Redis (`data/redis`)

- `data/redis/client.go` - Redis client factory with connectivity checks and configuration application.
- `data/redis/client_test.go` - Unit tests for Redis client setup and failure scenarios.
- `data/redis/config.go` - Redis config structs/options used by client creation.

---

## foundation Module

### Module Manifest

- `foundation/go.mod` - Module definition and direct dependencies for foundation utilities.
- `foundation/go.sum` - Dependency checksums for reproducible module builds.

### contactutil (`foundation/contactutil`)

- `foundation/contactutil/normalize.go` - Email/phone normalization helpers.

### domain (`foundation/domain`)

- `foundation/domain/event.go` - Core domain event interface/types and base event behavior.
- `foundation/domain/event_buffer.go` - In-memory event buffer implementation.
- `foundation/domain/base_event_test.go` - Unit tests for base event validation/metadata behavior.
- `foundation/domain/event_buffer_test.go` - Unit tests for buffer operations and edge cases.

### errors (`foundation/errors`)

- `foundation/errors/model.go` - Canonical error model (`ErrorResponse`, reasons, violations, helpers).
- `foundation/errors/constructors.go` - Constructors/presets for common error categories.
- `foundation/errors/presets.go` - Predefined error response templates and convenience builders.
- `foundation/errors/domain.go` - Domain error types, detection, and conversion helpers.
- `foundation/errors/adapt.go` - Generic adaptation (`error` -> internal error model) entrypoints.
- `foundation/errors/grpc.go` - Mapping to/from gRPC `status` and `codes`.
- `foundation/errors/grpc_extras.go` - Additional gRPC adaptation helpers.
- `foundation/errors/http.go` - HTTP status mapping and serialization helpers.
- `foundation/errors/validation_adapters.go` - Validation error adaptation into structured field violations.
- `foundation/errors/model_test.go` - Unit tests for core error model methods.
- `foundation/errors/domain_test.go` - Unit tests for domain error conversion (including wrapped errors).
- `foundation/errors/grpc_test.go` - Unit tests for gRPC error adapters.
- `foundation/errors/http_test.go` - Unit tests for HTTP error adapters.
- `foundation/errors/validation_adapters_test.go` - Unit tests for validation adaptation behavior.

### errx (`foundation/errx`)

- `foundation/errx/errx.go` - Small shared error utility helpers used across packages.

### geo (`foundation/geo`)

- `foundation/geo/country.go` - Country/ISO utility helpers and normalization.

### hash (`foundation/hash`)

- `foundation/hash/sha256_util.go` - SHA-256 helper functions.
- `foundation/hash/hash_test.go` - Unit tests for hash helpers.

### logger (`foundation/logger`)

- `foundation/logger/interface.go` - Logger interfaces and abstraction contracts.
- `foundation/logger/logger.go` - Logger initialization/configuration implementation.
- `foundation/logger/logger_test.go` - Unit tests for logger behavior and options.

### logutil (`foundation/logutil`)

- `foundation/logutil/redact.go` - Sensitive data redaction/sanitization utilities.
- `foundation/logutil/sanitize_validation_errors_test.go` - Tests for validation error sanitization.

### netutil (`foundation/netutil`)

- `foundation/netutil/timeout.go` - Timeout normalization/clamping helpers.
- `foundation/netutil/sanitize_timeout_test.go` - Unit tests for timeout sanitization.

### piiutil (`foundation/piiutil`)

- `foundation/piiutil/mask.go` - PII masking utilities.

### retry (`foundation/retry`)

- `foundation/retry/retry.go` - Retry primitives with backoff and context awareness.
- `foundation/retry/retry_test.go` - Unit tests for retry timing/cancel/error behavior.

### timeutil (`foundation/timeutil`)

- `foundation/timeutil/clock.go` - Clock abstraction and time helper methods.
- `foundation/timeutil/clock_test.go` - Unit tests for clock helpers and determinism.
- `foundation/timeutil/period.go` - Time-period helper functions.

### validator (`foundation/validator`)

- `foundation/validator/tagmap.go` - Validation tag-to-reason mapping utilities.
- `foundation/validator/validator.go` - Validator wrapper and error shaping.
- `foundation/validator/validator_test.go` - Unit tests for validator behavior.

---

## runtime Module

### Module Manifest

- `runtime/go.mod` - Module definition and dependencies for runtime infra packages.

### graceful (`runtime/graceful`)

- `runtime/graceful/metrics.go` - Graceful lifecycle metrics definitions and recorders.
- `runtime/graceful/metrics_test.go` - Unit tests for graceful metrics behavior.

### metrics (`runtime/metrics`)

- `runtime/metrics/handler.go` - HTTP handler exposing `/metrics` and `/health` with timeout/concurrency controls.
- `runtime/metrics/metrics_test.go` - Unit tests for endpoints, health behavior, timeouts, and concurrency limit.

### shutdown (`runtime/shutdown`)

- `runtime/shutdown/manager.go` - Graceful shutdown manager orchestration for multiple components.
- `runtime/shutdown/manager_test.go` - Unit tests for shutdown manager control flow and error handling.
- `runtime/shutdown/manager_integration_grpc_test.go` - Integration-like tests for gRPC shutdown behavior.
- `runtime/shutdown/manager_signals_unix_test.go` - Unix signal handling tests.

### shutdown adapters (`runtime/shutdown/adapters`)

- `runtime/shutdown/adapters/grpc.go` - Adapter from gRPC server to shutdown manager interface.
- `runtime/shutdown/adapters/grpc_test.go` - Unit tests for gRPC shutdown adapter.
- `runtime/shutdown/adapters/http.go` - Adapter from HTTP server to shutdown manager interface.
- `runtime/shutdown/adapters/http_test.go` - Unit tests for HTTP shutdown adapter.

---

## security Module

### Module Manifest

- `security/go.mod` - Module definition and dependencies for security packages.

### hmacotp (`security/hmacotp`)

- `security/hmacotp/hmacotp.go` - HMAC-based one-time code generation and verification logic.
- `security/hmacotp/hmacotp_test.go` - Unit tests for OTP validity windows and verification rules.

### jwt (`security/jwt`)

- `security/jwt/jwks_verifier.go` - JWKS-backed JWT verification, key cache, refresh policy, signature and claim checks.
- `security/jwt/verifier.go` - OBO claim types and strict token validation rules (audience/actor/ttl/replay/PoP).
- `security/jwt/verifier_test.go` - Unit tests for nil-claims guard and unknown-`kid` refresh behavior.

### mtls (`security/mtls`)

- `security/mtls/config.go` - mTLS config struct for cert paths/server name/reload interval.
- `security/mtls/loader.go` - Certificate/key/CA loading and bundle creation helpers.
- `security/mtls/client.go` - Client TLS config builder with reload-safe cert/CA sourcing.
- `security/mtls/server.go` - Server TLS config builder with strict client cert verification and reload support.
- `security/mtls/reload.go` - Polling-based certificate reloader and change detection.
- `security/mtls/test_helpers.go` - Test utilities for temporary cert generation/material setup.
- `security/mtls/loader_test.go` - Unit tests for loader and bundle parsing.
- `security/mtls/client_test.go` - Unit tests for client TLS config behavior.
- `security/mtls/server_test.go` - Unit tests for server TLS config behavior.
- `security/mtls/reload_test.go` - Unit tests for reload triggering and idempotent stop.

### replay (`security/replay`)

- `security/replay/replay.go` - Replay checker interfaces and in-memory implementation with TTL handling.
- `security/replay/replay_test.go` - Unit tests for sub-second TTL and default TTL fallback behavior.

### scope (`security/scope`)

- `security/scope/checker.go` - Scope matching helpers (`all`/`any`) for authorization decisions.

### tlsutil (`security/tlsutil`)

- `security/tlsutil/x5t.go` - TLS certificate thumbprint (`x5t`) helper functions.

---

## transport Module

### Module Manifest

- `transport/go.mod` - Module definition and dependencies for transport packages.

### grpc/creds (`transport/grpc/creds`)

- `transport/grpc/creds/creds.go` - gRPC credential helper constructors.

### grpc/dial (`transport/grpc/dial`)

- `transport/grpc/dial/dial.go` - gRPC dialing helpers and common connection options.

### grpc/metadata (`transport/grpc/metadata`)

- `transport/grpc/metadata/metadata.go` - Metadata extraction/injection helpers.

### grpc/middleware/authz (`transport/grpc/middleware/authz`)

- `transport/grpc/middleware/authz/context.go` - Identity/claims context storage and retrieval helpers.
- `transport/grpc/middleware/authz/helpers.go` - Policy and skip-auth resolver helper builders.
- `transport/grpc/middleware/authz/interceptor.go` - Unary/stream authz interceptors (JWT verify, OBO validation, PoP, scope checks).
- `transport/grpc/middleware/authz/interceptor_test.go` - Tests for authz success/failure paths in unary and stream flows.

### grpc/middleware/chain (`transport/grpc/middleware/chain`)

- `transport/grpc/middleware/chain/unary.go` - Middleware chain composition helpers.

### grpc/middleware/circuitbreaker (`transport/grpc/middleware/circuitbreaker`)

- `transport/grpc/middleware/circuitbreaker/interceptor.go` - Circuit breaker interceptor implementation.
- `transport/grpc/middleware/circuitbreaker/logger_adapter.go` - Logging adapter for circuit breaker internals.
- `transport/grpc/middleware/circuitbreaker/interceptor_test.go` - Unit tests for breaker transitions and request handling.

### grpc/middleware/contextcancel (`transport/grpc/middleware/contextcancel`)

- `transport/grpc/middleware/contextcancel/interceptor.go` - Context cancellation propagation interceptor.
- `transport/grpc/middleware/contextcancel/interceptor_test.go` - Unit tests for context cancel behavior.

### grpc/middleware/errorsmw (`transport/grpc/middleware/errorsmw`)

- `transport/grpc/middleware/errorsmw/interceptor.go` - Error normalization interceptor (domain/internal -> gRPC status).
- `transport/grpc/middleware/errorsmw/interceptor_test.go` - Unit tests for error mapping behavior.

### grpc/middleware/metricsmw (`transport/grpc/middleware/metricsmw`)

- `transport/grpc/middleware/metricsmw/interceptor.go` - Metrics collection interceptor shell.
- `transport/grpc/middleware/metricsmw/unary_full_test.go` - Unit tests for full unary metrics flow.

### grpc/middleware/metricsmw/promreporter (`transport/grpc/middleware/metricsmw/promreporter`)

- `transport/grpc/middleware/metricsmw/promreporter/reporter.go` - Prometheus-backed metrics reporter implementation.
- `transport/grpc/middleware/metricsmw/promreporter/reporter_test.go` - Unit tests for Prometheus reporter output and labels.

---

## Development Commands

Typical commands used in this repo:

```bash
# per-module unit tests
go test -count=1 -tags=unit ./...

# postgres unit with hooks
go test -count=1 -tags="unit testhooks" -v ./data/postgres

# integration tests with postgres container
docker compose -f data/postgres/docker-compose.test.yml up -d --wait --wait-timeout 60
go test -count=1 -tags=integration -v ./...
docker compose -f data/postgres/docker-compose.test.yml down -v

# static checks
go vet ./...
```

See `AGENTS.md` and `Makefile` for full workflow conventions.
