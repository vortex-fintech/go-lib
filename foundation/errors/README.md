# errors

Transport-agnostic error model with HTTP and gRPC adapters.

Use this package when you want one stable error contract across domain, HTTP, and gRPC.

## Error Contract

`ErrorResponse` contains:

- `code` - gRPC status code
- `reason` - machine-readable error reason
- `message` - human-readable message
- `domain` - service identifier (e.g., "payment-service")
- `details` - additional context (map)
- `violations` - field-level validation errors

## Quick Reference

| Preset | gRPC Code | HTTP Status | Use Case |
|--------|-----------|-------------|----------|
| `BadRequest` | InvalidArgument | 400 | Validation errors |
| `Unauthorized` | Unauthenticated | 401 | Missing auth |
| `Forbidden` | PermissionDenied | 403 | Insufficient permissions |
| `NotFound` | NotFound | 404 | Resource not found |
| `Conflict` | AlreadyExists | 409 | Duplicate resource |
| `RateLimited` | ResourceExhausted | 429 | Quota exceeded |
| `Internal` | Internal | 500 | Unexpected error |
| `Unavailable` | Unavailable | 503 | Service down |

## Example

### Service Layer

```go
package payment

import (
    "context"
    
    ferrors "github.com/vortex-fintech/go-lib/foundation/errors"
)

type Service struct {
    repo Repository
}

func (s *Service) CreatePayment(ctx context.Context, req *CreatePaymentRequest) error {
    // Business validation
    if req.Amount <= 0 {
        return ferrors.ValidationFields(map[string]string{
            "amount": "must be positive",
        })
    }
    
    // Check account exists
    account, err := s.repo.GetAccount(ctx, req.AccountID)
    if err != nil {
        return ferrors.NotFoundID("account", req.AccountID)
    }
    
    // Check KYC status
    if !account.KYCCompleted {
        return ferrors.Precondition("kyc_required", map[string]string{
            "account_id": req.AccountID,
            "status":     account.Status,
        })
    }
    
    // Check for duplicate
    existing, _ := s.repo.GetPaymentByReference(ctx, req.Reference)
    if existing != nil {
        return ferrors.Conflict("reference", req.Reference)
    }
    
    // Process payment...
    return nil
}
```

### HTTP Handler

```go
package api

import (
    "net/http"
    
    ferrors "github.com/vortex-fintech/go-lib/foundation/errors"
)

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
    var req CreatePaymentRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        ferrors.BadRequest("invalid_json").ToHTTP(w)
        return
    }
    
    if err := h.service.CreatePayment(r.Context(), &req); err != nil {
        ferrors.ToErrorResponse(err).ToHTTP(w)
        return
    }
    
    w.WriteHeader(http.StatusCreated)
}
```

### gRPC Service

```go
package grpc

import (
    "context"
    
    ferrors "github.com/vortex-fintech/go-lib/foundation/errors"
)

func (s *Server) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.Payment, error) {
    if err := s.service.CreatePayment(ctx, fromProto(req)); err != nil {
        return nil, ferrors.ToErrorResponse(err).ToGRPC()
    }
    
    return &pb.Payment{Id: "123"}, nil
}
```

### With Validation

```go
import (
    "github.com/vortex-fintech/go-lib/foundation/errors"
    "github.com/vortex-fintech/go-lib/foundation/validator"
)

func (s *Service) RegisterUser(ctx context.Context, req *RegisterRequest) error {
    // Validate request
    if errs := validator.Validate(req); errs != nil {
        return errors.ValidationFields(errs)
    }
    
    // Business logic...
    return nil
}
```

### Rate Limiting

```go
func (s *Service) CallPartner(ctx context.Context) error {
    if s.rateLimiter.IsExceeded() {
        return ferrors.RateLimited(2 * time.Second)
    }
    
    // Call partner...
    return nil
}
```

### Domain Invariants

```go
import "github.com/vortex-fintech/go-lib/foundation/errors"

func (a *Account) Withdraw(amount int64) error {
    // State invariant
    if a.Status != "active" {
        return errors.StateInvariant(
            nil, 
            "status",
            "account_must_be_active",
        )
    }
    
    // Domain invariant
    if amount > a.Balance {
        return errors.DomainInvariant("amount", "insufficient_funds")
    }
    
    // Transition invariant
    if a.PendingWithdrawal {
        return errors.TransitionInvariant(
            nil,
            "withdrawal",
            "concurrent_withdrawal_not_allowed",
        )
    }
    
    return nil
}
```

### Error Adaptation

```go
func (h *Handler) HandleError(w http.ResponseWriter, err error) {
    // Convert any error to ErrorResponse
    resp := ferrors.ToErrorResponse(err)
    
    // Log with context
    log.Errorw("request failed",
        "code", resp.Code.String(),
        "reason", resp.Reason,
        "message", resp.Message,
    )
    
    // Send to client
    resp.ToHTTP(w)
}
```

## Business Examples

### Payment Flow

```go
// 1. Validation
if amount <= 0 {
    return ferrors.ValidationFields(map[string]string{
        "amount": "must_be_positive",
    })
}

// 2. Not found
account, err := repo.GetAccount(ctx, accountID)
if err != nil {
    return ferrors.NotFoundID("account", accountID)
}

// 3. Business rule
if !account.KYCCompleted {
    return ferrors.Precondition("kyc_required", map[string]string{
        "account_id": accountID,
    })
}

// 4. Conflict
if exists := repo.GetByReference(ctx, ref); exists != nil {
    return ferrors.Conflict("reference", ref)
}

// 5. Rate limit
if limiter.Exceeded() {
    return ferrors.RateLimited(5 * time.Second)
}

// 6. Success
return nil
```

### Auth Flow

```go
// Missing token
if token == "" {
    return ferrors.Unauthorized("bearer", "api")
}

// Invalid token
if !ValidateToken(token) {
    return ferrors.Unauthenticated().WithReason("invalid_token")
}

// Insufficient permissions
if !user.HasPermission("payment:write") {
    return ferrors.Forbidden("payment:write", "payments")
}
```

## HTTP Mapping

| gRPC Code | HTTP Status |
|-----------|-------------|
| Canceled | 499 |
| InvalidArgument | 400 |
| DeadlineExceeded | 504 |
| NotFound | 404 |
| AlreadyExists | 409 |
| PermissionDenied | 403 |
| ResourceExhausted | 429 |
| FailedPrecondition | 412 |
| Aborted | 409 |
| OutOfRange | 400 |
| Unimplemented | 501 |
| Internal | 500 |
| Unavailable | 503 |
| DataLoss | 500 |
| Unauthenticated | 401 |

## Best Practices

1. **Use presets** - Don't create ErrorResponse directly
2. **Add reason** - Always set machine-readable reason
3. **Use details** - Add context, never PII
4. **Use violations** - For field-level validation errors
5. **Set domain** - For cross-service analytics

## Testing

```bash
go test ./foundation/errors/... -cover
go vet ./foundation/errors/...
```
