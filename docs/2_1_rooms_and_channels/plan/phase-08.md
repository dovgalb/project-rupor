---
phase: 8
name: Room transport HTTP
layer: transport
depends_on: [phase-04]
plan: ./README.md
---

# Phase 8: HTTP-транспорт room (handlers + dto + routes + error_mapper + HTTP-тесты)

## Цель

Реализовать `internal/room/transport/http/`: 7 хендлеров (по одному на use case), DTO с json-тегами, error_mapper для ROOM-NNN кодов, регистрацию роутов под защитой `RequireAuth`, HTTP-тесты через `httptest`.

## Контекст

После Phase 04 у нас есть все 7 use case. Транспорт **импортирует только usecase + domain + chi + pkg/httpx + auth/transport/http/middleware** (для `RequireAuth` и `UserIDFromContext`). Никакого repo, никакого channel.

Эталон стиля — `internal/auth/transport/http/` (см. `../research.md` §«Transport HTTP»). API-контракт — `../08-api-contract.md`. Sequences — `../02-behavior.md` UC-R1..UC-R7.

## Файлы для создания

### DTO

#### `internal/room/transport/http/dto.go`

Все типы приватные, с camelCase json-тегами, по образцу `internal/auth/transport/http/dto.go:9-51`:

```go
package httproom

import (
    "encoding/json"
    "io"
    "net/http"
    "time"
)

type createRoomRequest struct {
    Name string `json:"name"`
}

type roomResponse struct {
    ID        string    `json:"id"`
    OwnerID   string    `json:"ownerId"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"createdAt"`
}

type roomWithRoleResponse struct {
    ID        string    `json:"id"`
    OwnerID   string    `json:"ownerId"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"createdAt"`
    Role      string    `json:"role"`
}

type listRoomsResponse struct {
    Items []roomWithRoleResponse `json:"items"`
}

type memberResponse struct {
    UserID   string    `json:"userId"`
    Role     string    `json:"role"`
    JoinedAt time.Time `json:"joinedAt"`
}

type listMembersResponse struct {
    Items []memberResponse `json:"items"`
}

type inviteResponse struct {
    Code      string    `json:"code"`
    CreatedBy string    `json:"createdBy"`
    CreatedAt time.Time `json:"createdAt"`
}

func jsonDecode(r io.Reader, v any) error { return json.NewDecoder(r).Decode(v) }

func jsonEncode(w http.ResponseWriter, v any) error { return json.NewEncoder(w).Encode(v) }
```

Точные поля — см. `../08-api-contract.md` для каждого эндпоинта.

### Error mapper

#### `internal/room/transport/http/error_mapper.go`

По образцу `internal/auth/transport/http/error_mapper.go:17-54`:

```go
package httproom

import (
    "errors"
    "net/http"

    "github.com/dovgalb/project-rupor/internal/room/domain"
    authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
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
    case errors.Is(err, domain.ErrInvalidRoomName):
        return httpError{http.StatusBadRequest, "ROOM-001", "invalid room name"}
    case errors.Is(err, domain.ErrInvalidInviteCode):
        return httpError{http.StatusBadRequest, "ROOM-008", "invalid invite code"}

    // 404
    case errors.Is(err, domain.ErrRoomNotFound):
        return httpError{http.StatusNotFound, "ROOM-002", "room not found"}
    case errors.Is(err, domain.ErrInviteNotFound):
        return httpError{http.StatusNotFound, "ROOM-007", "invite not found or revoked"}

    // 403
    case errors.Is(err, domain.ErrNotMember):
        return httpError{http.StatusForbidden, "ROOM-003", "not a member"}
    case errors.Is(err, domain.ErrInsufficientRole):
        // ROOM-005 для DeleteRoom (owner only) и ROOM-004 для admin/owner операций
        // Поскольку err sentinel один, выбор кода делаем по контексту хендлера —
        // реализуется через wrapper-функции mapDeleteRoomError / mapInviteError ниже.
        return httpError{http.StatusForbidden, "ROOM-004", "insufficient role: admin or owner required"}

    // 409
    case errors.Is(err, domain.ErrAlreadyMember):
        return httpError{http.StatusConflict, "ROOM-006", "already a member"}

    // 401 — auth-domain (доступ без валидного токена защитная ветка после RequireAuth)
    case errors.Is(err, authdom.ErrAccessTokenInvalid):
        return httpError{http.StatusUnauthorized, "AUTH-010", "access token invalid"}

    default:
        return httpError{http.StatusInternalServerError, "INTERNAL", "internal"}
    }
}

// mapDeleteRoomError — обёртка для UC-R4: ErrInsufficientRole здесь означает
// "требуется owner". Возвращает ROOM-005 вместо ROOM-004.
func mapDeleteRoomError(err error) httpError {
    if errors.Is(err, domain.ErrInsufficientRole) {
        return httpError{http.StatusForbidden, "ROOM-005", "only owner can delete room"}
    }
    return mapError(err)
}

func writeError(w http.ResponseWriter, e httpError) {
    httpx.WriteJSONError(w, e.status, e.code, e.msg)
}

func writeBadBody(w http.ResponseWriter) {
    writeError(w, httpError{http.StatusBadRequest, "ROOM-009", "invalid request body"})
}
```

