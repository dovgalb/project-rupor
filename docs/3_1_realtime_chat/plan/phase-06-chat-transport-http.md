---
phase: 6
name: chat-transport-http
layer: transport
depends_on: [phase-05]
plan: ./README.md
---

# Phase 6: HTTP-транспорт `internal/chat/transport/http/`

## Цель

Создать REST-эндпоинт `GET /api/v1/channels/{channelID}/messages?before=&limit=` с курсорной пагинацией, защищённый `RequireAuth`. После этой фазы chat доступен через HTTP.

## Контекст

Phase-05 создала use case `ListMessages`. Шаблон HTTP-транспорта — `internal/channel/transport/http/` (5 файлов: routes, handler, dto, error_mapper, setup_test). См. `internal/channel/transport/http/list_channels_handler.go` как handler-шаблон.

API-контракт — [`../08-api-contract.md §REST`](../08-api-contract.md). Маппинг ошибок — [`../08-api-contract.md §Error responses`](../08-api-contract.md).

## Файлы для создания

### `internal/chat/transport/http/routes.go`

```go
package httpchat

import (
    "github.com/go-chi/chi/v5"

    authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
    authuc "github.com/dovgalb/project-rupor/internal/auth/usecase"
    "github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type Deps struct {
    ListMessages *usecase.ListMessages
    TokenIssuer  authuc.TokenIssuer
    Clock        authuc.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
    r.Route("/channels/{channelID}/messages", func(r chi.Router) {
        r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock))
        r.Get("/", NewListMessagesHandler(deps.ListMessages).ServeHTTP)
    })
}
```

**Замечание:** path — `/channels/{channelID}/messages`. При монтировании в `cmd/server/main.go` под `/api/v1` получится `GET /api/v1/channels/{channelID}/messages`. Совпадает с [API-контрактом](../08-api-contract.md).

### `internal/chat/transport/http/dto.go`

```go
package httpchat

import (
    "encoding/json"
    "io"
    "time"

    "github.com/dovgalb/project-rupor/internal/chat/domain"
)

type messageResponse struct {
    ID        string    `json:"id"`
    ChannelID string    `json:"channelId"`
    AuthorID  string    `json:"authorId"`
    Text      string    `json:"text"`
    CreatedAt time.Time `json:"createdAt"`
}

type listMessagesResponse struct {
    Items      []messageResponse `json:"items"`
    NextBefore *string           `json:"nextBefore"` // null если страница последняя
}

func messageToResponse(m *domain.Message) messageResponse {
    return messageResponse{
        ID:        m.ID().String(),
        ChannelID: m.ChannelID().String(),
        AuthorID:  m.AuthorID().String(),
        Text:      m.Text().String(),
        CreatedAt: m.CreatedAt(),
    }
}

func jsonEncode(w io.Writer, v any) error {
    return json.NewEncoder(w).Encode(v)
}
```

**Замечание:** `NextBefore *string` — указатель, чтобы при `null` JSON-сериализатор писал `"nextBefore": null`. Альтернатива (`omitempty`) ломает контракт.

### `internal/chat/transport/http/error_mapper.go`

```go
package httpchat

import (
    "errors"
    "net/http"

    authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
    "github.com/dovgalb/project-rupor/internal/chat/domain"
    "github.com/dovgalb/project-rupor/pkg/httpx"
)

type httpError struct {
    status int
    code   string
    msg    string
}

func mapError(err error) httpError {
    switch {
    case errors.Is(err, domain.ErrInvalidMessageText):
        return httpError{http.StatusBadRequest, "CHAT-001", "invalid message text"}
    case errors.Is(err, domain.ErrChannelNotFound):
        return httpError{http.StatusNotFound, "CHAT-002", "channel not found"}
    case errors.Is(err, domain.ErrChannelNotText):
        return httpError{http.StatusBadRequest, "CHAT-003", "channel is not text"}
    case errors.Is(err, domain.ErrChatAccessDenied):
        return httpError{http.StatusForbidden, "CHAT-004", "access denied: not a room member"}
    case errors.Is(err, domain.ErrInvalidMessageID),
         errors.Is(err, domain.ErrInvalidChannelID),
         errors.Is(err, domain.ErrInvalidAuthorID):
        return httpError{http.StatusBadRequest, "CHAT-005", "invalid uuid in path/query"}

    // 401 — защитная ветка при !ok в UserIDFromContext (после RequireAuth не должно).
    case errors.Is(err, authdom.ErrAccessTokenInvalid):
        return httpError{http.StatusUnauthorized, "AUTH-010", "access token invalid"}

    default:
        return httpError{http.StatusInternalServerError, "INTERNAL", "internal"}
    }
}

func writeError(w http.ResponseWriter, e httpError) {
    httpx.WriteJSONError(w, e.status, e.code, e.msg)
}

func writeLimitError(w http.ResponseWriter) {
    writeError(w, httpError{http.StatusBadRequest, "CHAT-006", "limit must be 1..100"})
}

func writeBadUUIDError(w http.ResponseWriter) {
    writeError(w, httpError{http.StatusBadRequest, "CHAT-005", "invalid uuid in path/query"})
}
```

