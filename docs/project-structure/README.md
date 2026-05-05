---
date: 2026-05-04
feature: project-structure
status: draft
research: ../../.thoughts/research/2026-05-04-project-structure.md
---

# Project Structure — Документы дизайна

## Бизнес-контекст

Репозиторий project-rupor сейчас в pre-code состоянии: есть только `go.mod`, методические стандарты в `prompts/`, метаданные `.claude/`. Ни одной директории из целевой структуры (`cmd/`, `internal/`, `pkg/`, `migrations/`, `config/`), ни сборочных артефактов (`Makefile`, `docker-compose.yml`, `.gitignore`), ни рабочих точек входа нет — `make run` упирается в пустоту.

Цель этой фичи — заложить **скелет монорепо**, после которого можно начинать Фазу 1.2 «Авторизация» из `.claude/plans/general_plan.md:29-34`. Скелет должен быть «живым»: `make run` поднимает HTTP-сервер с health-check, `make dc-up` поднимает PostgreSQL, `make migrate-up` накатывает initial-миграцию, `make test` / `make lint` проходят на пустом проекте, CI в GitHub Actions зелёный.

Скелет фиксирует терминологию слоёв из `prompts/Architecture Layers.txt:14-66` (`domain/usecase/transport/repository`) физически — папки уже есть с `.gitkeep`, и каждый следующий домен добавляется без обсуждений «куда класть».

## Scope

В этой фиче:
- Структура директорий: `cmd/server/`, `internal/{auth,user,room,channel,chat,voice}/{domain,usecase,transport/http,repository/postgres}/`, `pkg/websocket/`, `migrations/`, `config/`
- Точка входа `cmd/server/main.go` с health-check и graceful shutdown
- Загрузчик конфигурации `config/config.go` (env-переменные, fail-fast валидация)
- `Makefile` с целями run / build / test / lint / migrate-up / migrate-down / sqlc / dc-up / dc-down / dc-logs
- `docker-compose.yml` с PostgreSQL 16
- `migrations/0001_init.up.sql` / `.down.sql` (без бизнес-таблиц, только метаданные)
- `sqlc.yaml`, `.golangci.yml`, корневой `.gitignore`, `.env.example`
- CI: `.github/workflows/ci.yml` (build + test + lint)

Вне scope (отдельные тикеты):
- `web/` — фронтенд (Фаза 5 в `general_plan.md:60-65`)
- Бизнес-таблицы и доменный код (Фазы 1.2–4 в `general_plan.md:29-58`)
- Redis (не упомянут в `general_plan.md`, не нужен для MVP)
- Production Dockerfile для Go-сервера (локальная разработка через `make run`)
- Pre-commit hooks, secret scanning

## Критерии приёмки

1. После `git clone` и `make dc-up && make migrate-up && make run` сервер слушает на `SERVER_PORT` (по умолчанию 8080), `curl http://localhost:8080/api/v1/health` возвращает `{"status":"ok"}` со статусом 200.
2. `make build` собирает бинарник `bin/server`, `go build ./...` проходит без ошибок.
3. `make test` проходит — присутствует хотя бы один тест на загрузчик конфигурации (`config/config_test.go`) с покрытием happy path и валидации.
4. `make lint` (`golangci-lint run`) проходит без warnings на минимальном наборе линтеров (`govet`, `staticcheck`, `errcheck`, `ineffassign`, `unused`, `gofmt`, `goimports`).
5. `make migrate-up` накатывает `0001_init.up.sql`, `make migrate-down` откатывает.
6. `make dc-up` поднимает контейнер PostgreSQL, `make dc-down` останавливает; healthcheck `pg_isready` зелёный.
7. Конфигурация читается только из env-переменных (`SERVER_PORT` опциональна, default 8080; `DATABASE_URL`, `JWT_SECRET` обязательные), fail-fast при отсутствии обязательных.
8. Все целевые поддиректории `internal/{домен}/{слой}/` существуют с `.gitkeep` и tracked в git.
9. Граф импортов соответствует правилам из `prompts/Architecture Layers.txt:67-74`: `domain ← usecase ← transport/repository ← cmd/server/config`. Это проверяется отдельным импорт-тестом или линтером (см. `04-testing.md`).
10. CI workflow `.github/workflows/ci.yml` срабатывает на push/PR в `main` и проходит build + test + lint.
11. Сервер корректно завершается по `SIGTERM`/`SIGINT` за время не больше `5s` (graceful shutdown), активные запросы обрабатываются до конца.
12. Граф зависимостей `go.mod` содержит только согласованные библиотеки: `github.com/go-chi/chi/v5` (роутинг). Никаких других сторонних зависимостей в этой фиче.

## Документы

| Файл | Разрез | Описание |
|------|--------|----------|
| [01-architecture.md](./01-architecture.md) | Logical | C4 L1 → L2 → L3, дерево директорий, граф зависимостей слоёв |
| [02-behavior.md](./02-behavior.md) | Process | DFD, sequence-диаграммы (запуск, миграции, /health, graceful shutdown) |
| [03-decisions.md](./03-decisions.md) | Decision | ADR, альтернативы, риски, open questions |
| [04-testing.md](./04-testing.md) | Quality | Стратегия тестирования: config unit + smoke + CI gate |
| [07-standards.md](./07-standards.md) | Standards | Compliance-матрица по 8 файлам в `prompts/` |
| [08-api-contract.md](./08-api-contract.md) | API | Контракт `GET /api/v1/health` |

Не создаются (не применимы к этой фиче):
- `05-events.md` — нет доменных событий, нет WebSocket в этой фиче
- `06-repo-model.md` — нет сущностей в БД, миграция `0001_init` не вводит бизнес-таблиц
