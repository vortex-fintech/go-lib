# gRPC Metadata

Utilities for working with gRPC metadata (HTTP/2 headers).

## Where to use it

- Set authentication headers on outgoing gRPC calls
- Read headers from incoming requests
- Pass tracing/auth context between services

## Basic usage

### Client side (outgoing)

```go
import "github.com/vortex-fintech/go-lib/transport/grpc/metadata"

func callService(ctx context.Context, token, thumbprint string) {
    ctx = metadata.WithBearer(ctx, token)
    ctx = metadata.WithPoP(ctx, thumbprint)
    ctx = metadata.WithAZP(ctx, "mobile-app")

    resp, err := client.SomeMethod(ctx, req)
    // ...
}
```

### Server side (incoming)

```go
func (s *server) SomeMethod(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    token := metadata.Get(ctx, metadata.HeaderAuthorization)
    if token == "" {
        return nil, status.Error(codes.Unauthenticated, "missing token")
    }

    thumbprint := metadata.Get(ctx, metadata.HeaderPoP)
    // ...
}
```

## Standard headers

| Constant | Header | Description |
|----------|--------|-------------|
| `HeaderAuthorization` | `authorization` | Bearer token |
| `HeaderPoP` | `x-pop` | mTLS proof-of-possession (x5t#S256) |
| `HeaderAZP` | `x-azp` | Authorized party (client source) |

## Functions

### WithBearer

Adds `Authorization: Bearer <token>` to outgoing metadata.

```go
ctx = metadata.WithBearer(ctx, "my-token")
// or with existing prefix - normalized
ctx = metadata.WithBearer(ctx, "Bearer my-token")
```

Empty or whitespace-only tokens are ignored (no metadata added).

### WithPoP

Adds `X-PoP` header for mTLS proof-of-possession.

```go
ctx = metadata.WithPoP(ctx, "x5t#S256-thumbprint")
```

### WithAZP

Adds `X-AZP` header identifying the authorized party.

```go
ctx = metadata.WithAZP(ctx, "mobile-app")
```

### Get

Returns first value for a key from incoming or outgoing metadata.

```go
token := metadata.Get(ctx, "authorization")
```

Priority: incoming > outgoing. Returns empty string if not found.

### GetAll

Returns all values for a key (for multi-value headers).

```go
values := metadata.GetAll(ctx, "x-custom")
```

Returns `nil` if not found.

## Chaining

All `With*` functions can be chained:

```go
ctx = metadata.WithBearer(ctx, token)
ctx = metadata.WithPoP(ctx, thumbprint)
ctx = metadata.WithAZP(ctx, "web-app")
```

Or in one line:

```go
ctx = metadata.WithAZP(
    metadata.WithPoP(
        metadata.WithBearer(ctx, token),
        thumbprint,
    ),
    "web-app",
)
```

## Production notes

- Empty values are silently ignored (no error, no metadata)
- Keys are lowercased per gRPC/HTTP2 spec
- `Get` returns first value only - use `GetAll` for multi-value headers
- Existing metadata is preserved when adding new values
