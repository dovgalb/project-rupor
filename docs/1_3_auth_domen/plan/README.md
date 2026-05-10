---
date: 2026-05-09
feature: 1_3_auth_domen
design: ../README.md
status: draft
---

# План кода: 1_3_auth_domen

## Overview

Реализуется домен `internal/auth/`: четыре слоя Clean Architecture (`domain`, `usecase`, `transport/http`, `repository/`) + три адаптера (`repository/postgres`, `repository/jwt`, `repository/bcrypt`). По итогу появляются эндпоинты `POST /api/v1/auth/{register,login,refresh}` и `GET /api/v1/auth/me`, расширяется `config/` (TTL access/refresh), вводится `sqlc.yaml` и подключается граф зависимостей в `cmd/server/main.go`. Дизайн зафиксирован в `../README.md`, архитектура — `../01-architecture.md`, репо-модель — `../06-repo-model.md`, контракт — `../08-api-contract.md`, тесты — `../04-testing.md`, решения — `../03-decisions.md`.

Scope строго совпадает с разделом «Критерии приёмки» дизайна (`../README.md:32-47`). Никаких миграций (они закрыты фазой 1.2), никакого middleware (фаза 1.4), никаких интеграционных прогонов в CI (фаза 1.5).

## Phase Strategy

**Bottom-up по слоям Clean Architecture** (`prompts/Architecture Layers.txt:60-65`). Каждая фаза заканчивается зелёным `go build ./...` и зелёным `make test` (для фаз, в которых уже есть код, требующий unit-тестов). Слой пишется вместе со своими тестами в той же фазе — слой без тестов не считается завершённым (`prompts/Tests Style.txt:14-22`).

Зависимости между фазами строго однонаправленные: каждая следующая фаза опирается на пакеты, готовые в предыдущей. Это даёт возможность останавливаться после любой фазы, не оставляя битый `internal/auth/`.

## Phases

