---
phase: 9
name: Channel transport HTTP
layer: transport
depends_on: [phase-05]
plan: ./README.md
---

# Phase 9: HTTP-транспорт channel (handlers + dto + routes + error_mapper + HTTP-тесты)

## Цель

Реализовать `internal/channel/transport/http/`: 3 хендлера, DTO, error_mapper для CHANNEL-NNN, регистрацию роутов под `RequireAuth`, HTTP-тесты.

## Контекст

После Phase 05 у нас есть все 3 use case. Транспорт **импортирует только usecase + domain + chi + pkg/httpx + auth/transport/http/middleware** (для `RequireAuth` и `UserIDFromContext`).

API-контракт — `../08-api-contract.md`. Sequences — `../02-behavior.md` UC-C1..UC-C3. Стиль — `internal/auth/transport/http/` и аналогичный room/transport (Phase 08).

## Файлы для создания

### DTO

#### `internal/channel/transport/http/dto.go`

```go
package httpchannel

import (
    "encoding/json"
    "io"
    "net/http"
    "time"

    "github.com/dovgalb/project-rupor/internal/channel/domain"
)

type createChannelRequest struct {
    Name string `json:"name"`
    Kind string `json:"kind"`
}

type channelResponse struct {
    ID        string    `json:"id"`
    RoomID    string    `json:"roomId"`
    Name      string    `json:"name"`
    Kind      string    `json:"kind"`
    CreatedAt time.Time `json:"createdAt"`
}

type listChannelsResponse struct {
    Items []channelResponse `json:"items"`
}

func channelToResponse(c *domain.Channel) channelResponse {
    return channelResponse{
        ID:        c.ID().String(),
        RoomID:    c.RoomID().String(),
        Name:      c.Name().String(),
        Kind:      c.Kind().String(),
        CreatedAt: c.CreatedAt(),
    }
}

func jsonDecode(r io.Reader, v any) error { return json.NewDecoder(r).Decode(v) }
func jsonEncode(w http.ResponseWriter, v any) error { return json.NewEncoder(w).Encode(v) }
```

### Error mapper

#### `internal/channel/transport/http/error_mapper.go`

```go
package httpchannel

import (
    "errors"
    "net/http"

    authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
    "github.com/dovgalb/project-rupor/internal/channel/domain"
    "github.com/dovgalb/project-rupor/pkg/httpx"
)

type httpError struct {
    status int
    code   string
    msg    string
}

func mapError(err error) httpError {
    switch {
    // 400
    case errors.Is(err, domain.ErrInvalidChannelName):
        return httpError{http.StatusBadRequest, "CHANNEL-001", "invalid channel name"}
    case errors.Is(err, domain.ErrInvalidChannelKind):
        return httpError{http.StatusBadRequest, "CHANNEL-002", "invalid channel kind"}

    // 404
    case errors.Is(err, domain.ErrChannelNotFound):
        return httpError{http.StatusNotFound, "CHANNEL-003", "channel or room not found"}

    // 409
    case errors.Is(err, domain.ErrChannelNameAlreadyTaken):
        return httpError{http.StatusConflict, "CHANNEL-004", "channel name already taken"}

    // 403
    case errors.Is(err, domain.ErrChannelAccessDenied):
        return httpError{http.StatusForbidden, "CHANNEL-006", "access denied: not a member"}
    case errors.Is(err, domain.ErrChannelInsufficientRole):
        return httpError{http.StatusForbidden, "CHANNEL-007", "insufficient role: admin or owner required"}

    // 401
    case errors.Is(err, authdom.ErrAccessTokenInvalid):
        return httpError{http.StatusUnauthorized, "AUTH-010", "access token invalid"}

    default:
        return httpError{http.StatusInternalServerError, "INTERNAL", "internal"}
    }
}

func writeError(w http.ResponseWriter, e httpError) {
    httpx.WriteJSONError(w, e.status, e.code, e.msg)
}

func writeBadBody(w http.ResponseWriter) {
    writeError(w, httpError{http.StatusBadRequest, "CHANNEL-005", "invalid request body"})
}
```

### Routes

#### `internal/channel/transport/http/routes.go`

```go
package httpchannel

import (
    "github.com/go-chi/chi/v5"

    authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
    authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
    "github.com/dovgalb/project-rupor/internal/channel/usecase"
)

type Deps struct {
    CreateChannel *usecase.CreateChannel
    ListChannels  *usecase.ListChannels
    DeleteChannel *usecase.DeleteChannel
    TokenIssuer   authuc.TokenIssuer
    Clock         authuc.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
    r.Route("/rooms/{roomID}/channels", func(r chi.Router) {
        r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock))

        r.Post("/", NewCreateChannelHandler(deps.CreateChannel).ServeHTTP)
        r.Get("/", NewListChannelsHandler(deps.ListChannels).ServeHTTP)
        r.Delete("/{channelID}", NewDeleteChannelHandler(deps.DeleteChannel).ServeHTTP)
    })
}
```

**Внимание о пересечении путей с room-роутами**: в Phase 08 room-роутер уже занимает `/rooms`. Здесь mount `/rooms/{roomID}/channels` — отдельная подгруппа. chi различает пути по template'ам и обрабатывает обе ветки. Порядок Mount в `cmd/server/main.go` (Phase 10) — сначала room, потом channel; при необходимости можно переставить, на routing это не влияет.

### Handlers

Шаблон по `internal/auth/transport/http/me_handler.go` и room-handler'ам из Phase 08.

