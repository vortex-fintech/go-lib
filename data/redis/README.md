# redis package

Shared Redis client factory based on `go-redis` universal client.

## What this package provides

- single, sentinel, and cluster bootstrap through one config,
- startup ping health check,
- optional TLS setup (minimum TLS 1.2),
- strict config validation before client creation.

## Supported modes

- `single`: exactly one address, optional `DB` selection.
- `sentinel`: one or more sentinel addresses plus required `MasterName`.
- `cluster`: two or more addresses, `DB` must be `0`.

## Core usage pattern

1. Build `redis.Config` from service config.
2. Call `NewRedisClient(ctx, cfg)` once on startup.
3. Reuse returned `redis.UniversalClient`.
4. Close it on service shutdown.

## Config examples (env-driven)

Use one config model and switch behavior by environment variables.

### Dev (single)

```env
REDIS_MODE=single
REDIS_ADDR=localhost:6380
REDIS_DB=0
```

### Prod (sentinel)

```env
REDIS_MODE=sentinel
REDIS_ADDRS=10.0.0.11:26379,10.0.0.12:26379,10.0.0.13:26379
REDIS_MASTER_NAME=mymaster
REDIS_DB=0
```

### Prod (cluster)

```env
REDIS_MODE=cluster
REDIS_ADDRS=10.1.0.21:6379,10.1.0.22:6379,10.1.0.23:6379
REDIS_DB=0
```

Service business code can stay unchanged when mode changes, because
`NewRedisClient` returns `redis.UniversalClient` for all modes.

## Validation behavior

- missing addresses are rejected,
- unknown mode is rejected,
- sentinel without `MasterName` is rejected,
- `MasterName` outside sentinel mode is rejected,
- negative `DB` is rejected.

## Tests

- Unit: `go test ./redis`
- Integration: `go test -tags integration ./redis`
- Start local Redis for integration tests:

```bash
docker compose -f "redis/docker-compose.test.yml" up -d
```

Optional integration scenarios:

- Sentinel test uses `REDIS_TEST_SENTINEL_ADDRS` and `REDIS_TEST_SENTINEL_MASTER`.
- Cluster test uses `REDIS_TEST_CLUSTER_ADDRS`.
- If these vars are not set, related tests are skipped.