| # | Фаза | Слой | Зависимости | Status |
|---|------|------|-------------|--------|
| 1 | [Зависимости, config, sqlc.yaml](phase-01.md) | infra | — | ☐ |
| 2 | [`internal/auth/domain/`](phase-02.md) | domain | фаза 1 (uuid в go.mod) | ☐ |
| 3 | [`internal/auth/usecase/` — ports + скелеты use case'ов](phase-03.md) | usecase | фаза 2 | ☐ |
| 4 | [`internal/auth/repository/postgres/`](phase-04.md) | adapter | фазы 1 (sqlc.yaml, pgx), 3 (порты) | ☐ |
| 5 | [`internal/auth/repository/{jwt,bcrypt}/`](phase-05.md) | adapter | фазы 1 (deps), 3 (порты) | ☐ |
| 6 | [`internal/auth/usecase/` — реализация `Execute()`](phase-06.md) | usecase | фазы 2, 3 | ☐ |
| 7 | [`internal/auth/transport/http/`](phase-07.md) | transport | фазы 3, 6 | ☐ |
| 8 | [Composition root + архитектурный smoke-test](phase-08.md) | infra | фазы 4, 5, 7 | ☐ |

## File Map

### New Files

`config/` (фаза 1):
- расширение существующего `config/config.go` — без новых файлов; модификации см. ниже.

`sqlc.yaml` (фаза 1, корень репо).

`internal/auth/domain/` (фаза 2):
- `user.go` — Entity `User` + конструкторы `NewUser` / `ReconstructUser`.
- `refresh_token.go` — Entity `RefreshToken` + методы `Revoke`, `IsActive`, `IsRevoked`, `IsExpired`.
- `email.go`, `username.go`, `password.go`, `password_hash.go` — VO над `string`.
- `user_id.go`, `refresh_token_id.go` — VO над `uuid.UUID`.
- `token_hash.go` — VO над `[32]byte`.
- `errors.go` — sentinel-ошибки `ErrInvalidEmail`, `ErrEmailAlreadyTaken`, … (полный список — `../01-architecture.md:227-228`).
- `*_test.go` — unit-тесты по `../04-testing.md:65-115` (16 тестов).

`internal/auth/usecase/` (фазы 3 и 6):
- `ports.go` — интерфейсы `UserRepository`, `RefreshTokenRepository`, `PasswordHasher`, `TokenIssuer`, `Clock`, `UUIDGenerator`, `RandomBytes` (`../01-architecture.md:243-271`).
- `register_user.go`, `login_user.go`, `refresh_access.go`, `get_current_user.go` — четыре use case-структуры.
- `*_test.go` — 21 тест с фейками (`../04-testing.md:117-199`).
- `fakes_test.go` — общие фейки (`../04-testing.md:121-156`).

`internal/auth/repository/postgres/` (фаза 4):
- `queries/users.sql`, `queries/refresh_tokens.sql` — sqlc-исходники (`../06-repo-model.md:111-150`).
- `db/` — сгенерированный sqlc-пакет (создаётся `make sqlc`, коммитится в git).
- `user_repository.go`, `refresh_token_repository.go` — реализации портов.
- `mapper.go` — `userRowToDomain`, `domainToInsertUserParams`, `refreshTokenRowToDomain`, `domainToInsertRefreshTokenParams`.
- `pgerr.go` — `isUniqueViolation`.
- `*_integration_test.go` — `//go:build integration` тесты (`../04-testing.md:201-231`).

`internal/auth/repository/jwt/` (фаза 5):
- `token_issuer.go` — реализация `usecase.TokenIssuer` через `golang-jwt/jwt/v5`.
- `token_issuer_test.go` — 6 тестов (`../04-testing.md:233-244`).

`internal/auth/repository/bcrypt/` (фаза 5):
- `password_hasher.go` — реализация `usecase.PasswordHasher`.
- `password_hasher_test.go` — 3 теста (`../04-testing.md:246-254`).

`internal/auth/transport/http/` (фаза 7):
- `dto.go` — request/response/error структуры.
- `register_handler.go`, `login_handler.go`, `refresh_handler.go`, `me_handler.go`.
- `error_mapper.go` — `mapError(err) (status, code, message)`.
- `routes.go` — `RegisterRoutes(r chi.Router, deps Deps)`.
- `*_test.go` — 19 тестов (`../04-testing.md:256-298`).

`arch_test.go` (фаза 8, корень репо или `internal/`) — активация архитектурного smoke (`../04-testing.md:299-311`).

`Makefile` — добавить цель `sqlc` (фаза 1).

`manual_qa/auth/test-flow.http` (фаза 7, опциональный smoke по `../08-api-contract.md:281-296`).

### Modified Files

- `go.mod`, `go.sum` (фаза 1) — четыре новые прямые зависимости: `github.com/jackc/pgx/v5`, `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid`, `golang.org/x/crypto` (`../README.md:44`).
- `config/config.go` (фаза 1) — поля `jwtAccessTTL`, `jwtRefreshTTL`, геттеры, парсинг `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL`, коды `CONFIG-004`, `CONFIG-005` (`../README.md:40`).
- `config/config_test.go` (фаза 1) — 4 новых теста по `../04-testing.md:313-322`.
- `cmd/server/main.go` (фаза 8) — новые шаги composition root, регистрация роутов, `pgxpool` lifecycle (`../01-architecture.md:316-370`).

### Files NOT touched

- `migrations/*` — закрыты фазой 1.2.
- `pkg/websocket/*` — пустой каркас, не относится к auth.
- `web/*` — фронтенд, фаза 5.
- `docker-compose.yml`, `Dockerfile`, CI-yaml — без изменений (стек тот же, новые ENV — опциональные с дефолтами).

## DI Integration

Точка сборки графа зависимостей — `cmd/server/main.go` (фаза 8). Порядок:

1. `config.Load(config.OsLookuper{})` — теперь возвращает в т.ч. `JWTAccessTTL()`, `JWTRefreshTTL()`.
2. `pgxpool.New(ctx, cfg.DatabaseURL())` — пул соединений; `pool.Close()` в graceful shutdown.
3. `db.New(pool)` — sqlc-сгенерированный конструктор `*db.Queries`.
4. `postgres.NewUserRepository(q)`, `postgres.NewRefreshTokenRepository(pool)`.
5. `bcrypt.NewPasswordHasher(10)`, `jwt.NewTokenIssuer([]byte(cfg.JWTSecret()), cfg.JWTAccessTTL())`.
6. Реальные `Clock`/`UUIDGenerator`/`RandomBytes` — inline-структуры в `main.go` (`../03-decisions.md` OQ-9).
7. Use case-конструкторы `usecase.NewRegisterUser(...)` и т.д.
8. `httpauth.RegisterRoutes(r chi.Router, deps Deps)` — регистрация под `r.Route("/api/v1/auth", ...)`.

Полная диаграмма — `../01-architecture.md:316-370`.

## Error Codes

Новые коды, вводимые этой фичей:

| Code | Источник | Where mapped |
|------|----------|--------------|
| `AUTH-001..AUTH-012` | `internal/auth/domain/errors.go` (sentinel) | `internal/auth/transport/http/error_mapper.go` (фаза 7) — таблица в `../08-api-contract.md:32-46` |
| `CONFIG-004` | `config/config.go`, парсинг `JWT_ACCESS_TTL` | возвращается из `config.Load` (фаза 1) |
| `CONFIG-005` | `config/config.go`, парсинг `JWT_REFRESH_TTL` | возвращается из `config.Load` (фаза 1) |
| `INTERNAL` | `internal/auth/transport/http/error_mapper.go` | дефолтная ветка для всех 5xx |

Полная таблица соответствия `domain.Err* ↔ HTTP status ↔ AUTH-XXX` — `../08-api-contract.md` плюс матрица покрытия `../04-testing.md:14-64`.

## Success Criteria

- [ ] Все 14 пунктов «Критериев приёмки» из `../README.md:33-47` выполнены.
- [ ] `go build ./...` зелёный после каждой фазы.
- [ ] `make lint` зелёный после фазы 8.
- [ ] `make test` зелёный после фазы 8 — все 71 unit-тестов (без integration-tag) проходят. Integration-тесты (`//go:build integration`, ~12 шт.) по умолчанию пропускаются.
- [ ] `go test ./... -race -count=1` — race-detector чистый.
- [ ] `make sqlc` отрабатывает идемпотентно (на чистом checkout даёт тот же `db/`-пакет, что в git).
- [ ] Архитектурный smoke (`arch_test.go`) — зелёный; whitelist `domain` и `usecase` соблюдён.
- [ ] Manual QA по `../08-api-contract.md:281-296` (10 шагов) — все шаги дают ожидаемые ответы при `make dc-up && make migrate-up && make run`.
- [ ] `git diff --name-only` ограничен путями: `go.mod`, `go.sum`, `sqlc.yaml`, `Makefile`, `config/`, `cmd/server/main.go`, `internal/auth/**`, `arch_test.go`, `docs/1_3_auth_domen/**`, опционально `manual_qa/auth/**`. Никаких посторонних файлов.
- [ ] `internal/auth/domain/` импортирует только stdlib + `github.com/google/uuid`. `internal/auth/usecase/` — только stdlib + `uuid` + `internal/auth/domain`. Проверяется `arch_test.go`.
