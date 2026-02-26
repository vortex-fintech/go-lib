# AGENTS.md - Coding Agent Guidelines

Guidelines for agentic coding agents in the `go-lib/security` repository.

**Module:** `github.com/vortex-fintech/go-lib/security` | **Go:** 1.25+

## Build, Lint, Test Commands

```bash
go test ./...                      # All tests
go test -race ./...                # With race detector
go test ./hmac                     # Single package
go test ./scope -run TestHasAll    # Single test
go test ./jwt -run TestVerify -v   # Single test verbose
go test -cover ./...               # Coverage
go fmt ./...                       # Format
go vet ./...                       # Vet
go build ./...                     # Build
go mod tidy                        # Tidy deps
```

## Code Style

### Imports

```go
import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)
```

### Error Handling

```go
var (
	ErrEmptySecret = errors.New("secret cannot be empty")
	ErrEmptyMAC    = errors.New("expected MAC cannot be empty")
)
```

- Return errors as last value
- Never expose secrets in error messages

### Naming

- **Exported:** `PascalCase` (e.g., `JWKSConfig`, `RedisChecker`)
- **Unexported:** `camelCase` (e.g., `jwk`, `decodeClaims`)
- **Interfaces:** Simple nouns (e.g., `Checker`, `Verifier`)
- **Constructors:** `New<TypeName>` (e.g., `NewJWKSVerifier`)
- **Acronyms:** `JTI`, `JWT`, `TLS` (uppercase)

### Structs

```go
type JWKSConfig struct {
	URL            string
	RefreshEvery   time.Duration
	ExpectedIssuer string
}

type Claims struct {
	Subject string `json:"sub"`
	Scopes  []string `json:"scopes,omitempty"`
}
```

### Interfaces

```go
type Checker interface {
	SeenJTI(ctx context.Context, namespace, jti string, ttl time.Duration) (seen bool, err error)
}

type Verifier interface {
	Verify(ctx context.Context, rawToken string) (*Claims, error)
}
```

### Comments

- Bilingual: English or Russian
- Document exported items

## Testing

### Table-Driven Tests

```go
func TestHasAll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scopes []string
		need   []string
		want   bool
	}{
		{"all present", []string{"a", "b"}, []string{"a"}, true},
		{"missing", []string{"a"}, []string{"b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scope.HasAll(tt.scopes, tt.need...); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
```

### Conventions

- External test packages: `package hmac_test`
- `t.Parallel()` for parallel tests
- `t.Fatalf()` for fatal, `t.Errorf()` for non-fatal
- Name: `Test<Function>_<Scenario>` (e.g., `TestVerify_EmptySecret`)

## Security

- Never log secrets/tokens
- Use `hmac.Equal()` for constant-time comparison
- Validate inputs at boundaries
- Use `context` for cancellation

## Common Patterns

### Options Pattern

```go
type RedisOptions struct {
	Prefix   string
	FailOpen bool
}

func NewRedisChecker(rdb redis.UniversalClient, opt RedisOptions) *RedisChecker {
	prefix := opt.Prefix
	if prefix == "" {
		prefix = "obo:jti"
	}
	return &RedisChecker{rdb: rdb, prefix: prefix}
}
```

### Adapter Pattern

```go
func (r *RedisChecker) AsAuthzCallback(namespace string, ttl time.Duration) func(string) bool {
	return func(jti string) bool {
		seen, _ := r.SeenJTI(context.Background(), namespace, jti, ttl)
		return seen
	}
}
```

## Package Structure

```
security/
├── jwt/      # JWT verification, OBO validation
├── mtls/     # Mutual TLS with hot reload
├── hmac/     # HMAC-SHA256
├── replay/   # Anti-replay JTI tracking
├── scope/    # Scope evaluation utilities
└── go.mod
```

## Before Committing

1. `go fmt ./...`
2. `go vet ./...`
3. `go test -race ./...`
4. `go mod tidy`
