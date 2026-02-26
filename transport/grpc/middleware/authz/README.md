# gRPC Authorization Interceptor

OBO (On-Behalf-Of) JWT authorization with mTLS proof-of-possession and scope-based access control.

## Where to use it

- gRPC services requiring JWT authentication
- Internal service-to-service communication with OBO tokens
- APIs with scope-based authorization
- High-security services with mTLS binding

## Architecture

```
Request → Bearer Token → JWT Verification → OBO Validation → Scope Check → Handler
                                    ↓
                            mTLS PoP (x5t#S256)
```

## Basic usage

```go
verifier, _ := jwt.NewJWKSVerifier(jwt.JWKSConfig{
    URL:            "https://sso.internal/.well-known/jwks.json",
    ExpectedIssuer: "https://sso.internal",
})

cfg := authz.Config{
    Verifier:   verifier,
    Audience:   "wallet",
    Actor:      "api-gateway",
    RequirePoP: true,
}

if err := authz.ValidateConfig(cfg); err != nil {
    log.Fatalf("invalid authz config: %v", err)
}

authInterceptor := authz.UnaryServerInterceptor(cfg)

server := grpc.NewServer(
    grpc.UnaryInterceptor(authInterceptor),
)
```

`UnaryServerInterceptor` и `StreamServerInterceptor` не паникуют при невалидной конфигурации и возвращают `codes.Internal`.

## Config options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `Verifier` | Yes | - | JWT verifier (JWKS-based) |
| `Audience` | Yes | - | This service's audience (e.g., "wallet") |
| `Actor` | No | - | Expected actor (e.g., "api-gateway") |
| `AllowedAZP` | No | - | Allowed authorized parties |
| `Leeway` | No | 45s | Time leeway for exp/iat checks |
| `MaxTTL` | No | 5m | Maximum token lifetime |
| `RequireScopes` | No | false | Require non-empty scopes |
| `RequirePoP` | No | false | Require mTLS proof-of-possession |
| `MTLSThumbprint` | No | auto | Function to extract x5t#S256 from peer |
| `SeenJTI` | No | - | Anti-replay callback |
| `RequiredScopes` | No | - | Global scope requirements |
| `ResolvePolicy` | No | - | Per-method policy resolver |
| `SkipAuth` | No | - | Skip authentication for specific methods |

## Policy-based authorization

```go
authInterceptor := authz.UnaryServerInterceptor(authz.Config{
    Verifier:  verifier,
    Audience:  "wallet",
    RequirePoP: true,
    ResolvePolicy: authz.MapResolver(map[string]authz.Policy{
        "/wallet.Wallet/GetBalance": {Any: []string{"wallet:read", "wallet:admin"}},
        "/wallet.Wallet/Transfer":   {All: []string{"wallet:write", "payments:create"}},
        "/admin.Admin/DeleteUser":   {All: []string{"admin:write"}},
    }),
})
```

**Policy rules:**
- `All`: User must have ALL listed scopes
- `Any`: User must have at least ONE of the listed scopes

## Skip authentication

```go
authInterceptor := authz.UnaryServerInterceptor(authz.Config{
    Verifier:  verifier,
    Audience:  "wallet",
    RequirePoP: false,
    SkipAuth: authz.PrefixSkipAuth(
        "/grpc.health.v1.Health/",
        "/grpc.reflection.v1.",
    ),
})
```

Or with explicit methods:

```go
SkipAuth: authz.SliceSkipAuth(
    "/svc.Public/Method1",
    "/svc.Public/Method2",
),
```

## Anti-replay protection

```go
checker := replay.NewRedisChecker(rdb, replay.RedisOptions{
    Prefix: "obo:jti",
})

authInterceptor := authz.UnaryServerInterceptor(authz.Config{
    Verifier:  verifier,
    Audience:  "wallet",
    SeenJTI:   checker.AsAuthzCallback("wallet", 5*time.Minute),
})
```

## Accessing identity in handlers

```go
func (s *server) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.Balance, error) {
    id, err := authz.RequireIdentity(ctx)
    if err != nil {
        return nil, err
    }
    
    walletID, err := authz.RequireWalletID(ctx)
    if err != nil {
        return nil, err
    }
    
    return s.getBalance(ctx, id.UserID, walletID)
}
```

### Context helpers

| Function | Returns | Error |
|----------|---------|-------|
| `IdentityFrom(ctx)` | `Identity, bool` | - |
| `RequireIdentity(ctx)` | `Identity` | standardized domain error |
| `RequireUserID(ctx)` | `uuid.UUID` | standardized domain error |
| `ClaimsFrom(ctx)` | `*Claims, bool` | - |
| `RequireWalletID(ctx)` | `string` | `ErrWalletCtxMissing` |
| `RequireWalletMatch(ctx, want)` | - | `ErrWalletMismatch` |

### Identity struct

```go
type Identity struct {
    UserID   uuid.UUID
    Scopes   []string
    SID      string    // session ID
    DeviceID string
}
```

## Authorize function (reusable)

For HTTP middleware or custom use cases:

```go
result, err := authz.Authorize(ctx, "/svc.Method", authz.Config{
    Verifier:  verifier,
    Audience:  "wallet",
    RequirePoP: true,
})
if err != nil {
    return err
}
if result != nil {
    userID := result.Identity.UserID
    scopes := result.Identity.Scopes
    walletID := result.Claims.WalletID
}
```

## Error codes

| Condition | gRPC Code |
|-----------|-----------|
| Missing/invalid metadata | `Unauthenticated` |
| Invalid/expired token | `Unauthenticated` |
| Missing mTLS certificate | `Unauthenticated` |
| Token expired / IAT in future | `Unauthenticated` |
| OBO validation failed | `PermissionDenied` |
| Insufficient scopes | `PermissionDenied` |

## Stream support

```go
streamInterceptor := authz.StreamServerInterceptor(authz.Config{
    Verifier:  verifier,
    Audience:  "wallet",
    RequirePoP: true,
})

server := grpc.NewServer(
    grpc.UnaryInterceptor(authInterceptor),
    grpc.StreamInterceptor(streamInterceptor),
)
```

## Production notes

- Always set `Audience` to your service name
- Enable `RequirePoP` for high-security services
- Use `SeenJTI` with Redis for distributed replay protection
- Keep `MaxTTL` ≤ 5 minutes
- Use `ResolvePolicy` for method-level authorization
- Monitor `Unauthenticated` vs `PermissionDenied` ratios for security insights
