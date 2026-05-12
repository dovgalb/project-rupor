---
phase: 9
name: composition-root
layer: infrastructure
depends_on: [phase-01, phase-02, phase-03, phase-04, phase-05, phase-06, phase-07, phase-08]
plan: ./README.md
---

# Phase 9: Composition root + sqlc.yaml + arch_test + Logger sanitizer

## Цель

Финальная склейка: расширение `sqlc.yaml` для chat, обновление composition root `cmd/server/main.go` (новые репозитории, hub, use case'ы, регистрация роутов, graceful shutdown hub'а), маскирование `?token=` в access-логе, расширение `arch_test.go` правилами для `chat/*` и `pkg/websocket`. После этой фазы фича 3.1 полностью склеена и работает end-to-end.

## Контекст

Все предыдущие фазы реализованы. Phase-04 написала `queries/messages.sql`, но не подключила к sqlc-конфигу (мы делали временный локальный конфиг). Тесты `pkg/websocket`, `chat/*`, расширения `room` — изолированно зелёные. Cмонтировать в систему — задача этой фазы.

## Файлы для модификации

### `sqlc.yaml`

**Добавить 4-й `sql:` блок** для chat-репозитория. Точный текст — из [`../06-repo-model.md §sqlc.yaml — изменения`](../06-repo-model.md):

```yaml
  - engine: "postgresql"
    schema: "migrations"
    queries: "internal/chat/repository/postgres/queries"
    gen:
      go:
        sql_package: "pgx/v5"
        package: "db"
        out: "internal/chat/repository/postgres/db"
        emit_interface: false
        emit_json_tags: false
        emit_db_tags: false
        emit_pointers_for_null_types: false
        emit_prepared_queries: false
        overrides:
          - db_type: "uuid"
            nullable: false
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
          - db_type: "timestamptz"
            nullable: false
            go_type: "time.Time"
          - db_type: "pg_catalog.timestamptz"
            nullable: false
            go_type: "time.Time"
```

После — запустить `make sqlc`. Если уже сгенерировано в phase-04 (через временный конфиг) — пересгенерировать поверх.

### `cmd/server/main.go`

Изменения вокруг существующих секций. Точные диапазоны зависят от текущего состояния, ниже — описание изменений и их места.

**1. Добавить импорты:**

```go
chatpg "github.com/dovgalb/project-rupor/internal/chat/repository/postgres"
httpchat "github.com/dovgalb/project-rupor/internal/chat/transport/http"
wschat "github.com/dovgalb/project-rupor/internal/chat/transport/ws"
chatusecase "github.com/dovgalb/project-rupor/internal/chat/usecase"

pws "github.com/dovgalb/project-rupor/pkg/websocket"
```

**2. После существующей секции "=== Channel composition ===" (`main.go:124-130`)** — добавить новую секцию:

```go
// === WebSocket Hub ===
hub := pws.NewHub(logger)
defer hub.Shutdown(context.Background())  // или интегрировать в graceful shutdown ниже

// === Chat composition ===
messageRepo := chatpg.NewMessageRepository(pool)
membershipForChat := roompg.NewMembershipQueryChatAdapter(pool)
chatBroadcaster := newHubChatBroadcaster(hub)        // wrapper
roomEventsPublisher := newHubRoomEventsPublisher(hub) // wrapper

sendMessageUC := chatusecase.NewSendMessage(messageRepo, membershipForChat, chatBroadcaster, clock, uuids)
listMessagesUC := chatusecase.NewListMessages(messageRepo, membershipForChat)
```

**3. В секции инстанцирования `joinByCodeUC` (`main.go:122`) — добавить `roomEventsPublisher` последним параметром:**

```go
joinByCodeUC := roomusecase.NewJoinByCode(inviteRepo, membershipRepo, roomRepo, clock, roomEventsPublisher)
```

**4. В секции `mux.Route("/api/v1", func(r chi.Router) { ... })` (`main.go:148-176`) — добавить регистрацию роутов после `httpchannel.RegisterRoutes`:**

```go
httpchat.RegisterRoutes(r, httpchat.Deps{
    ListMessages: listMessagesUC,
    TokenIssuer:  issuer,
    Clock:        clock,
})
wschat.RegisterWSRoute(r, wschat.WSDeps{
    Hub:                hub,
    SendMessage:        sendMessageUC,
    MembershipForChat:  membershipForChat,
    MembershipForRooms: roomIDsAdapter{repo: membershipRepo},  // thin adapter
    TokenIssuer:        issuer,
    Clock:              clock,
    OriginPatterns:     stripScheme(cfg.CORSAllowedOrigins()),
})
```

**5. Расширить graceful shutdown:**

Заменить:
```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
defer cancel()
if err := srv.Shutdown(shutdownCtx); err != nil {
    return err
}
```

