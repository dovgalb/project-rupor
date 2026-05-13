---
phase: 7
name: chat-transport-ws
layer: transport
depends_on: [phase-03, phase-05]
plan: ./README.md
---

# Phase 7: WS-транспорт `internal/chat/transport/ws/`

## Цель

Реализовать WebSocket-эндпоинт `GET /api/v1/ws?token=<jwt>`: аутентификация по query-токену, апгрейд через `pkg/websocket.Upgrade`, регистрация в hub, автоподписка на room-topic'и для всех членств пользователя, цикл чтения JSON-фреймов и роутинг команд (`subscribe`, `message.send`).

## Контекст

Phase-03 создала hub. Phase-05 создала use case `SendMessage`. Для авто-подписки на room-topic нужен дополнительный порт — `MembershipReader` (упрощённый «список room_id'ов пользователя»). Реализация — существующий `room/repository/postgres.MembershipRepository.ListByUser` (уже есть в проекте — см. `internal/room/repository/postgres/membership_repository.go:43-60`).

Шаблон handler'а — `internal/channel/transport/http/list_channels_handler.go` для structure, но WS-handler сложнее (long-lived connection, read-loop).

WS-протокол — [`../08-api-contract.md §WebSocket`](../08-api-contract.md). События — [`../05-events.md`](../05-events.md).

## Файлы для создания

### `internal/chat/transport/ws/routes.go`

```go
package wschat

import (
    "github.com/go-chi/chi/v5"

    authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
    "github.com/dovgalb/project-rupor/internal/chat/usecase"
    "github.com/dovgalb/project-rupor/pkg/websocket"
)

type WSDeps struct {
    Hub                *websocket.Hub
    SendMessage        *usecase.SendMessage
    MembershipForChat  usecase.MembershipQuery
    MembershipForRooms MembershipReader  // см. ниже
    TokenIssuer        authuc.TokenIssuer
    Clock              authuc.Clock
    OriginPatterns     []string          // из cfg.CORSAllowedOrigins (без scheme)
}

func RegisterWSRoute(r chi.Router, deps WSDeps) {
    r.Get("/ws", NewWSHandler(deps).ServeHTTP)
}
```

### `internal/chat/transport/ws/ports.go`

```go
package wschat

import (
    "context"

    "github.com/google/uuid"
)

// MembershipReader — миниатюрный порт для автоподписки на room-topic'и при connect.
// Реализуется существующим room/repository/postgres.MembershipRepository (метод ListByUser).
type MembershipReader interface {
    ListRoomIDsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}
```

**Замечание:** существующий `room/repository/postgres.MembershipRepository.ListByUser` возвращает `[]*room/domain.Membership`. Нам нужен только список `room_id`. Варианты:
- (a) Добавить метод `ListRoomIDsByUser(ctx, userID) ([]uuid.UUID, error)` в `room/repository/postgres.MembershipRepository` + новый SQL `ListRoomIDsByUser`.
- (b) Создать thin-adapter в `cmd/server/main.go`, который оборачивает `ListByUser` и извлекает `roomID`.

**Решение:** (b). Это минимальное изменение, не трогаем room. Adapter в phase-09:
```go
type roomIDsAdapter struct{ repo *roompg.MembershipRepository }

func (a roomIDsAdapter) ListRoomIDsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
    ms, err := a.repo.ListByUser(ctx, /* конвертация uuid → room.domain.UserID */)
    if err != nil { return nil, err }
    out := make([]uuid.UUID, 0, len(ms))
    for _, m := range ms { out = append(out, m.RoomID().UUID()) }
    return out, nil
}
```

Если в room-репо нет `ListByUser` (только `ListByRoom`) — добавить в фазе 7. **Проверить перед стартом фазы.** Если требуется новый метод, расширить phase-04 (room-extensions).

### `internal/chat/transport/ws/event_dto.go`

```go
package wschat

import (
    "encoding/json"

    "github.com/google/uuid"
)

type inboundEvent struct {
    Type      string          `json:"type"`
    ChannelID string          `json:"channel_id,omitempty"`
    Text      string          `json:"text,omitempty"`
    Raw       json.RawMessage `json:"-"`
}

// EventType constants
const (
    EventTypeSubscribe   = "subscribe"
    EventTypeMessageSend = "message.send"

    OutSubscribed   = "subscribed"
    OutMessageSent  = "message.sent"
    OutError        = "error"
)

func errorFrame(code, msg string) map[string]any {
    return map[string]any{
        "type": OutError,
        "data": map[string]any{
            "code":    code,
            "message": msg,
        },
    }
}

func subscribedFrame(channelID uuid.UUID) map[string]any {
    return map[string]any{
        "type": OutSubscribed,
        "data": map[string]any{"channel_id": channelID.String()},
    }
}

func messageSentFrame(messageID, channelID uuid.UUID, createdAt string) map[string]any {
    return map[string]any{
        "type": OutMessageSent,
        "data": map[string]any{
            "id":         messageID.String(),
            "channel_id": channelID.String(),
            "created_at": createdAt,
        },
    }
}
```

### `internal/chat/transport/ws/error_mapper.go`

```go
package wschat

import (
    "errors"

    "github.com/dovgalb/project-rupor/internal/chat/domain"
)

func mapDomainError(err error) (code, msg string) {
    switch {
    case errors.Is(err, domain.ErrInvalidMessageText):
        return "CHAT-001", "invalid message text"
    case errors.Is(err, domain.ErrChannelNotFound):
        return "CHAT-002", "channel not found"
    case errors.Is(err, domain.ErrChannelNotText):
        return "CHAT-003", "channel is not text"
    case errors.Is(err, domain.ErrChatAccessDenied):
        return "CHAT-004", "access denied: not a room member"
    case errors.Is(err, domain.ErrInvalidMessageID),
         errors.Is(err, domain.ErrInvalidChannelID),
         errors.Is(err, domain.ErrInvalidAuthorID):
        return "CHAT-005", "invalid uuid in payload"
    default:
        return "INTERNAL", "internal"
    }
}
```

### `internal/chat/transport/ws/handler.go`

Главный файл — handler-сценарий. Объём — крупный (~200 строк), поэтому пишем поэтапно. Псевдокод:

```go
package wschat

import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"

    "github.com/google/uuid"
    ws "nhooyr.io/websocket"

    authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
    authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
    "github.com/dovgalb/project-rupor/internal/chat/usecase"
    "github.com/dovgalb/project-rupor/pkg/httpx"
    pws "github.com/dovgalb/project-rupor/pkg/websocket"
)

type WSHandler struct {
    deps WSDeps
}

func NewWSHandler(deps WSDeps) *WSHandler {
    return &WSHandler{deps: deps}
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    logger := slog.Default()  // или из r.Context() если будет middleware

    // 1. Аутентификация ДО апгрейда.
    token := r.URL.Query().Get("token")
    if token == "" {
        httpx.WriteJSONError(w, http.StatusUnauthorized, "AUTH-010", "access token invalid")
        return
    }
    userID, err := h.deps.TokenIssuer.VerifyAccess(token, h.deps.Clock.Now())
    if err != nil {
        switch {
        case errors.Is(err, authdom.ErrAccessTokenExpired):
            httpx.WriteJSONError(w, http.StatusUnauthorized, "AUTH-011", "access token expired")
        default:
            httpx.WriteJSONError(w, http.StatusUnauthorized, "AUTH-010", "access token invalid")
        }
        return
    }

    // 2. Список room_id пользователя — для авто-подписки.
    roomIDs, err := h.deps.MembershipForRooms.ListRoomIDsByUser(ctx, userID.UUID())
    if err != nil {
        logger.Error("ws: list room ids", slog.Any("err", err))
        httpx.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL", "internal")
        return
    }

    // 3. Апгрейд.
    rawConn, err := pws.Upgrade(w, r, pws.UpgradeOptions{
        OriginPatterns: h.deps.OriginPatterns,
    })
    if err != nil {
        logger.Warn("ws: upgrade failed", slog.Any("err", err))
        return
    }

    // 4. Регистрация и автоподписка.
    conn := pws.NewConn(userID.UUID(), rawConn)  // конструктор экспортируется в phase-03 (пометить экспорт)
    cleanup := h.deps.Hub.Register(conn)
    defer cleanup()

    for _, roomID := range roomIDs {
        h.deps.Hub.Subscribe(conn, pws.RoomTopic(roomID))
    }

    // 5. Read loop.
    h.readLoop(ctx, conn, userID.UUID())
}

func (h *WSHandler) readLoop(ctx context.Context, conn *pws.Conn, userID uuid.UUID) {
    for {
        var in inboundEvent
        if err := conn.ReadJSON(ctx, &in); err != nil {
            // close-code зависит от типа ошибки; nhooyr возвращает специфичные
            return
        }
        switch in.Type {
        case EventTypeSubscribe:
            h.handleSubscribe(ctx, conn, userID, in)
        case EventTypeMessageSend:
            h.handleMessageSend(ctx, conn, userID, in)
        default:
            _ = conn.WriteJSON(ctx, errorFrame("CHAT-007", "unsupported event type"))
        }
    }
}

func (h *WSHandler) handleSubscribe(ctx context.Context, conn *pws.Conn, userID uuid.UUID, in inboundEvent) {
    channelID, err := uuid.Parse(in.ChannelID)
    if err != nil {
        _ = conn.WriteJSON(ctx, errorFrame("CHAT-005", "invalid channel_id uuid"))
        return
    }
    // Проверка прав через chat/usecase.MembershipQuery.
    chatChannelID, _ := domain.NewChannelID(channelID)
    chatUserID, _ := domain.NewUserID(userID)
    if err := h.deps.MembershipForChat.Require(ctx, chatChannelID, chatUserID, usecase.RoleAnyMember); err != nil {
        code, msg := mapDomainError(err)
        _ = conn.WriteJSON(ctx, errorFrame(code, msg))
        return
    }
    h.deps.Hub.Subscribe(conn, pws.ChannelTopic(channelID))
    _ = conn.WriteJSON(ctx, subscribedFrame(channelID))
}

func (h *WSHandler) handleMessageSend(ctx context.Context, conn *pws.Conn, userID uuid.UUID, in inboundEvent) {
    channelID, err := uuid.Parse(in.ChannelID)
    if err != nil {
        _ = conn.WriteJSON(ctx, errorFrame("CHAT-005", "invalid channel_id uuid"))
        return
    }
    out, err := h.deps.SendMessage.Execute(ctx, usecase.SendMessageInput{
        ActorID:   userID,
        ChannelID: channelID,
        Text:      in.Text,
    })
    if err != nil {
        code, msg := mapDomainError(err)
        _ = conn.WriteJSON(ctx, errorFrame(code, msg))
        return
    }
    _ = conn.WriteJSON(ctx, messageSentFrame(out.MessageID, out.ChannelID, out.CreatedAt.Format(time.RFC3339Nano)))
}
```

**Замечания:**
- 401-ответы ДО апгрейда — через `httpx.WriteJSONError` (как REST). Это позволяет клиенту распарсить HTTP-ответ как JSON.
- После апгрейда — все ошибки только через WS-фреймы.
- `pws.NewConn(userID, rawConn)` — экспортированный конструктор в `pkg/websocket/conn.go` (в phase-03 он `newConn` — приватный). Сделать экспортированным.
- Внутренний state handler'а минимален: один read-loop на одной горутине. Конкурентные write'ы — через `conn.WriteJSON` (mutex внутри).
- Disconnect — через `defer cleanup()` срабатывает на любой return из `readLoop` (включая close-frame, error, context cancel).

### `internal/chat/transport/ws/setup_test.go`

Шаблон — `internal/channel/transport/http/setup_test.go`. Поднимает `httptest.NewServer` с реальным WSHandler, реальным `Hub`, реальным `SendMessage` use case, fake-репо. Клиент-сторона — `nhooyr.io/websocket.Dial`.

Хелперы:
- `connectWS(t, env, token) *websocket.Conn`
- `readFrame(t, c) inboundEvent`
- `writeFrame(t, c, v any)`

### `internal/chat/transport/ws/handler_test.go`

14 тестов из [`../04-testing.md §WSHandler`](../04-testing.md).

## Файлы для модификации

### `pkg/websocket/conn.go` (phase-03)

Экспортировать конструктор `NewConn`:

```go
// NewConn создаёт Conn вокруг уже принятого *websocket.Conn.
// Используется handler'ами после Upgrade.
func NewConn(userID uuid.UUID, raw *ws.Conn) *Conn {
    return &Conn{id: uuid.New(), userID: userID, ws: raw}
}
```

В phase-03 он был приватным `newConn`. Изменить на публичный.

## Ключевые решения

- **Аутентификация до апгрейда** — HTTP 401 с JSON-body. После апгрейда — close-frame или WS error-frame.
- **Авто-подписка на room-topic'и при connect** — [§ D-09](../03-decisions.md). Один SQL-запрос (`ListRoomIDsByUser`) + N hub.Subscribe вызовов.
- **Read-loop на той же горутине, что и handler** — нет лишних горутин. `defer cleanup()` гарантирует unregister на любой return.
- **Адаптер `roomIDsAdapter` живёт в `cmd/server/main.go`** — единственное место, где chat-handler видит реализацию `MembershipReader`. Сам handler видит только порт.
- **`mapDomainError` отдельно от REST'овского** — потому что WS-фрейм с кодом, а не HTTP-status. Логически — те же CHAT-коды, разный формат.

## Verification

- [ ] `go build ./internal/chat/transport/ws/...` без ошибок.
- [ ] `go test ./internal/chat/transport/ws/...` зелёный.
- [ ] `go test -race ./internal/chat/transport/ws/...` зелёный.
- [ ] Все 14 тестов из [04-testing.md §WSHandler](../04-testing.md) написаны и проходят.
- [ ] Импорты — только stdlib + uuid + chi + `internal/chat/{domain,usecase}` + `internal/auth/{domain,usecase}` + `pkg/{httpx,websocket}` + `nhooyr.io/websocket`. (Arch-тест в phase-09.)
- [ ] 401-ответы (AUTH-010, AUTH-011) возвращаются ДО апгрейда — клиент видит HTTP-status, не close-frame.
- [ ] `TestWS_MessageSend_AsMember_PersistsAndBroadcasts` — client A шлёт, client B получает `message.new` через свою подписку.
- [ ] `TestWS_ClientCloses_HubUnregistersAndCleansSubscriptions` — после close в hub пусто.
- [ ] `TestWS_ServerShutdown_ClosesWith1001` — при `hub.Shutdown` клиенты получают close-frame 1001.
