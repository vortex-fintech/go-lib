# go-lib

Пакет с переиспользуемыми утилитами и инфраструктурными компонентами для Go‑сервисов Vortex.

Фокус: строгая обработка ошибок (HTTP/gRPC), надёжная остановка сервисов, метрики, gRPC‑middleware, Postgres/Redis клиенты, JWT/JWKS верификация и mTLS.

## Быстрый старт

- Требования: Go 1.25+ (toolchain go1.25.x)
- Установка:

```bash
go get github.com/vortex-fintech/go-lib@latest
```

- Импортируйте нужные пакеты, примеры ниже.

## Состав пакетов и примеры

Ниже только самое полезное для продакшена. В коде много тестов — их можно посмотреть как дополнительную документацию.

### errors — единый формат ошибок для HTTP и gRPC

Структура ErrorResponse со следующими возможностями:
- код gRPC (`codes.Code`) + человекочитаемое сообщение
- машинный `Reason`, `Domain`, `Details` (k/v)
- валидационные нарушения (BadRequest Violations)
- конвертация в gRPC status и HTTP ответ

Пример:

```go
import (
    gliberr "github.com/vortex-fintech/go-lib/errors"
    "google.golang.org/grpc/codes"
)

func CreateUser() error {
    // Валидация
    return gliberr.InvalidArgument().
        WithReason("invalid_input").
        WithDetails(map[string]string{"email":"invalid"}).
        WithViolations([]gliberr.FieldViolation{{Field:"email", Reason:"invalid"}})
}

func ToGRPC(err error) error {
    return gliberr.Internal().WithReason("unexpected").ToGRPC()
}

func ToHTTP(w http.ResponseWriter) {
    gliberr.ResourceExhausted().
        WithDetail("retry","10s").
        ToHTTPWithRetry(w, 10*time.Second)
}
```

См. также адаптеры для gRPC: `grpc/middleware/errorsmw` ниже.

### grpc/middleware — цепочки и полезные перехватчики

- chain: сборка единой цепочки unary‑перехватчиков с правильным порядком
- errorsmw: перевод ошибок домена/контекста в статус gRPC
- metricsmw (+ promreporter): наблюдаемость RPC
- contextcancel: корректное завершение при отмене контекста
- circuitbreaker: простой CB c HALF_OPEN пробами
- authz: аутентификация/авторизация по OBO‑JWT с PoP (mTLS)

Сборка сервера с цепочкой:

```go
import (
    "google.golang.org/grpc"
    chain "github.com/vortex-fintech/go-lib/grpc/middleware/chain"
    errorsmw "github.com/vortex-fintech/go-lib/grpc/middleware/errorsmw"
    metricsmw "github.com/vortex-fintech/go-lib/grpc/middleware/metricsmw"
    promrep "github.com/vortex-fintech/go-lib/grpc/middleware/metricsmw/promreporter"
)

// Ваши пром‑метрики
type myRPCMetrics struct { /* ... */ }
func (m *myRPCMetrics) ObserveRPC(svc, method, code string, sec float64) {}
func (m *myRPCMetrics) IncError(typ, svc, method string) {}

rep := promrep.Reporter{M: &myRPCMetrics{}}

srv := grpc.NewServer(chain.Default(chain.Options{
    Pre:  []grpc.UnaryServerInterceptor{metricsmw.UnaryFull(rep)},
    Post: []grpc.UnaryServerInterceptor{},
    // AuthzInterceptor: см. ниже
}))
```

#### authz — OBO‑JWT + PoP (mTLS) + scopes

Перехватчик проверяет:
- подпись JWT через ваш `Verifier` (например, JWKS)
- политику OBO (`aud`, `act.sub`, `exp/iat` + `leeway`, `max TTL`)
- обязательную привязку к клиентскому сертификату (PoP, `x5t#S256`) — по умолчанию включено
- наличие/достаточность скоупов (All/Any/глобальные)

В контекст прокидывается `Identity{UserID uuid.UUID, Scopes []string, SID, DeviceID}`.

Пример настройки с JWKS‑верификатором:

```go
import (
    "github.com/vortex-fintech/go-lib/grpc/middleware/authz"
    libjwt "github.com/vortex-fintech/go-lib/security/jwt"
)

verifier, _ := libjwt.NewJWKSVerifier(libjwt.JWKSConfig{
    URL:            "https://sso.internal/.well-known/jwks.json",
    RefreshEvery:   5 * time.Minute,
    Timeout:        5 * time.Second,
    ExpectedIssuer: "https://sso.internal",
})

az := authz.UnaryServerInterceptor(authz.Config{
    Verifier:       verifier,
    Audience:       "wallet",       // этот сервис
    Actor:          "api-gateway",  // ожидаемый актёр
    Leeway:         45 * time.Second,
    MaxTTL:         5 * time.Minute,
    RequireScopes:  true,
    RequirePoP:     true, // по умолчанию true
    ResolvePolicy:  authz.MapResolver(map[string]authz.Policy{"/pkg.Service/Method":{All:[]string{"wallet:read"}}}),
    SkipAuth:       authz.SliceSkipAuth("/pkg.Health/Check"),
})

srv := grpc.NewServer(chain.Default(chain.Options{AuthzInterceptor: az}))
```

Доступ к идентичности в бизнес‑коде:

```go
id, err := authz.RequireIdentity(ctx) // или только UUID: authz.RequireUserID(ctx)
```

### security/jwt — JWKS‑верификация и строгая политика OBO

