---
date: 2026-05-12
feature: PR-3_1_realtime_chat
---

# Ресерч: PR-3_1 — Реалтайм-чат

Полный отчёт исследования — `/.thoughts/research/2026-05-12-3_1-realtime-chat.md` (коммит `60bb055`, ветка `PR-3_1-realtime_chat`). Здесь — сжатая сводка фактов, релевантных для дизайна.

## Резюме

Фазы 1 и 2 общего плана завершены: домены `auth`, `room`, `channel`, миграции `0001-0007`, HTTP-роуты под `/api/v1/`, JWT-аутентификация, middleware-стек, набор интеграционных и юнит-тестов. Готовы переиспользуемые порты и реализации: `usecase.TokenIssuer.VerifyAccess(token, now)` (`internal/auth/usecase/ports.go:29-32`); кросс-доменный порт `channel/usecase.MembershipQuery.Require(...)` с реализацией `MembershipQueryAdapter` в `internal/room/repository/postgres/membership_query.go:29-66`; helpers `WithUserID`/`UserIDFromContext`. Текущий `RequireAuth` (`internal/auth/transport/http/middleware/auth.go:17-38`) читает только `Authorization: Bearer` — для WebSocket-апгрейда потребуется отдельный аутентификатор, читающий JWT из query.

Фаза 3 в коде ещё не начата: `pkg/websocket/` содержит только `.gitkeep`; библиотека `nhooyr.io/websocket` отсутствует в `go.mod` (`go.mod:5-19`); `internal/chat/` не существует; миграция `0008_messages` не написана; sqlc-конфиг не расширен для chat; arch_test.go не покрывает chat/websocket.

Архитектурные паттерны проекта стабильны и применимы для chat: rich domain model с value objects и приватными полями (см. `prompts/Domain Model.txt`), кросс-доменные порты в usecase-слое (как `channel/usecase.MembershipQuery`), маппинг через `Reconstruct...` в репозитории, sqlc через `pgx/v5`. Стиль тестов — без mock-генераторов, ручные fakes (`prompts/Tests Style.txt`), `t.Parallel`, AAA-структура, SUT-паттерн для use case-тестов, `httptest.NewServer` для HTTP-интеграции.

## Структура проекта (обнаружено)

- **Init chain** (`cmd/server/main.go:71-176`): config → pgxpool → sqlc Queries → repositories → usecases → chi router → middleware chain → routes → server.
- **Роутер**: chi, монтаж под `/api/v1` (`main.go:148-176`), глобальные middleware (RequestID, Recover, Logger, CORS) перед `Route`. Каждый домен экспортирует `RegisterRoutes(r chi.Router, deps Deps)`. Внутри группы домен сам применяет `r.Use(authmw.RequireAuth(...))` (`internal/channel/transport/http/routes.go:21`).
- **Сущности**: rich model с приватными полями, конструкторы `NewXxx` + `ReconstructXxx`, value objects для всех типизированных значений (`internal/channel/domain/channel.go:5-55`).
- **Репозиторий**: конструктор принимает `*pgxpool.Pool`, держит `*db.Queries`, маппит через `domainToInsertXxxParams` / `xxxRowToDomain`, обрабатывает unique-violation через `pgerr.go::isUniqueViolation(err, constraintName)` (`internal/channel/repository/postgres/channel_repository.go:23-31`).
- **Используемые диапазоны кодов ошибок**:
  - `AUTH-001..AUTH-012` (`internal/auth/transport/http/error_mapper.go`)
  - `ROOM-001..ROOM-009` (`internal/room/transport/http/error_mapper.go`)
  - `CHANNEL-001..CHANNEL-007` (`internal/channel/transport/http/error_mapper.go`)
  - `CONFIG-001..CONFIG-006` (`config/config.go`)
  - `INTERNAL` — общий fallback
- **Следующий свободный префикс**: `CHAT-001..CHAT-NNN` (см. `08-api-contract.md`).

## Обзор архитектуры

Clean Architecture (`prompts/Architecture Layers.txt`): `domain` ← `usecase` ← `transport`/`repository`. Кросс-доменные зависимости только через порты в usecase-слое; единственное санкционированное исключение зафиксировано в `arch_test.go:259-280` — `room/repository/postgres` реализует `channel/usecase.MembershipQuery`. arch_test.go (`arch_test.go:1-418`) проверяет правила импортов через AST-парсер.

