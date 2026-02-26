# Idempotency Store Contract

This package stores and coordinates idempotency state for payment-like operations.

## Where to use it

- Keep handlers and business decisions in services.
- Use this package for reusable orchestration (`Begin`, `Finish`, `Reacquire`) and the Postgres `Store` implementation.

## Storage model

- `idempotency_keys` is the source of truth.
- Uniqueness key: `(principal, grpc_method, idempotency_key)`.
- `request_hash` must match for repeated calls with the same idempotency key.

## Service flow

1. Call `Begin(...)`.
   - `EXECUTE`: new operation, run business logic.
   - `REPLAY`: already completed (`SUCCEEDED` or `FAILED_FINAL`), return stored response.
   - `IN_PROGRESS`: another request is running, return a retry-later style response.
   - `RETRYABLE`: previous run ended with `FAILED_RETRYABLE`, trigger retry policy.
2. After business logic, call `Finish(...)`.
3. For retry workers, call `Reacquire(...)` with a new lease token (`updatedAt`), then `Finish(...)`.

## Handler template (service layer)

```go
begin, err := idempotency.Begin(ctx, store, run, idempotency.BeginInput{
    Principal:      principal,
    GRPCMethod:     fullMethod,
    IdempotencyKey: idemKey,
    RequestHash:    reqHash,
    ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
})
if err != nil {
    return nil, err
}

switch begin.Decision {
case idempotency.BeginDecisionReplay:
    return decodePayload(begin.Existing.ResponsePayload)
case idempotency.BeginDecisionInProgress:
    return nil, errInProgress
case idempotency.BeginDecisionRetryable:
    return nil, errRetryLater
case idempotency.BeginDecisionExecute:
    // run business operation
}

lease := *begin.Lease
done := idempotency.Completion{
    Status:          idempotency.StatusSucceeded,
    ResponseCode:    0,
    ResponsePayload: responseBytes,
    // UpdatedAt is optional here: Finish uses lease.UpdatedAt when empty.
}

ok, err := idempotency.Finish(ctx, store, run, lease, done)
if err != nil {
    return nil, err
}
if !ok {
    return nil, errStaleWorker
}

return response, nil
```

## Concurrency and safety

- `Complete(...)` uses optimistic lock: `status='IN_PROGRESS' AND updated_at=<lease-token>`.
- A stale worker cannot complete a newer retried attempt.
- Timestamps are normalized to UTC microseconds before DB comparison.

## Production notes

- Apply `schema.sql` before using the store.
- Keep idempotency source of truth in Postgres for payment-grade consistency.
- Use Redis only as cache, not as the primary idempotency store.
- For module-level checklist, see `../README.md`.