**Внимание о двух mapper-функциях**: в дизайне `ErrInsufficientRole` — единственный sentinel, а коды `ROOM-004` и `ROOM-005` различаются по контексту операции. В `DeleteRoomHandler` используется `mapDeleteRoomError`, в остальных — `mapError`. Альтернатива (два sentinel'а в domain) обсуждалась и отвергнута — код роли в HTTP-таблице, а не в домене.

Импорт `authdom` нужен только для error code AUTH-010 при защитной ветке `!ok` в `UserIDFromContext`. Архитектурно это допустимо: транспорт может импортировать чужие domain'ы для error mapping.

### Routes

#### `internal/room/transport/http/routes.go`

По образцу `internal/auth/transport/http/routes.go:10-31`:

```go
package httproom

import (
    "github.com/go-chi/chi/v5"

    authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
    "github.com/dovgalb/project-rupor/internal/room/usecase"
    authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
)

type Deps struct {
    CreateRoom       *usecase.CreateRoom
    GetRoom          *usecase.GetRoom
    ListUserRooms    *usecase.ListUserRooms
    DeleteRoom       *usecase.DeleteRoom
    ListMembers      *usecase.ListMembers
    RegenerateInvite *usecase.RegenerateInvite
    JoinByCode       *usecase.JoinByCode
    TokenIssuer      authuc.TokenIssuer
    Clock            authuc.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
    r.Route("/rooms", func(r chi.Router) {
        r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock))

        r.Post("/", NewCreateRoomHandler(deps.CreateRoom).ServeHTTP)
        r.Get("/", NewListRoomsHandler(deps.ListUserRooms).ServeHTTP)
        r.Get("/{roomID}", NewGetRoomHandler(deps.GetRoom).ServeHTTP)
        r.Delete("/{roomID}", NewDeleteRoomHandler(deps.DeleteRoom).ServeHTTP)
        r.Get("/{roomID}/members", NewListMembersHandler(deps.ListMembers).ServeHTTP)
        r.Post("/{roomID}/invite", NewRegenerateInviteHandler(deps.RegenerateInvite).ServeHTTP)
        r.Post("/join/{code}", NewJoinByCodeHandler(deps.JoinByCode).ServeHTTP)
    })
}
```

**Внимание о `Clock` и `TokenIssuer`**: они импортируются из `internal/auth/usecase`, потому что `RequireAuth` (фаза 1.4) типизирован этими интерфейсами. Это та же практика, что и санкционированный кросс-доменный middleware-импорт. Альтернатива — продублировать интерфейсы Clock/TokenIssuer в каждом транспорте — переусложнение.

### Handlers

Каждый handler — отдельный файл. Шаблон (по `internal/auth/transport/http/me_handler.go:11-41`):

```go
type CreateRoomHandler struct{ uc *usecase.CreateRoom }

func NewCreateRoomHandler(uc *usecase.CreateRoom) *CreateRoomHandler { ... }

func (h *CreateRoomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    uid, ok := authmw.UserIDFromContext(r.Context())
    if !ok {
        writeError(w, mapError(authdom.ErrAccessTokenInvalid))
        return
    }

    var req createRoomRequest
    if err := jsonDecode(r.Body, &req); err != nil {
        writeBadBody(w)
        return
    }

    out, err := h.uc.Execute(r.Context(), usecase.CreateRoomInput{
        ActorID: uid.UUID(),
        Name:    req.Name,
    })
    if err != nil {
        writeError(w, mapError(err))
        return
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusCreated)
    _ = jsonEncode(w, roomToResponse(out.Room))
}
```

Список файлов:

#### `internal/room/transport/http/create_room_handler.go` (POST /rooms → 201)
#### `internal/room/transport/http/list_rooms_handler.go` (GET /rooms → 200)
- Не требует body.
- Маппит `[]usecase.RoomWithRole` в `listRoomsResponse{Items: ...}` — каждый `roomWithRoleResponse` включает `Role` (через `m.Role.String()`).

#### `internal/room/transport/http/get_room_handler.go` (GET /rooms/{roomID} → 200)
- `roomIDStr := chi.URLParam(r, "roomID")`.
- Парсить через `uuid.Parse` — на ошибке `mapError(domain.ErrRoomNotFound)` → 404.

#### `internal/room/transport/http/delete_room_handler.go` (DELETE /rooms/{roomID} → 204)
- Использует `mapDeleteRoomError` вместо `mapError`.
- На успехе — `w.WriteHeader(http.StatusNoContent)` (без тела).

#### `internal/room/transport/http/list_members_handler.go` (GET /rooms/{roomID}/members → 200)
- Маппит `[]*domain.Membership` в `[]memberResponse`.

#### `internal/room/transport/http/regenerate_invite_handler.go` (POST /rooms/{roomID}/invite → 200)
- Body игнорируется.
- На успехе — `inviteResponse{Code, CreatedBy, CreatedAt}`.

#### `internal/room/transport/http/join_by_code_handler.go` (POST /rooms/join/{code} → 200)
- `code := chi.URLParam(r, "code")`.
- Body игнорируется.
- На успехе — `roomResponse`.

### Helpers (mapping domain → response)

#### `internal/room/transport/http/dto.go` (дополнить)

Добавить функции:

```go
func roomToResponse(r *domain.Room) roomResponse {
    return roomResponse{
        ID:        r.ID().String(),
        OwnerID:   r.OwnerID().String(),
        Name:      r.Name().String(),
        CreatedAt: r.CreatedAt(),
    }
}

func roomWithRoleToResponse(rwr usecase.RoomWithRole) roomWithRoleResponse {
    return roomWithRoleResponse{
        ID:        rwr.Room.ID().String(),
        OwnerID:   rwr.Room.OwnerID().String(),
        Name:      rwr.Room.Name().String(),
        CreatedAt: rwr.Room.CreatedAt(),
        Role:      rwr.Role.String(),
    }
}

func membershipToResponse(m *domain.Membership) memberResponse {
    return memberResponse{
        UserID:   m.UserID().String(),
        Role:     m.Role().String(),
        JoinedAt: m.JoinedAt(),
    }
}

func inviteToResponse(i *domain.Invite) inviteResponse {
    return inviteResponse{
        Code:      i.Code().String(),
        CreatedBy: i.CreatedBy().String(),
        CreatedAt: i.CreatedAt(),
    }
}
```

### HTTP Tests

#### `internal/room/transport/http/setup_test.go`

По образцу `internal/auth/transport/http/setup_test.go:163-216`. Создаёт реальный chi-router с `RequireAuth`, реальные usecase (из Phase 04), фейковые репозитории (взять реализацию из `internal/room/usecase/fakes_test.go`, **скопировать** в этот пакет — не делиться через `internal_test`).

Важно для тестов: нужен реальный `TokenIssuer` (из `internal/auth/repository/jwt`) с тестовым секретом. Хелпер `issueTestToken(t, userID)` для генерации Bearer-токенов. По образцу auth setup_test.

#### Тесты по handler'у
Каждый файл `<handler>_test.go` покрывает: 200/2xx happy-path + основные 4xx из coverage mapping (`../04-testing.md`):

- `internal/room/transport/http/create_room_handler_test.go`:
  - `TestCreateRoom_BadBody_Returns400` (ROOM-009)
  - `TestCreateRoom_Valid_Returns201`
  - `TestCreateRoom_NoToken_Returns401`
  - `TestCreateRoom_InvalidName_Returns400` (ROOM-001 — приходит из usecase через domain)
- `internal/room/transport/http/get_room_handler_test.go`:
  - `TestGetRoom_InvalidUUID_Returns404`
  - `TestGetRoom_AsMember_Returns200`
  - `TestGetRoom_NotMember_Returns403`
- `internal/room/transport/http/list_rooms_handler_test.go`:
  - `TestListRooms_HasItems_Returns200`
  - `TestListRooms_NoItems_ReturnsEmptyArray`
- `internal/room/transport/http/delete_room_handler_test.go`:
  - `TestDeleteRoom_NotFound_Returns404`
  - `TestDeleteRoom_AsOwner_Returns204`
  - `TestDeleteRoom_AsAdmin_Returns403_ROOM005`
  - `TestDeleteRoom_NotMember_Returns403_ROOM003`
- `internal/room/transport/http/list_members_handler_test.go`:
  - `TestListMembers_InvalidUUID_Returns404`
  - `TestListMembers_AsMember_Returns200`
- `internal/room/transport/http/regenerate_invite_handler_test.go`:
  - `TestRegenerateInvite_InvalidUUID_Returns404`
  - `TestRegenerateInvite_AsMember_Returns403`
  - `TestRegenerateInvite_AsAdmin_Returns200`
- `internal/room/transport/http/join_by_code_handler_test.go`:
  - `TestJoinByCode_InvalidFormat_Returns400`
  - `TestJoinByCode_NoActiveInvite_Returns404`
  - `TestJoinByCode_AlreadyMember_Returns409`
  - `TestJoinByCode_NewMember_Returns200`

## Файлы для модификации

- `internal/room/transport/http/.gitkeep` — удалить.

## Ключевые решения

- **Один файл — один handler** — повторяем стиль auth.
- **Все DTO приватные** — экспортируется только `RegisterRoutes` и `Deps`. Это уменьшает поверхность пакета.
- **`mapDeleteRoomError` — отдельный wrapper** — `ErrInsufficientRole` маппится в разные коды (ROOM-004 vs ROOM-005) в зависимости от контекста use case. Альтернатива (два domain sentinel) хуже, потому что требует знание HTTP-семантики в domain.
- **`Clock` и `TokenIssuer` импортируются из `internal/auth/usecase`** — для типизации `RequireAuth`. Если в будущем room/channel будут иметь свои Clock/Issuer — введём общий пакет; пока переиспользуем auth-овские.
- **HTTP-тесты используют реальный `TokenIssuer` (jwt-адаптер)** + реальные usecase — это полу-интеграционные тесты, проверяющие правильность роутинга, middleware, маппинга ошибок. Repository — фейковый, БД не нужна.
- **`uuid.Parse` ошибка → 404 ROOM-002**, а не 400. Семантически: ресурс не найден (по такому ID не может существовать). Совпадает с дизайном UC-R2.

## Verification

- [ ] `dto.go`, `error_mapper.go`, `routes.go` созданы.
- [ ] 7 handler-файлов созданы.
- [ ] `setup_test.go` создан (с фейковыми репо и реальным TokenIssuer).
- [ ] 7 `<handler>_test.go` файлов созданы; покрывают coverage mapping из `../04-testing.md`.
- [ ] `go build ./internal/room/transport/http/...` чистый.
- [ ] `go test ./internal/room/transport/http/... -race -count=1` зелёный.
- [ ] Импорты non-test файлов: stdlib + `chi` + `pkg/httpx` + свой usecase/domain + `internal/auth/transport/http/middleware` + `internal/auth/domain` (для AUTH-010 mapping) + `internal/auth/usecase` (для типов в Deps). **Нет** `internal/room/repository`, `internal/channel/...`.
- [ ] `golangci-lint run ./internal/room/transport/http/...` чистый.
- [ ] `internal/room/transport/http/.gitkeep` удалён.
- [ ] Ручная сверка через `manual_qa/2_1_rooms_and_channels/01_create_room.http` (после фазы 10) подтверждает форматы JSON.
