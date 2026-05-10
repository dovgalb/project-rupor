---
phase: 1
name: Build & Tooling Infra
layer: infrastructure
depends_on: none
plan: ./README.md
---

# Phase 1: Build & Tooling Infra

## Цель

Создать сборочную инфраструктуру монорепо: конфиги линтера, Makefile, docker-compose с PostgreSQL, sqlc-конфиг, .gitignore, .env.example, CI workflow. После этой фазы доступны команды `make lint`, `make dc-up`, `make dc-down`. Go-кода и миграций ещё нет.

## Контекст

Стартовое состояние репо: только `go.mod` с `module github.com/dovgalb/project-rupor` и `go 1.25`, без `require`. Никаких сборочных артефактов нет (см. `.thoughts/research/2026-05-04-project-structure.md`).

Эта фаза не создаёт Go-код, поэтому `go.mod` и `go.sum` не трогаются.

## Файлы для создания

### `.gitignore`

**Назначение:** исключения для Go-проекта. Все стандартные паттерны + `.env`, `.idea/*` (кроме `.idea/.gitignore`, который уже tracked), build-артефакты.

**Детали реализации:**

```gitignore
# Build artifacts
/bin/
/dist/
/coverage.out
/coverage.html
*.test
*.out

# Go
/vendor/

# Env
/.env
/.env.local
!/.env.example

# OS
.DS_Store
Thumbs.db

# IDE — оставляем .idea/.gitignore tracked
/.idea/*
!/.idea/.gitignore

# Logs
*.log

# tmp
/tmp/
```

### `.env.example`

**Назначение:** шаблон env-переменных, который разработчик копирует в `.env` после клона. Дефолты соответствуют `docker-compose.yml`.

**Детали реализации:**

```env
# PostgreSQL connection string (используется make migrate-up и приложением)
DATABASE_URL=postgres://rupor:rupor@localhost:5432/rupor?sslmode=disable

# Секрет для подписи JWT-токенов. В проде заменить на криптостойкий.
JWT_SECRET=change-me-in-prod

# Порт HTTP-сервера (опционально, default 8080)
SERVER_PORT=8080
```

Никаких реальных секретов в `.env.example` — это шаблон. В проде значения подаются через ENV в systemd/docker/k8s.

### `.golangci.yml`

**Назначение:** конфигурация `golangci-lint` с минимальным набором линтеров (ADR-008).

**Детали реализации:**

```yaml
run:
  timeout: 3m
  go: "1.25"

linters:
  disable-all: true
  enable:
    - govet
    - staticcheck
    - errcheck
    - ineffassign
    - unused
    - gofmt
    - goimports

linters-settings:
  goimports:
    local-prefixes: github.com/dovgalb/project-rupor
  govet:
    enable-all: true

issues:
  exclude-use-default: false
```

Опция `goimports.local-prefixes` группирует импорты проекта отдельно от stdlib и сторонних — единый стиль.

### `Makefile`

**Назначение:** все основные команды разработчика. Target по `prompts/Go style.txt:122-125` (`gofmt`, `go vet`, `golangci-lint run`, `go test`).

**Детали реализации:**

```makefile
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
```

Замечания:
- `run`/`build` ссылаются на `./cmd/server`, который появится в Phase 4. До тех пор `make run` будет падать — это ожидаемо.
- `migrate-up` ссылается на `migrations/`, появятся в Phase 2.
- `sqlc generate` потребует валидного `sqlc.yaml` (создаётся в этой же фазе).

### `docker-compose.yml`

**Назначение:** локальная PostgreSQL для разработки.

**Детали реализации:**

```yaml
services:
  postgres:
    image: postgres:16-alpine
    container_name: rupor-postgres
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-rupor}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-rupor}
      POSTGRES_DB: ${POSTGRES_DB:-rupor}
    ports:
      - "${POSTGRES_PORT:-5432}:5432"
    volumes:
      - rupor-pg-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-rupor}"]
      interval: 5s
      timeout: 3s
      retries: 5

volumes:
  rupor-pg-data:
```

Версия `postgres:16-alpine` (ADR-009 — встроенный `pgcrypto`). Маппинг порта через `${POSTGRES_PORT:-5432}` решает проблему конфликта с локально установленным Postgres (Q-edge case в `../02-behavior.md`).

### `sqlc.yaml`

**Назначение:** конфигурация sqlc для генерации Go-кода из SQL.

**Детали реализации (вариант с пустым массивом, проверяется на реализации):**

```yaml
version: "2"
sql: []
```

**Fallback (если `sql: []` не валиден для sqlc v2)** — закомментированный шаблон per-domain:

```yaml
version: "2"
sql:
  # При появлении первого домена раскомментируй и заполни.
  #
  # - schema: "./migrations"
  #   queries: "./internal/<domain>/repository/postgres/queries"
  #   engine: "postgresql"
  #   gen:
  #     go:
  #       package: "db"
  #       out: "./internal/<domain>/repository/postgres/db"
  #       sql_package: "pgx/v5"
  - schema: "./migrations"
    queries: "./.sqlc-placeholder"
    engine: "postgresql"
    gen:
      go:
        package: "placeholder"
        out: "./.sqlc-placeholder/gen"
```

Если выбран fallback — создать пустую папку `./.sqlc-placeholder/` с `.gitkeep` и добавить её в `.gitignore`. Это ад-хок-обходной путь до появления первого реального домена в Фазе 1.2.

**Verification:**
- `sqlc compile` (или `sqlc generate --dry-run`) проходит без ошибок на выбранном варианте.
- Если `sql: []` не работает — переключиться на fallback и зафиксировать в commit-сообщении.

### `.github/workflows/ci.yml`

**Назначение:** CI на push/PR в `main` (ADR-011).

**Детали реализации:**

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build-test-lint:
    name: Build, Test, Lint
    runs-on: ubuntu-latest
    timeout-minutes: 10
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.25"
          cache: true

      - name: go mod download
        run: go mod download

      - name: go build
        run: go build ./...

      - name: go test
        run: go test ./... -race -count=1

      - uses: golangci/golangci-lint-action@v6
        with:
          version: v1.62
          args: --timeout=3m
```

Замечания:
- `actions/setup-go@v5` кэширует `~/go/pkg/mod` по `go.sum` — после первого прогона CI быстрый.
- `golangci-lint-action@v6` ставит свою версию линтера, не зависим от того, что у разработчика локально.
- `go.sum` появится в Phase 4 после `go get chi/v5`. До тех пор CI будет работать с пустым go.sum (нет зависимостей).
- В `go test ./...` пока нет тестов (Phase 3 добавит). На пустом проекте `go test ./...` возвращает `no test files` — exit 0, не failure.

## Файлы для модификации

В этой фазе модификаций существующих файлов нет.

## Ключевые решения

- **ADR-001** (раскладка папок) — папки `internal/<домен>/<слой>/` создаются в Phase 2, не здесь.
- **ADR-005** (sqlc + golang-migrate) — `sqlc.yaml` создаётся, миграции — в Phase 2.
- **ADR-008** (минимальный линт) — выбраны 7 линтеров (см. выше).
- **ADR-010** (golang-migrate как локальный CLI) — Makefile вызывает `migrate` напрямую.
- **ADR-011** (CI на push/PR в main) — workflow в этой фазе.
- **ADR-013** (.env.example) — создаётся здесь, в `.gitignore` запрет реального `.env`.

См. `../03-decisions.md` для полного контекста.

## Verification

- [ ] `golangci-lint run` — exit 0 (на пустом проекте 0 issues)
- [ ] `gofmt -l .` — пустой вывод
- [ ] `make help` показывает список целей
- [ ] `make dc-up` поднимает контейнер `rupor-postgres`, `docker ps` показывает его в `Up (healthy)` через 5–15с
- [ ] `make dc-down` останавливает и удаляет контейнер
- [ ] `docker compose config` — валидирует синтаксис без warnings
- [ ] `sqlc compile` (или `sqlc generate --dry-run`) — exit 0 на выбранном варианте `sqlc.yaml`
- [ ] `cat .gitignore | grep -E "^/.env$"` — паттерн запрета `.env` присутствует
- [ ] `cat .gitignore | grep -E "^!/.env.example$"` — `.env.example` НЕ игнорируется
- [ ] `cat .env.example` — содержит три обязательных переменные
- [ ] `cat .github/workflows/ci.yml | yq '.jobs."build-test-lint".steps | length'` ≥ 6 шагов (или просто прочитать визуально — должны быть checkout/setup-go/download/build/test/lint)
- [ ] `make lint` — exit 0 на пустом проекте
- [ ] `make test` — exit 0 (no test files — это норма)
- [ ] Все 7 файлов созданы и tracked в git: `.gitignore`, `.env.example`, `.golangci.yml`, `Makefile`, `docker-compose.yml`, `sqlc.yaml`, `.github/workflows/ci.yml`

## Что НЕ делает эта фаза

- Не создаёт папки `internal/`, `pkg/`, `migrations/` — это Phase 2.
- Не добавляет Go-зависимости в `go.mod` — это Phase 4 (chi).
- Не создаёт `cmd/server/` — это Phase 4.
- Не модифицирует существующий `README.MD` — это Phase 5.
