---
date: 2026-05-12
feature: PR-3_1_realtime_chat
design: ../README.md
status: draft
---

# План кода: PR-3_1 — Реалтайм-чат

## Overview

Реализация фазы 3.1 общего плана: миграция `messages`, домен `chat` (Message + VO), use case'ы `SendMessage` / `ListMessages`, реализация репозитория, доменно-агностичный hub в `pkg/websocket/`, chat-handler в `internal/chat/transport/ws/`, REST-эндпоинт `GET /api/v1/channels/:id/messages`, событие `member.joined` от `room/usecase.JoinByCode`, склейка в composition root.

Дизайн — [../README.md](../README.md) (утверждён 2026-05-12). Все архитектурные решения — в [../03-decisions.md](../03-decisions.md). JSON-формы — в [../08-api-contract.md](../08-api-contract.md). Тестовая стратегия — в [../04-testing.md](../04-testing.md).

## Phase Strategy

**Bottom-up.** Migrations → Domain → (parallel: Repository + Hub) → UseCase → Transport (HTTP + WS) → Room-extension → Composition Root.

Почему:
- Каждая фаза тестируется изолированно (domain без БД, hub без chat, repository — integration).
- `pkg/websocket` не зависит от `internal/chat`, можно реализовать параллельно с doman/repository (phase-03 и phase-02/phase-04 — независимые).
- Use case требует домен + порты, но не требует адаптеров (фейки в тестах).
- Транспорт требует use case.
- Composition root — последний шаг, склейка всего.

Vertical slice не выбран: WS-handler и REST-handler делят один use case, нет смысла дублировать domain/repository по фазам endpoint'ов. Adapter-first отброшен: домен не зависит от схемы БД, поэтому начало с миграции + домена корректно.

## Phases

| # | Фаза | Слой | Зависимости | Status |
|---|---|---|---|---|
| 01 | Миграция БД `0008_messages` | migrations | none | ☐ |
| 02 | Домен `chat/domain` | domain | none | ☐ |
| 03 | `pkg/websocket` (Hub + Conn + Topic + Upgrade) | infrastructure | none | ☐ |
| 04 | Репозиторий `chat/repository/postgres` + расширение `room/repository/postgres` | repository | 01, 02 | ☐ |
| 05 | Use case `chat/usecase` (ports + SendMessage + ListMessages) | usecase | 02 | ☐ |
| 06 | Транспорт HTTP `chat/transport/http` | transport | 05 | ☐ |
| 07 | Транспорт WS `chat/transport/ws` | transport | 03, 05 | ☐ |
| 08 | Расширение `room/usecase.JoinByCode` (RoomEventsPublisher) | usecase | 03 | ☐ |
| 09 | Composition root + sqlc.yaml + go.mod + arch_test + Logger sanitizer | infrastructure | 01..08 | ☐ |

## File Map

### New Files

#### Миграции
- `migrations/0008_messages.up.sql` — таблица `messages` + индекс `(channel_id, created_at DESC, id DESC)` + индекс `author_id`
- `migrations/0008_messages.down.sql` — `DROP TABLE IF EXISTS messages`

#### Домен `internal/chat/domain/`
- `internal/chat/domain/message.go` — `Message` rich entity, `NewMessage`, `ReconstructMessage`, геттеры
- `internal/chat/domain/message_id.go` — VO `MessageID` (uuid)
- `internal/chat/domain/message_text.go` — VO `MessageText` (1..4000 рун, TrimSpace, control-chars whitelist)
- `internal/chat/domain/channel_id.go` — VO `ChannelID`
- `internal/chat/domain/user_id.go` — VO `UserID`
- `internal/chat/domain/room_id.go` — VO `RoomID`
- `internal/chat/domain/errors.go` — `ErrInvalidMessageText`, `ErrInvalidMessageID`, `ErrInvalidChannelID`, `ErrInvalidAuthorID`, `ErrInvalidCreatedAt`, `ErrChannelNotFound`, `ErrChannelNotText`, `ErrChatAccessDenied`, `ErrChatInsufficientRole`, `ErrMessageNotFound`

