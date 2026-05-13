---
parent: ./README.md
view: process
---

# 02 — Behavior (Process View)

DFD и sequence diagrams по основным сценариям. Один sequence — на use case (включая happy path и сводный блок error/edge cases).

## Data Flow Diagrams

### DFD-1: Отправка и доставка сообщения

```mermaid
flowchart LR
    Client["WS Client A"] -->|"message.send<br/>(channel_id, text)"| WsH["WSHandler"]
    WsH -->|"input"| SendUC["chat.SendMessage"]
    SendUC -->|"Require"| MQ["MembershipQuery"]
    MQ -->|"GetMemberForChannel"| Pool[(Postgres)]
    SendUC -->|"ChannelOf"| MR["MessageRepository"]
    MR -->|"GetChannelKind"| Pool
    SendUC -->|"domain.NewMessage"| Msg["Message entity"]
    SendUC -->|Save| MR
    MR -->|InsertMessage| Pool
    SendUC -->|"PublishToChannel<br/>(message.new)"| Hub["pkg/websocket.Hub"]
    Hub -->|"broadcast JSON"| ClientA["Subs in channel:id"]
    Hub -->|"broadcast JSON"| ClientB["..."]
```

### DFD-2: Чтение истории канала (REST курсор)

```mermaid
flowchart LR
    Client["HTTP Client"] -->|"GET /channels/:id/messages?before=&limit="| Handler["ListMessagesHandler"]
    Handler -->|"parse + UserIDFromContext"| ListUC["chat.ListMessages"]
    ListUC -->|"Require RoleAnyMember"| MQ["MembershipQuery"]
    MQ -->|"GetRoomMember"| Pool[(Postgres)]
    ListUC -->|"ListByChannel"| MR["MessageRepository"]
    MR -->|"ListMessagesBeforeCursor"| Pool
    MR -->|"rows → domain"| ListUC
    ListUC -->|"items"| Handler
    Handler -->|"JSON"| Client
```

### DFD-3: Вступление в комнату + member.joined

```mermaid
flowchart LR
    Client["HTTP Client"] -->|"POST /rooms/join/:code"| JoinH["JoinByCodeHandler"]
    JoinH -->|"input"| JoinUC["room.JoinByCode"]
    JoinUC -->|"inviteRepo.Find"| Pool[(Postgres)]
    JoinUC -->|"membershipRepo.Add"| Pool
    JoinUC -->|"PublishMemberJoined<br/>(room_id, user_id)"| Hub["pkg/websocket.Hub"]
    Hub -->|"broadcast JSON"| OtherSubs["Subs in room:id"]
    JoinH -->|"JSON 200"| Client
```

### DFD-4: Подключение к WS и подписка

```mermaid
flowchart LR
    Client["WS Client"] -->|"GET /api/v1/ws?token=jwt<br/>Upgrade: websocket"| WsH["WSHandler"]
    WsH -->|"VerifyAccess"| Issuer["TokenIssuer"]
    WsH -->|"membership.ListUserRooms"| MR["MembershipRepository"]
    MR --> Pool[(Postgres)]
    WsH -->|"Accept"| Up["pkg/websocket.Upgrade"]
    WsH -->|"Register"| Hub
    WsH -->|"Subscribe room:id ×N"| Hub
    Client -->|"subscribe<br/>(channel_id)"| WsH
    WsH -->|"Require RoleAnyMember"| MQ["MembershipQuery"]
    WsH -->|"Subscribe channel:id"| Hub
```

---

## Sequence Diagrams

### UC-1: Отправка сообщения (`chat.SendMessage`)

**Триггер:** WS-клиент шлёт `{"type":"message.send","channel_id":"<uuid>","text":"<string>"}`.

