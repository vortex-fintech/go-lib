APP_NAME := go-lib
PKG := ./...
DC := docker-compose -f db/postgres/docker-compose.test.yml
CONTAINER := go-lib-test-postgres

.PHONY: all tidy build test test-integration up down wait-db test-all test-race cover test-integration-core

all: tidy build

tidy:
	go mod tidy

build:
	go build -v $(PKG)

# Юнит-тесты (в т.ч. Name() и т.п.)
test:
	go test -tags=unit -v $(PKG)

# Интеграция (Manager+gRPC, сигнал на Unix). Если не нужен Postgres — используй test-integration-core.
test-integration: up wait-db
	go test -tags=integration -v $(PKG)
	$(MAKE) down

# Интеграция без инфраструктуры (если БД не требуется)
test-integration-core:
	go test -tags=integration -v $(PKG)

# Последовательно все тесты
test-all:
	$(MAKE) test
	$(MAKE) test-integration

# С гонщиком
test-race:
	go test -race -tags=unit $(PKG)
	go test -race -tags=integration $(PKG)

# Покрытие по юнитам (при желании добавь интеграцию отдельно)
cover:
	go test -coverprofile=coverage.out -tags=unit $(PKG)
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser"

up:
	$(DC) up -d

down:
	$(DC) down

wait-db:
	@echo "Waiting for Postgres to be healthy..."
	@until docker inspect --format='{{.State.Health.Status}}' $(CONTAINER) | grep -q healthy; do sleep 1; done
