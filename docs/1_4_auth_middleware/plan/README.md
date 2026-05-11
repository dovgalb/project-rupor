---
date: 2026-05-10
feature: auth_middleware
design: ../README.md
status: draft
---

# План кода: 1.4 Auth Middleware

## Overview

Реализуется HTTP-middleware-stack: новый общий пакет `pkg/httpx/` (примитивы + cross-cutting middleware) и новый подпакет `internal/auth/transport/http/middleware/` (`RequireAuth`). По итогу:
- Все ответы идут через `RequestID → Recover → Logger → CORS` глобально.
- Приватные маршруты защищены `RequireAuth` через `chi.Group + Use`.
- `MeHandler` упрощается до чтения `userID` из контекста.
- `error_mapper.go:writeError` делегирует сериализацию envelope в `pkg/httpx.WriteJSONError`.
- `config/` дополнен `CORS_ALLOWED_ORIGINS` (env, default `http://localhost:5173`, валидация → `CONFIG-006`).

Дизайн зафиксирован в `../README.md`, архитектура — `../01-architecture.md`, поведение — `../02-behavior.md`, решения — `../03-decisions.md`, тесты — `../04-testing.md`, контракт — `../08-api-contract.md`. Coverage mapping (`AUTH-010/011`, `INTERNAL`, `CONFIG-006`) — `../04-testing.md:14-65`.

Scope строго совпадает с разделом «Критерии приёмки» дизайна (`../README.md:32-46`). Никаких новых эндпоинтов, никаких новых доменных сущностей, никаких миграций.

## Phase Strategy

**Bottom-up по зависимостям** (`prompts/Architecture Layers.txt:60-65`).

`pkg/httpx/` примитивы → cross-cutting middleware → auth-specific middleware → расширение config → интеграция в роутер → arch-тесты + manual_qa. Каждая фаза заканчивается зелёным `go build ./...` и `go test ./<пакет>/...` для своего пакета. Слой пишется вместе со своими тестами в той же фазе — слой без тестов не считается завершённым (`prompts/Tests Style.txt:14-22`).

Зависимости — строго однонаправленные: каждая следующая фаза опирается на пакеты, готовые в предыдущей. Фаза 4 (config) технически независима от 1–3 и может идти параллельно, но в линейной последовательности — после 3, перед 5, чтобы интеграция в фазе 5 имела готовый `cfg.CORSAllowedOrigins()`.

## Phases

| # | Фаза | Слой | Зависимости | Status |
|---|------|------|-------------|--------|
| 1 | [`pkg/httpx/` примитивы](phase-01.md) | infra | — | ☐ |
| 2 | [`pkg/httpx/middleware/` cross-cutting](phase-02.md) | infra | фаза 1 | ☐ |
| 3 | [`internal/auth/transport/http/middleware/` (RequireAuth)](phase-03.md) | adapter | фаза 1 (httpx), 1.3 (TokenIssuer) | ☐ |
| 4 | [`config/` `CORS_ALLOWED_ORIGINS`](phase-04.md) | infra | — | ☐ |
| 5 | [Интеграция: cmd/server, routes, MeHandler, error_mapper](phase-05.md) | infra + transport | фазы 1–4 | ☐ |
| 6 | [`arch_test.go` + `manual_qa/`](phase-06.md) | tests | фазы 1–5 | ☐ |

## File Map

### New Files

`pkg/httpx/` (фаза 1):
- `pkg/httpx/jsonerror.go` — функция `WriteJSONError(w, status, code, message)` (`../01-architecture.md:106-114`).
- `pkg/httpx/contextkeys.go` — типизированный `requestIDKey`, `WithRequestID`, `RequestIDFromContext` (`../01-architecture.md:118-126`).
- `pkg/httpx/jsonerror_test.go`, `pkg/httpx/contextkeys_test.go` — 4 теста (`../04-testing.md:36-50`).

`pkg/httpx/middleware/` (фазы 1 и 2):
- `pkg/httpx/middleware/responsewriter.go` (фаза 1) — общий wrapper `responseWriter` для Recover и Logger (`../01-architecture.md:78-95`).
- `pkg/httpx/middleware/responsewriter_test.go` (фаза 1) — 1–2 теста на идемпотентность `WriteHeader`.
- `pkg/httpx/middleware/recover.go` + `recover_test.go` (фаза 2) — Recover + 4 теста (`../04-testing.md:118-127`).
- `pkg/httpx/middleware/logger.go` + `logger_test.go` (фаза 2) — Logger + 6 тестов (`../04-testing.md:129-138`).
- `pkg/httpx/middleware/cors.go` + `cors_test.go` (фаза 2) — CORS + 7 тестов (`../04-testing.md:140-150`).
- `pkg/httpx/middleware/requestid.go` + `requestid_test.go` (фаза 2) — RequestID + 5 тестов (`../04-testing.md:108-116`).

