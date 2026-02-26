# JWT Verifier

JWKS-based JWT verification with OBO (On-Behalf-Of) token validation.

## Where to use it

- Verify internal OBO tokens from SSO
- Validate API tokens with JWKS endpoint
- Enforce mTLS binding, scopes, and replay protection

## Basic usage

```go
verifier, err := jwt.NewJWKSVerifier(jwt.JWKSConfig{
    URL:            "https://sso.internal/.well-known/jwks.json",
    ExpectedIssuer: "https://sso.internal",
    RefreshEvery:   5 * time.Minute,
})
if err != nil {
    log.Fatal(err)
}

claims, err := verifier.Verify(ctx, tokenString)
if err != nil {
    return err
}
```

## OBO token validation

```go
opt := jwt.OBOValidateOptions{
    WantAudience:    "wallet",
    WantActor:       "api-gateway",
    AllowedAZP:      []string{"vortex-web", "mobile-app"},
    Leeway:          5 * time.Second,
    MaxTTL:          time.Hour,
    MTLSThumbprint:  clientCertThumbprint,
    SeenJTI:         replayChecker.AsAuthzCallback("wallet", time.Hour),
    RequireScopes:   true,
}

if err := jwt.ValidateOBO(time.Now(), claims, opt); err != nil {
    return err
}
```

## Require scopes

```go
if err := jwt.RequireScopes(now, claims, opt, "wallet:read", "payments:create"); err != nil {
    return err
}
```

## Require wallet access

```go
if err := jwt.RequireWallet(now, claims, opt, walletID, "wallet:read"); err != nil {
    return err
}
```

## Claims structure

```go
type Claims struct {
    Issuer   string   `json:"iss"`
    Subject  string   `json:"sub"`
    Audience []string `json:"aud"`
    Iat      int64    `json:"iat"`
    Exp      int64    `json:"exp"`
    Sid      string   `json:"sid,omitempty"`
    Jti      string   `json:"jti,omitempty"`
    Scopes   []string `json:"scopes,omitempty"`
    Azp      string   `json:"azp,omitempty"`
    Act      *Actor   `json:"act,omitempty"`
    Cnf      *Cnf     `json:"cnf,omitempty"`
    SrcTH    string   `json:"src_th,omitempty"`
    ACR      string   `json:"acr,omitempty"`
    AMR      []string `json:"amr,omitempty"`
    WalletID string   `json:"wallet_id,omitempty"`
    DeviceID string   `json:"device_id,omitempty"`
}

func (c Claims) ExpiresAt() time.Time
func (c Claims) EffectiveScopes() []string  // sorted copy
func (c Claims) HasScopes(required ...string) bool
```

## JWKSConfig options

| Option | Default | Description |
|--------|---------|-------------|
| `URL` | required | JWKS endpoint URL |
| `ExpectedIssuer` | none | Validate `iss` claim |
| `RefreshEvery` | 5m | Max refresh interval |
| `Timeout` | 5s | HTTP timeout for JWKS requests |
| `Leeway` | 5s | Time leeway for exp/iat checks |

## Supported algorithms

- RS256 (RSA PKCS#1 v1.5)
- PS256 (RSA PSS)

## Validation errors

| Error | Condition |
|-------|-----------|
| `ErrNilClaims` | Claims pointer is nil |
| `ErrBadSubject` | Subject is not a valid UUID |
| `ErrAudMismatch` | Audience doesn't match expected |
| `ErrMissingActor` | Actor claim is missing |
| `ErrActorMismatch` | Actor doesn't match expected |
| `ErrAZPMismatch` | AZP not in allowed list |
| `ErrExpired` | Token has expired |
| `ErrIATInFuture` | Issued-at time is in the future |
| `ErrTTLTooLong` | Token lifetime exceeds MaxTTL |
| `ErrMissingJTI` | JTI claim is missing |
| `ErrReplay` | JTI already seen (replay attack) |
| `ErrMTLSBindingMismatch` | Certificate thumbprint doesn't match |
| `ErrMissingScopes` | Required scopes not present |
| `ErrWalletMismatch` | Wallet ID doesn't match |

## JWKS caching

- Keys cached in memory with automatic refresh
- Respects `Cache-Control: max-age` header
- Falls back to `RefreshEvery` if no header
- Supports `ETag` / `If-None-Match` for efficient revalidation
- Unknown `kid` triggers immediate refresh
- Malformed JWK entries are skipped (valid keys are still usable)
- Existing key cache is kept if refresh response has no valid RSA keys

## mTLS binding

```go
// Get thumbprint from client certificate
tlsConfig, _, _ := mtls.TLSConfigClient(mtls.Config{...})
// After TLS handshake, extract certificate and compute thumbprint
// (typically done in middleware or connection handler)
thumbprint := jwt.X5tS256FromCert(peerCert)
opt.MTLSThumbprint = thumbprint

if err := jwt.ValidateOBO(now, claims, opt); err != nil {
    if errors.Is(err, jwt.ErrMTLSBindingMismatch) {
        // Token was not issued for this client certificate
    }
}
```

## Production notes

- Set `MaxTTL` to limit token lifetime (e.g., 1 hour)
- Always validate `aud` matches your service
- Use `SeenJTI` callback with Redis for distributed replay protection
- Enable mTLS binding for high-security services
- Keep `Leeway` small (5s recommended)

## Utility functions

```go
thumbprint := jwt.X5tS256FromCert(cert)
```

If `cert == nil`, `X5tS256FromCert` returns an empty string.