#### Тесты домена
- `internal/chat/domain/message_test.go`
- `internal/chat/domain/message_text_test.go`
- `internal/chat/domain/message_id_test.go`
- `internal/chat/domain/channel_id_test.go`
- `internal/chat/domain/user_id_test.go`
- `internal/chat/domain/room_id_test.go`
- `internal/chat/domain/testing_builders_test.go` — test builder для `Message`

#### `pkg/websocket/`
- `pkg/websocket/hub.go` — `Hub` (регистрация, подписки, broadcast)
- `pkg/websocket/conn.go` — `Conn` (обёртка над nhooyr conn + write mutex)
- `pkg/websocket/topic.go` — `Topic` (struct с `Kind` + `UUID`, `Key() string`)
- `pkg/websocket/upgrade.go` — `Upgrade(ctx, w, r, opts)`
- `pkg/websocket/hub_test.go` — тесты hub'а (subscribe, publish, disconnect, shutdown, slow consumer, race)

#### Use case `internal/chat/usecase/`
- `internal/chat/usecase/ports.go` — `MessageRepository`, `MembershipQuery`, `RoleRequirement`, `Broadcaster`, `Clock`, `UUIDGenerator`, `ChannelInfo`
- `internal/chat/usecase/send_message.go` — `SendMessage`
- `internal/chat/usecase/list_messages.go` — `ListMessages`
- `internal/chat/usecase/send_message_test.go`
- `internal/chat/usecase/list_messages_test.go`
- `internal/chat/usecase/fakes_test.go` — ручные фейки портов

#### Транспорт HTTP `internal/chat/transport/http/`
- `internal/chat/transport/http/routes.go` — `Deps`, `RegisterRoutes(r chi.Router, deps Deps)`
- `internal/chat/transport/http/list_messages_handler.go`
- `internal/chat/transport/http/dto.go` — `messageResponse`, `listMessagesResponse`, `messageToResponse`, `jsonEncode`/`jsonDecode`
- `internal/chat/transport/http/error_mapper.go` — `mapError`, `writeError`, `writeBadBody`
- `internal/chat/transport/http/list_messages_handler_test.go`
- `internal/chat/transport/http/setup_test.go`

#### Транспорт WS `internal/chat/transport/ws/`
- `internal/chat/transport/ws/routes.go` — `WSDeps`, `RegisterWSRoute(r chi.Router, deps WSDeps)`
- `internal/chat/transport/ws/handler.go` — `WSHandler.ServeHTTP` (auth → upgrade → register → read-loop)
- `internal/chat/transport/ws/event_dto.go` — `inboundEvent`, `outboundEvent`, error-frame helpers
- `internal/chat/transport/ws/handler_test.go`
- `internal/chat/transport/ws/setup_test.go`

#### Репозиторий `internal/chat/repository/postgres/`
- `internal/chat/repository/postgres/message_repository.go`
- `internal/chat/repository/postgres/mapper.go`
- `internal/chat/repository/postgres/pgerr.go`
- `internal/chat/repository/postgres/queries/messages.sql` — `InsertMessage :exec`, `GetMessageByID :one`, `ListMessagesBeforeCursor :many`, `GetChannelKind :one`
- `internal/chat/repository/postgres/db/` — сгенерируется sqlc (`db.go`, `models.go`, `messages.sql.go`)
- `internal/chat/repository/postgres/compile_check_test.go`
- `internal/chat/repository/postgres/integration_helpers_test.go`
- `internal/chat/repository/postgres/message_repository_integration_test.go`

#### Room extensions (новые файлы)
- `internal/room/repository/postgres/membership_query_chat.go` — `MembershipQueryChatAdapter`, реализует `chat/usecase.MembershipQuery`
- `internal/room/repository/postgres/membership_query_chat_integration_test.go`