На:
```go
shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
defer cancel()

hub.Shutdown(shutdownCtx)  // закрыть все WS перед остановкой HTTP-сервера

if err := srv.Shutdown(shutdownCtx); err != nil {
    return err
}
```

И удалить `defer hub.Shutdown(...)` из шага 2 (двойной вызов идемпотентен через `sync.Once`, но лишний).

**6. Добавить тонкие адаптеры в новый файл `cmd/server/ws_adapters.go`** (отдельный файл, чтобы не раздувать main.go):

```go
package main

import (
    "time"

    "github.com/google/uuid"

    chatdom "github.com/dovgalb/project-rupor/internal/chat/domain"
    roomdom "github.com/dovgalb/project-rupor/internal/room/domain"
    roompg "github.com/dovgalb/project-rupor/internal/room/repository/postgres"
    pws "github.com/dovgalb/project-rupor/pkg/websocket"
)

// hubChatBroadcaster адаптирует *pws.Hub под chat/usecase.Broadcaster.
type hubChatBroadcaster struct{ hub *pws.Hub }

func newHubChatBroadcaster(hub *pws.Hub) *hubChatBroadcaster {
    return &hubChatBroadcaster{hub: hub}
}

func (b *hubChatBroadcaster) PublishToChannel(channelID chatdom.ChannelID, eventType string, payload any) {
    b.hub.Publish(context.Background(), pws.ChannelTopic(channelID.UUID()), eventType, payload)
}

func (b *hubChatBroadcaster) PublishToRoom(roomID chatdom.RoomID, eventType string, payload any) {
    b.hub.Publish(context.Background(), pws.RoomTopic(roomID.UUID()), eventType, payload)
}

// hubRoomEventsPublisher адаптирует *pws.Hub под room/usecase.RoomEventsPublisher.
type hubRoomEventsPublisher struct{ hub *pws.Hub }

func newHubRoomEventsPublisher(hub *pws.Hub) *hubRoomEventsPublisher {
    return &hubRoomEventsPublisher{hub: hub}
}

func (p *hubRoomEventsPublisher) PublishMemberJoined(roomID roomdom.RoomID, userID roomdom.UserID, joinedAt time.Time) {
    payload := map[string]any{
        "room_id":   roomID.String(),
        "user_id":   userID.String(),
        "joined_at": joinedAt.Format(time.RFC3339Nano),
    }
    p.hub.Publish(context.Background(), pws.RoomTopic(roomID.UUID()), "member.joined", payload)
}

// roomIDsAdapter адаптирует *roompg.MembershipRepository под wschat.MembershipReader.
type roomIDsAdapter struct{ repo *roompg.MembershipRepository }

func (a roomIDsAdapter) ListRoomIDsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
    uid, err := roomdom.NewUserID(userID)
    if err != nil {
        return nil, err
    }
    ms, err := a.repo.ListByUser(ctx, uid)
    if err != nil {
        return nil, err
    }
    out := make([]uuid.UUID, 0, len(ms))
    for _, m := range ms {
        out = append(out, m.RoomID().UUID())
    }
    return out, nil
}

// stripScheme приводит CORS origin'ы к формату OriginPatterns nhooyr (host без схемы).
func stripScheme(origins []string) []string {
    out := make([]string, 0, len(origins))
    for _, o := range origins {
        u, err := url.Parse(o)
        if err != nil || u.Host == "" {
            continue
        }
        out = append(out, u.Host)
    }
    return out
}
```

⚠️ **Проверка:** `roompg.MembershipRepository.ListByUser` — существует ли такой метод? Если нет, нужно добавить в phase-04 (room-extensions). Проверить перед стартом фазы 9. Если нет — расширить phase-04 нужным методом.

### `pkg/httpx/middleware/logger.go`

**Назначение изменения:** маскировать `?token=...` в логируемом URL для WS-пути. Соответствует [`../07-standards.md §Уточнение 4`](../07-standards.md).

**Подход:** добавить функцию-санитайзер URL, применяемую перед логированием:

```go
// sanitizeURL маскирует значение query-параметра "token" в логах.
// Применяется к URL целиком, не зависит от path.
func sanitizeURL(u *url.URL) string {
    if u == nil { return "" }
    q := u.Query()
    if q.Get("token") != "" {
        q.Set("token", "REDACTED")
    }
    redacted := *u
    redacted.RawQuery = q.Encode()
    return redacted.RequestURI()
}
```

В существующей функции логирования (`Logger(...)`) — заменить `r.URL.RequestURI()` (или аналогичный вызов) на `sanitizeURL(r.URL)`.

Применять ко ВСЕМ путям (не только `/api/v1/ws`), чтобы быть консервативным. Стоимость — две операции с query на каждый лог, пренебрежимо.

