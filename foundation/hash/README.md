# hash

Deterministic hashing helpers.

## Functions

- `HashStringsCanonical(parts ...string) string`
  - SHA-256 over canonical length-prefixed tuple encoding
- `HMACStringsCanonical(key, parts ...string) string`
  - keyed HMAC-SHA256 over canonical length-prefixed tuple encoding

## Security Notes

For fintech write paths and idempotency keys:

- use `HashStringsCanonical` for untrusted/raw inputs
- use `HMACStringsCanonical` for low-entropy or sensitive identifiers

## Recommended Implementation (Fintech)

Use explicit field order and canonical hashing for idempotency/outbox keys.

```go
// Good: deterministic, unambiguous tuple hashing.
idempotencyKey := hash.HashStringsCanonical(
	"payment.create.v1", // operation namespace + version
	tenantID,
	merchantID,
	externalRequestID,
	amountMinor,
	currency,
)
```

For low-entropy or sensitive values, use keyed HMAC with a secret from Vault/KMS.

```go
// Good: keyed hash protects against offline dictionary attacks.
fingerprint := hash.HMACStringsCanonical(
	serviceSecretKey,
	"customer.lookup.v1",
	normalizedEmail,
	normalizedPhone,
)
```

Do:

- keep field order stable and versioned (`*.v1`, `*.v2`)
- normalize input before hashing (trim/case rules in caller layer)
- store only hash output, not raw sensitive values

Avoid:

- mixing optional fields without explicit placeholders/versioning