Для chat паттерн повторяется: `chat/domain` чистый; `chat/usecase` объявляет порты `MessageRepository`, `MembershipQuery` (зеркало для chat), `Broadcaster` (новое), `Clock`, `UUIDGenerator`; `chat/transport/http` и `chat/transport/ws` парсят запросы; `chat/repository/postgres` реализует `MessageRepository`. Hub в `pkg/websocket/` реализует порт `Broadcaster`; адаптер расширяется в `room/repository/postgres/membership_query.go` для нового порта `chat/usecase.MembershipQuery` (по аналогии с уже существующим).

## Существующие паттерны

| Паттерн | Где смотреть | Применение для chat |
|---|---|---|
| Rich domain model + VO | `internal/channel/domain/channel.go`, `channel_name.go`, `channel_kind.go` | `Message`, `MessageText`, `MessageID` |
| `NewXxx` + `ReconstructXxx` | `internal/channel/domain/channel.go:13-49` | Те же два конструктора у `Message` |
| Доменные ошибки | `internal/channel/domain/errors.go` | `chat/domain/errors.go` |
| Кросс-доменный порт + адаптер | `internal/channel/usecase/ports.go:33-35` + `internal/room/repository/postgres/membership_query.go` | Зеркало для `chat/usecase.MembershipQuery` + расширение того же адаптера |
| Use case SUT-паттерн | `internal/channel/usecase/create_channel_test.go:15-48` | `send_message_test.go`, `list_messages_test.go` |
| Ручные fakes без gomock | `internal/channel/usecase/fakes_test.go` | `fakes_test.go` в `chat/usecase` |
| HTTP-handler шаблон | `internal/channel/transport/http/list_channels_handler.go` | `list_messages_handler.go` |
| `mapError` + `httpx.WriteJSONError` | `internal/channel/transport/http/error_mapper.go` | Аналогичный mapper для chat |
| chi-роутер с `r.Use(RequireAuth(...))` | `internal/channel/transport/http/routes.go:20-26` | REST chat: `r.Use(RequireAuth(...))`; WS chat: ручная проверка токена из query |
| Postgres-репо + sqlc + mapper + pgerr | `internal/channel/repository/postgres/{channel_repository.go, mapper.go, pgerr.go}` | `message_repository.go`, `mapper.go`, `pgerr.go` |
| Шаблон миграции таблицы | `migrations/0007_channels.up.sql` | `migrations/0008_messages.up.sql` |
| Compile-check теста | `internal/channel/repository/postgres/compile_check_test.go` | Аналог для `MessageRepository` |
| Integration-helpers | `internal/channel/repository/postgres/integration_helpers_test.go` | Расширить `seedRoom` до `seedChannel` |
| arch_test.go | `arch_test.go:316-417` (channel) | Добавить аналогичные тесты для `chat` и `pkg/websocket` |

## Точки интеграции

- **`cmd/server/main.go`** (composition root): добавить инстанцирование `chat`-репозитория, usecase'ов, hub'а, ws-handler; зарегистрировать routes; добавить `httpchat.RegisterRoutes(...)` под `/api/v1` и `ws` handler на `/api/v1/ws`.
- **`go.mod`**: новая зависимость `nhooyr.io/websocket`.
- **`config/config.go`**: при необходимости добавить env-переменные для WS (max message size, deadlines, ping interval) — отдельный код ошибки `CONFIG-007+`.
- **`sqlc.yaml`**: 4-й блок `sql:` для `internal/chat/repository/postgres/`.
- **`migrations/`**: `0008_messages.up.sql` / `0008_messages.down.sql`.
- **`arch_test.go`**: новые тесты для `internal/chat/*` и `pkg/websocket`.
- **`internal/room/repository/postgres/membership_query.go`**: расширить адаптер реализацией `chat/usecase.MembershipQuery` (новая публичная структура `MembershipQueryChatAdapter` или метод-фабрика — детали в `03-decisions.md`).
- **`internal/room/usecase/`**: добавление порта-нотификатора для `member.joined` (см. `05-events.md`); затрагивает `JoinByCode`.
- **JWT**: переиспользование `TokenIssuer.VerifyAccess(token, now)` напрямую из ws-handler'а (`internal/auth/usecase/ports.go:31`).
- **`pkg/httpx`**: формат ошибок остаётся для REST. Для WS — отдельный формат `error`-фреймов (см. `08-api-contract.md`).
