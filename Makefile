SHELL := bash
.SHELLFLAGS := -lc

MODULES := foundation security transport data runtime messaging/kafka/franzgo messaging/kafka/schemaregistry
POSTGRES_DC := docker compose -f data/postgres/docker-compose.test.yml
REDIS_DC := docker compose -f data/redis/docker-compose.test.yml
REDPANDA_NAME := redpanda-test
REDPANDA_IMAGE := docker.redpanda.com/redpandadata/redpanda:v25.1.5
KAFKA_BROKER ?= localhost:9092

.PHONY: all tidy build test test-unit vet test-integration-core test-integration test-race cover test-all up wait-redpanda down

all: tidy build

tidy:
	@set -euo pipefail; \
	for m in $(MODULES); do \
		(cd "$$m" && go mod tidy); \
	done

build:
	@set -euo pipefail; \
	for m in $(MODULES); do \
		(cd "$$m" && go build ./...); \
	done

test:
	@set -euo pipefail; \
	for m in $(MODULES); do \
		(cd "$$m" && go test -count=1 ./...); \
	done

test-unit:
	(cd foundation && go test -count=1 -tags unit ./...)
	(cd runtime && go test -count=1 -tags unit ./...)
	(cd data && go test -count=1 -tags "unit testhooks" ./postgres)

vet:
	@set -euo pipefail; \
	for m in $(MODULES); do \
		(cd "$$m" && go vet ./...); \
	done

test-integration-core:
	(cd data && go test -count=1 -tags integration ./...)
	(cd runtime && go test -count=1 -tags integration ./...)
	(cd messaging/kafka/franzgo && KAFKA_BROKER=$(KAFKA_BROKER) go test -count=1 -tags integration ./...)

test-integration:
	@set -euo pipefail; \
	trap '$(MAKE) --no-print-directory down >/dev/null 2>&1 || true' EXIT; \
	$(MAKE) --no-print-directory up; \
	$(MAKE) --no-print-directory test-integration-core

test-race:
	docker run --rm -v "$$(pwd):/work" golang:1.25.7 sh -c 'set -e; \
		cd /work/foundation && go mod download && go test -race -count=1 ./...; \
		cd /work/security && go mod download && go test -race -count=1 ./...; \
		cd /work/transport && go mod download && go test -race -count=1 ./...; \
		cd /work/data && go mod download && go test -race -count=1 ./...; \
		cd /work/runtime && go mod download && go test -race -count=1 ./...; \
		cd /work/messaging/kafka/franzgo && go mod download && go test -race -count=1 ./...; \
		cd /work/messaging/kafka/schemaregistry && go mod download && go test -race -count=1 ./...'

cover:
	@set -euo pipefail; \
	root="$$(pwd)"; \
	rm -rf "$$root/coverage"; \
	mkdir -p "$$root/coverage"; \
	for m in $(MODULES); do \
		out="$${m//\//_}"; \
		(cd "$$m" && go test -count=1 -coverprofile="$$root/coverage/$${out}.out" ./...); \
	done

test-all: test test-unit vet test-integration test-race

up:
	@set -euo pipefail; \
	$(POSTGRES_DC) up -d --wait --wait-timeout 90; \
	$(REDIS_DC) up -d --wait --wait-timeout 90; \
	docker rm -f $(REDPANDA_NAME) >/dev/null 2>&1 || true; \
	docker run -d --name $(REDPANDA_NAME) -p 9092:9092 -p 9644:9644 \
		$(REDPANDA_IMAGE) \
		redpanda start --overprovisioned --smp 1 --memory 1G --reserve-memory 0M \
		--node-id 0 --check=false \
		--kafka-addr PLAINTEXT://0.0.0.0:9092 \
		--advertise-kafka-addr PLAINTEXT://localhost:9092 >/dev/null; \
	$(MAKE) --no-print-directory wait-redpanda

wait-redpanda:
	@set -euo pipefail; \
	for i in $$(seq 1 30); do \
		if docker exec $(REDPANDA_NAME) rpk cluster info >/dev/null 2>&1; then \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	docker exec $(REDPANDA_NAME) rpk cluster info || true; \
	exit 1

down:
	@set -euo pipefail; \
	$(POSTGRES_DC) down -v || true; \
	$(REDIS_DC) down -v || true; \
	docker rm -f $(REDPANDA_NAME) >/dev/null 2>&1 || true
