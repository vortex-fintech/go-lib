# postgres package

Shared Postgres access layer built on `pgx`.

## What this package provides

- connection bootstrap from URL or structured DB config,
- pooled runner abstraction (`Runner`) for pool and tx paths,
- transaction helpers (`WithTx`, `WithTxRO`, `WithTxOpts`),
- serializable retries (`WithSerializable`),
- savepoint helper (`WithSavepoint`),
- SQLSTATE helpers for constraint errors.

## Core usage pattern

1. Open one `Client` on service startup.
2. Use `RunnerFromPool()` for non-transactional queries.
3. Use `WithTx(...)` for atomic write flows.
4. Use `WithTxRO(...)` for consistent read-only multi-query reads.
5. Inside tx callback, use `MustRunnerFromContext(txCtx)` only in internal layers with strict invariants; on public boundaries, prefer `RunnerFromContextOrError(txCtx)` and return `(value, error)`.

## Interfaces

- `TxManager`: minimal contract (`WithTx`, `WithTxRO`) for higher layers.
- `AdvancedTxManager`: extended contract with tx options, serializable retry, and savepoints.

## Reliability notes

- `WithSerializable` retries SQLSTATE `40001` and `40P01`.
- Transaction cleanup is panic-safe.
- Savepoint rollback/release cleanup errors are preserved and returned.

## Tests

- Unit: `go test ./postgres`
- Integration: `go test -tags integration ./postgres`
- Race (Docker):

```bash
docker run --rm -v "C:/Vortex Services/go-lib/data:/work" golang:1.25.7 sh -c 'cd /work && go mod download && go test -race ./postgres'
```