### Modified Files

| Файл | Что меняется |
|---|---|
| `go.mod` | + `nhooyr.io/websocket v1.8.x` |
| `go.sum` | автоматически (`go mod tidy`) |
| `sqlc.yaml` | + 4-й `sql:` блок для `internal/chat/repository/postgres/` |
| `cmd/server/main.go` | + composition chat-домена; + hub'а; + регистрация роутов REST и WS; + биндинг `RoomEventsPublisher` в `room.JoinByCode`; + чистый shutdown hub'а |
| `internal/room/usecase/ports.go` | + интерфейс `RoomEventsPublisher` |
| `internal/room/usecase/join_by_code.go` | + поле `events RoomEventsPublisher` в struct; + параметр в `NewJoinByCode`; + вызов `events.PublishMemberJoined(...)` после `Add` |
| `internal/room/usecase/join_by_code_test.go` | + fake publisher; + тесты публикации |
| `internal/room/repository/postgres/queries/room_members.sql` | + `GetMemberForChannel :one`; + `ChannelExists :one` |
| `internal/room/repository/postgres/db/room_members.sql.go` | автоматически (`make sqlc`) |
| `arch_test.go` | + тесты для `chat/{domain,usecase,repository/postgres,transport/http,transport/ws}` + расширение разрешения `room/repository/postgres` на импорт `chat/{usecase,domain}` + правило для `pkg/websocket` |
| `pkg/httpx/middleware/logger.go` | + санитайз `?token=` в логируемом URL для пути `/api/v1/ws` |
| `pkg/httpx/middleware/logger_test.go` | + `TestLogger_StripsTokenFromWSPath` |

## DI Integration

### Init chain position

Биндинг идёт в существующем порядке `cmd/server/main.go:71-176`:

1. `pool := pgxpool.New(...)` (`main.go:74`) — без изменений
2. `queries := db.New(pool)` — без изменений
3. **(новое)** `chatDB := chatdb.New(pool)` — генерация sqlc для chat
4. Существующая сборка `auth`, `room`, `channel` — без изменений
5. **(новое)** Hub:
   ```go
   hub := websocket.NewHub(logger)
   chatBroadcaster := newChatBroadcasterAdapter(hub)   // тонкий враппер
   roomEvents      := newRoomEventsAdapter(hub)        // тонкий враппер
   ```
6. **(новое)** Chat composition:
   ```go
   messageRepo := chatpg.NewMessageRepository(pool)
   membershipForChat := roompg.NewMembershipQueryChatAdapter(pool)
   sendMessageUC := chatuc.NewSendMessage(messageRepo, membershipForChat, chatBroadcaster, clock, uuids)
   listMessagesUC := chatuc.NewListMessages(messageRepo, membershipForChat)
   ```
7. **(модификация)** `joinByCodeUC := roomusecase.NewJoinByCode(inviteRepo, membershipRepo, roomRepo, clock, roomEvents)` — добавляется параметр `roomEvents`.
8. **(новое)** Регистрация роутов:
   ```go
   httpchat.RegisterRoutes(r, httpchat.Deps{
       ListMessages: listMessagesUC,
       TokenIssuer:  issuer,
       Clock:        clock,
   })
   wschat.RegisterWSRoute(r, wschat.WSDeps{
       Hub:              hub,
       SendMessage:      sendMessageUC,
       MembershipForChat: membershipForChat,
       MembershipForRooms: membershipRepo,   // для авто-подписки на room-topics при connect
       TokenIssuer:      issuer,
       Clock:            clock,
   })
   ```
9. **(модификация)** Graceful shutdown: перед `srv.Shutdown(shutdownCtx)` вызывается `hub.Shutdown(shutdownCtx)`.

### Composition root changes

Точные диапазоны строк — указаны в `phase-09-composition-root.md`. Общий объём изменений в `main.go` — ~50 новых строк, ~10 модифицированных.

### Initialization order (последовательность)