`internal/auth/transport/http/middleware/` (фаза 3):
- `internal/auth/transport/http/middleware/contextkeys.go` — `userIDKey`, `WithUserID`, `UserIDFromContext` (`../01-architecture.md:140-148`).
- `internal/auth/transport/http/middleware/auth.go` — `RequireAuth(issuer, clock)` (`../01-architecture.md:130-138`).
- `internal/auth/transport/http/middleware/contextkeys_test.go`, `auth_test.go` — 10 тестов (`../04-testing.md:170-200`).

`cmd/server/main_test.go` (фаза 5, опционально) — `TestHealthHandler_ThroughMiddlewareStack` smoke (`../04-testing.md:243-249`).

`manual_qa/1_4_auth_middleware/` (фаза 6):
- `README.md` — описание стенда (по образцу `manual_qa/1_3_auth/README.md`).
- `00_smoke.http` — общая прогонка через middleware-stack.
- `01_cors_preflight.http` — preflight + cross-origin запросы.
- `02_request_id.http` — приём клиентского `X-Request-ID`, отвержение невалидного.
- `03_require_auth.http` — все 401-сценарии RequireAuth.

### Modified Files

- `config/config.go` (фаза 4) — поле `corsAllowedOrigins []string`, геттер, парсинг env `CORS_ALLOWED_ORIGINS`, валидация → `CONFIG-006` (`../02-behavior.md`, раздел «Конфигурация при старте»).
- `config/config_test.go` (фаза 4) — 4 новых теста (`../04-testing.md:230-236`).
- `.env`, `.env.example` (фаза 4) — добавить `CORS_ALLOWED_ORIGINS=http://localhost:5173`.
- `cmd/server/main.go` (фаза 5) — добавление `mux.Use(...)` для четырёх глобальных middleware, формирование `userIDHook`, передача через `Logger` (`../01-architecture.md:155-180`, ADR-002).
- `internal/auth/transport/http/routes.go` (фаза 5) — `chi.Group` для public (`/register`, `/login`, `/refresh`) и protected (`/me` под `r.Use(authmw.RequireAuth(...))`) (`../01-architecture.md:184-198`).
- `internal/auth/transport/http/me_handler.go` (фаза 5) — упрощение: убрать парсинг `Authorization`, убрать `issuer`/`clock` поля, читать `userID` через `UserIDFromContext` (`../01-architecture.md:202-222`). Defensive ветка `if !ok { writeError(...) }`.
- `internal/auth/transport/http/error_mapper.go` (фаза 5) — `writeError(w, he)` внутри использует `pkg/httpx.WriteJSONError(w, he.status, he.code, he.message)`. Контракт `httpError` сохраняется (ADR-011).
- `internal/auth/transport/http/setup_test.go` (фаза 5) — минимальное обновление: тесты регрессии 1.3 продолжают использовать `httpauth.RegisterRoutes`, который теперь сам подключает `RequireAuth`. Изменений в коде тестов нет (`../04-testing.md:202-222`).
- `arch_test.go` (фаза 6) — добавить `TestArchitecture_PkgHttpxImports` и `TestArchitecture_AuthMiddlewareImports` (`../04-testing.md:259-264`).

### Files NOT touched

- `migrations/*` — фаза 1.2 закрыта.
- `internal/auth/domain/*` — domain не меняется (1.3).
- `internal/auth/usecase/*` — usecase не меняется (контракт `TokenIssuer.VerifyAccess` стабилен).
- `internal/auth/repository/{postgres,jwt,bcrypt}/*` — адаптеры не меняются.
- `internal/auth/transport/http/{register,login,refresh}_handler.go` — public-хендлеры не трогаем.
- `internal/auth/transport/http/dto.go` — DTO не меняются.
- `pkg/websocket/*` — пустой каркас, фаза 3.
- `web/*` — фронтенд, фаза 5.
- `docker-compose.yml`, `Dockerfile`, `.github/workflows/ci.yml` — без изменений.
- `go.mod`, `go.sum` — без изменений (нет новых прямых зависимостей; `github.com/google/uuid` уже есть с 1.3).
- `Makefile` — без изменений.
- `sqlc.yaml` — без изменений.

## DI Integration

Точка сборки middleware-stack — `cmd/server/main.go` (фаза 5).

**Init chain position**: middleware-stack подключается ПОСЛЕ создания `mux := chi.NewRouter()` (`cmd/server/main.go:100`), но ДО `mux.Route("/api/v1", ...)` (`cmd/server/main.go:101`). Это вставка из 4 строк `mux.Use(...)` между этими двумя точками + дополнительная сборка `userIDHook`.

**Composition root changes**:
- `cmd/server/main.go:100` — после `mux := chi.NewRouter()` добавить:
  ```go
  uuidGen := func() string { return uuid.New().String() }
  userIDHook := func(ctx context.Context) []slog.Attr {
      uid, ok := authmw.UserIDFromContext(ctx)
      if !ok { return nil }
      return []slog.Attr{slog.String("user_id", uid.String())}
  }
  mux.Use(httpxmw.RequestID(uuidGen))
  mux.Use(httpxmw.Recover(logger))
  mux.Use(httpxmw.Logger(logger, userIDHook))
  mux.Use(httpxmw.CORS(cfg.CORSAllowedOrigins(), false))
  ```
