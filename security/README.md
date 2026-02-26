# security

Security utilities for Vortex services: JWT verification, mTLS, HMAC, replay protection, and scope helpers.

## Packages

| Package | Description |
|---------|-------------|
| [jwt](./jwt) | JWKS-based JWT verification with OBO validation |
| [mtls](./mtls) | Mutual TLS configuration with hot reload |
| [hmac](./hmac) | HMAC-SHA256 computation and verification |
| [replay](./replay) | Anti-replay protection for JTI tracking |
| [scope](./scope) | OAuth/JWT scope evaluation utilities |

## Quick Start

```go
import (
    "github.com/vortex-fintech/go-lib/security/jwt"
    "github.com/vortex-fintech/go-lib/security/mtls"
    "github.com/vortex-fintech/go-lib/security/replay"
    "github.com/vortex-fintech/go-lib/security/scope"
)

func main() {
    verifier, _ := jwt.NewJWKSVerifier(jwt.JWKSConfig{
        URL:            "https://sso.internal/.well-known/jwks.json",
        ExpectedIssuer: "https://sso.internal",
    })

    claims, err := verifier.Verify(ctx, tokenString)
    if err != nil {
        panic(err)
    }

    if !scope.HasAll(claims.EffectiveScopes(), "wallet:read", "payments:create") {
        panic("insufficient permissions")
    }
}
```

## By Category

### Authentication

| Package | Purpose |
|---------|---------|
| [jwt](./jwt) | JWT verification, OBO token validation, mTLS binding |
| [mtls](./mtls) | Service-to-service authentication, certificate rotation |

### Integrity & Signatures

| Package | Purpose |
|---------|---------|
| [hmac](./hmac) | Webhook signatures, API request signing, one-time tokens |

### Protection

| Package | Purpose |
|---------|---------|
| [replay](./replay) | Token replay prevention, JTI deduplication |

### Authorization

| Package | Purpose |
|---------|---------|
| [scope](./scope) | Scope checking, permission validation |

## Common Patterns

### JWT Verification with OBO

```go
verifier, _ := jwt.NewJWKSVerifier(jwt.JWKSConfig{
    URL:            "https://sso.internal/.well-known/jwks.json",
    ExpectedIssuer: "https://sso.internal",
    RefreshEvery:   5 * time.Minute,
})

claims, err := verifier.Verify(ctx, tokenString)
if err != nil {
    return err
}

opt := jwt.OBOValidateOptions{
    WantAudience:   "wallet",
    WantActor:      "api-gateway",
    AllowedAZP:     []string{"vortex-web", "mobile-app"},
    MaxTTL:         time.Hour,
    MTLSThumbprint: clientCertThumbprint,
    SeenJTI:        replayChecker.AsAuthzCallback("wallet", time.Hour),
}

if err := jwt.ValidateOBO(time.Now(), claims, opt); err != nil {
    return err
}
```

### mTLS Server

```go
cfg := mtls.Config{
    CACertPath:     "/certs/ca.pem",
    CertPath:       "/certs/server.pem",
    KeyPath:        "/certs/server-key.pem",
    ReloadInterval: time.Minute,
}

tlsConfig, reloader, _ := mtls.TLSConfigServer(cfg)
defer reloader.Stop()

server := &http.Server{
    Addr:      ":8443",
    TLSConfig: tlsConfig,
}
server.ListenAndServeTLS("", "")
```

### mTLS Client

```go
cfg := mtls.Config{
    CACertPath: "/certs/ca.pem",
    CertPath:   "/certs/client.pem",
    KeyPath:    "/certs/client-key.pem",
    ServerName: "api.internal",
}

tlsConfig, reloader, _ := mtls.TLSConfigClient(cfg)
defer reloader.Stop()

client := &http.Client{
    Transport: &http.Transport{TLSClientConfig: tlsConfig},
}
```

### Webhook Signature Verification

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("X-Signature")

    ok, err := hmac.Verify(string(body), webhookSecret, signature)
    if err != nil || !ok {
        http.Error(w, "invalid signature", 401)
        return
    }

    // Process trusted webhook
}
```

### Replay Protection with Redis

```go
rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
checker := replay.NewRedisChecker(rdb, replay.RedisOptions{
    Prefix:   "obo:jti",
    FailOpen: false,
})

seen, err := checker.SeenJTI(ctx, "wallet-service", jti, time.Hour)
if err != nil {
    return err
}
if seen {
    return errors.New("replay detected")
}
```

### Scope Middleware

```go
func RequireScopes(need ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := ClaimsFromContext(r.Context())
            if !scope.HasAll(claims.EffectiveScopes(), need...) {
                http.Error(w, "forbidden", 403)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Testing

```bash
# Unit tests
go test ./...

# With race detector (requires CGO + C compiler)
go test -race ./...

# With race detector in Docker (Linux/macOS/WSL)
docker run --rm -v "${PWD}:/src" -w /src golang:1.25 go test -race ./...

# With race detector in Docker (Git Bash on Windows)
MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL="*" docker run --rm -v "$(pwd -W):/src" -w /src golang:1.25 go test -race ./...

# Single package
go test ./jwt
go test ./mtls
go test ./hmac
go test ./replay
go test ./scope
```
