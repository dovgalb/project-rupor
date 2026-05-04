.PHONY: run build test lint fmt vet migrate-up migrate-down sqlc dc-up dc-down dc-logs help

# Загружаем .env, если есть. Не падаем, если нет.
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

DATABASE_URL ?= postgres://rupor:rupor@localhost:5432/rupor?sslmode=disable
SERVER_PORT  ?= 8080

help:
	@echo "Available targets:"
	@echo "  run           — go run ./cmd/server"
	@echo "  build         — собрать бинарник в bin/server"
	@echo "  test          — go test ./... -race -count=1"
	@echo "  lint          — golangci-lint run"
	@echo "  fmt           — gofmt + goimports"
	@echo "  vet           — go vet ./..."
	@echo "  migrate-up    — накатить миграции"
	@echo "  migrate-down  — откатить последнюю миграцию"
	@echo "  sqlc          — sqlc generate"
	@echo "  dc-up         — docker compose up -d"
	@echo "  dc-down       — docker compose down"
	@echo "  dc-logs       — docker compose logs -f"

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -race -count=1

lint:
	golangci-lint run

fmt:
	gofmt -w .
	goimports -w -local github.com/dovgalb/project-rupor .

vet:
	go vet ./...

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

sqlc:
	sqlc generate

dc-up:
	docker compose up -d

dc-down:
	docker compose down

dc-logs:
	docker compose logs -f