- `NewJWKSVerifier(JWKSConfig)` — безопасный клиент JWKS с кэшированием, поддержкой Cache‑Control/ETag
- `ValidateOBO(now, claims, OBOValidateOptions)` — строгие проверки aud/act/времени/JTI/PoP/scopes
- Утилиты: `X5tS256FromCert` для mTLS привязки

### security/mtls — TLS для клиента и сервера, живой перезагрузчик

См. пакет `security/mtls`: генерация `*tls.Config` для сервера/клиента, загрузка из файлов, перезагрузка при изменении.

### graceful/shutdown — единый менеджер остановки

- оркеструет запуск и остановку многих серверов
- различает «нормальные» ошибки serve (например, http.ErrServerClosed)
- ограничение по времени с форс‑остановкой
- сигналы ОС (SIGINT/SIGTERM), интеграция логирования, метрики Prometheus

Пример см. выше (блок Quickstart).

### db/postgres — pgxpool + удобные раннеры и транзакции

Высокоуровневое открытие по URL и низкоуровневое по структуре конфигурации.

```go
import (
    "context"
    "time"
    "github.com/vortex-fintech/go-lib/db/postgres"
)

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Через DBConfig (host/port/...)
cli, err := postgres.OpenWithDBConfig(ctx, postgres.DBConfig{
    Host:"localhost", Port:"5433", User:"testuser", Password:"testpass",
    DBName:"testdb", SSLMode:"disable",
    MaxOpenConns:10, MaxIdleConns:5,
    ConnMaxLifetime:10 * time.Minute, ConnMaxIdleTime:2 * time.Minute,
})
if err != nil { /* ... */ }
defer cli.Close()

// Выполнение запросов
run := cli.RunnerFromPool()
row := run.QueryRow(ctx, "SELECT 1")

// Транзакция
_ = cli.WithTx(ctx, func(txrun postgres.Runner) error {
    _, err := txrun.Exec(ctx, "INSERT ...")
    return err
})
```

Обработчики ошибок Postgres: `Constraint(err)`, `IsUniqueViolation(err)` и др.

### db/redis — универсальный клиент (single/sentinel/cluster)

```go
rdb, err := redis.NewRedisClient(ctx, redis.Config{Addr: "127.0.0.1:6379", TLSEnabled: false})
if err != nil { /* ... */ }
defer rdb.Close()
```

Поддержка TLS (минимум TLS 1.2), пинг при инициализации с таймаутом.

### metrics — /metrics и /health в одном handler’е

```go
import (
    "net/http"
    "github.com/vortex-fintech/go-lib/metrics"
)

mux, reg := metrics.New(metrics.Options{
    Register: func(r prometheus.Registerer) error { /* регистрируем свои метрики */; return nil },
    Health:   func(ctx context.Context, r *http.Request) error { return nil },
})
_ = http.ListenAndServe(":9100", mux)
```

- GET/HEAD‑только маршруты, таймаут health по умолчанию 500ms
- регистрируются стандартные Go/Process метрики

### logger — лёгкая обёртка над zap

```go
log := logger.Init("my-service", "production")
defer log.SafeSync()
log.Infow("start", "version", "1.2.3")
```

Поддерживаются окружения: development, debug, production, unknown.

### retry — быстрые ретраи

`RetryInit(ctx, fn)` и `RetryFast(ctx, fn)` — экспонента и быстрый режим соответственно, уважают context.

### validator — обёртка над go‑playground/validator

`Validate(any) map[string]string` и `Instance()` для доступа к оригинальному валидатору. См. `validator/tagmap.go` для маппинга кодов ошибок.

### Прочее

- hash: SHA‑256 утилиты
- timeutil: UTC‑часы, оффсеты, sleep с отменой
- netutil: санитация таймаутов
- logutil: маскировка/санитизация ошибок в зависимости от окружения
- grpc/creds: обёртки для gRPC transport credentials

## Тестирование

Проект разделяет юнит‑ и интеграционные тесты через build‑теги.

### Юнит‑тесты

Запуск (Windows PowerShell):

```powershell
go test -count=1 -tags=unit ./...
go test -count=1 -tags "unit testhooks" ./db/postgres
```

Или через Make (требуется Bash, например Git Bash/WSL):

```bash
make test
```

### Интеграционные тесты (Postgres + Docker)

Требуются: Docker и docker compose. БД поднимается по `db/postgres/docker-compose.test.yml` (порт 5433).

```bash
make test-integration    # up -> wait -> go test -tags=integration -> down
```

Эквивалент вручную (PowerShell):

```powershell
docker compose -f db/postgres/docker-compose.test.yml up -d --wait --wait-timeout 60
go test -count=1 -tags integration ./...
docker compose -f db/postgres/docker-compose.test.yml down -v
```

## Совместимость и версии

- Go 1.25+
- Модуль: `github.com/vortex-fintech/go-lib`
- Семантические версии (SemVer). Уточняйте совместимость мажорных релизов по changelog (в рамках мажора — обратная совместимость API).

## Лицензия

MIT — см. `LICENSE`.

## Вклад и вопросы

PR/issue приветствуются. Для security‑вопросов используйте приватный канал; не публикуйте чувствительные детали в публичных задачах.

## Диагностика прод‑готовности (состояние репозитория)

- Build: PASS (`go build ./...`)
- Vet: PASS (`go vet ./...`)
- Unit tests: PASS (`go test -tags=unit ./...` и `-tags "unit testhooks" ./db/postgres`)
- Integration tests: требуют Docker; запускаются по `make test-integration` — см. раздел «Тестирование»

Рекомендации по развитию (не блокеры):
- при необходимости добавить линтеры (golangci-lint) и CI‑workflow
- описать политику релизов/changelog
- расширить README примерами по security/mtls при необходимости
