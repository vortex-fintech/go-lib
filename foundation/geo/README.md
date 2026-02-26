# geo

Country code normalization helpers.

## Functions

- `NormalizeISO2(code) (normalized string, ok bool)`
- `IsValidISO2(code) bool`

## Behavior

- `NormalizeISO2` and `IsValidISO2`:
  - trim input
  - uppercase ASCII letters
  - accept only ASCII letters and exact length 2
  - format-only validation (`ZZ` is valid format)

For strict ISO 3166-1 alpha-2 validation, use reference service.

## Example

```go
package main

import (
    "fmt"
    
    "github.com/vortex-fintech/go-lib/foundation/geo"
)

func main() {
    // Format normalization
    code, ok := geo.NormalizeISO2("  us  ")
    if !ok {
        panic("invalid format")
    }
    fmt.Println(code) // "US"
    
    // Format validation
    if geo.IsValidISO2("gb") {
        fmt.Println("valid format")
    }
    
    // Note: "ZZ", "UK" pass format validation
    // Use reference service for strict ISO validation
}
```

## Business Examples

- **Raw event ingestion:**
  - input `" us "` -> `NormalizeISO2` returns `"US"`
  - event can be accepted early for pipeline continuity
- **Input sanitization:**
  - normalize user input before sending to reference service
  - ensures consistent format in API calls
