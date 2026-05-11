---
date: 2026-05-10
feature: auth_middleware
status: draft
research: ./research.md
---

# 1.4 Auth Middleware — Документы дизайна

## Бизнес-контекст

Задача 1.4 «Middleware» (`.claude/plans/general_plan.md:114-118`) закрывает HTTP-каркас проекта. Сегодня:
- Авторизация по access-токену живёт инлайново в одном хендлере (`internal/auth/transport/http/me_handler.go:23-37`); в `request.Context()` userID не кладётся.
- Никаких глобальных middleware — нет ни одного `Use(...)` во всём репо. `panic` в любом хендлере уронит процесс, нет access-логирования, нет CORS-заголовков.
- Фаза 2 (`/rooms`, `/channels`, `/messages`) и фаза 3 (`/ws`) опираются на наличие userID в контексте запроса и единого механизма защиты приватных маршрутов; без 1.4 они запустятся в инлайн-режиме каждый раз, что ломает стандарт.

После 1.4:
- Все приватные маршруты защищаются единой middleware, которая извлекает `Authorization: Bearer …`, валидирует через стабильный порт `usecase.TokenIssuer.VerifyAccess` (контракт зафиксирован в фазе 1.3, см. `docs/1_3_auth_domen/03-decisions.md`, ADR-009) и кладёт `domain.UserID` в контекст.
- Каждый запрос проходит через access-log с метриками (method/path/status/duration/request_id), recover ловит panic и возвращает стандартный 500 INTERNAL вместо краха процесса, CORS-headers выставляются по whitelist origins.
- Хендлер `/auth/me` упрощается до одной функции: `userID, _ := authmw.UserIDFromContext(ctx)` → `usecase.GetCurrentUser`.
- Появляется первый общий пакет HTTP-обвязки `pkg/httpx/middleware/`, который в фазе 2/3 переиспользуют `room`, `channel`, `chat`, WebSocket-handshake.

Scope строго ограничен подпунктами тикета (`general_plan.md:115-118`):
- JWT-middleware (извлечение `Authorization: Bearer`, валидация, проброс userID в контекст).
- Логирование запросов (access-log).
- Recover middleware.
- CORS (для будущего фронта).
- Request-ID middleware (согласовано на этапе 0.6 — нужен для коррелятора в логах).
- Перевод `/auth/me` на новую middleware (вместо текущего инлайн-парсинга).
- Конфиг `CORS_ALLOWED_ORIGINS` (ENV, default `http://localhost:5173`).

Вне scope:
- Авторизация (RBAC) и роли — фаза 2 (`general_plan.md:128-131`).
- Rate limiting на login/refresh — отдельная фича.
- Метрики Prometheus / OpenTelemetry — отдельная фича «observability».
- HTTP/2, gRPC, gzip-сжатие — не в MVP.
- WebSocket-авторизация (`/api/v1/ws?token=...`) — фаза 3 (отдельный механизм через query-string, не Bearer).

## Критерии приёмки

