# go-lib

Набор battle-tested библиотек, утилит и инфраструктурных компонентов, которые используются во всех Go‑сервисах Vortex. Здесь собраны решения для ошибок, сетевой безопасности, транспорта gRPC/HTTP, хранилищ данных, метрик, graceful shutdown и инструментов разработчика. Репозиторий можно подключать целиком или выборочно, импортируя только нужные пакеты.

## Содержание

- [go-lib](#go-lib)
  - [Содержание](#содержание)
  - [Назначение](#назначение)
  - [Основные возможности](#основные-возможности)
  - [Структура репозитория](#структура-репозитория)
  - [Требования и установка](#требования-и-установка)
  - [Быстрый старт](#быстрый-старт)
  - [Ключевые пакеты](#ключевые-пакеты)
    - [errors](#errors)
    - [grpc/middleware](#grpcmiddleware)
    - [security](#security)
    - [data layer](#data-layer)
    - [сервисная инфраструктура](#сервисная-инфраструктура)
    - [вспомогательные пакеты](#вспомогательные-пакеты)
  - [Наблюдаемость и эксплуатация](#наблюдаемость-и-эксплуатация)
  - [Тестирование и качество](#тестирование-и-качество)
    - [Быстрые проверки (PowerShell)](#быстрые-проверки-powershell)
    - [Makefile цели (Bash)](#makefile-цели-bash)
    - [Интеграционные тесты вручную](#интеграционные-тесты-вручную)
    - [Проверки перед релизом](#проверки-перед-релизом)
  - [Интеграция в CI/CD](#интеграция-в-cicd)
  - [Версионирование и совместимость](#версионирование-и-совместимость)
  - [Вклад и поддержка](#вклад-и-поддержка)
  - [Лицензия](#лицензия)

## Назначение

`go-lib` стандартизирует инфраструктурные слои, чтобы боевые сервисы могли сосредоточиться на бизнес‑логике и при этом соответствовали внутренним требованиям по безопасности, наблюдаемости и отказоустойчивости. Каждый пакет сопровождается тестами и может использоваться в продакшене без доработок.

## Основные возможности

- единый формат ошибок с адаптерами под HTTP/gRPC и EM-friendly полями;
- middleware-цепочки для gRPC (authz, circuit breaker, context cancel, metrics, error mapping);
- клиенты Postgres (pgxpool) и Redis с безопасной конфигурацией и helper’ами для транзакций;
- библиотека для JWT/JWKS, PoP (mTLS) и OBO‑политик, а также утилиты для HMAC‑одноразовых паролей;
- TLS helpers (сервер/клиент, динамическая перезагрузка, проверка цепочек) и анти‑replay механизмы;
- metrics handler с `/metrics` и `/health`, совместимый с Prometheus и Kubernetes probes;
- retry/time/net/hash/validator/logging утилиты, используемые всеми сервисами;
- инструменты graceful shutdown/metrics, покрывающие запуск, ожидание и мягкую остановку нескольких серверов.

## Структура репозитория

| Директория | Назначение |
| --- | --- |
| `security/scope` | Проверка скоупов и политики доступа на уровне бизнес‑операций. |
| `data/postgres` | Подключение к Postgres (pgxpool), runners, транзакции, тестовые хуки, Docker‑компоуз для интеграций. |
| `data/redis` | Клиент Redis с поддержкой TLS, Sentinel, Cluster и проверкой доступности. |
| `foundation/errors` | Конструкторы доменных ошибок, gRPC/HTTP адаптеры, пресеты и валидационные адаптеры. |
| `runtime/graceful` | Метрики graceful-цикла, менеджер остановки (`runtime/shutdown`) и адаптеры для HTTP/gRPC. |
| `transport/grpc` | Набор middleware (authz, chain, circuit breaker, context cancel, errors, metrics), dialer и metadata/helpers. |
| `foundation/hash` | SHA‑256 утилиты и тесты. |
| `foundation/logger` | Обёртка над zap с безопасным синком и профилями окружений. |
| `foundation/logutil` | Санитизация/редакция данных перед логированием. |
| `runtime/metrics` | HTTP‑handler для `/metrics` и `/health` с встроенными стандартными метриками. |
| `foundation/netutil` | Нормализация таймаутов и сетевые helpers. |
| `foundation/retry` | Экспоненциальные и быстрые ретраи, уважающие контекст. |
| `security/hmacotp` | Генерация и проверка HMAC‑одноразовых кодов (e.g. device binding). |
| `security/jwt` | JWKS‑верификатор, строгая OBO‑валидация, PoP-утилиты. |
| `security/mtls` | mTLS конфигурации, загрузка сертификатов, hot reload, helpers для тестов. |
| `security/replay` | Защита от повторного воспроизведения запросов. |
| `security/tlsutil` | Расчёт `x5t` и связанные TLS‑утилиты. |
| `foundation/timeutil` | UTC‑clock, sleep с отменой и helpers для тестирования времени. |
| `foundation/validator` | Надстройка над `go-playground/validator` с маппингом тэгов в коды ошибок. |

В корне также находятся `Makefile` для унификации тестов/линтеров, `LICENSE` (MIT), `go.work` и `go.mod` внутри каждого модуля.

## Требования и установка

- Go 1.25+ (используем toolchain `go1.25.1`).
- Docker + Docker Compose (для интеграционных тестов Postgres).
- Git Bash/WSL или иная Bash‑совместимая оболочка для команд `make`.

Установка:

```bash
go get github.com/vortex-fintech/go-lib/foundation@latest
```

После установки импортируйте только нужные подпакеты — репозиторий разделён на независимые модули (`foundation`, `security`, `transport`, `data`, `runtime`) и не тянет лишние зависимости.

## Быстрый старт

```go
package main

import (
    "context"
    "time"

    gliberr "github.com/vortex-fintech/go-lib/foundation/errors"
    "github.com/vortex-fintech/go-lib/data/postgres"
    metricsmw "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metricsmw"
    promrep "github.com/vortex-fintech/go-lib/transport/grpc/middleware/metricsmw/promreporter"
    chain "github.com/vortex-fintech/go-lib/transport/grpc/middleware/chain"
    "google.golang.org/grpc"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    db, err := postgres.Open(ctx, "postgres://user:pass@localhost:5432/app?sslmode=disable")
    if err != nil {
        panic(gliberr.Internal().WithReason("db_unavailable"))
    }
    defer db.Close()

    rep := promrep.NewDefault()
    srv := grpc.NewServer(chain.Default(chain.Options{
        Pre: []grpc.UnaryServerInterceptor{metricsmw.UnaryFull(rep)},
    }))

    _ = srv // регистрируйте сервисы, используйте db.Runner() для работы с БД
}
```

Код выше демонстрирует типичный каркас: готовые ошибки, безопасный Postgres‑клиент и gRPC‑сервер с метриками. Остальные подсистемы (authz, logger, graceful shutdown) подключаются аналогичным образом.

## Ключевые пакеты

### errors

- конструкторы `Internal()`, `InvalidArgument()`, `Unauthenticated()` и др. возвращают неизменяемые шаблоны;
- методы `WithReason`, `WithDomain`, `WithDetails`, `WithViolations` формируют payload для клиентов;
- адаптеры `ToGRPC`, `ToHTTP`, `FromStatus` обеспечивают симметрию между протоколами;
- предусмотрены пресеты и адаптеры валидации (`validation_adapters.go`).

```go
func CreateUser() error {
    return gliberr.InvalidArgument().
        WithReason("invalid_email").
        WithViolations([]gliberr.FieldViolation{{Field: "email", Reason: "invalid"}})
}
```

### grpc/middleware

- `chain`: правильная сборка pre/post цепочек и authz‑интерсептора;
- `authz`: проверка OBO‑JWT, PoP (мэппинг `x5t#S256` с TLS), скоупы и identity helpers;
- `circuitbreaker`: half-open стратегия, лимиты попыток и метрики ошибок;
- `contextcancel`: гарантирует отмену RPC при потере клиента;
- `errorsmw`: конвертирует доменные ошибки в gRPC status/k8s-friendly сообщения;
- `metricsmw` + `promreporter`: готовые счетчики/гистограммы для Prometheus с минимальными настройками.

### security

- `security/jwt`: JWKS‑клиент с кэшированием, проверкой хедеров (kid, alg), поддержкой ETag/Cache-Control, функциями `ValidateOBO` и PoP (`X5tS256FromCert`).
- `security/hmacotp`: безопасные одноразовые коды (включая window, TTL, попытки) для device binding и подтверждений операций.
- `security/mtls`: загрузчик TLS‑материалов, валидация цепочек, клиентские и серверные конфиги, hot reload, `test_helpers.go` для integration tests.
- `security/replay`: реплей-протекция с хранением отпечатков и TTL.
- `security/tlsutil`: расчёт `x5t` и вспомогательные функции для PoP и mutual TLS.

### data layer

- `data/postgres`: работа через `pgxpool.Pool`, runners (`RunnerFromPool`, `RunnerFromConn`), транзакции `WithTx`, обработка ошибок `IsUniqueViolation`, `Constraint`. В комплекте build-теги `unit`, `integration`, `testhooks` и docker-compose для CI.
- `data/redis`: создание клиента с пингом и поддержкой TLS 1.2+, Sentinel/Cluster, graceful закрытие.

### сервисная инфраструктура

- `runtime/shutdown`: менеджер, который запускает/останавливает несколько серверов, подписывается на SIGINT/SIGTERM, умеет различать «нормальные» ошибки (`http.ErrServerClosed`) и фатальные.
- `runtime/graceful`: метрики времени остановки/ожидания, репортинг в Prometheus.
- `runtime/metrics`: HTTP‑handler комбинирует `/metrics` и `/health` (GET/HEAD), автоматически регистрирует Go/process метрики и предоставляет простой API для ваших health‑проверок.
- `foundation/logger`: инициализация zap логгера с безопасным `SafeSync` и пресетами профилей (`development`, `debug`, `production`).
- `foundation/retry`, `foundation/timeutil`, `foundation/netutil`: контролируемые ретраи, mockable часы, санитария таймаутов — используются во всех остальных пакетах.

### вспомогательные пакеты

- `security/scope`: набор структур для описания политик (All/Any/Gloabl scopes) и функция `checker` для их проверки;
- `foundation/hash`, `foundation/logutil`, `foundation/validator`, `transport/grpc/metadata`, `transport/grpc/dial`, `transport/grpc/creds` и др. обеспечивают мелкие, но важные куски инфраструктуры.

## Наблюдаемость и эксплуатация

- `/metrics` и `/health` поднимаются вызовом `runtime/metrics.New`, который возвращает mux и Prometheus registerer. Таймаут health по умолчанию 500 мс, маршруты поддерживают только GET/HEAD.
- gRPC наблюдаемость достигается комбинацией `metricsmw` и `promreporter`. Пакет предоставляет интерфейсы, совместимые с нашими стандартами именования.
- `runtime/graceful` дополнительно публикует время запуска/остановки серверов и количество активных goroutine.
- Логирование централизовано через `logger.Init(name, env)` — в production профиле включены JSON-логи, stacktrace только для ошибок, а SafeSync гарантирует flush.

## Тестирование и качество

Проект разделяет юнит‑, интеграционные и race‑тесты через build‑теги и цели Makefile.

### Быстрые проверки (PowerShell)

```powershell
go test -count=1 -tags=unit ./...
go test -count=1 -tags "unit testhooks" ./data/postgres
go vet ./...
```

### Makefile цели (Bash)

- `make test` — все unit + testhooks для Postgres;
- `make test-integration` — поднимает Docker с Postgres, запускает `go test -tags=integration ./...`, выключает инфраструктуру;
- `make test-all` — unit + integration последовательно;
- `make test-race` — unit и testhooks с включённой `-race`;
- `make cover` — собирает отчёты покрытия и открывает `coverage.html`.

### Интеграционные тесты вручную

```powershell
docker compose -f data/postgres/docker-compose.test.yml up -d --wait --wait-timeout 60
go test -count=1 -tags integration ./...
docker compose -f data/postgres/docker-compose.test.yml down -v
```

### Проверки перед релизом

- `go build ./...`
- `go vet ./...`
- `go test -count=1 -tags=unit ./...`
- `go test -count=1 -tags "unit testhooks" ./data/postgres`
- `make test-integration` (при изменениях БД или TLS инфраструктуры)

## Интеграция в CI/CD

1. Подключите `go env -w GOTOOLCHAIN=auto` либо используйте локальный toolchain 1.25.1 (см. `go.work` и `go.mod` в модулях).
2. Выполняйте `go test ./...` с нужными тегами в параллельных job’ах (unit/testhooks/integration).
3. При необходимости запускайте `make up` для развёртывания Postgres перед интеграционными тестами и `make down`, чтобы гарантированно очистить тома.
4. Публикуйте артефакты покрытий (`coverage.out`, `coverage.dbpgx.out`) для анализа качества.

## Версионирование и совместимость

- Модули `github.com/vortex-fintech/go-lib/{foundation,security,transport,data,runtime}` следуют SemVer.
- Мажорные релизы могут вносить ломающие изменения, но внутри одного мажора API стабильны.
- Минимальная поддерживаемая версия Go — 1.25. Старшие версии также поддерживаются (используется `toolchain go1.25.1`).

## Вклад и поддержка

- Issues и Pull Requests приветствуются; старайтесь прикладывать тесты и описывать сценарии отказа.
- Перед PR запускайте unit + testhooks + integration (если затронуты БД или TLS пакеты).
- Security‑вопросы лучше отправлять приватно (см. внутренние инструкции команды безопасности).

## Лицензия

MIT — подробности в `LICENSE`.
