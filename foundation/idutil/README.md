# idutil

Type-safe UUID wrapper with compile-time distinction between ID types.

## Features

- Generic `ID[T]` wrapper around `uuid.UUID`
- UUID v7 by default (time-sortable)
- Compile-time type safety (`AccountID` != `CredentialID`)
- Inherits all `uuid.UUID` methods (JSON, SQL, binary marshaling)

## API

```go
NewID[T]() (ID[T], error)    // Generate new UUID v7
ParseID[T](s) (ID[T], error) // Parse string to ID
(id ID[T]).IsZero() bool     // Check for zero value
(id ID[T]).String() string   // Get string representation
```

## Example

### Defining Type-Safe IDs

```go
package domain

import "github.com/vortex-fintech/go-lib/foundation/idutil"

type (
	accountIDTag    struct{}
	credentialIDTag struct{}
	orderIDTag      struct{}
)

type (
	AccountID    = idutil.ID[accountIDTag]
	CredentialID = idutil.ID[credentialIDTag]
	OrderID      = idutil.ID[orderIDTag]
)

func NewAccountID() (AccountID, error)            { return idutil.NewID[accountIDTag]() }
func ParseAccountID(s string) (AccountID, error) { return idutil.ParseID[accountIDTag](s) }

func NewCredentialID() (CredentialID, error)            { return idutil.NewID[credentialIDTag]() }
func ParseCredentialID(s string) (CredentialID, error) { return idutil.ParseID[credentialIDTag](s) }
```

### Using IDs in Domain Types

```go
type Account struct {
	ID        AccountID
	Email     string
	CreatedAt time.Time
}

type Credential struct {
	ID        CredentialID
	AccountID AccountID
	Type      string
}

func CreateAccount(email string) (*Account, error) {
	id, err := NewAccountID()
	if err != nil {
		return nil, err
	}
	return &Account{
		ID:        id,
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}, nil
}
```

### Type Safety at Compile Time

```go
func ProcessAccount(id AccountID) { /* ... */ }
func ProcessCredential(id CredentialID) { /* ... */ }

accountID, _ := NewAccountID()
credentialID, _ := NewCredentialID()

ProcessAccount(accountID)       // OK
ProcessCredential(credentialID) // OK

// ProcessAccount(credentialID)  // Compile error!
// ProcessCredential(accountID)  // Compile error!
```

### Parsing from Request

```go
func (h *Handler) GetAccount(ctx context.Context, idStr string) (*Account, error) {
	id, err := ParseAccountID(idStr)
	if err != nil {
		return nil, ErrInvalidAccountID
	}
	
	if id.IsZero() {
		return nil, ErrAccountNotFound
	}
	
	return h.repo.GetAccount(ctx, id)
}
```

### JSON and Database Support

```go
type Account struct {
	ID AccountID `json:"id" db:"id"`
}

// ID automatically marshals/unmarshals as UUID string in JSON
// ID automatically works with SQL drivers via uuid.UUID embedding

func (r *Repository) Create(ctx context.Context, acc *Account) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO accounts (id, email) VALUES ($1, $2)",
		acc.ID, acc.Email,
	)
	return err
}
```

### Logging with IDs

```go
func (s *Service) ProcessOrder(ctx context.Context, orderID OrderID) error {
	s.log.Infow("processing order",
		"order_id", orderID.String(),
	)
	
	// ...
	return nil
}
```

## Business Examples

- **Account ID**: `type AccountID = idutil.ID[accountIDTag]`
- **Credential ID**: `type CredentialID = idutil.ID[credentialIDTag]`
- **Order ID**: `type OrderID = idutil.ID[orderIDTag]`
- **Session ID**: `type SessionID = idutil.ID[sessionIDTag]`
- **Transaction ID**: `type TransactionID = idutil.ID[transactionIDTag]`

## Why UUID v7

UUID v7 is time-sortable, which provides:

- Better database index locality
- Chronological ordering in logs
- Efficient range queries by creation time

## Notes

- Zero value (`ID[T]{}`) is valid Go but represents an empty/unset ID
- Use `IsZero()` to check for unset IDs before database operations
- Tag types are empty structs with zero memory overhead
