# gRPC Idempotency Middleware

Extracts idempotency key and request hash from incoming gRPC requests.

## Where to use it

- Payment operations
- Any operation that must not be duplicated
- Services using `data/idempotency` package

## How it works

1. Extracts `idempotency-key` header from gRPC metadata
2. Hashes request payload with SHA-256 (deterministic serialization)
3. Puts `Metadata` struct into context for service layer

**Important:** This middleware only extracts metadata. The service layer must call `idempotency.Begin/Finish` for actual idempotency handling.

## Basic usage

```go
import (
    "github.com/vortex-fintech/go-lib/transport/grpc/middleware/idempotencymw"
    "github.com/vortex-fintech/go-lib/data/idempotency"
)

server := grpc.NewServer(
    grpc.UnaryInterceptor(idempotencymw.Unary(idempotencymw.Config{})),
)
```

## In service handler

```go
func (s *Service) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
    meta, ok := idempotencymw.FromContext(ctx)
    if !ok {
        return nil, status.Error(codes.Internal, "idempotency metadata missing")
    }

    begin, err := idempotency.Begin(ctx, s.store, s.db, idempotency.BeginInput{
        Principal:      meta.Principal,
        GRPCMethod:     meta.GRPCMethod,
        IdempotencyKey: meta.IdempotencyKey,
        RequestHash:    meta.RequestHash,
        ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
    })
    if err != nil {
        return nil, err
    }

    switch begin.Decision {
    case idempotency.BeginDecisionReplay:
        return decodeResponse(begin.Existing.ResponsePayload)
    case idempotency.BeginDecisionInProgress:
        return nil, status.Error(codes.Aborted, "request in progress")
    case idempotency.BeginDecisionRetryable:
        return nil, status.Error(codes.Unavailable, "retry later")
    case idempotency.BeginDecisionExecute:
        resp, err := s.createOrder(ctx, req)
        if err != nil {
            return nil, err
        }
        done := idempotency.Completion{
            Status:          idempotency.StatusSucceeded,
            ResponsePayload: encodeResponse(resp),
        }
        idempotency.Finish(ctx, s.store, s.db, *begin.Lease, done)
        return resp, nil
    }
    return nil, status.Error(codes.Internal, "unknown decision")
}
```

## Config

| Field | Default | Description |
|-------|---------|-------------|
| `RequireKey` | false | Return error if key missing |
| `Header` | `idempotency-key` | Header name for key |
| `MaxKeyLength` | 128 | Maximum key length |
| `IsMethodEnabled` | all enabled | Filter which methods use idempotency |
| `ResolvePrincipal` | "unknown" | Extract user/tenant from context |

## Metadata struct

```go
type Metadata struct {
    Principal      string  // User/tenant identifier
    GRPCMethod     string  // Full gRPC method (e.g., "/svc/CreateOrder")
    IdempotencyKey string  // Client-provided key
    RequestHash    string  // SHA-256 of request payload
}
```

## ResolvePrincipal example

```go
idempotencymw.Unary(idempotencymw.Config{
    ResolvePrincipal: func(ctx context.Context, md metadata.MD) string {
        if authzCtx, ok := authz.FromContext(ctx); ok {
            return authzCtx.UserID
        }
        return "anonymous"
    },
})
```

## Method filtering

```go
idempotencymw.Unary(idempotencymw.Config{
    IsMethodEnabled: func(method string) bool {
        mutating := []string{"/svc/Create", "/svc/Update", "/svc/Delete"}
        for _, m := range mutating {
            if method == m {
                return true
            }
        }
        return false
    },
})
```

## Limitations

- **Unary only** - streaming RPCs not supported (no single request to hash)
- Request must be a protobuf message

## Production notes

- Place before business logic middleware
- Use with `data/idempotency` package for full idempotency
- Key should be client-generated UUID
- Same key + same request = same hash (safe to retry)
- Same key + different request = rejected by `idempotency.Begin` (hash mismatch)