```mermaid
sequenceDiagram
    actor Client
    participant WsH as WSHandler
    participant Send as SendMessage UC
    participant MQ as MembershipQuery
    participant Repo as MessageRepository
    participant Hub as websocket.Hub
    participant Subs as Other Subs

    Client->>WsH: WS frame {type:message.send, channel_id, text}
    WsH->>Send: Execute(actorID, channelID, text)
    Send->>MQ: Require(channelID, actorID, RoleAnyMember)
    MQ-->>Send: nil
    Send->>Repo: ChannelOf(channelID)
    Repo-->>Send: {roomID, kind: text}
    Send->>Send: domain.NewMessage(id, channelID, authorID, MessageText, now)
    Send->>Repo: Save(msg)
    Repo-->>Send: nil
    Send->>Hub: PublishToChannel(channelID, "message.new", payload)
    Hub-->>Subs: ws frame {type:message.new, data:{...}}
    Send-->>WsH: SendMessageOutput{messageID, createdAt}
    WsH-->>Client: ws frame {type:message.sent, data:{message_id, created_at}}
```

**Error cases:**

| Условие | Доменная ошибка | WS-ответ | REST-эквивалент (на будущее) |
|---|---|---|---|
| Пустой / только пробелы / > 4000 рун / control-char | `ErrInvalidMessageText` | `{type:"error", code:"CHAT-001"}` | 400 |
| Канал не найден | `ErrChannelNotFound` (из chat/domain) | `{type:"error", code:"CHAT-002"}` | 404 |
| Канал не текстовый | `ErrChannelNotText` | `{type:"error", code:"CHAT-003"}` | 400 |
| Не член комнаты | `ErrChatAccessDenied` | `{type:"error", code:"CHAT-004"}` | 403 |
| Невалидный channel_id (не uuid) | парсинг handler-level | `{type:"error", code:"CHAT-005"}` | 400 |
| Внутренний сбой Save | обёрнутая `fmt.Errorf` | `{type:"error", code:"INTERNAL"}` | 500 |

**Edge cases:**
- **Гонка между подпиской и публикацией.** Если клиент B успел подписаться на канал ровно в момент broadcast'а, и в этот момент клиент A слал message.send — порядок недетерминирован. Принимаемое поведение: B либо получает, либо нет; история возвращается через REST `ListMessages` (см. UC-2).
- **Отвалившийся клиент.** Если `conn.WriteJSON` упал внутри `Hub.Publish`, hub удаляет соединение и продолжает рассылку остальным. Сообщение в БД сохранено — клиент при reconnect+REST подтянет его.
- **Запись быстрее broadcast'а.** Save и Publish не атомарны. Возможен сценарий: сообщение в БД, но broadcast не отправлен (panic/crash сервера между Save и Publish). Митигация — клиент при reconnect получает историю через REST до резюмирования подписки.
- **Race на write в один conn.** Внутри `Conn.WriteJSON` — sync.Mutex (см. `03-decisions.md` § D-03). Многократный writer (`Hub.Publish` из разных горутин) безопасен.

### UC-2: Чтение истории канала (`chat.ListMessages`)

**Триггер:** HTTP `GET /api/v1/channels/{channelID}/messages?before=<uuid>&limit=<n>`.

```mermaid
sequenceDiagram
    actor Client
    participant H as ListMessagesHandler
    participant MW as RequireAuth
    participant UC as ListMessages UC
    participant MQ as MembershipQuery
    participant Repo as MessageRepository
    participant DB as Postgres

    Client->>MW: GET /channels/:id/messages?before=&limit=<br/>Authorization: Bearer
    MW->>MW: VerifyAccess(token) → userID
    MW->>H: ctx with userID
    H->>H: parse channelID, before, limit
    H->>UC: Execute(actor, channelID, before, limit)
    UC->>MQ: Require(channelID, actor, RoleAnyMember)
    MQ-->>UC: nil
    UC->>Repo: ListByChannel(channelID, before, limit)
    Repo->>DB: SELECT ... WHERE channel_id=.. AND (id, created_at) < cursor ORDER BY created_at DESC, id DESC LIMIT n
    DB-->>Repo: rows
    Repo-->>UC: []*Message
    UC-->>H: ListMessagesOutput{Items, NextBefore}
    H-->>Client: 200 {items, nextBefore}
```

**Error cases:**

