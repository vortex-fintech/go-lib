# data

Shared data-layer building blocks for Vortex services.

## Packages

| Package | Description |
|---------|-------------|
| [postgres](./postgres) | pgx pool + transaction helpers |
| [redis](./redis) | Redis client factory (single/sentinel/cluster) |
| [idempotency](./idempotency) | Payment-grade idempotency state store |

## Quick Start

### Postgres

```go
import "github.com/vortex-fintech/go-lib/data/postgres"

func main() {
    cfg := postgres.Config{
        URL: "postgres://user:pass@localhost:5432/db?sslmode=disable",
    }
    
    client, err := postgres.Open(cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()
    
    err = postgres.WithTx(ctx, client, func(txCtx context.Context, runner postgres.Runner) error {
        _, err := runner.Exec(txCtx, "INSERT INTO users (id) VALUES ($1)", userID)
        return err
    })
}
```

### Redis

```go
import "github.com/vortex-fintech/go-lib/data/redis"

func main() {
    cfg := redis.Config{
        Mode:  "single",
        Addrs: []string{"localhost:6379"},
    }
    
    client, err := redis.NewRedisClient(ctx, cfg)
    if err != nil {
        panic(err)
    }
    defer client.Close()
}
```

### Idempotency

```go
import "github.com/vortex-fintech/go-lib/data/idempotency"

func (s *Service) ProcessPayment(ctx context.Context, req *PaymentRequest) (*PaymentResponse, error) {
    begin, err := idempotency.Begin(ctx, store, runner, idempotency.BeginInput{
        Principal:      principal,
        GRPCMethod:     "/payment.v1.PaymentService/Create",
        IdempotencyKey: req.IdempotencyKey,
        RequestHash:    hashRequest(req),
        ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
    })
    if err != nil {
        return nil, err
    }
    
    switch begin.Decision {
    case idempotency.BeginDecisionReplay:
        return decodeResponse(begin.Existing.ResponsePayload)
    case idempotency.BeginDecisionInProgress:
        return nil, ErrInProgress
    case idempotency.BeginDecisionRetryable:
        return nil, ErrRetryLater
    }
    
    resp, err := s.executePayment(ctx, req)
    if err != nil {
        return nil, err
    }
    
    done := idempotency.Completion{
        Status:          idempotency.StatusSucceeded,
        ResponsePayload: encodeResponse(resp),
    }
    
    ok, err := idempotency.Finish(ctx, store, runner, *begin.Lease, done)
    if err != nil {
        return nil, err
    }
    if !ok {
        return nil, ErrStaleWorker
    }
    
    return resp, nil
}
```

## By Category

### Database

| Package | Purpose |
|---------|---------|
| [postgres](./postgres) | Connection pool, transactions, savepoints, serializable retries |

### Cache

| Package | Purpose |
|---------|---------|
| [redis](./redis) | Unified client for single/sentinel/cluster modes |

### Reliability

| Package | Purpose |
|---------|---------|
| [idempotency](./idempotency) | Idempotent operation coordination |

## Common Patterns

### Transaction with Runner

```go
err := postgres.WithTx(ctx, client, func(txCtx context.Context, runner postgres.Runner) error {
    if _, err := runner.Exec(txCtx, "UPDATE accounts SET balance = balance - $1 WHERE id = $2", amount, fromID); err != nil {
        return err
    }
    if _, err := runner.Exec(txCtx, "UPDATE accounts SET balance = balance + $1 WHERE id = $2", amount, toID); err != nil {
        return err
    }
    return nil
})
```

### Serializable Retry

```go
err := postgres.WithSerializable(ctx, client, func(txCtx context.Context, runner postgres.Runner) error {
    // Automatically retries on serialization failures (SQLSTATE 40001, 40P01)
    return businessLogic(txCtx, runner)
})
```

### Redis Mode Switching

```go
// Dev
cfg := redis.Config{Mode: "single", Addrs: []string{"localhost:6379"}}

// Prod Sentinel
cfg := redis.Config{
    Mode:       "sentinel",
    Addrs:      []string{"sentinel1:26379", "sentinel2:26379"},
    MasterName: "mymaster",
}

// Prod Cluster
cfg := redis.Config{
    Mode:  "cluster",
    Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
}
```

### Idempotency Retry Worker

```go
func (w *Worker) RetryPending(ctx context.Context) error {
    records, err := store.FindRetryable(ctx, runner)
    if err != nil {
        return err
    }
    
    for _, rec := range records {
        lease, ok, err := idempotency.Reacquire(ctx, store, runner, rec)
        if err != nil {
            return err
        }
        if !ok {
            continue // already retried by another worker
        }
        
        resp, err := w.execute(ctx, rec)
        done := idempotency.Completion{
            Status:          idempotency.StatusSucceeded,
            ResponsePayload: resp,
        }
        idempotency.Finish(ctx, store, runner, lease, done)
    }
    return nil
}
```

## Production Checklist

### Idempotency Setup

1. Apply `idempotency/schema.sql` via migrations
2. Wire middleware metadata (principal, method, key, hash)
3. Call `Begin` → business logic → `Finish`
4. Run retry workers with `Reacquire`
5. Schedule periodic `DeleteExpired` cleanup

### Postgres

- Use `WithTx` for write operations
- Use `WithTxRO` for consistent reads
- Use `WithSerializable` for concurrent hotspots
- Set appropriate pool size via config

### Redis

- Use sentinel/cluster in production
- Enable TLS for external networks
- Configure connection pool via config

## Testing

```bash
# Unit tests
go test ./...

# Integration tests (requires Docker)
docker compose -f postgres/docker-compose.test.yml up -d
docker compose -f redis/docker-compose.test.yml up -d

go test -tags integration ./idempotency ./postgres
go test -tags integration ./redis
```
