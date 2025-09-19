# go-lib

Reusable Go utilities and infrastructure for internal services at Vortex.

## 📦 Packages

### `db/postgres`

Wrapper to initialize and configure a PostgreSQL connection using `database/sql`.

#### 🔧 Usage

```go
import (
    "context"
    "time"

    "github.com/vortex-fintech/go-lib/db/postgres"
)

func main() {
    cfg := postgres.DBConfig{
        Host:            "localhost",
        Port:            "5433",
        User:            "testuser",
        Password:        "testpass",
        DBName:          "testdb",
        SSLMode:         "disable",
        MaxOpenConns:    10,
        MaxIdleConns:    5,
        ConnMaxLifetime: 10 * time.Minute,
        ConnMaxIdleTime: 2 * time.Minute,
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    db, err := postgres.NewPostgresClient(ctx, cfg)
    if err != nil {
        panic(err)
    }
    defer db.Close()
}
```

Notes:
- The client pings the DB with the provided context; on ping error it closes the connection and returns the error.
- `ConnMaxIdleTime` is supported in addition to `ConnMaxLifetime`.

### `dbsql`

Helpers for working with SQL databases, providing a unified interface for both `*sql.DB` and `*sql.Tx`.

#### Features

- `Executor` interface abstracts both `*sql.DB` and `*sql.Tx`
- `UseExecutor` to choose between a DB and an active transaction
- Context-aware methods:
  - `ExecContext`
  - `QueryContext`
  - `QueryRowContext`

#### 🔧 Usage

```go
import (
    "context"
    "database/sql"

    dbsql "github.com/vortex-fintech/go-lib/db/dbsql"
)

func DoSomething(ctx context.Context, exec dbsql.Executor) error {
    _, err := exec.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", "user")
    return err
}

func example(ctx context.Context, db *sql.DB) error {
    // use DB directly
    if err := DoSomething(ctx, dbsql.UseExecutor(db, nil)); err != nil {
        return err
    }

    // use Tx
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if err := DoSomething(ctx, dbsql.UseExecutor(db, tx)); err != nil {
        return err
    }
    return tx.Commit()
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

### `graceful/shutdown`

Unified graceful start/stop manager for coordinating multiple servers (HTTP, gRPC, etc.).

#### Features

- Central orchestration of serving and coordinated shutdown across many servers
- Differentiates between normal (expected) serve errors (e.g. `http.ErrServerClosed`) and fatal errors
- Graceful timeout after which a force stop is executed
- Optional OS signal handling (SIGINT, SIGTERM)
- Pluggable logging callback (integrate zap / zerolog / custom)
- Optional Prometheus metrics export
- Extensible normal error predicate (`IsNormalError`)
- Adapters pattern for HTTP, gRPC (and you can add your own)

#### 🔧 Quickstart

```go
import (
    "context"
    "net"
    "net/http"
    "time"

    "github.com/vortex-fintech/go-lib/graceful/shutdown"
    "github.com/vortex-fintech/go-lib/graceful/shutdown/adapters" // HTTP / gRPC adapters
)

func main() {
    // 1) HTTP server
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })
    httpSrv := &http.Server{Handler: mux}
    httpLn, _ := net.Listen("tcp", ":8080")

    // 2) gRPC server (example)
    // grpcSrv := grpc.NewServer()
    // grpcLn, _ := net.Listen("tcp", ":9090")

    // 3) Manager
    mgr := shutdown.New(shutdown.Config{
        ShutdownTimeout: 15 * time.Second, // grace period
        HandleSignals:   true,             // catch SIGINT/SIGTERM
        Logger: func(level, msg string, kv ...any) {
            // integrate your structured logger here
            // log.With(kv...).Log(level, msg)
        },
        // IsNormalError: override if you need to extend default classification
        // IsNormalError: func(err error) bool { return shutdown.DefaultIsNormalErr(err) },
    })

    // 4) Register servers via adapters
    mgr.Add(&adapters.HTTP{Srv: httpSrv, Lis: httpLn, NameStr: "http"})
    // mgr.Add(&adapters.GRPC{Srv: grpcSrv, Lis: grpcLn, NameStr: "grpc"})

    // 5) Run (blocking)
    if err := mgr.Run(context.Background()); err != nil {
        // fatal (non-normal) error that triggered shutdown
        // log.Error("shutdown failed", "err", err)
    }
}
```

#### 🧰 Prometheus Metrics (optional)

```go
import (
    "net/http"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "github.com/vortex-fintech/go-lib/graceful/shutdown"
)

reg := prometheus.NewRegistry()
pm  := shutdown.NewPromMetrics(reg, "vortex", "graceful")

mgr := shutdown.New(shutdown.Config{
    ShutdownTimeout: 15 * time.Second,
    HandleSignals:   true,
    Logger:          myLogger,
    Metrics:         pm, // enable metrics
})

// Expose metrics (could be a separate server)
go func() {
    _ = http.ListenAndServe(":9100", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}()
```

Exported metrics (label cardinality kept low):

```
vortex_graceful_graceful_stop_total{result="success|force"}
vortex_graceful_server_serve_errors_total{name}
vortex_graceful_server_stop_result_total{name,result="success|force"}
vortex_graceful_graceful_duration_seconds (histogram)
```

#### 🔌 Adapters

- `adapters.HTTP`: graceful via `(*http.Server).Shutdown(ctx)`, force via `Close()`. Respects `BaseContext` if set for request scoping.
- `adapters.GRPC`: graceful via `(*grpc.Server).GracefulStop()`, force via `Stop()`.

You can implement your own by satisfying the adapter interface (see package).

#### ☸️ Kubernetes Recommendations

- `terminationGracePeriodSeconds` ≥ `ShutdownTimeout` + 5–10s buffer (gives time for force path and network propagation).
- Optional `preStop` hook:
  - HTTP: call a `/drain` endpoint to flip readiness (stop new traffic) before SIGTERM.
  - gRPC: stop accepting new streams / connections before SIGTERM.
- Probes:
  - `livenessProbe`: only fails on unrecoverable internal faults.
  - `readinessProbe`: must return NOT ready during graceful phase to drain traffic.
- Signals: with `HandleSignals: true` the manager listens to SIGINT/SIGTERM and initiates graceful shutdown automatically.

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
├── dbsql/
│   ├── helper.go
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
graceful/
    ├── shutdown/
        ├── (manager, adapters, metrics)
```

## 🛠️ Dependencies

- [lib/pq](https://github.com/lib/pq)
- [sqlmock](https://github.com/DATA-DOG/go-sqlmock)
- [testify](https://github.com/stretchr/testify)
- [grpc](https://github.com/grpc/grpc-go)
- [cenkalti/backoff](https://github.com/cenkalti/backoff)
- [go-playground/validator](https://github.com/go-playground/validator)
- [prometheus/client_golang](https://github.com/prometheus/client_golang)