| Условие | Код | HTTP Status |
|---|---|---|
| Невалидный токен / нет токена | AUTH-010 | 401 |
| Истёкший токен | AUTH-011 | 401 |
| `channelID` не UUID | CHAT-005 | 400 |
| `before` не UUID (если задан) | CHAT-005 | 400 |
| `limit` < 1 или > 100 | CHAT-006 | 400 |
| Канал не найден | CHAT-002 | 404 |
| Не член комнаты канала | CHAT-004 | 403 |
| Внутренняя ошибка БД | INTERNAL | 500 |

**Edge cases:**
- **Пустая история.** Возвращаем `{"items":[],"nextBefore":null}` со статусом 200.
- **`limit` опущен.** Дефолт 50.
- **`before` опущен.** Самые последние сообщения (LIMIT n DESC).
- **Меньше n записей до начала истории.** `nextBefore = null` в ответе.
- **`before`-id не существует.** Курсор интерпретируется как «всё, что строго ДО этого id». Если id неизвестен — возвращаем пустой набор (а не 404), чтобы клиент мог писать «прокручиваю с какого-то места».

### UC-3: Подключение WebSocket (`WSHandler.ServeHTTP`)

**Триггер:** клиент инициирует `GET /api/v1/ws?token=<jwt>` с заголовком `Upgrade: websocket`.

```mermaid
sequenceDiagram
    actor Client
    participant H as WSHandler
    participant Iss as TokenIssuer
    participant MR as MembershipRepository
    participant Up as websocket.Upgrade
    participant Hub as Hub
    participant Conn as Conn

    Client->>H: GET /api/v1/ws?token=jwt<br/>Upgrade: websocket
    H->>H: token = query.Get("token")
    H->>Iss: VerifyAccess(token, now)
    Iss-->>H: userID
    H->>MR: ListByUser(userID)
    MR-->>H: []Membership (roomIDs)
    H->>Up: Upgrade(w, r, opts)
    Up-->>H: *Conn
    H->>Hub: Register(conn, userID)
    Hub-->>H: closeFn
    loop for each roomID
        H->>Hub: Subscribe(conn, RoomTopic(roomID))
    end
    H->>Conn: read loop
    Note over Conn: см. UC-4 и UC-1<br/>при получении событий
    Client--xConn: close / network error
    Conn-->>H: read err
    H->>Hub: closeFn()
```

**Error cases (до апгрейда — HTTP-ответ):**

| Условие | Код | HTTP Status |
|---|---|---|
| Нет токена в query | AUTH-010 | 401 |
| Невалидный токен | AUTH-010 | 401 |
| Истёкший токен | AUTH-011 | 401 |
| `Upgrade` не запрошен | — | 400 (nhooyr Accept вернёт ошибку, отдаём stock-HTTP) |
| Origin не в whitelist | — | 403 (через `pkg/websocket.Upgrade` opts из CORS-config) |

**Error cases (после апгрейда — WS-close с кодом):**

| Условие | Close code | Reason |
|---|---|---|
| Server shutdown | 1001 | "going away" |
| Невалидный JSON-фрейм | 1003 | "invalid frame" |
| Превышение `max message size` | 1009 | "message too large" |
| Внутренний сбой | 1011 | "internal error" |

**Edge cases:**
- **Нет членств у пользователя.** Авто-подписки на room: не создаются. Клиент остаётся подключённым и может позже `subscribe` на channel (если получит доступ), хотя в фазе 3 подписка на channel требует членство.
- **Ребаланс при `JoinByCode` после connect.** Если пользователь вступил в новую комнату ПОСЛЕ открытия WS, авто-подписка на её room-topic не происходит до следующего reconnect. Это сознательный компромисс MVP (см. `03-decisions.md` § D-09).
- **Дубликат коннектов.** Один user может иметь несколько активных WS-коннектов (с разных вкладок). Hub не объединяет — каждый коннект независим.

### UC-4: Подписка на канал (`subscribe`)

**Триггер:** клиент шлёт `{"type":"subscribe","channel_id":"<uuid>"}`.

