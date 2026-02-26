# Replay Protection

Anti-replay protection for JTI (JWT ID) tracking.

## Where to use it

- Prevent token replay attacks
- Deduplicate API requests
- Track processed message IDs

## Redis checker

```go
import "github.com/vortex-fintech/go-lib/security/replay"

rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
checker := replay.NewRedisChecker(rdb, replay.RedisOptions{
    Prefix:   "obo:jti",
    FailOpen: false,
})

seen, err := checker.SeenJTI(ctx, "wallet-service", "jti-12345", time.Hour)
if err != nil {
    return err
}
if seen {
    return errors.New("replay detected")
}
```

`ttl` must be greater than zero. Passing `ttl <= 0` is treated as a configuration error.

### Fail-open vs fail-closed

```go
// Fail-closed (default): Redis error = treat as replay
replay.RedisOptions{FailOpen: false}

// Fail-open: Redis error = allow request
replay.RedisOptions{FailOpen: true}
```

The same policy is applied to local misconfiguration errors (for example, non-positive TTL or nil Redis client):

- `FailOpen=false` => `SeenJTI` returns `seen=true` with error (block request)
- `FailOpen=true` => `SeenJTI` returns `seen=false, nil` (allow request)

## In-memory checker

For development or single-instance deployments:

```go
checker := replay.NewInMemoryChecker(replay.MemoryOptions{
    TTL:      time.Hour,
    MaxItems: 100000,
})

seen, _ := checker.SeenJTI(ctx, "wallet", "jti-abc", time.Hour)
```

When `MaxItems` is reached, oldest entries are evicted.

## Integration with JWT validation

```go
checker := replay.NewRedisChecker(rdb, replay.RedisOptions{})

opt := jwt.OBOValidateOptions{
    WantAudience: "wallet",
    SeenJTI:      checker.AsAuthzCallback("wallet", time.Hour),
}

if err := jwt.ValidateOBO(time.Now(), claims, opt); err != nil {
    if errors.Is(err, jwt.ErrReplay) {
        // Replay attack blocked
    }
}
```

## AsAuthzCallback adapter

Converts checker to simple callback for JWT validation:

```go
seenFunc := checker.AsAuthzCallback("wallet-service", time.Hour)

if seenFunc("jti-123") {
    // Already processed
}
```

## Interface

```go
type Checker interface {
    SeenJTI(ctx context.Context, namespace, jti string, ttl time.Duration) (seen bool, err error)
}
```

Returns `true` if JTI was already seen (replay), `false` if new.

## Production notes

- Use Redis for distributed systems
- Always pass a positive TTL to `SeenJTI` (usually slightly longer than token MaxTTL)
- Monitor Redis connectivity in fail-closed mode
- In-memory checker suitable only for single-instance deployments
- Namespace JTIs by service to avoid collisions
