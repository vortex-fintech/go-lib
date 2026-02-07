APP_NAME := go-lib
PKG := ./...
MODULES := foundation security transport data runtime
DC := docker compose -f data/postgres/docker-compose.test.yml
CONTAINER := go-lib-test-postgres

# Используем bash (Git Bash / WSL)
SHELL := bash
.SHELLFLAGS := -lc

.PHONY: all tidy build test test-integration test-integration-core test-all test-race cover up wait-db down

# === Базовые ===
all: tidy build

tidy:
	@set -e; \
	for m in $(MODULES); do \
		(cd $$m && go mod tidy); \
	done

build:
	go build -v $(PKG)

# === Тесты ===
# Юнит-тесты (включая testhooks)
test:
	go test -count=1 -tags=unit -v $(PKG)
	go test -count=1 -tags="unit testhooks" -v ./data/postgres

# Интеграция с Postgres
test-integration: up wait-db
	@set -e; \
	go test -count=1 -tags=integration -v $(PKG); \
	status=$$?; \
	$(DC) down -v; \
	exit $$status

# Интеграция без инфраструктуры
test-integration-core:
	go test -count=1 -tags=integration -v $(PKG)

# Все тесты по порядку
test-all:
	$(MAKE) test
	$(MAKE) test-integration

# С гонщиком (race)
test-race:
	go test -race -count=1 -tags=unit $(PKG)
	go test -race -count=1 -tags="unit testhooks" -v ./data/postgres

# Покрытие
cover:
	go test -count=1 -coverprofile=coverage.out -tags=unit $(PKG)
	go test -count=1 -coverprofile=coverage.dbpgx.out -tags="unit testhooks" ./data/postgres
	go tool cover -html=coverage.out -o coverage.html
	@echo "Открой coverage.html в браузере."

# === Инфраструктура ===
up:
	$(DC) up -d --wait --wait-timeout 60

wait-db: ; @echo "Postgres is up (compose --wait handled it)."

down:
	$(DC) down -v
