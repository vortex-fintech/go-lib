# Development

Этот документ описывает команды для разработки и проверки workspace `go-lib`.

## Requirements

- Go `1.25`
- Toolchain `go1.25.7`
- Docker (для integration/race сценариев)

## Modules

```bash
foundation
security
transport
data
runtime
messaging/kafka/franzgo
messaging/kafka/schemaregistry
```

## Download dependencies

```bash
for m in foundation security transport data runtime messaging/kafka/franzgo messaging/kafka/schemaregistry; do
  (cd "$m" && go mod download)
done
```

## Build

```bash
for m in foundation security transport data runtime messaging/kafka/franzgo messaging/kafka/schemaregistry; do
  (cd "$m" && go build ./...)
done
```

## Tests

### Default sweep

```bash
for m in foundation security transport data runtime messaging/kafka/franzgo messaging/kafka/schemaregistry; do
  (cd "$m" && go test -count=1 ./...)
done
```

### Unit-tagged suites

```bash
(cd foundation && go test -count=1 -tags unit ./...)
(cd runtime && go test -count=1 -tags unit ./...)
(cd data && go test -count=1 -tags "unit testhooks" ./postgres)
```

## Vet

```bash
for m in foundation security transport data runtime messaging/kafka/franzgo messaging/kafka/schemaregistry; do
  (cd "$m" && go vet ./...)
done
```

## Integration tests (Docker)

### Start infrastructure

```bash
docker compose -f data/postgres/docker-compose.test.yml up -d
docker compose -f data/redis/docker-compose.test.yml up -d
docker run -d --name redpanda-test -p 9092:9092 -p 9644:9644 \
  docker.redpanda.com/redpandadata/redpanda:v25.1.5 \
  redpanda start --overprovisioned --smp 1 --memory 1G --reserve-memory 0M \
  --node-id 0 --check=false \
  --kafka-addr PLAINTEXT://0.0.0.0:9092 \
  --advertise-kafka-addr PLAINTEXT://localhost:9092
```

### Run integration suites

```bash
(cd data && go test -count=1 -tags integration ./...)
(cd runtime && go test -count=1 -tags integration ./...)
(cd messaging/kafka/franzgo && KAFKA_BROKER=localhost:9092 go test -count=1 -tags integration ./...)
```

### Cleanup

```bash
docker compose -f data/postgres/docker-compose.test.yml down -v
docker compose -f data/redis/docker-compose.test.yml down -v
docker rm -f redpanda-test
```

## Race tests (Docker)

```bash
docker run --rm -v "${PWD}:/work" golang:1.25.7 sh -c '
set -e
cd /work/foundation && go mod download && go test -race -count=1 ./...
cd /work/security && go mod download && go test -race -count=1 ./...
cd /work/transport && go mod download && go test -race -count=1 ./...
cd /work/data && go mod download && go test -race -count=1 ./...
cd /work/runtime && go mod download && go test -race -count=1 ./...
cd /work/messaging/kafka/franzgo && go mod download && go test -race -count=1 ./...
cd /work/messaging/kafka/schemaregistry && go mod download && go test -race -count=1 ./...
'
```

## CI/CD workflows

Файлы GitHub Actions находятся в `.github/workflows`:

- `ci.yml`
- `integration-race.yml`
- `release.yml`
