# Scope Helpers

Simple utilities for evaluating OAuth/JWT scopes.

## Where to use it

- Check if user has required permissions
- Validate API access scopes
- Filter by scope in middleware

## Basic usage

```go
import "github.com/vortex-fintech/go-lib/security/scope"

scopes := []string{"wallet:read", "wallet:write", "payments:create"}

// Check all required scopes present
if !scope.HasAll(scopes, "wallet:read", "wallet:write") {
    return errors.New("insufficient permissions")
}

// Check at least one scope present
if !scope.HasAny(scopes, "admin", "superuser") {
    return errors.New("requires admin or superuser")
}
```

## Index for repeated checks

```go
idx := scope.Index(claims.Scopes)

for _, req := range requiredScopes {
    if _, ok := idx[req]; !ok {
        return errors.New("missing scope: " + req)
    }
}
```

## API reference

### `Index(scopes []string) map[string]struct{}`

Creates a set from scope slice for O(1) lookups.

### `HasAll(scopes []string, need ...string) bool`

Returns true if all `need` scopes are present in `scopes`.

### `HasAny(scopes []string, any ...string) bool`

Returns true if at least one of `any` scopes is present in `scopes`.

## Examples

### Middleware

```go
func RequireScopes(need ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            claims := ClaimsFromContext(r.Context())
            if !scope.HasAll(claims.Scopes, need...) {
                http.Error(w, "forbidden", 403)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Conditional logic

```go
if scope.HasAny(claims.Scopes, "admin", "moderator") {
    // Show admin panel
}

if scope.HasAll(claims.Scopes, "payments:create", "payments:approve") {
    // Allow payment creation and approval
}
```

## Production notes

- Scopes are case-sensitive
- Empty `need` / `any` always returns true
- Works with nil/empty scope slices
