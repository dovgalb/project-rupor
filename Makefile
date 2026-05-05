.PHONY: run build test lint fmt vet migrate-up migrate-down sqlc dc-up dc-down dc-logs install-hooks init-project help

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
	@echo "  install-hooks — включить git-хуки из .githooks/"
	@echo "  init-project  — полная инициализация окружения (env, deps, hooks, db, migrations)"

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

install-hooks:
	git config core.hooksPath .githooks
	@echo "git hooks подключены из .githooks/"

init-project:
	@echo "==> 1/6 Проверка обязательных инструментов"
	@command -v go >/dev/null 2>&1 || { echo "ERROR: go не установлен (нужен Go 1.25+)"; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "ERROR: docker не установлен"; exit 1; }
	@command -v migrate >/dev/null 2>&1 || { echo "ERROR: golang-migrate не установлен. brew install golang-migrate"; exit 1; }
	@echo "==> 2/6 Проверка опциональных инструментов"
	@command -v golangci-lint >/dev/null 2>&1 || echo "WARN: golangci-lint не установлен (нужен для pre-commit). brew install golangci-lint"
	@command -v sqlc >/dev/null 2>&1 || echo "WARN: sqlc не установлен. brew install sqlc"
	@echo "==> 3/6 Подготовка .env"
	@if [ ! -f .env ]; then cp .env.example .env && echo "    .env создан из .env.example"; else echo "    .env уже существует, пропуск"; fi
	@echo "==> 4/6 Загрузка Go-зависимостей"
	go mod download
	@echo "==> 5/6 Подключение git-хуков"
	@$(MAKE) install-hooks
	@echo "==> 6/6 Запуск БД и миграций"
	@$(MAKE) dc-up
	@echo "    ожидание готовности PostgreSQL..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		docker compose exec -T postgres pg_isready -U rupor >/dev/null 2>&1 && break || sleep 1; \
	done
	@$(MAKE) migrate-up
	@echo ""
	@echo "Готово. Запусти сервер: make run"
