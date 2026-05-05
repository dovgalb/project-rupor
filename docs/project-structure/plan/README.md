---
date: 2026-05-04
feature: project-structure
design: ../README.md
status: draft
---

# План кода: Project Structure

## Overview

Реализация скелета монорепо по дизайну в `../README.md`. После прохождения всех фаз `make dc-up && make migrate-up && make run && curl localhost:8080/api/v1/health` отрабатывает end-to-end и возвращает `{"status":"ok"}`. CI на push/PR в `main` зелёный.

Дизайн-документы:
- `../01-architecture.md` — структура папок, граф зависимостей слоёв
- `../02-behavior.md` — sequence-диаграммы запуска, миграций, graceful shutdown, CI
- `../03-decisions.md` — 14 ADR (раскладка папок, sqlc, chi, env-конфиг и пр.)
- `../04-testing.md` — coverage mapping и список тестов
- `../07-standards.md` — compliance-матрица по `prompts/`
- `../08-api-contract.md` — контракт `GET /api/v1/health`

## Phase Strategy

**Bottom-up адаптированный**: сборочная инфра → папочный скелет → пакет конфигурации → точка входа → доводка документации.

Почему именно так:
- **Phase 1 (Build & Tooling) первой** — она даёт `.golangci.yml`, без которого `make lint` не пройдёт в последующих фазах. Также вводит `Makefile` и `docker-compose.yml`, на которые ссылаются остальные фазы.
- **Phase 2 (Repo Skeleton) до Phase 3** — потому что `internal/<домен>/` папки и initial-миграция логически предшествуют коду. После Phase 2 `make migrate-up` накатывает `0001_init.up.sql`.
- **Phase 3 (Config) до Phase 4 (Server)** — `cmd/server/main.go` импортирует `config.Load`. Каждая фаза тестируется изолированно: после Phase 3 `go test ./config/...` зелёный без `cmd/server/`.
- **Phase 4 (Server) последняя из реализации** — собирает всё вместе, добавляет `chi/v5` в `go.mod`, запускает `make run`.
- **Phase 5 (Doc & Cleanup) — финальная** — обновляет `README.MD` с quickstart, удаляет `init_project`. Не блокирует основные критерии приёмки, но закрывает Suggestions из architect review.

## Phases

| # | Фаза | Слой | Зависимости | Status |
|---|------|------|-------------|--------|
| 1 | [Build & Tooling Infra](./phase-01.md) | infrastructure | none | ☐ |
| 2 | [Repo Skeleton](./phase-02.md) | infrastructure | phase-01 | ☐ |
| 3 | [Config Package](./phase-03.md) | application | phase-01 | ☐ |
| 4 | [Server Entry Point](./phase-04.md) | composition root | phase-01, phase-03 | ☐ |
| 5 | [Doc & Cleanup](./phase-05.md) | docs | phase-04 | ☐ |

## File Map

### New Files