### `pkg/httpx/middleware/logger_test.go`

**Добавить тест:**

```go
func TestLogger_StripsTokenFromAccessLog(t *testing.T) {
    t.Parallel()

    var buf bytes.Buffer
    logger := slog.New(slog.NewJSONHandler(&buf, nil))
    mw := middleware.Logger(logger, nil)

    h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))

    req := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token=secret123&channel_id=abc", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)

    if strings.Contains(buf.String(), "secret123") {
        t.Fatalf("token leaked into log: %s", buf.String())
    }
    if !strings.Contains(buf.String(), "REDACTED") {
        t.Fatalf("expected REDACTED marker in log: %s", buf.String())
    }
    if !strings.Contains(buf.String(), "channel_id=abc") {
        t.Fatalf("other params should remain: %s", buf.String())
    }
}
```

### `arch_test.go`

**Добавить тесты** для chat-пакетов и pkg/websocket. По шаблону существующих:

```go
// chat/domain — только stdlib + uuid.
func TestArchitecture_ChatDomainImports(t *testing.T) {
    t.Parallel()
    allowed := []string{"github.com/google/uuid"}
    forbidden := []string{
        "github.com/dovgalb/project-rupor/internal",
        "github.com/dovgalb/project-rupor/pkg",
        "nhooyr.io/websocket",
        "github.com/jackc/pgx",
    }
    assertImports(t, "internal/chat/domain", allowed, forbidden)
}

// chat/usecase — только stdlib + uuid + chat/domain.
func TestArchitecture_ChatUseCaseImports(t *testing.T) {
    t.Parallel()
    allowed := []string{
        "github.com/google/uuid",
        "github.com/dovgalb/project-rupor/internal/chat/domain",
    }
    forbidden := []string{
        "github.com/dovgalb/project-rupor/internal/channel",
        "github.com/dovgalb/project-rupor/internal/room",
        "github.com/dovgalb/project-rupor/internal/auth",
        "github.com/dovgalb/project-rupor/pkg/websocket",
    }
    assertImports(t, "internal/chat/usecase", allowed, forbidden)
}

// chat/repository/postgres — изолирован.
func TestArchitecture_ChatRepoIsolated(t *testing.T) {
    t.Parallel()
    allowed := []string{ /* pgx, uuid, chat/{domain,usecase} */ }
    forbidden := []string{
        "github.com/dovgalb/project-rupor/internal/auth",
        "github.com/dovgalb/project-rupor/internal/room",
        "github.com/dovgalb/project-rupor/internal/channel",
        "github.com/dovgalb/project-rupor/internal/chat/transport",
    }
    assertImports(t, "internal/chat/repository/postgres", allowed, forbidden)
}

// chat/transport/http — может импортировать auth.transport.middleware.
func TestArchitecture_ChatTransportHTTPImports(t *testing.T) {
    t.Parallel()
    allowed := []string{
        "github.com/google/uuid",
        "github.com/go-chi/chi",
        "github.com/dovgalb/project-rupor/internal/chat/domain",
        "github.com/dovgalb/project-rupor/internal/chat/usecase",
        "github.com/dovgalb/project-rupor/internal/auth/domain",
        "github.com/dovgalb/project-rupor/internal/auth/usecase",
        "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware",
        "github.com/dovgalb/project-rupor/pkg/httpx",
    }
    forbidden := []string{
        "github.com/dovgalb/project-rupor/internal/room",
        "github.com/dovgalb/project-rupor/internal/channel",
    }
    assertImports(t, "internal/chat/transport/http", allowed, forbidden)
}

// chat/transport/ws — может импортировать pkg/websocket.
func TestArchitecture_ChatTransportWSImports(t *testing.T) {
    t.Parallel()
    allowed := []string{
        "github.com/google/uuid",
        "github.com/go-chi/chi",
        "github.com/dovgalb/project-rupor/internal/chat/domain",
        "github.com/dovgalb/project-rupor/internal/chat/usecase",
        "github.com/dovgalb/project-rupor/internal/auth/domain",
        "github.com/dovgalb/project-rupor/internal/auth/usecase",
        "github.com/dovgalb/project-rupor/pkg/httpx",
        "github.com/dovgalb/project-rupor/pkg/websocket",
        "nhooyr.io/websocket",  // допустимо для close-кодов
    }
    forbidden := []string{
        "github.com/dovgalb/project-rupor/internal/room",
        "github.com/dovgalb/project-rupor/internal/channel",
    }
    assertImports(t, "internal/chat/transport/ws", allowed, forbidden)
}

// pkg/websocket — никаких internal/*.
func TestArchitecture_PkgWebsocketIsolated(t *testing.T) {
    t.Parallel()
    allowed := []string{
        "github.com/google/uuid",
        "nhooyr.io/websocket",
    }
    forbidden := []string{
        "github.com/dovgalb/project-rupor/internal",
        "github.com/dovgalb/project-rupor/pkg/httpx",
    }
    assertImports(t, "pkg/websocket", allowed, forbidden)
}

// room/repository/postgres — расширение: разрешён chat/{domain,usecase}.
// (Существующий TestArchitecture_RoomRepoMayImplementChannelPort расширить.)
```

