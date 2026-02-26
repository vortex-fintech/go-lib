# AGENTS.md

Operational guide for agentic coding assistants working in this repository.

## Scope
- Modules (per go.work):
  - `./messaging/kafka/franzgo`
  - `./messaging/kafka/schemaregistry`
- Language: Go
- Go version: `go 1.25`
- Toolchain in `go.mod`: `go1.25.7`
- Main packages: `franzgo` (Kafka client), `schemaregistry` (Schema Registry + Protobuf serde)
- This repository is a shared library module (not an app entrypoint)

## Source-of-truth docs
- `README.md` (root messaging)
- `kafka/franzgo/README.md`
- `kafka/schemaregistry/README.md`

## Cursor / Copilot rules
- `.cursorrules`: not present
- `.cursor/rules/`: not present
- `.github/copilot-instructions.md`: not present
- If any are added later, treat them as higher-priority constraints than this file.

## Build / format / lint / test

### Build
- Build franzgo: `go build ./...` (in `messaging/kafka/franzgo`)
- Build schemaregistry: `go build ./...` (in `messaging/kafka/schemaregistry`)
- Build from repo root:
  - `go build ./messaging/kafka/franzgo/...`
  - `go build ./messaging/kafka/schemaregistry/...`

### Format
- Format code: `go fmt ./...`
- Optional formatting check: `gofmt -l .`

### Lint / static analysis
- Run vet: `go vet ./...`
- No `golangci-lint` config exists in this repo currently.

### Default tests (unit-style)
- Run franzgo tests: `go test ./...` (in `messaging/kafka/franzgo`)
- Run schemaregistry tests: `go test ./...` (in `messaging/kafka/schemaregistry`)
- Run from repo root:
  - `go test ./messaging/kafka/franzgo/...`
  - `go test ./messaging/kafka/schemaregistry/...`
- Run with race detector in Docker:
  ```bash
  docker run --rm -v "C:/Vortex Services/go-lib:/work" golang:1.25.7 \
      sh -c 'cd /work && go test -race ./messaging/kafka/franzgo/... && go test -race ./messaging/kafka/schemaregistry/...'
  ```

### Running a single test (important)
- Single franzgo test:
  - `go test ./... -run '^TestNewClient_Defaults$'` (in `messaging/kafka/franzgo`)
  - `go test ./messaging/kafka/franzgo/... -run '^TestProducer_ProduceBatch_NilRecord$'` (from root)
- Single schemaregistry test:
  - `go test ./... -run '^TestProtoSerializer_Caching$'` (in `messaging/kafka/schemaregistry`)
  - `go test ./messaging/kafka/schemaregistry/... -run '^TestProtoDeserializer_Deserialize$'` (from root)
- Prefix match:
  - `go test ./... -run '^TestNewClient_'` (in `messaging/kafka/franzgo`)

### Tagged tests (integration)
- Integration tests require a running Kafka broker.
- Run franzgo integration tests:
  - `KAFKA_BROKER=localhost:9092 go test -tags integration ./...` (in `messaging/kafka/franzgo`)
  - From root: `KAFKA_BROKER=localhost:9092 go test -tags integration ./messaging/kafka/franzgo/...`
- schemaregistry has unit tests only (no `-tags integration` needed).

### Integration environment setup
- Start Kafka (Redpanda) for integration tests:
  ```bash
  docker run -d --name redpanda-test -p 9092:9092 -p 9644:9644 \
      docker.redpanda.com/redpandadata/redpanda:v25.1.5 \
      redpanda start --overprovisioned --smp 1 --memory 1G --reserve-memory 0M \
      --node-id 0 --check=false \
      --kafka-addr PLAINTEXT://0.0.0.0:9092 \
      --advertise-kafka-addr PLAINTEXT://localhost:9092
  ```
- Verify Kafka is ready:
  - `docker exec redpanda-test rpk cluster info`
- Stop Kafka:
  - `docker rm -f redpanda-test`

### Integration test defaults and env vars
- Kafka broker: `localhost:9092` (override with `KAFKA_BROKER` env var)
- Schema Registry is not required for current tests (unit only for schemaregistry).

