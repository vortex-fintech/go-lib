# textutil

Text canonicalization and policy utilities.

## Main APIs

- `CanonicalizeStrict(input, CanonicalPolicy)` - strict text canonicalization
- `NormalizeText(input, TextPolicy)` - canonicalization with policy validation
- `FirstNonEmpty(values...)` - returns first non-empty string

## Features

### CanonicalizeStrict

- Collapses whitespace to single spaces
- Rejects invalid UTF-8
- Rejects control characters
- Optionally allows newlines
- Optionally allows format characters (Cf)
- Enforces max runes limit

### NormalizeText

All features of CanonicalizeStrict plus:

- **NormalizeNFKC** - Unicode NFKC normalization (full-width → normal, ligatures → separate chars)
- **AllowedCharset** - restrict to specific characters
- **Pattern** - regex validation
- **MinRunes/MaxBytes** - length constraints

### AllowedCharset Options

- `AllowLetters` - allow unicode letters
- `AllowDigits` - allow unicode digits
- `AllowSpace` - allow space character
- `ExtraAllowed` - additional allowed characters (e.g., "._-@")
- `AllowedScripts` - restrict to specific scripts (Latin, Cyrillic, etc.)
- `DisallowMixedScripts` - reject mixed scripts in same text

## Example

```go
package main

import (
    "regexp"
    "unicode"
    
    "github.com/vortex-fintech/go-lib/foundation/textutil"
)

func main() {
    // Simple name normalization
    name, err := textutil.NormalizeText("  Ana   Maria  ", textutil.TextPolicy{
        MinRunes:   1,
        MaxRunes:   64,
        AllowEmpty: false,
    })
    // name = "Ana Maria"
    
    // Name with charset restriction (Latin only)
    latinName, err := textutil.NormalizeText("John", textutil.TextPolicy{
        MinRunes:   1,
        MaxRunes:   64,
        AllowEmpty: false,
        AllowedCharset: &textutil.AllowedCharset{
            AllowLetters:   true,
            AllowSpace:     true,
            AllowedScripts: []*unicode.RangeTable{unicode.Latin},
        },
    })
    
    // Email-like identifier
    email, err := textutil.NormalizeText("USER@EXAMPLE.COM", textutil.TextPolicy{
        MinRunes:   1,
        MaxRunes:   128,
        AllowEmpty: false,
        AllowedCharset: &textutil.AllowedCharset{
            AllowLetters: true,
            AllowDigits:  true,
            ExtraAllowed: "@._-",
        },
        Pattern: regexp.MustCompile(`^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+$`),
    })
    
    // Multi-line note with NFKC normalization
    note, err := textutil.NormalizeText("Ｈｅｌｌｏ\nWorld", textutil.TextPolicy{
        MinRunes:      1,
        MaxRunes:      500,
        AllowEmpty:    false,
        AllowNewlines: true,
        NormalizeNFKC: true,
    })
    // note = "Hello\nWorld" (full-width normalized)
    
    // First non-empty from multiple sources
    value := textutil.FirstNonEmpty("", "  ", "active", "fallback")
    // value = "active"
    
    _, _, _, _ = name, latinName, email, note
}
```

## Business Examples

### KYC Profile Name

```go
var personNamePolicy = textutil.TextPolicy{
    MinRunes:   1,
    MaxRunes:   80,
    MaxBytes:   320,
    AllowEmpty: false,
    AllowedCharset: &textutil.AllowedCharset{
        AllowLetters:   true,
        AllowSpace:     true,
        ExtraAllowed:   "'.-",
        DisallowMixedScripts: true,
    },
}

func NormalizePersonName(input string) (string, error) {
    return textutil.NormalizeText(input, personNamePolicy)
}
```

### Address Field with Newlines

```go
var addressPolicy = textutil.TextPolicy{
    MinRunes:      1,
    MaxRunes:      200,
    AllowEmpty:    false,
    AllowNewlines: true,
}

func NormalizeAddress(input string) (string, error) {
    return textutil.NormalizeText(input, addressPolicy)
}
```

### Product Code (Alphanumeric)

```go
var productCodePolicy = textutil.TextPolicy{
    MinRunes:   1,
    MaxRunes:   32,
    AllowEmpty: false,
    AllowedCharset: &textutil.AllowedCharset{
        AllowLetters: true,
        AllowDigits:  true,
        ExtraAllowed: "-_",
    },
    Pattern: regexp.MustCompile(`^[A-Z0-9_-]+$`),
}
```

### Username (Single Script)

```go
var usernamePolicy = textutil.TextPolicy{
    MinRunes:   3,
    MaxRunes:   32,
    AllowEmpty: false,
    AllowedCharset: &textutil.AllowedCharset{
        AllowLetters:         true,
        AllowDigits:          true,
        ExtraAllowed:         "_",
        AllowedScripts:       []*unicode.RangeTable{unicode.Latin},
        DisallowMixedScripts: true,
    },
}
```

### Normalizing Full-Width Input

```go
// Japanese users often input full-width characters
var searchQueryPolicy = textutil.TextPolicy{
    MinRunes:     1,
    MaxRunes:     100,
    AllowEmpty:   false,
    NormalizeNFKC: true, // ＡＢＣ → ABC
}

func NormalizeSearchQuery(input string) (string, error) {
    return textutil.NormalizeText(input, searchQueryPolicy)
}
```

## Security/Validation Notes

- Invalid UTF-8, control chars, and newline-like runes are rejected by default
- Rune limits are enforced for DoS resistance
- AllowedCharset prevents injection of unexpected characters
- DisallowMixedScripts helps detect homograph attacks
- NormalizeNFKC helps normalize visually similar characters

## Compatibility Note

When adopting stricter text policy rules in an existing API, use an observe phase before hard enforcement.
