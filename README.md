# go-lib

Reusable Go utilities and infrastructure for internal services at Vortex.

## 📦 Packages

### `db/postgres`

Simple wrapper to initialize and configure a PostgreSQL connection using `database/sql`.

#### 🔧 Usage

```go
import (
    "context"
    "github.com/vortex-fintech/go-lib/db/postgres"
    "time"
)

func main() {
    cfg := postgres.DBConfig{
        Host:            "localhost",
        Port:            "5432",
        User:            "user",
        Password:        "pass",
        DBName:          "example",
        SSLMode:         "disable",
        MaxOpenConns:    10,
        MaxIdleConns:    5,
        ConnMaxLifetime: 30 * time.Minute,
    }

    db, err := postgres.NewPostgresClient(context.Background(), cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
```

---

### `errors`

Unified error response helpers for gRPC and HTTP APIs.

#### 🔧 Usage

```go
import (
    "github.com/vortex-fintech/go-lib/errors"
    "google.golang.org/grpc/codes"
)

func SomeHandler() error {
    // Validation error with details
    return errors.ValidationError(map[string]string{
        "email": "invalid format",
    })

    // Predefined error
    // return errors.NotFoundError

    // Custom error
    // return errors.NewError("custom message", codes.Aborted, nil)
}
```

- All errors implement `error` and have fields: `Code`, `Message`, `Details`.
- Use `.ToGRPC()` to convert to gRPC error with details.
- Predefined errors: `NotFoundError`, `InternalError`, etc.

---

### `hash`

Helpers for hashing strings with SHA-256 and a custom separator.

#### 🔧 Usage

```go
import (
    "github.com/vortex-fintech/go-lib/hash"
)

func main() {
    h := hash.HashStringsWithSep("foo", "bar", "baz")
    // h is a SHA-256 hex string, unique for this set and order of strings
}
```

- Uses a non-printable separator to avoid collisions.
- Always returns a 64-character hex string.

---

### `logger`

Simple and fast structured logger based on [zap](https://github.com/uber-go/zap).

#### 🔧 Usage

```go
import (
    "github.com/vortex-fintech/go-lib/logger"
)

func main() {
    log := logger.Init("my-service", "development")
    defer log.SafeSync() // flush logs on exit

    log.Info("service started")
    log.Infow("user login", "userID", 123)
    log.Warnf("disk space low: %d%%", 5)

    l2 := log.With("request_id", "abc-123")
    l2.Error("something went wrong")
}
```

- Supports environments: `"development"`, `"debug"`, `"production"`, `"unknown"`.
- Implements `LoggerInterface` (see `logger/interface.go`).
- Use `.With(...)` for contextual logging.
- Use `.SafeSync()` to flush logs (safe for tests and production).

---

### `retry`

Helpers for retrying operations with exponential backoff or fixed attempts.

#### 🔧 Usage

```go
import (
    "context"
    "github.com/vortex-fintech/go-lib/retry"
)

func main() {
    err := retry.RetryInit(context.Background(), func() error {
        // your operation here
        return nil
    })
    if err != nil {
        // handle error after retries
    }

    err = retry.RetryFast(context.Background(), func() error {
        // your operation here
        return nil
    })
}
```

- `RetryInit` — exponential backoff, up to ~20 seconds.
- `RetryFast` — 3 attempts with a short delay.
- Both methods support cancellation via context.

---

### `validator`

Helpers for struct validation using [go-playground/validator](https://github.com/go-playground/validator).

#### 🔧 Usage

```go
import (
    "github.com/vortex-fintech/go-lib/validator"
)

type User struct {
    Email string `validate:"required,email"`
    Age   int    `validate:"min=18"`
}

func main() {
    u := User{Email: "test@example.com", Age: 25}
    if fields := validator.Validate(u); fields != nil {
        // handle validation errors
    }
}
```

- Returns `map[string]string` with field names and error codes.
- See `validator/tagmap.go` for error code mapping.
- Use `validator.Instance()` to get the underlying validator instance.

---

## 🧪 Testing

This project supports **unit** and **integration** tests with proper separation via Go build tags.

### ✅ Unit Tests

- Use [`sqlmock`](https://github.com/DATA-DOG/go-sqlmock) for database code
- Fast, isolated
- No external services required

Run:
```bash
make test
```

### 🐳 Integration Tests

- Launches a real PostgreSQL instance via Docker
- Tests real connection and configuration
- Located in `client_integration_test.go` with `//go:build integration`

Run:
```bash
make test-integration
```

This will:
- Start a Docker container (`postgres:14`)
- Wait until the DB is healthy
- Run integration tests with `-tags=integration`
- Tear down the Docker container

You can also manually run:

```bash
make up          # Start Postgres container
make down        # Stop and remove container
```

### 🧪 Build Tags

| File                          | Tag           | Included in...             |
|-------------------------------|---------------|----------------------------|
| `client_test.go`              | `unit`        | `make test`                |
| `client_integration_test.go`  | `integration` | `make test-integration`    |
| `errors_test.go`              | `unit`        | `make test`                |
| `hash_test.go`                | `unit`        | `make test`                |
| `logger_test.go`              | `unit`        | `make test`                |
| `retry_test.go`               | `unit`        | `make test`                |
| `validator_test.go`           | `unit`        | `make test`                |

## 📂 Structure

```
db/
└── postgres/
    ├── client.go
    ├── config.go
    ├── client_test.go
    ├── client_integration_test.go
    └── docker-compose.test.yml
errors/
    ├── errors.go
    ├── response.go
    ├── errors_test.go
hash/
    ├── sha256_util.go
    ├── hash_test.go
logger/
    ├── logger.go
    ├── interface.go
    ├── logger_test.go
retry/
    ├── retry.go
    ├── retry_test.go
validator/
    ├── validator.go
    ├── tagmap.go
    ├── validator_test.go
```

## 🛠️ Dependencies

- [lib/pq](https://github.com/lib/pq)
- [sqlmock](https://github.com/DATA-DOG/go-sqlmock)
- [testify](https://github.com/stretchr/testify)
- [grpc](https://github.com/grpc/grpc-go)
- [cenkalti/backoff](https://github.com/cenkalti/backoff)
- [go-playground/validator](https://github.com/go-playground/validator)