# netutil

Networking-oriented utility helpers.

## Functions

- `SanitizeTimeout(d, min, fallback) time.Duration`
- `SanitizeTimeoutAllowZero(d, min, fallback) time.Duration`

## Rules

- negative timeout -> `fallback`
- timeout below positive `min` -> `min`
- otherwise -> original value
- for `SanitizeTimeout`, `d=0` with positive `min` is clamped to `min`
- for `SanitizeTimeoutAllowZero`, `d=0` is preserved as `0`

## Example

```go
package main

import (
    "net/http"
    "time"
    
    "github.com/vortex-fintech/go-lib/foundation/netutil"
)

func main() {
    // Config values (could come from env/config file)
    timeout := 50 * time.Millisecond   // too low
    minTimeout := 200 * time.Millisecond
    fallbackTimeout := 3 * time.Second
    
    // Sanitize timeout for external API calls
    safeTimeout := netutil.SanitizeTimeout(timeout, minTimeout, fallbackTimeout)
    // Result: 200ms (clamped to minimum)
    
    client := &http.Client{
        Timeout: safeTimeout,
    }
    
    // For long-running operations (exports, batch jobs)
    exportTimeout := 0 // means "no timeout"
    safeExport := netutil.SanitizeTimeoutAllowZero(exportTimeout, minTimeout, fallbackTimeout)
    // Result: 0 (preserved)
    
    _ = client
    _ = safeExport
}
```

### HTTP Client Configuration

```go
func NewHTTPClient(cfg Config) *http.Client {
    timeout := netutil.SanitizeTimeout(
        cfg.RequestTimeout,    // from config
        100*time.Millisecond,  // minimum allowed
        30*time.Second,        // fallback for invalid values
    )
    
    return &http.Client{
        Timeout: timeout,
    }
}
```

### Partner API with Retry Protection

```go
func CallPartnerAPI(ctx context.Context, cfg PartnerConfig) error {
    // Prevent too-aggressive timeouts that cause retry storms
    timeout := netutil.SanitizeTimeout(
        cfg.Timeout,
        500*time.Millisecond,  // minimum for partner API
        10*time.Second,        // safe fallback
    )
    
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    // ... make request
    return nil
}
```

## Business Examples

- **Partner API calls (strict floor):**
  - config `timeout=50ms`, `min=200ms`, `fallback=3s`
  - `SanitizeTimeout` returns `200ms`, preventing too-aggressive timeouts and retry storms
- **Long-running export endpoint (explicit no-timeout):**
  - config `timeout=0`, `min=200ms`, `fallback=10s`
  - `SanitizeTimeoutAllowZero` returns `0`, so operation can run without client-side deadline
- **Broken config safety:**
  - config `timeout=-1s`, `fallback=5s`
  - both functions return `5s`, avoiding invalid negative timeout in runtime clients
