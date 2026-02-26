# validator

Request validation wrapper with custom fintech-friendly rules.

## Functions

- `Instance() *validator.Validate` - singleton validator instance
- `Validate(input any) map[string]string` - validate struct, returns field->error map

## Behavior

- returns `nil` when valid
- returns `field -> error_code` map when invalid
- nested fields use dotted paths (e.g., `User.Email`)
- unknown error types return `{"_error": "validation_failed"}`

## Custom Validators

### `ascii_email`

Validates email with ASCII-only characters. Rejects Unicode to prevent:
- Homograph attacks (Cyrillic 'а' vs Latin 'a')
- Compatibility issues with non-SMTPUTF8 systems

```go
type Request struct {
    Email string `validate:"required,ascii_email"`
}
```

## Error Code Mapping

| Tag | Error Code |
|-----|------------|
| `required` | `required` |
| `email` | `invalid_email` |
| `ascii_email` | `invalid_email` |
| `e164` | `invalid_phone` |
| `min` | `too_short` |
| `max` | `too_long` |
| `len` | `invalid_length` |
| `gt` / `lt` / `gte` / `lte` | comparison codes |
| `oneof` | `invalid_choice` |
| ... | ... |

## Example

### Basic Validation

```go
package handler

import (
    "github.com/vortex-fintech/go-lib/foundation/validator"
)

type RegisterRequest struct {
    Email    string `validate:"required,ascii_email"`
    Password string `validate:"required,min=8,max=64"`
    Age      int    `validate:"min=18,max=120"`
}

func HandleRegister(req *RegisterRequest) error {
    if errs := validator.Validate(req); errs != nil {
        return errs // {"Email": "invalid_email", "Password": "too_short"}
    }
    return nil
}
```

### Nested Struct Validation

```go
type OrderRequest struct {
    Customer struct {
        Email string `validate:"required,ascii_email"`
        Phone string `validate:"required,e164"`
    } `validate:"required"`
    Items []struct {
        SKU    string `validate:"required"`
        Amount int    `validate:"min=1,max=10000"`
    } `validate:"required,dive"`
}

func HandleOrder(req *OrderRequest) error {
    if errs := validator.Validate(req); errs != nil {
        return errs // {"Customer.Email": "required", "Items[0].Amount": "too_small"}
    }
    return nil
}
```

### Payment Request

```go
type PaymentRequest struct {
    FromEmail string `validate:"required,ascii_email"`
    ToEmail   string `validate:"required,ascii_email"`
    Amount    int64  `validate:"required,min=1,max=100000000"`
    Currency  string `validate:"required,len=3"`
}

func (s *Service) ProcessPayment(req *PaymentRequest) error {
    if errs := validator.Validate(req); errs != nil {
        return errs
    }
    
    // Process payment...
    return nil
}
```

### ASCII Email Protection

```go
type KYCRequest struct {
    // ASCII-only email prevents:
    // - Homograph attacks: user@exаmple.com (Cyrillic 'а')
    // - Unicode confusion: usuário@exemplo.com
    Email string `validate:"required,ascii_email"`
}

func (s *Service) SubmitKYC(req *KYCRequest) error {
    if errs := validator.Validate(req); errs != nil {
        return errs
    }
    // Email is guaranteed ASCII-only
    // ...
}
```

## Adding Custom Validators

```go
func init() {
    v := validator.Instance()
    v.RegisterValidation("custom_rule", func(fl validator.FieldLevel) bool {
        // Custom validation logic
        return true
    })
}
```

## Business Examples

- **User registration**: Validate email format, password strength, age limits
- **Payment processing**: Validate amounts, currency codes, account identifiers
- **KYC onboarding**: ASCII-only email prevents homograph attacks
- **API requests**: Nested struct validation for complex payloads

## Testing

```go
func TestValidation(t *testing.T) {
    req := &RegisterRequest{
        Email:    "юзер@example.com", // Cyrillic
        Password: "123",
        Age:      15,
    }
    
    errs := validator.Validate(req)
    // errs = {
    //   "Email": "invalid_email",
    //   "Password": "too_short", 
    //   "Age": "too_short"
    // }
}
```
