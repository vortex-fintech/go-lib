# logutil

Helpers for log-safe output.

## Functions

- `SanitizeValidationErrors(fields, env, replacement, sensitiveKeys...)`
- `SanitizeValidationErrorsStrict(fields, replacement, sensitiveKeys...)`

## Behavior

- `development`/`debug`: values are not redacted
- other environments: sensitive field names are redacted
- input map is never mutated
- returned map is a copy (no aliasing with input)
- strict mode always redacts regardless of environment

Default sensitive matcher targets field tokens such as:

- `password`, `pass`, `secret`, `token`, `otp`

Also includes common fintech tokens:

- `pin`, `cvv`, `cvc`, `pan`, `iban`, `account`, `routing`, `swift`

## Example

```go
package main

import (
    "github.com/vortex-fintech/go-lib/foundation/logutil"
    "github.com/vortex-fintech/go-lib/foundation/logger"
)

func main() {
    log, _ := logger.New("kyc-service", "production")
    
    // Validation errors from request
    errors := map[string]string{
        "pin":           "must be 4 digits",
        "cvv":           "required",
        "account_number": "invalid format",
        "email":         "invalid email format",
        "name":          "required",
    }
    
    // Strict mode - always redact sensitive fields
    sanitized := logutil.SanitizeValidationErrorsStrict(
        errors,
        "[REDACTED]",
        "session", // optional custom sensitive key
    )
    
    // Result:
    // {
    //   "pin": "[REDACTED]",
    //   "cvv": "[REDACTED]",
    //   "account_number": "[REDACTED]",
    //   "email": "invalid email format",
    //   "name": "required"
    // }
    
    log.Warnw("validation failed", "errors", sanitized)
}
```

### Environment-Aware Mode

```go
// For local debugging tools only
sanitized := logutil.SanitizeValidationErrors(
    errors,
    "development", // no redaction in dev
    "[REDACTED]",
)
// Result: original values preserved for debugging
```

### Custom Sensitive Keys

```go
sanitized := logutil.SanitizeValidationErrorsStrict(
    errors,
    "[REDACTED]",
    "device_fingerprint", // custom keys
    "session_id",
    "api_key",
)
```

## Recommended Implementation (Fintech)

For production and audit log paths, prefer strict mode.

```go
sanitized := logutil.SanitizeValidationErrorsStrict(
    validationErrors,
    "[REDACTED]",
    "session", "device", // optional service-specific sensitive keys
)

logger.Warnw("validation failed",
    "errors", sanitized,
    "request_id", reqID,
)
```

Use environment-aware mode only for local debugging utilities.

## Tokenization Examples

The tokenizer splits camelCase, snake_case, and dot-notation:

| Input Field | Tokens | Matched |
|-------------|--------|---------|
| `NewPassword` | `new`, `password` | ✅ (password) |
| `user_pin` | `user`, `pin` | ✅ (pin) |
| `card.cvv` | `card`, `cvv` | ✅ (cvv) |
| `beneficiarySwift` | `beneficiary`, `swift` | ✅ (swift) |
| `compassion` | `compassion` | ❌ (no match) |
| `passenger` | `passenger` | ❌ (no match) |
