# go-lib

`go-lib` — набор переиспользуемых Go-модулей для backend-сервисов: базовые утилиты, безопасность, транспорт, доступ к данным, runtime-компоненты и Kafka-интеграции.

## Workspace overview

Этот репозиторий — Go workspace (`go.work`) из нескольких независимых модулей, которые можно подключать по отдельности:

- `foundation`
- `security`
- `transport`
- `data`
- `runtime`
- `messaging/kafka/franzgo`
- `messaging/kafka/schemaregistry`

## Quick start

### Requirements

- Go `1.25`
- Toolchain `go1.25.7`

### Подключение одного модуля

Обычно сервису нужен только один или несколько конкретных модулей, а не весь workspace.

```bash
go get github.com/vortex-fintech/go-lib/foundation@latest
```

Пример `go.mod`:

```go
module your/service

go 1.25

toolchain go1.25.7

require github.com/vortex-fintech/go-lib/foundation vX.Y.Z
```

### Минимальный пример использования

```go
package main

import (
	"fmt"

	"github.com/vortex-fintech/go-lib/foundation/timeutil"
)

func main() {
	fmt.Println(timeutil.NowUTC())
}
```

## Module guide

- [`foundation`](foundation/README.md) — общие примитивы: ошибки, логирование, валидация, retry/time/text/domain-утилиты.
- [`security`](security/README.md) — JWT/JWKS, mTLS, replay protection и scopes для защиты API.
- [`transport`](transport/README.md) — инфраструктура gRPC: metadata, credentials, dial helpers, middleware.
- [`data`](data/README.md) — клиенты Postgres/Redis и механизмы idempotency.
- [`runtime`](runtime/README.md) — graceful shutdown, health/metrics handlers и runtime-паттерны.
- [`messaging`](messaging/README.md) — Kafka-интеграции (`franzgo`) и Schema Registry/protobuf serde.

## Versioning & compatibility

- Каждый модуль versioned как отдельный Go module и может релизиться независимо.
- Внутри одного сервиса рекомендуется фиксировать версии модулей явно в `go.mod`.
- Для информации по доступным версиям/релизам смотрите теги и GitHub Releases репозитория.

## Development

Подробные команды для локальной разработки, CI/test orchestration, Docker-интеграций и race-проверок вынесены в [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