```mermaid
sequenceDiagram
    actor Client
    participant H as WSHandler
    participant MQ as MembershipQuery
    participant Hub as Hub

    Client->>H: ws frame {type:subscribe, channel_id}
    H->>H: parse channel_id
    H->>MQ: Require(channelID, userID, RoleAnyMember)
    alt member
        MQ-->>H: nil
        H->>Hub: Subscribe(conn, ChannelTopic(channelID))
        H-->>Client: {type:"subscribed", channel_id}
    else not member
        MQ-->>H: ErrChatAccessDenied
        H-->>Client: {type:"error", code:"CHAT-004"}
    end
```

**Error cases:** см. таблицу UC-1, плюс невалидный `channel_id` (CHAT-005).

**Edge cases:**
- **Двойная подписка.** Повторный `subscribe` на тот же канал — no-op (set-семантика). Hub возвращает `{type:"subscribed"}` повторно.
- **Подписка на channel:<id> чужой комнаты.** `MembershipQuery.Require` вернёт `ErrChatAccessDenied`, клиент получит `{type:"error", code:"CHAT-004"}`.

### UC-5: Вступление в комнату с публикацией `member.joined` (`room.JoinByCode`)

**Триггер:** HTTP `POST /api/v1/rooms/join/{code}`.

Существующий handler / use case дополняется ОДНИМ вызовом publisher'а после `Add(membership)`. Логика остальных шагов (валидация кода, проверка `ErrAlreadyMember`) не меняется.

```mermaid
sequenceDiagram
    actor Newcomer
    participant H as JoinByCodeHandler
    participant UC as room.JoinByCode
    participant Invites as InviteRepository
    participant Mems as MembershipRepository
    participant Pub as RoomEventsPublisher
    participant Hub as Hub
    participant Subs as Existing room subs

    Newcomer->>H: POST /rooms/join/:code
    H->>UC: Execute(userID, code)
    UC->>Invites: FindByCode(code)
    Invites-->>UC: invite
    UC->>Mems: Add(membership member)
    Mems-->>UC: nil
    UC->>Pub: PublishMemberJoined(roomID, userID, joinedAt)
    Pub->>Hub: PublishToRoom(roomID, "member.joined", payload)
    Hub-->>Subs: ws frame {type:"member.joined", data:{room_id, user_id, joined_at}}
    UC-->>H: JoinByCodeOutput
    H-->>Newcomer: 200 OK
```

**Error cases / edge cases** для `JoinByCode` не меняются по сравнению с фазой 2 (см. `internal/room/usecase/join_by_code.go`). Дополнительные:

| Сценарий | Поведение |
|---|---|
| Publish не отвечает или паникует | Не откатывает membership. Логируется WARN. HTTP-ответ — 200. |
| Newcomer уже имеет открытый WS | Newcomer НЕ получает свой `member.joined`, потому что подписался на room-topic ещё до Join и подписан только на свои существующие комнаты. Для подписки на новую комнату — reconnect или ручной `subscribe` (см. § D-09). |

---

## Дополнительные сценарии

### S-1: Graceful shutdown

**Триггер:** `signal.NotifyContext` (SIGINT / SIGTERM) → `srv.Shutdown(ctx)`.

Поведение:
1. `srv.Shutdown` останавливает приём новых HTTP/WS-соединений.
2. `Hub.Shutdown(ctx)` итерируется по всем коннектам, на каждом вызывает `conn.Close(1001, "going away")`.
3. Сервер ждёт максимум `shutdownTimeout = 5s` (`cmd/server/main.go:39`); после — принудительно завершает.

### S-2: Disconnect клиента

**Триггер:** клиент закрывает соединение / network drop / `read` возвращает ошибку.

Поведение:
1. Read-loop в `WSHandler` ловит ошибку.
2. Вызывает `closeFn` от `Hub.Register`.
3. Hub удаляет соединение из `subscriptions` (все topic'и) и реестра.
4. `conn.Close(1000, "bye")` — best-effort.

### S-3: Read-deadline превышен (пинг-пинг)

В MVP не реализуется специальный keepalive: `nhooyr.io/websocket` поддерживает ping/pong нативно через `*Conn.Ping(ctx)`. Параметризация ping-интервала и read-deadline — в фазе 3.2 (вне scope текущего PR). В рамках текущей фазы используются дефолты библиотеки.
