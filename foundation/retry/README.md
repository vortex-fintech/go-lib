# retry

Retry helpers for transient failures in startup flows and short operations.

## What to use

- `RetryInit(ctx, fn)`
  - exponential backoff
  - defaults: initial `500ms`, multiplier `2.0`, max interval `5s`
  - max elapsed time: `20s`
- `RetryFast(ctx, fn)`
  - fixed attempts: `3`
  - fixed delay: `200ms` between attempts
- `Permanent(err)`
  - marks an error as non-retryable
- `IsPermanent(err)`
  - detects both `retry.Permanent(err)` and `backoff.Permanent(err)`

## Correct usage pattern

Use retry only for transient failures, and mark business-final failures as permanent.

```go
import (
    "context"
    "fmt"
    "net/http"

    "github.com/vortex-fintech/go-lib/foundation/retry"
)

func callProviderWithRetry(ctx context.Context, doRequest func(context.Context) (int, error)) error {
    return retry.RetryFast(ctx, func() error {
        statusCode, err := doRequest(ctx)
        if err != nil {
            // Network timeout / temporary transport failure -> retry.
            return err
        }

        switch statusCode {
        case http.StatusUnauthorized, http.StatusForbidden:
            // Wrong credentials/permissions -> do not retry.
            return retry.Permanent(fmt.Errorf("provider auth failed: status=%d", statusCode))
        case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
            // Transient provider-side errors -> retry.
            return fmt.Errorf("provider temporary failure: status=%d", statusCode)
        }

        return nil
    })
}
```

## Behavior guarantees

- checks context cancellation before every attempt
- returns immediately when context is canceled
- applies no extra sleep after final failed attempt
- stops immediately on permanent errors

## Business examples

- Payments provider returns `503`: retry, because outage is usually transient
- Provider returns `401`: stop immediately, because credentials are invalid until reconfigured
- Service startup waits for DB: use `RetryInit` to survive short warm-up period

## Testing

Package tests are under `unit` build tag.

```bash
go test ./retry -tags unit -cover
go vet ./retry
```