## Code style guidelines

### General style
- Follow idiomatic Go and keep code `gofmt`-clean.
- Prefer small focused functions and early returns.
- Keep behavior deterministic; avoid hidden side effects.
- Preserve backward-compatible public APIs unless explicitly changing contracts.
- **DO NOT ADD COMMENTS** unless explicitly asked.

### Imports
- Use standard `gofmt` grouping: stdlib, blank line, external modules.
- Alias imports for clarity or collisions (e.g., `kgo` for `github.com/twmb/franz-go/pkg/kgo`).
- Avoid dot imports.

### Formatting and layout
- Do not manually align spacing; let `gofmt` decide.
- Keep JSON struct tags on one line with spaces: `json:"field,omitempty"`.

### Types and interfaces
- Prefer concrete structs for domain/config payloads (`Config`, `Message`, `ProtoSerializer`).
- Keep interfaces narrow and capability-focused (`RegistryClient`).
- Use typed string constants for finite states.
- Keep zero-value semantics intentional and validated.

### Naming conventions
- Exported identifiers: `PascalCase`.
- Unexported identifiers: `camelCase`.
- Exported sentinel errors: `ErrXxx` (e.g., `ErrConsumerHandlerNil`, `ErrSchemaNotCached`).
- Test names: `Test<Subject>_<Scenario>` (e.g., `TestNewClient_GroupOnlyOptionsWithoutGroup`).
- Package-level constants: `PascalCase` for exported, `camelCase` for internal.

### Error handling
- Validate inputs up front and fail fast.
- Return sentinel errors for expected invalid-input paths.
- Wrap propagated errors with `%w` when adding context.
- Use `errors.Is` and `errors.As` instead of string matching.
- Provide detailed sentinel errors for diagnostics (e.g., `ErrConsumerClientNil`, `ErrProducerRecordNil`).

### Context, time, and cancellation
- Accept `context.Context` as the first argument of operations.
- Use `context.WithTimeout` around external I/O and cleanup paths.
- Always `defer cancel()` for derived contexts.
- Consumer `Consume` should respect context cancellation and return `ctx.Err()`.

### Kafka client patterns
- Use `franzgo.NewClient` with `Config` for initialization.
- Group-only options (`DisableAutoCommit`, `AutoCommitMarks`, `AutoCommitInterval`) require `ConsumerGroup` to be set.
- Always `defer client.Close()` after creating client.
- Use `client.Ping(ctx)` to verify connectivity.
- Producer uses `ProduceSync` for synchronous production.
- Consumer `Consume` returns fetch errors; do not silently ignore `fetches.Errors()`.

### Schema Registry patterns
- Use `schemaregistry.NewClient` with `Config.URL` for Schema Registry connection.
- Use `ProtoSerializer` for encoding Protobuf messages to Confluent wire format.
- Use `ProtoDeserializer` for decoding Confluent wire format to Protobuf payload.
- The serializer derives the message-index path from the Protobuf descriptor; do not hardcode `[0]`.
- Cache schema IDs per subject to avoid repeated registry lookups.

### Testing conventions
- Prefer table-driven tests for validation matrices.
- Use `t.Parallel()` for independent unit tests (optional).
- Use same-package tests for unexported behavior.
- Use external package tests (`<pkg>_test`) for public API validation.
- Use `t.Fatalf()` in same-package tests.
- For protobuf serializer tests, verify the message-index path matches the descriptor.
- Integration tests should use unique consumer group names (e.g., with `time.Now().UnixNano()`).

## Agent workflow expectations
- Before completing substantial changes, run:
  - `go test ./...` (in each module)
  - `go vet ./...`
- If integration behavior changes, run:
  - `KAFKA_BROKER=localhost:9092 go test -tags integration ./...` (in `messaging/kafka/franzgo`)
- If formatting-sensitive files change, run `go fmt ./...`.
- If exported behavior changes, update the relevant package README.
- Avoid adding new dependencies unless necessary.
- Run race tests in Docker before finalizing substantial changes.