**Phase 1 — Build & Tooling Infra:**
- `.gitignore` — стандартные исключения для Go-проекта (bin/, vendor/, .env, *.test, coverage.out, .DS_Store, .idea/* кроме .gitignore)
- `.env.example` — шаблон env с дефолтами для dev (DATABASE_URL, JWT_SECRET, SERVER_PORT)
- `.golangci.yml` — конфиг golangci-lint, набор линтеров: govet, staticcheck, errcheck, ineffassign, unused, gofmt, goimports
- `Makefile` — цели run, build, test, lint, migrate-up, migrate-down, sqlc, dc-up, dc-down, dc-logs
- `docker-compose.yml` — сервис postgres:16-alpine с healthcheck
- `sqlc.yaml` — version: 2, заготовка с закомментированным шаблоном per-domain блока
- `.github/workflows/ci.yml` — workflow `Build & Test`: setup-go 1.25 → go build → go test -race → golangci-lint

**Phase 2 — Repo Skeleton:**
- `internal/auth/domain/.gitkeep`
- `internal/auth/usecase/.gitkeep`
- `internal/auth/transport/http/.gitkeep`
- `internal/auth/repository/postgres/.gitkeep`
- (то же × 6 доменов: auth, user, room, channel, chat, voice — итого 24 файла)
- `pkg/websocket/.gitkeep`
- `migrations/0001_init.up.sql` — `CREATE EXTENSION IF NOT EXISTS pgcrypto;`
- `migrations/0001_init.down.sql` — `DROP EXTENSION IF EXISTS pgcrypto;`

**Phase 3 — Config Package:**
- `config/config.go` — типы `Config`, `Lookuper`, `osLookuper`, `ValidationError`, sentinel `ErrConfigInvalid`, функция `Load(Lookuper) (*Config, error)`
- `config/config_test.go` — `mapLookuper`, 6 тестов (см. `../04-testing.md`)

**Phase 4 — Server Entry Point:**
- `cmd/server/main.go` — `main()`, инициализация slog, chi.Mux, http.Server, graceful shutdown
- `cmd/server/health.go` — `healthHandler(w, r)` (приватная функция уровня пакета main)
- `cmd/server/health_test.go` — `TestHealthHandler_Returns200` (white-box, package main)

**Phase 5 — Doc & Cleanup:**
- (нет новых файлов)

### Modified Files

**Phase 1:**
- (нет — все файлы новые)

**Phase 2:**
- (нет)

**Phase 3:**
- (нет — пакет `config/` новый)

**Phase 4:**
- `go.mod` — добавляется `require github.com/go-chi/chi/v5 vX.Y.Z` через `go get github.com/go-chi/chi/v5`
- `go.sum` — создаётся автоматически после `go mod tidy`

**Phase 5:**
- `README.MD` — добавляется раздел `## Запуск локально` с инструкцией `cp .env.example .env && make dc-up && make migrate-up && make run`
- `init_project` — удаляется (заметка стала избыточной после реализации)

## DI Integration

В этой фиче «DI цепочка» — это `cmd/server/main.go`. Появится впервые в Phase 4, поэтому это единственная точка склейки.

**Init chain position:** `main()` — первая и единственная функция, которая собирает граф зависимостей.

**Composition root changes:** в Phase 4 создаётся `cmd/server/main.go:main()` со следующим порядком инициализации:

1. `cfg, err := config.Load(config.OsLookuper{})` — читает env, fail-fast при ошибке.
2. `logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))` — `slog`-логгер JSON в stdout.
3. `slog.SetDefault(logger)` — для использования из других пакетов в будущем.
4. `mux := chi.NewRouter()` — chi-роутер.
5. `mux.Route("/api/v1", func(r chi.Router) { r.Get("/health", healthHandler) })` — регистрация health.
6. `srv := &http.Server{Addr: ":" + strconv.Itoa(cfg.ServerPort), Handler: mux, ReadHeaderTimeout: 10 * time.Second}` — сервер.
7. `ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` — подписка на сигналы.
8. Запуск `srv.ListenAndServe()` в горутине, ожидание `ctx.Done()`.
9. `shutdownCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)` → `srv.Shutdown(shutdownCtx)` → `os.Exit(0)`.

**Initialization order (зависимости):**
- `config.Load` не зависит ни от чего, кроме `Lookuper`.
- `slog.Logger` не зависит от `config` (level можно сделать ENV-настраиваемым в будущем — пока хардкод `Info`).
- `chi.Mux` не зависит от `config`.
- `http.Server` зависит от `config.ServerPort` и `chi.Mux`.
- `signal.NotifyContext` не зависит ни от чего.

Цикл импортов отсутствует. Будущие фазы (1.2 — auth) добавят перед шагом 5 инициализацию `pgxpool.Pool`, репозиториев, use case'ов, регистрацию хендлеров через `r.Route("/auth", authTransport.Register)`.

## Error Codes

**Range:** `CONFIG-001..CONFIG-010` (зарезервирован за пакетом `config/`).

**Conflict check:** Verified — это первая фича в проекте, никаких существующих диапазонов нет (см. `../research.md` или ресерч `.thoughts/research/2026-05-04-project-structure.md`). Префикс `CONFIG-` зарезервирован за конфигурацией; будущие фичи выбирают свои префиксы (`AUTH-`, `ROOM-`, `CHANNEL-`, `CHAT-`, `VOICE-`, `USER-`, `WS-`).

| Code | Description | HTTP Status | Слой |
|------|-------------|-------------|------|
| CONFIG-001 | `JWT_SECRET` not set | (exit 1 при старте, не HTTP) | config |
| CONFIG-002 | `DATABASE_URL` not set | (exit 1 при старте, не HTTP) | config |
| CONFIG-003 | `SERVER_PORT` invalid (parse error or out of range 1..65535) | (exit 1 при старте, не HTTP) | config |
| CONFIG-004..CONFIG-010 | (зарезервированы для будущих расширений конфига — например, валидации DATABASE_URL формата, JWT_SECRET длины и т.п.) | — | — |

Все CONFIG-* коды — это типизированные ошибки из `config.ValidationError{Code, Field, Reason}`. Runtime-сбои сервера (порт занят, shutdown timeout) кодов не имеют — это лог-события (`../02-behavior.md` UC2).

## Success Criteria

- [ ] Все 5 фаз завершены и проверены (см. Verification в каждом `phase-NN.md`)
- [ ] `go build ./...` — exit 0
- [ ] `go test ./... -race -count=1` — все тесты проходят (минимум 7: 6 в `config/` + 1 в `cmd/server/`)
- [ ] `golangci-lint run` — exit 0, 0 issues
- [ ] `make dc-up` поднимает PostgreSQL, healthcheck зелёный
- [ ] `make migrate-up` накатывает `0001_init.up.sql` без ошибок, в БД появляется `schema_migrations` с `version=1, dirty=false`
- [ ] `make migrate-down` откатывает миграцию
- [ ] `make run` стартует сервер на порту из `SERVER_PORT` (default 8080)
- [ ] `curl http://localhost:8080/api/v1/health` возвращает `200` и `{"status":"ok"}`
- [ ] Сервер корректно завершается по `SIGINT`/`SIGTERM` за время ≤5с (graceful shutdown)
- [ ] CI workflow `.github/workflows/ci.yml` зелёный на тестовом push/PR
- [ ] Все 24 поддиректории `internal/<домен>/<слой>/.gitkeep` существуют и tracked в git
- [ ] `pkg/websocket/.gitkeep` существует и tracked
- [ ] `go.mod` содержит только `github.com/go-chi/chi/v5` как сторонней зависимостью (никаких других)
- [ ] Все 12 критериев приёмки из `../README.md` выполнены
- [ ] Coverage mapping из `../04-testing.md` соблюдён: каждый CONFIG-XXX имеет соответствующий тест

## Known Open Questions из дизайна (которые надо закрыть к плану)

Из `../03-decisions.md`:

- **Q3** (валидность пустого `sql: []` в sqlc.yaml v2) — закрывается в Phase 1: реальная проверка `sqlc compile` или `sqlc generate --dry-run`. Если массив `sql: []` не принимается sqlc v2, переключаемся на закомментированный шаблон с одним блоком (план Phase 1 содержит fallback).
- **Q9** (chi/middleware.Logger vs свой slog-wrapper) — закрывается в Phase 4: используем `slog`-wrapper, чтобы все логи в одном формате. Финальное решение зафиксируется как ADR-015 в `../03-decisions.md` (Phase 5 опционально).

Остальные Open Questions (Q1, Q2, Q4–Q8, Q10) не блокируют реализацию — текущие дефолты в дизайне их закрывают.