### `internal/chat/transport/http/list_messages_handler.go`

```go
package httpchat

import (
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"

    authdom "github.com/dovgalb/project-rupor/internal/auth/domain"
    authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
    "github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type ListMessagesHandler struct {
    uc *usecase.ListMessages
}

func NewListMessagesHandler(uc *usecase.ListMessages) *ListMessagesHandler {
    return &ListMessagesHandler{uc: uc}
}

func (h *ListMessagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    uid, ok := authmw.UserIDFromContext(r.Context())
    if !ok {
        writeError(w, mapError(authdom.ErrAccessTokenInvalid))
        return
    }

    channelID, err := uuid.Parse(chi.URLParam(r, "channelID"))
    if err != nil {
        writeBadUUIDError(w)
        return
    }

    var beforeID uuid.UUID
    if raw := r.URL.Query().Get("before"); raw != "" {
        beforeID, err = uuid.Parse(raw)
        if err != nil {
            writeBadUUIDError(w)
            return
        }
    }

    limit := usecase.DefaultLimit
    if raw := r.URL.Query().Get("limit"); raw != "" {
        n, perr := strconv.Atoi(raw)
        if perr != nil || n < usecase.MinLimit || n > usecase.MaxLimit {
            writeLimitError(w)
            return
        }
        limit = n
    }

    out, err := h.uc.Execute(r.Context(), usecase.ListMessagesInput{
        ActorID:   uid.UUID(),
        ChannelID: channelID,
        Before:    beforeID,
        Limit:     limit,
    })
    if err != nil {
        writeError(w, mapError(err))
        return
    }

    items := make([]messageResponse, 0, len(out.Items))
    for _, m := range out.Items {
        items = append(items, messageToResponse(m))
    }
    var nextBeforeStr *string
    if out.NextBefore != uuid.Nil {
        s := out.NextBefore.String()
        nextBeforeStr = &s
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    _ = jsonEncode(w, listMessagesResponse{Items: items, NextBefore: nextBeforeStr})
}
```

### `internal/chat/transport/http/setup_test.go`

Шаблон — `internal/channel/transport/http/setup_test.go`. Поднимает `httptest.NewServer` с реальным chi-роутером, реальным `ListMessages` usecase, fake-репо (под `sync.Mutex`), реальным `jwtadapter.NewTokenIssuer`. Хелпер `doGet(env, path, token)` возвращает `*http.Response`.

### `internal/chat/transport/http/list_messages_handler_test.go`

10 тестов из [`../04-testing.md §ListMessagesHandler`](../04-testing.md). Каждый проверяет: HTTP-код, формат JSON-тела (точно из [API-контракта](../08-api-contract.md)), отсутствие side-effect'ов в fake-репо при ошибочных сценариях.

## Файлы для модификации

Нет в этой фазе. Регистрация роута в `cmd/server/main.go` — phase-09.

## Ключевые решения

- **Парсинг limit на handler-уровне** — раньше, чем в use case. Возвращает CHAT-006 (HTTP 400) для невалидного limit, не INTERNAL.
- **`NextBefore` как `*string`** — для корректного `null` в JSON. Контракт фиксирован в [`../08-api-contract.md`](../08-api-contract.md).
- **camelCase в REST JSON** — `channelId`, `authorId`, `createdAt`, `nextBefore`. Отличается от snake_case WS-payload'а. См. [`../07-standards.md §Уточнение 3`](../07-standards.md).
- **`error_mapper.mapError` — единственная точка маппинга** — handler не маппит руками, всё через `mapError(err)`.

## Verification

- [ ] `go build ./internal/chat/transport/http/...` без ошибок.
- [ ] `go test ./internal/chat/transport/http/...` зелёный.
- [ ] `go test -race ./internal/chat/transport/http/...` зелёный.
- [ ] Все 10 тестов из [04-testing.md](../04-testing.md) написаны и проходят.
- [ ] Импорты — только stdlib + uuid + chi + `internal/chat/{domain,usecase}` + `internal/auth/{domain,transport/http/middleware}` + `pkg/httpx`. (Arch-тест в phase-09.)
- [ ] JSON-формат ответа совпадает с [API-контракта](../08-api-contract.md) (тест `TestListMessages_AsMember_Returns200` явно сверяет ключи и значения).
- [ ] Все коды ошибок CHAT-001..006 и AUTH-010, AUTH-011 покрыты тест-кейсами.
