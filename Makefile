APP_NAME := go-lib
PKG := ./...
DC := docker-compose -f db/postgres/docker-compose.test.yml
CONTAINER := go-lib-test-postgres

.PHONY: all tidy build test test-integration up down wait-db

all: tidy build

tidy:
	go mod tidy

build:
	go build -v $(PKG)

test:
	go test -tags=unit -v $(PKG)

test-integration: up wait-db
	go test -tags=integration -v $(PKG)
	$(MAKE) down

up:
	$(DC) up -d

down:
	$(DC) down

wait-db:
	@echo "Waiting for Postgres to be healthy..."
	@until docker inspect --format='{{.State.Health.Status}}' $(CONTAINER) | grep -q healthy; do sleep 1; done