- `cmd/server/main.go:103-110` — `httpauth.Deps` поля **не меняются**: `Register/Login/Refresh/Me/TokenIssuer/Clock` остаются. `RequireAuth` конструируется внутри `RegisterRoutes`, не передаётся через `Deps`.

**Initialization order**:
1. `config.Load(config.OsLookuper{})` — теперь возвращает в т. ч. `CORSAllowedOrigins()`.
2. `pgxpool.New(...)` — без изменений.
3. `db.New(pool)`, `postgres.NewUserRepository(...)`, `postgres.NewRefreshTokenRepository(...)` — без изменений.
4. `bcrypt.NewPasswordHasher(...)`, `jwt.NewTokenIssuer(...)` — без изменений.
5. `realClock`, `realUUID`, `cryptoRand` — без изменений.
6. `usecase.NewRegisterUser/Login/Refresh/GetCurrentUser` — без изменений.
7. `mux := chi.NewRouter()` — без изменений.
8. **НОВОЕ**: `mux.Use(...)` — четыре глобальных middleware (RequestID, Recover, Logger, CORS).
9. `mux.Route("/api/v1", ...)` — внутри `httpauth.RegisterRoutes(r, deps)` теперь сам делит на public/protected группы.

Полная диаграмма — `../01-architecture.md:374-415` (граф зависимостей модулей) и `../01-architecture.md:425-462` (composition root).

## Error Codes

Range фичи: `AUTH-010..AUTH-013` (без новых внутри, только переезд с inline на middleware), `CONFIG-006` (новый).

**Conflict check** (просканировано в research): существующие диапазоны:
- `AUTH-001..AUTH-012` — фаза 1.3 (`docs/1_3_auth_domen/08-api-contract.md:32-46`).
- `AUTH-010` — invalid access token (1.3, теперь генерируется в `RequireAuth`).
- `AUTH-011` — expired access token (1.3, теперь генерируется в `RequireAuth`).
- `INTERNAL` — все 5xx (1.3, расширенно использование Recover middleware).
- `CONFIG-001..CONFIG-005` — фазы 1.1 (`SERVER_PORT`) и 1.3 (`JWT_*`). `CONFIG-006` — следующий свободный.

Новых `AUTH-*` кодов в этой фиче **нет**: middleware использует уже существующие `AUTH-010` (любой невалидный bearer/подпись/sub) и `AUTH-011` (только expired). Это сознательно для сохранения регрессии 1.3.

| Code | Description | HTTP Status | Источник |
|------|-------------|-------------|----------|
| AUTH-010 | access token invalid | 401 | RequireAuth (фаза 3); fallback в MeHandler (фаза 5) |
| AUTH-011 | access token expired | 401 | RequireAuth (фаза 3) |
| INTERNAL | internal server error | 500 | Recover (фаза 2) — panic recovery; default branch error_mapper (1.3) |
| CONFIG-006 | invalid CORS_ALLOWED_ORIGINS | сервер не стартует | config.Load (фаза 4) |

## Success Criteria

- [ ] Все 14 пунктов «Критериев приёмки» из `../README.md:33-46` выполнены.
- [ ] `go build ./...` зелёный после каждой фазы.
- [ ] `make lint` зелёный после фазы 6.
- [ ] `make test` зелёный после фазы 6 — все ~65 тестов (46 новых + 19 регрессии 1.3) проходят.
- [ ] `go test ./... -race -count=1` — race-detector чистый.
- [ ] `arch_test.go` — все 6 тестов (4 существующих + 2 новых) зелёные.
- [ ] Manual QA по `../08-api-contract.md:213-264` — все шаги (00_smoke / 01_cors / 02_request_id / 03_require_auth) дают ожидаемые ответы при `make dc-up && make migrate-up && make run`.
- [ ] API-контракт совпадает с реализацией:
  - На каждом ответе — `X-Request-ID`.
  - На запросах с whitelisted `Origin` — `Access-Control-Allow-Origin` + `Vary: Origin`.
  - Preflight `OPTIONS` для whitelisted origin → 204 + полный набор `Access-Control-*`.
  - 401-ответы для `/auth/me` сохраняют те же тела, что и в 1.3.
- [ ] `git diff --name-only` ограничен путями: `pkg/httpx/**`, `internal/auth/transport/http/middleware/**`, `internal/auth/transport/http/{routes,me_handler,error_mapper}.go`, `internal/auth/transport/http/setup_test.go`, `config/config.go`, `config/config_test.go`, `.env*`, `cmd/server/main.go` (опц. `cmd/server/main_test.go`), `arch_test.go`, `docs/auth_middleware/**`, `manual_qa/1_4_auth_middleware/**`. Никаких посторонних файлов.
- [ ] `pkg/httpx/...` не импортирует `internal/...`. Проверяется `arch_test.go`.
- [ ] `internal/auth/transport/http/middleware/` импортирует только `usecase`, `domain`, `pkg/httpx`, stdlib, uuid. Проверяется `arch_test.go`.
- [ ] `go.mod` без новых прямых зависимостей.
- [ ] Production-код не содержит `panic(...)` (кроме `pkg/httpx/middleware/Recover` — re-panic для `http.ErrAbortHandler`).