#### `internal/channel/transport/http/create_channel_handler.go` (POST /rooms/{roomID}/channels → 201)

```go
type CreateChannelHandler struct{ uc *usecase.CreateChannel }

func NewCreateChannelHandler(uc *usecase.CreateChannel) *CreateChannelHandler { ... }

func (h *CreateChannelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    uid, ok := authmw.UserIDFromContext(r.Context())
    if !ok {
        writeError(w, mapError(authdom.ErrAccessTokenInvalid))
        return
    }

    roomIDStr := chi.URLParam(r, "roomID")
    roomID, perr := uuid.Parse(roomIDStr)
    if perr != nil {
        writeError(w, mapError(domain.ErrChannelNotFound))
        return
    }

    var req createChannelRequest
    if err := jsonDecode(r.Body, &req); err != nil {
        writeBadBody(w)
        return
    }

    out, err := h.uc.Execute(r.Context(), usecase.CreateChannelInput{
        ActorID: uid.UUID(),
        RoomID:  roomID,
        Name:    req.Name,
        Kind:    req.Kind,
    })
    if err != nil {
        writeError(w, mapError(err))
        return
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusCreated)
    _ = jsonEncode(w, channelToResponse(out.Channel))
}
```

#### `internal/channel/transport/http/list_channels_handler.go` (GET /rooms/{roomID}/channels → 200)
- Аналогично, без body. Отдаёт `listChannelsResponse{Items: ...}` (пустой массив если нет каналов).

#### `internal/channel/transport/http/delete_channel_handler.go` (DELETE /rooms/{roomID}/channels/{channelID} → 204)
- Парсить `roomID` И `channelID`. На любой ошибке парсинга → CHANNEL-003.
- На успехе — `WriteHeader(204)` без тела.

### HTTP Tests

#### `internal/channel/transport/http/setup_test.go`

По образцу room-setup из Phase 08:
- Реальный chi-router.
- Реальные channel-usecase + фейковые `ChannelRepository` и `MembershipQuery`.
- Реальный `TokenIssuer` (jwt) с тестовым секретом.
- Хелпер `issueTestToken(t, userID)` для Bearer-токенов.

Скопировать фейки из `internal/channel/usecase/fakes_test.go` (другой пакет — нельзя shared).

#### Тесты по handler'у

- `internal/channel/transport/http/create_channel_handler_test.go`:
  - `TestCreateChannel_BadBody_Returns400` (CHANNEL-005)
  - `TestCreateChannel_InvalidRoomUUID_Returns404` (CHANNEL-003)
  - `TestCreateChannel_InvalidName_Returns400` (CHANNEL-001)
  - `TestCreateChannel_InvalidKind_Returns400` (CHANNEL-002)
  - `TestCreateChannel_NotMember_Returns403_CHANNEL006`
  - `TestCreateChannel_AsMember_Returns403_CHANNEL007`
  - `TestCreateChannel_AsAdmin_Returns201`
  - `TestCreateChannel_DuplicateName_Returns409`

- `internal/channel/transport/http/list_channels_handler_test.go`:
  - `TestListChannels_InvalidRoomUUID_Returns404`
  - `TestListChannels_NotMember_Returns403`
  - `TestListChannels_AsMember_Returns200`
  - `TestListChannels_EmptyRoom_ReturnsEmptyArray`

- `internal/channel/transport/http/delete_channel_handler_test.go`:
  - `TestDeleteChannel_InvalidUUID_Returns404`
  - `TestDeleteChannel_NotFound_Returns404`
  - `TestDeleteChannel_NotMember_Returns403`
  - `TestDeleteChannel_AsMember_Returns403_CHANNEL007`
  - `TestDeleteChannel_AsAdmin_Returns204`

## Файлы для модификации

- `internal/channel/transport/http/.gitkeep` — удалить.

## Ключевые решения

- **Mount-паттерн `/rooms/{roomID}/channels`** — channel-транспорт регистрирует свою подсекцию внутри `/api/v1`. chi route-tree обрабатывает оба mount'а (room и channel) корректно благодаря template-matching (`/rooms/{roomID}` vs `/rooms/{roomID}/channels`).
- **404 на невалидный UUID** — семантически: «канал/комната не найдены». Совпадает с дизайном UC-C1/C2/C3.
- **Никакого `mapDeleteChannelError` wrapper'а** — в channel `ErrChannelInsufficientRole` всегда означает «нужен admin/owner» (CHANNEL-007); Owner-only операций здесь нет.
- **Импорт `authdom` (для AUTH-010)** — то же самое, что в room/transport. Защитная ветка после `RequireAuth`.

## Verification

- [ ] `dto.go`, `error_mapper.go`, `routes.go` созданы.
- [ ] 3 handler-файла созданы.
- [ ] `setup_test.go` создан.
- [ ] 3 `<handler>_test.go` файла созданы.
- [ ] `go build ./internal/channel/transport/http/...` чистый.
- [ ] `go test ./internal/channel/transport/http/... -race -count=1` зелёный.
- [ ] Импорты non-test файлов: stdlib + `chi` + `uuid` + `pkg/httpx` + свой usecase/domain + `internal/auth/transport/http/middleware` + `internal/auth/domain` + `internal/auth/usecase`. **Нет** `internal/channel/repository`, `internal/room/...`.
- [ ] `golangci-lint run ./internal/channel/transport/http/...` чистый.
- [ ] `internal/channel/transport/http/.gitkeep` удалён.
