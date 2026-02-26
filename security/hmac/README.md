# HMAC-SHA256

HMAC-SHA256 computation and verification with constant-time comparison.

## Where to use it

- Webhook signature verification (Stripe, Telegram, GitHub style)
- API request signing
- One-time tokens (password reset, email verification)
- Message authentication between services

## Basic usage

```go
import "github.com/vortex-fintech/go-lib/security/hmac"

mac := hmac.Compute(payload, secret)
```

## Verify signatures

```go
ok, err := hmac.Verify(receivedPayload, secret, receivedSignature)
if err != nil {
    return err
}
if !ok {
    return errors.New("invalid signature")
}
```

## Business examples

### Webhook verification

```go
func handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("Stripe-Signature")

    ok, err := hmac.Verify(string(body), stripeSecret, signature)
    if err != nil || !ok {
        http.Error(w, "invalid signature", 401)
        return
    }

    // Process trusted webhook
}
```

### Password reset token

```go
func generateResetToken(userID string) string {
    timestamp := time.Now().Unix()
    data := fmt.Sprintf("%s:%d", userID, timestamp)
    return hmac.Compute(data, resetSecret)
}

func verifyResetToken(userID, token string, maxAge time.Duration) bool {
    // Recompute and verify with timing-safe comparison
    data := fmt.Sprintf("%s:%d", userID, timestamp)
    ok, _ := hmac.Verify(data, resetSecret, token)
    return ok
}
```

### API request signing

```go
func signRequest(method, path, body string, secret []byte) string {
    payload := method + "\n" + path + "\n" + body
    return hmac.Compute(payload, secret)
}

func verifyRequest(r *http.Request, secret []byte) bool {
    body, _ := io.ReadAll(r.Body)
    expected := r.Header.Get("X-Signature")
    payload := r.Method + "\n" + r.URL.Path + "\n" + string(body)
    ok, _ := hmac.Verify(payload, secret, expected)
    return ok
}
```

## API reference

### `Compute(message string, secret []byte) string`

Returns hex-encoded HMAC-SHA256 of message with secret.

### `Verify(message string, secret []byte, expectedMAC string) (bool, error)`

Constant-time verification of HMAC signature.

| Error | Condition |
|-------|-----------|
| `ErrEmptySecret` | `secret` is empty |
| `ErrEmptyMAC` | `expectedMAC` is empty |
| `hex.InvalidByteError` | `expectedMAC` is not valid hex |

## Security notes

- `Verify` uses `hmac.Equal` for constant-time comparison (prevents timing attacks)
- Never use `==` to compare HMACs: `if computed == received` is vulnerable
- Keep secrets at least 32 bytes (256 bits) for SHA256
- Rotate secrets periodically

## Production notes

- Store secrets in secure storage (Vault, AWS Secrets Manager, etc.)
- Never log secrets or computed MACs
- Use different secrets for different purposes (webhooks, tokens, API)
- Consider adding timestamp to prevent replay attacks