1. config → pool → queries → repositories → use cases (auth, room, channel) — как сейчас
2. hub (new) — создаётся ДО chat-use case'ов, потому что они зависят от Broadcaster
3. chat repos и use cases (new)
4. router + middleware (как сейчас)
5. routes registration (REST + WS, дополнительно chat) — как сейчас плюс новые
6. server.ListenAndServe + signal context (как сейчас)
7. shutdown: hub.Shutdown → srv.Shutdown — порядок важен, чтобы клиенты получили close-frame до того, как HTTP-сервер прекратит ответ

## Error Codes

**Range:** `CHAT-001..CHAT-007` (новый префикс).

**Conflict check:** Verified — заняты `AUTH-001..012`, `ROOM-001..009`, `CHANNEL-001..007`, `CONFIG-001..006`, `INTERNAL`. Префикс `CHAT-*` свободен.

| Code | Description | HTTP Status | WS-фрейм |
|---|---|---|---|
| CHAT-001 | Невалидный текст (пустой, > 4000 рун, control-chars) | 400 | error |
| CHAT-002 | Канал не найден | 404 | error |
| CHAT-003 | Канал не текстовый (voice) | 400 | error |
| CHAT-004 | Не член комнаты канала | 403 | error |
| CHAT-005 | Невалидный UUID в path/query/payload | 400 | error |
| CHAT-006 | `limit` вне диапазона `1..100` | 400 | — |
| CHAT-007 | Неподдерживаемый WS-фрейм `type` (зарезервировано) | — | error |

См. [`../08-api-contract.md`](../08-api-contract.md) для точных JSON-форм.

## Success Criteria

- [ ] Все 9 фаз завершены и помечены ☑ в этой таблице
- [ ] `make test` проходит локально и в CI
- [ ] `go test -race ./...` проходит (включая hub-тесты)
- [ ] Все коды ошибок `CHAT-001..006` покрыты тестами (см. [`../04-testing.md`](../04-testing.md) §Coverage Mapping)
- [ ] Интеграционные тесты `chat/repository/postgres` зелёные против `TEST_DATABASE_URL`
- [ ] `go build ./...` чистый
- [ ] `make lint` чистый (включая новые правила в `arch_test.go`)
- [ ] API-контракт совпадает с реализацией: `08-api-contract.md` ≡ `chat/transport/http/dto.go` и WS-event-DTO
- [ ] Все 10 критериев приёмки из [`../README.md`](../README.md) §Критерии приёмки — выполнены
- [ ] WS-токен не утекает в access-лог (тест `TestLogger_StripsTokenFromWSPath` зелёный)
- [ ] `manual_qa/3_1_realtime_chat/` пополнен `.http`-файлами для REST и manual WS-сценариями (по аналогии с существующим `manual_qa/2_1_rooms_and_channels/00_flow.http`)
- [ ] `general_plan.md` отмечен чекбоксами `[x]` для всех пунктов фазы 3.1

## Phases (ссылки)

1. [phase-01-migration.md](./phase-01-migration.md) — Миграция БД
2. [phase-02-chat-domain.md](./phase-02-chat-domain.md) — Домен chat
3. [phase-03-pkg-websocket.md](./phase-03-pkg-websocket.md) — `pkg/websocket`
4. [phase-04-chat-repository.md](./phase-04-chat-repository.md) — Repository + расширение MembershipQuery
5. [phase-05-chat-usecase.md](./phase-05-chat-usecase.md) — Use case
6. [phase-06-chat-transport-http.md](./phase-06-chat-transport-http.md) — Транспорт HTTP
7. [phase-07-chat-transport-ws.md](./phase-07-chat-transport-ws.md) — Транспорт WS
8. [phase-08-room-extension.md](./phase-08-room-extension.md) — Расширение room (`member.joined`)
9. [phase-09-composition-root.md](./phase-09-composition-root.md) — Composition root + go.mod + arch_test + Logger
