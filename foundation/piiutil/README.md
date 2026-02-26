# piiutil

PII masking helpers for logs, diagnostics, and external responses.

## Functions

- `MaskEmail(email)`
- `MaskPhone(phone)`
- `MaskIDLast4(value)`

## Short-Value Safety

For phone/ID masking:

- if total digits `<= 4`, only the last **1** digit is kept
- if total digits `> 4`, the last **4** digits are kept

This prevents full disclosure of short values like OTP-like `1234`.

## Example

```go
package main

import (
    "github.com/vortex-fintech/go-lib/foundation/logger"
    "github.com/vortex-fintech/go-lib/foundation/piiutil"
)

func main() {
    log, _ := logger.New("payment-service", "production")
    
    user := struct {
        Email  string
        Phone  string
        TaxID  string
        SSN    string
    }{
        Email: "john.doe@example.com",
        Phone: "+1 (555) 123-4567",
        TaxID: "123-45-6789",
        SSN:   "123456789",
    }
    
    // Mask PII before logging
    log.Infow("user registered",
        "email", piiutil.MaskEmail(user.Email),
        "phone", piiutil.MaskPhone(user.Phone),
        "tax_id", piiutil.MaskIDLast4(user.TaxID),
        "ssn", piiutil.MaskIDLast4(user.SSN),
    )
    // Output:
    // email=j******e@example.com
    // phone=+* (***) ***-4567
    // tax_id=***-**-6789
    // ssn=*****6789
}
```

### HTTP Response Masking

```go
func (h *Handler) GetUserDebug(w http.ResponseWriter, r *http.Request) {
    user := getUserFromDB(r.Context())
    
    // Mask PII for debug/admin endpoints
    resp := map[string]string{
        "email":      piiutil.MaskEmail(user.Email),
        "phone":      piiutil.MaskPhone(user.Phone),
        "national_id": piiutil.MaskIDLast4(user.NationalID),
        "status":      user.Status, // non-PII, keep as-is
    }
    
    json.NewEncoder(w).Encode(resp)
}
```

### Payment Logging

```go
func (s *PaymentService) ProcessPayment(ctx context.Context, req *PaymentRequest) error {
    // Log with masked PII for audit trail
    s.log.Infow("payment initiated",
        "email", piiutil.MaskEmail(req.Email),
        "phone", piiutil.MaskPhone(req.Phone),
        "card_last4", piiutil.MaskIDLast4(req.CardNumber),
        "amount", req.Amount, // non-PII
        "currency", req.Currency,
    )
    
    // Process payment...
    if err := s.gateway.Charge(ctx, req); err != nil {
        s.log.Errorw("payment failed",
            "email", piiutil.MaskEmail(req.Email),
            "error", err.Error(),
        )
        return err
    }
    
    return nil
}
```

### Support Ticket Creation

```go
func (s *SupportService) CreateTicket(ctx context.Context, userID string, issue string) error {
    user := s.repo.GetUser(ctx, userID)
    
    ticket := &Ticket{
        UserID:      user.ID,
        EmailHint:   piiutil.MaskEmail(user.Email), // j******e@example.com
        PhoneHint:   piiutil.MaskPhone(user.Phone), // +*******4567
        Issue:       issue,
    }
    
    // Support agent sees hints, not full PII
    return s.repo.CreateTicket(ctx, ticket)
}
```

## Masking Examples

### Email Masking (`MaskEmail`)

Masks local-part, keeps first and last char. For 2-char local: first + `*`.

| Input | Output | Notes |
|-------|--------|-------|
| `user@example.com` | `u**r@example.com` | standard |
| `abc@example.com` | `a*c@example.com` | 3 chars |
| `ab@example.com` | `a*@example.com` | 2 chars → first + `*` |
| `u@example.com` | `u@example.com` | single char, unchanged |
| `john.doe@example.com` | `j******e@example.com` | long local |
| `юзер@example.com` | `ю**р@example.com` | unicode |
| `weird` | `w***d` | no @ → generic token |
| `ab` | `a*` | no @, 2 chars |
| `x` | `x` | single char |
| `@example.com` | `@**********m` | invalid → generic token |

### Phone Masking (`MaskPhone`)

Keeps format, masks digits. Keeps last 1 (≤4 digits) or 4 (>4 digits).

| Input | Output | Notes |
|-------|--------|-------|
| `+1234567890` | `+******7890` | >4 digits → keep 4 |
| `+1 (555) 123-4567` | `+* (***) ***-4567` | formatted |
| `+1234` | `+***4` | ≤4 digits → keep 1 |
| `123` | `**3` | ≤4 digits → keep 1 |
| `12` | `*2` | ≤4 digits → keep 1 |
| `1` | `1` | single digit |
| `AB-CD` | `**-*D` | no digits → mask letters |
| `()-` | `()-` | only separators |

### ID Masking (`MaskIDLast4`)

Same rules as phone: keep last 1 (≤4 digits) or 4 (>4 digits).

| Input | Output | Notes |
|-------|--------|-------|
| `123-45-6789` | `***-**-6789` | SSN format |
| `S1234567D` | `S***4567D` | NRIC (Singapore) |
| `123456789` | `*****6789` | 9 digits |
| `1234` | `***4` | ≤4 digits → keep 1 |
| `AB-1234-CD` | `AB-***4-CD` | formatted, ≤4 digits |
| `12-AB` | `*2-AB` | 2 digits |
| `ABCD` | `***D` | no digits → mask letters |
| `----` | `----` | only separators |

## Recommended Service Integration (Production)

Use masking at logging/response boundaries, not in your source-of-truth storage.

```go
log.Infow("payment declined",
    "user_email", piiutil.MaskEmail(user.Email),
    "phone", piiutil.MaskPhone(user.Phone),
    "tax_id", piiutil.MaskIDLast4(user.TaxID),
)

// Keep full value in DB where business logic requires it,
// but never emit raw PII to logs/metrics/errors.
```

## Business Examples

- **Support ticket logs**: show `u***@example.com` so agent can identify account without exposing full e-mail.
- **Fraud investigations**: keep last 4 digits of phone/ID for analyst correlation across systems.
- **Incident channels (Slack/PagerDuty)**: masked fields are safe to forward to broad on-call groups.

## Implementation Checklist

- Mask before writing structured logs (`Infow/Warnw/Errorw`).
- Mask before returning debug payloads from admin/internal APIs.
- Do not rely on masking as encryption; treat it as display/log hygiene.
- Add tests for local formats (country-specific phone/ID patterns).