⚠️ **Внимание:** существующий тест `TestArchitecture_RoomRepoMayImplementChannelPort` (`arch_test.go:259-280`) разрешает импорт `channel/usecase` и `channel/domain`. Расширить его, чтобы разрешить также `chat/usecase` и `chat/domain` — НЕ создавать отдельный тест, а добавить в `allowed` список существующего теста.

### `general_plan.md`

Отметить все пункты фазы 3 как `[x]` после успешной реализации (это финальный шаг — после verification).

### `manual_qa/3_1_realtime_chat/`

**Создать новую директорию** с `.http`-файлами для ручного QA. Шаблон — `manual_qa/2_1_rooms_and_channels/00_flow.http`. Содержание:

- `00_flow.http` — основной happy-path scenario: register A → register B → A creates room → A creates text channel → A generates invite → B joins → A sends message via REST? Нет, message только через WS. Поэтому добавить отдельные:
- `01_messages_rest.http` — `GET /channels/{id}/messages?before=&limit=`
- `02_ws_browser_test.md` — инструкция: «открыть browser console, выполнить `new WebSocket('ws://localhost:8080/api/v1/ws?token=...')`, отправить фреймы, наблюдать события».

Точное содержимое — на усмотрение implementer'а. Главное — покрыть happy-path для приёмочного тестирования.

## Файлы для создания

- `cmd/server/ws_adapters.go` (см. выше)
- `manual_qa/3_1_realtime_chat/00_flow.http`
- `manual_qa/3_1_realtime_chat/01_messages_rest.http`
- `manual_qa/3_1_realtime_chat/02_ws_browser_test.md`

## Ключевые решения

- **Adapters в `cmd/server/`, не в `pkg/websocket`** — потому что они знают про доменные типы (`chatdom.ChannelID`, `roomdom.RoomID`), а `pkg/websocket` агностичен ([§ D-02](../03-decisions.md)).
- **`stripScheme` — простая локальная утилита** — не нуждается в выносе в `config/`. Альтернатива (новое поле `cfg.WSOriginPatterns()`) — оверкилл.
- **`hub.Shutdown` вызывается ДО `srv.Shutdown`** — чтобы клиенты получили close-frame 1001 до того, как HTTP-сервер прекратит отвечать (см. [§ S-1 в 02-behavior.md](../02-behavior.md)).
- **Маскирование `token` в logger — applies ко всем путям** — не только WS, для консервативности.
- **arch_test.go расширяется минимально** — один новый тест на каждое новое правило + одно расширение существующего (room/repository).
- **`general_plan.md` обновляется** — отметка `[x]` фиксирует завершение фазы.

## Verification

- [ ] `make sqlc` без ошибок.
- [ ] `go build ./...` чистый.
- [ ] `make lint` чистый.
- [ ] `go test ./...` зелёный (все unit + integration с `TEST_DATABASE_URL`).
- [ ] `go test -race ./...` зелёный.
- [ ] `make migrate-up` применяет миграцию `0008_messages`; `make migrate-down` откатывает.
- [ ] `make run` запускает сервер; в логе видно `server starting addr=:8080`.
- [ ] `curl http://localhost:8080/api/v1/health` — 200 OK.
- [ ] Ручной QA по `manual_qa/3_1_realtime_chat/00_flow.http` — happy path работает: A регистрируется, создаёт комнату, канал, B джоинится, A видит `member.joined` через WS (если был подключен).
- [ ] Ручной QA WS через browser-console: connect, subscribe, message.send → клиент получает message.new и message.sent.
- [ ] WS-токен НЕ виден в access-логе (визуально проверить `make run` → подключиться WS → grep лога).
- [ ] `general_plan.md` обновлён.
- [ ] Все 10 критериев приёмки из [`../README.md §Критерии приёмки`](../README.md) выполнены.
- [ ] `arch_test.go` тесты для chat/* и pkg/websocket — зелёные.
- [ ] Существующий arch_test `TestArchitecture_RoomRepoMayImplementChannelPort` обновлён и разрешает chat-импорты.

## Post-completion

- Закоммитить с conventional commit message: `feat(chat): добавить домен chat и реалтайм-чат (PR-3.1)` (по образцу commit'а `a921a49 PR-2: feat(rooms, channels): добавить домены room и channel со всеми слоями`).
- Открыть PR в `main` через `gh pr create`.
- Дождаться CI зелёным.