1. В `pkg/httpx/middleware/` появляются четыре middleware: `Recover`, `Logger`, `CORS`, `RequestID`. Каждая — публичная функция-фабрика, возвращающая `func(http.Handler) http.Handler`. Без сторонних зависимостей (см. `03-decisions.md`, ADR-001).
2. В `internal/auth/transport/http/middleware/` появляется `RequireAuth` — middleware-фабрика, принимающая `usecase.TokenIssuer` и `usecase.Clock`, валидирующая `Authorization: Bearer` и кладущая `domain.UserID` в контекст. Соответствует правилам слоёв (`prompts/Architecture Layers.txt:67-74`).
3. В `internal/auth/transport/http/middleware/contextkeys.go` появляются типизированный ключ `userIDKey` и публичные хелперы `WithUserID(ctx, id) ctx` / `UserIDFromContext(ctx) (domain.UserID, bool)` (`prompts/Go style.txt:51` — `context.Value` только для cross-cutting).
4. В `cmd/server/main.go` middleware подключаются в правильном порядке: `RequestID → Recover → Logger → CORS` глобально на корневой роутер; `RequireAuth` — только на группе приватных маршрутов в `internal/auth/transport/http/routes.go`.
5. `internal/auth/transport/http/routes.go` разделяется на публичную и приватную группы через `chi.Group`. `/register`, `/login`, `/refresh` — публичные; `/me` — приватный.
6. `internal/auth/transport/http/me_handler.go` упрощается: больше не парсит `Authorization`, не вызывает `TokenIssuer.VerifyAccess`. Только `UserIDFromContext` + `usecase.GetCurrentUser.Execute`. Контракт (HTTP, JSON, error codes) не меняется.
7. `config/config.go` дополнен полем `corsAllowedOrigins []string` (env `CORS_ALLOWED_ORIGINS`, default `http://localhost:5173`, CSV-формат). Валидация: каждый origin — корректный URL с `scheme://host[:port]` без пути; пустой список — fail-fast `CONFIG-006`.
8. Все коды ошибок диапазона `AUTH-010..AUTH-013` для middleware зафиксированы (`08-api-contract.md`) и покрыты тестами (`04-testing.md`). Никаких новых `AUTH-*` кодов вне этого диапазона.
9. `arch_test.go` дополнен новыми проверками: (a) `pkg/httpx/middleware/` импортирует только stdlib + `github.com/google/uuid`; (b) `internal/auth/transport/http/` (включая `middleware/`) не импортирует `internal/auth/repository/...`. Имеющиеся тесты слоёв продолжают проходить.
10. `go build ./...`, `make lint`, `make test` — все три зелёные. Race-detector чистый.
11. `go.mod` пополняется ровно нулём новых прямых зависимостей (`prompts/Go style.txt:111-117`). Используем только `github.com/go-chi/chi/v5` (уже есть), `github.com/google/uuid` (уже есть), stdlib (`context`, `log/slog`, `net/http`, `strings`, `time`).
12. `manual_qa/1_4_auth_middleware/` содержит `.http`-файлы, проверяющие: (a) preflight `OPTIONS` → 204 с правильными `Access-Control-*`; (b) запрос с невалидным токеном → 401 + `AUTH-010`; (c) request-id виден в `X-Request-ID` ответа; (d) panic-маршрут (для тестов) → 500 + `INTERNAL`, процесс не упал.
13. Существующие 19 HTTP-тестов из фазы 1.3 (`docs/1_3_auth_domen/04-testing.md`) продолжают проходить без изменений в коде тестов — контракт ответов сохраняется. Меняется только `setup_test.go`, который теперь монтирует middleware в test-router.
14. Production-code не использует `panic` (`prompts/Go style.txt:44`). Recover middleware — единственное место, где `recover()` допустим.

## Документы

| Файл | Разрез | Описание |
|------|--------|----------|
| [01-architecture.md](./01-architecture.md) | Logical | C4 L1 → L2 → L3, граф зависимостей `pkg/httpx/middleware/` ↔ `internal/auth/transport/http/middleware/`, цепочка middleware, контекст userID |
| [02-behavior.md](./02-behavior.md) | Process | DFD по 4 типам запросов + sequence diagrams (auth/public/preflight/panic), error/edge cases |
| [03-decisions.md](./03-decisions.md) | Decision | ADR (нет внешних либ, разделение пакетов, порядок middleware, ENV CORS, request-id, тип ключа в ctx, …), риски, open questions |
| [04-testing.md](./04-testing.md) | Quality | Coverage mapping `AUTH-010..013` ↔ тесты, тест-кейсы по middleware, обновления HTTP-тестов |
| [07-standards.md](./07-standards.md) | Standards | Compliance-матрица по 8 стандартам `prompts/`, уточнения и расхождения |
| [08-api-contract.md](./08-api-contract.md) | Contract | Изменения API surface: preflight `OPTIONS`, новые заголовки в ответах (`X-Request-ID`, `Access-Control-*`), уточнение поведения 401 для `/auth/me` |
| [research.md](./research.md) | Research | Baseline: что уже есть в коде, какие точки внедрения middleware естественны, текущий контракт ошибок |

### Намеренно опущенные документы шаблона

- **`05-events.md`** — middleware не публикует доменных событий. Логирование запросов идёт через `slog`, не через шину.
- **`06-repo-model.md`** — middleware не имеет своих сущностей в БД, нет схемы хранения, нет sqlc-запросов.
