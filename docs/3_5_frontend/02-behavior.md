---
parent: ./README.md
view: process
feature: 3_5-frontend
---

# 3.5 Frontend — Behavior (DFD + Sequences)

## Data Flow Diagrams

### DFD-1: REST-вызов (общий поток)

```mermaid
flowchart LR
    User -->|click submit| Form
    Form -->|FormValues| Store
    Store -->|api function call| ApiFn
    ApiFn -->|apiFetch path init| SharedFetch
    SharedFetch -->|Authorization Bearer<br/>JSON body| Backend[(Rupor API)]
    Backend -->|JSON response<br/>or ErrorEnvelope| SharedFetch
    SharedFetch -->|ApiResult ok or error| ApiFn
    ApiFn -->|ViewModel или ошибка| Store
    Store -.->|state change| View
    View -.->|re-render| User
```

### DFD-2: WS-фрейм входящий

```mermaid
flowchart LR
    Backend[(Rupor WS Hub)] -->|JSON frame| WsClient
    WsClient -->|parseFrame| TypedFrame[IncomingFrame discriminated union]
    TypedFrame -->|switch by type| Dispatcher
    Dispatcher -->|subscribed message.new<br/>message.sent error| ChatStore[useChatStore]
    Dispatcher -->|member.joined| RoomsStore[useRoomsStore]
    ChatStore -.->|state change| View1[MessageList]
    RoomsStore -.->|state change| View2[MembersList]
```

### DFD-3: Auto-refresh при 401 AUTH-011

```mermaid
flowchart LR
    Call[any apiFetch call] -->|HTTP 401 AUTH-011| Fetch[shared api fetch]
    Fetch -->|refreshAccessOnce| Refresh[shared api refresh single-flight]
    Refresh -->|POST auth refresh| Backend1[(Rupor API)]
    Backend1 -->|200 new tokens| Refresh
    Refresh -->|update tokenStorage| Fetch
    Fetch -->|retry original request| Backend2[(Rupor API)]
    Backend2 -->|200 OK| Fetch
    Fetch -->|ApiResult ok| Call
```

---

## Sequence Diagrams

### Use Case 1: Регистрация

```mermaid
sequenceDiagram
    actor User
    participant View as RegisterForm
    participant Store as useAuthStore
    participant ApiR as authApi.register
    participant ApiL as authApi.login
    participant Me as authApi.me
    participant Fetch as shared/api/fetch
    participant BE as Rupor API

    User->>View: заполняет email/username/password
    User->>View: нажимает «Зарегистрироваться»
    View->>View: zod-validate (sync)
    View->>Store: register({ email, username, password })
    Store->>ApiR: register(req)
    ApiR->>Fetch: POST /auth/register
    Fetch->>BE: POST /api/v1/auth/register
    BE-->>Fetch: 201 { id, email, username, createdAt }
    Fetch-->>ApiR: ApiResult.ok
    ApiR-->>Store: UserResponse
    Store->>Store: status="loading"
    Note over Store: auto-login: вызываем login(...) с теми же email/password
    Store->>ApiL: login({ email, password })
    ApiL->>Fetch: POST /auth/login
    Fetch->>BE: POST /api/v1/auth/login
    BE-->>Fetch: 200 TokensResponse
    Store->>Store: tokenStorage.set(tokens)
    Store->>Me: me()
    Me->>BE: GET /api/v1/auth/me  [Authorization: Bearer ...]
    BE-->>Me: 200 CurrentUser
    Me-->>Store: ApiResult.ok
    Store->>Store: status="authenticated", currentUser=...
    View->>View: useNavigate("/rooms")
    View-->>User: переход на /rooms
```

**Error cases:**

| Условие | Source | UI | Store |
|---|---|---|---|
| `AUTH-001` invalid email | register | inline error на `email` | `status=error` |
| `AUTH-002` invalid username | register | inline error на `username` | `status=error` |
| `AUTH-003` invalid password | register | inline error на `password` | `status=error` |
| `AUTH-004` email taken | register | inline error на `email` | `status=error` |
| `AUTH-005` username taken | register | inline error на `username` | `status=error` |
| Network error | fetch | toast «Нет соединения» | без изменений |
| Auto-login после register упал | login | form-level error «Зарегистрировано, но не вошли — попробуйте login вручную» + navigate /login | `status=error` |

### Use Case 2: Логин

```mermaid
sequenceDiagram
    actor User
    participant View as LoginForm
    participant Store as useAuthStore
    participant ApiL as authApi.login
    participant Me as authApi.me
    participant TokStore as tokenStorage
    participant BE as Rupor API

    User->>View: вводит email/password, Enter / Click
    View->>View: zod-validate
    View->>Store: login(req)
    Store->>ApiL: login(req)
    ApiL->>BE: POST /api/v1/auth/login
    BE-->>ApiL: 200 TokensResponse
    ApiL-->>Store: ok
    Store->>TokStore: set(tokens)
    Store->>Me: me()
    Me->>BE: GET /api/v1/auth/me
    BE-->>Me: 200 CurrentUser
    Store->>Store: status="authenticated"
    View->>View: useNavigate("/rooms")
```

**Error cases:**

| Условие | UI | Store |
|---|---|---|
| `AUTH-006` invalid credentials | form-level «Неверный email или пароль» | `status=error` |
| `AUTH-012` malformed body | bug-баннер (UI-баг) | `status=error` |
| 500 INTERNAL | toast | `status=error` |

### Use Case 3: Auto-refresh access-токена (фоновый процесс)

```mermaid
sequenceDiagram
    participant App as Любой компонент
    participant Store as Любой store
    participant Fetch as shared/api/fetch
    participant Refresh as shared/api/refresh (single-flight)
    participant TokStore as tokenStorage
    participant BE as Rupor API

    App->>Store: action
    Store->>Fetch: apiFetch("/rooms")
    Fetch->>BE: GET /api/v1/rooms  [Bearer expired]
    BE-->>Fetch: 401 AUTH-011
    Fetch->>Refresh: refreshAccessOnce()
    Note over Refresh: currentRefresh==null → стартуем
    Refresh->>BE: POST /api/v1/auth/refresh  { refreshToken }
    BE-->>Refresh: 200 TokensResponse (new pair)
    Refresh->>TokStore: set(new tokens)
    Refresh-->>Fetch: ApiResult.ok
    Fetch->>BE: GET /api/v1/rooms (retry с новым access)
    BE-->>Fetch: 200 OK rooms
    Fetch-->>Store: ApiResult.ok rooms
    Store-->>App: state update
```

**Параллельные 401 (single-flight):**

```mermaid
sequenceDiagram
    participant A as Запрос A
    participant B as Запрос B
    participant Fetch as fetch.ts
    participant Refresh as refresh.ts
    participant BE as Rupor API

    A->>Fetch: GET /rooms
    B->>Fetch: GET /channels
    Fetch->>BE: GET /rooms
    Fetch->>BE: GET /channels
    BE-->>Fetch: 401 AUTH-011 (для A)
    BE-->>Fetch: 401 AUTH-011 (для B)
    Fetch->>Refresh: refreshAccessOnce() [A]
    Note over Refresh: currentRefresh==null → создаём promise
    Fetch->>Refresh: refreshAccessOnce() [B]
    Note over Refresh: currentRefresh != null → возвращаем тот же promise
    Refresh->>BE: POST /auth/refresh (один раз)
    BE-->>Refresh: 200 new tokens
    Refresh-->>Fetch: ok (для A)
    Refresh-->>Fetch: ok (для B)
    Fetch->>BE: retry GET /rooms
    Fetch->>BE: retry GET /channels
```

**Error cases auto-refresh:**

| Условие | Реакция |
|---|---|
| refresh вернул 401 `AUTH-007/008/009` | logout: clear stores, navigate /login |
| retry оригинального запроса снова дал 401 `AUTH-011` | logout (refresh-loop защита, см. `03-decisions.md` D-05) |
| Запрос был на сам `/auth/refresh` и упал с 401 AUTH-011 | logout (не пытаемся refresh refresh-а) |
| 500 на /auth/refresh | toast + logout |

### Use Case 4: Загрузка списка комнат и выбор активной

```mermaid
sequenceDiagram
    actor User
    participant Page as AppShellPage
    participant RoomsStore as useRoomsStore
    participant ApiR as roomsApi.listRooms
    participant Fetch as fetch.ts
    participant BE

    User->>Page: открывает /rooms
    Page->>RoomsStore: useEffect → loadRooms()
    RoomsStore->>ApiR: listRooms()
    ApiR->>Fetch: GET /rooms
    Fetch->>BE: GET /api/v1/rooms
    BE-->>Fetch: 200 { items: [...] }
    Fetch-->>RoomsStore: ApiResult.ok
    RoomsStore->>RoomsStore: rooms=..., status="ready"
    Page-->>User: рендер sidebar со списком

    User->>Page: click on RoomListItem
    Page->>RoomsStore: selectRoom(roomId)
    Page->>Page: useNavigate(`/rooms/${roomId}`)
```

### Use Case 5: Создание комнаты

```mermaid
sequenceDiagram
    actor User
    participant Modal as CreateRoomModal
    participant RoomsStore as useRoomsStore
    participant Api as roomsApi.createRoom
    participant BE

    User->>Modal: вводит name, click «Создать»
    Modal->>Modal: zod-validate (1..64 руны)
    Modal->>RoomsStore: createRoom(name)
    RoomsStore->>Api: createRoom({ name })
    Api->>BE: POST /api/v1/rooms { name }
    BE-->>Api: 201 RoomDto
    Api-->>RoomsStore: ApiResult.ok
    RoomsStore->>RoomsStore: rooms.push({...room, role:"owner"})
    Modal->>Modal: useNavigate(`/rooms/${room.id}`)
    Modal->>Modal: close()
```

**Error cases:** `ROOM-001 invalid name` → inline; `ROOM-009 invalid body` → bug-баннер.

### Use Case 6: Удаление комнаты (только owner)

```mermaid
sequenceDiagram
    actor User
    participant Confirm as DeleteRoomConfirm
    participant RoomsStore
    participant Api as roomsApi.deleteRoom
    participant BE

    User->>Confirm: click «Удалить»
    Confirm->>RoomsStore: deleteRoom(roomId)
    RoomsStore->>Api: deleteRoom(roomId)
    Api->>BE: DELETE /api/v1/rooms/{id}
    BE-->>Api: 204 No Content
    RoomsStore->>RoomsStore: rooms = rooms.filter(r=>r.id!==roomId)
    Note over RoomsStore: каскадно очищаем channelsByRoom[id] и messagesByChannel
    Confirm->>Confirm: useNavigate("/rooms")
```

**Error cases:** `ROOM-005 only owner` → toast «Удалять может только владелец»; `ROOM-003 not a member` → удалить из стора (мы там не состоим больше).

### Use Case 7: Создание/показ инвайт-кода

```mermaid
sequenceDiagram
    actor User
    participant Modal as InviteCodeModal
    participant RoomsStore
    participant Api as roomsApi.regenerateInvite
    participant BE

    User->>Modal: открывает «Пригласить»
    Note over Modal: если activeInviteByRoom[roomId] есть — показать; иначе — пусто
    User->>Modal: click «Сгенерировать новый код»
    Modal->>RoomsStore: regenerateInvite(roomId)
    RoomsStore->>Api: regenerateInvite(roomId)
    Api->>BE: POST /api/v1/rooms/{id}/invite
    BE-->>Api: 200 InviteCodeDto
    Api-->>RoomsStore: ok
    RoomsStore->>RoomsStore: activeInviteByRoom[roomId] = code
    Modal-->>User: показывает code, кнопка «Копировать»
```

**Side-effect:** при каждом нажатии ВСЕ предыдущие активные инвайты комнаты отзываются на бэке (это поведение бэка, см. baseline). UI отображает только последний.

**Error cases:** `ROOM-004 admin/owner required` → toast «Только админ/владелец могут создавать инвайты».

### Use Case 8: Вступление в комнату по коду

```mermaid
sequenceDiagram
    actor User
    participant Modal as JoinByCodeModal
    participant RoomsStore
    participant ChannelsStore
    participant Ws as wsClient
    participant Api as roomsApi.joinByCode
    participant BE

    User->>Modal: вводит код (8 chars)
    Modal->>Modal: zod-validate (длина, Crockford-алфавит)
    Modal->>RoomsStore: joinByCode(code)
    RoomsStore->>Api: joinByCode(code)
    Api->>BE: POST /api/v1/rooms/join/{code}
    BE-->>Api: 200 RoomDto
    Api-->>RoomsStore: ok
    RoomsStore->>RoomsStore: rooms.push({...room, role:"member"})
    Note over RoomsStore,Ws: КРИТИЧНО: reconnect WS<br/>чтобы получить подписку на room:<id>
    RoomsStore->>Ws: reconnect()
    Ws->>BE: close (1000) + new connect ?token=
    BE-->>Ws: re-subscribe room:<all rooms incl. new>
    Modal->>Modal: useNavigate(`/rooms/${room.id}`)
```

**Error cases:**
| Code | UI | Store |
|---|---|---|
| `ROOM-008 invalid code` | inline «Неверный формат кода» | — |
| `ROOM-007 invite not found` | inline «Код не найден или отозван» | — |
| `ROOM-006 already a member` | toast + navigate в существующую комнату | — |

### Use Case 9: Загрузка списка членов комнаты

```mermaid
sequenceDiagram
    actor User
    participant Page as AppShellPage
    participant RoomsStore
    participant Api as roomsApi.listMembers
    participant BE

    User->>Page: открывает /rooms/:id
    Page->>RoomsStore: useEffect → loadMembers(roomId)
    RoomsStore->>Api: listMembers(roomId)
    Api->>BE: GET /api/v1/rooms/{id}/members
    BE-->>Api: 200 { items: [{userId,role,joinedAt}] }
    Api-->>RoomsStore: ok
    RoomsStore->>RoomsStore: membersByRoom[id] = items
    Page-->>User: рендер MembersList
```

**WS дополнение:** при `member.joined` (см. UC-11) — `appendMember`. UUID-placeholder в UI для незнакомых членов.

### Use Case 10: CRUD каналов (создание / список / удаление)

```mermaid
sequenceDiagram
    actor User
    participant Page as AppShellPage
    participant ChannelsStore
    participant Api as channelsApi
    participant BE

    User->>Page: открывает /rooms/:rid
    Page->>ChannelsStore: useEffect → loadChannels(rid)
    ChannelsStore->>Api: listChannels(rid)
    Api->>BE: GET /api/v1/rooms/{rid}/channels
    BE-->>Api: 200 { items: [...] }
    Api-->>ChannelsStore: channelsByRoom[rid] = items

    Note over User: разделение на text/voice (voice — disabled)
    User->>Page: click text-канал
    Page->>ChannelsStore: selectChannel(cid)
    Page->>Page: navigate(`/rooms/${rid}/channels/${cid}`)

    Note over User: admin/owner: open CreateChannelModal
    User->>Page: click «+ канал»
    User->>Page: name=..., kind="text"
    Page->>ChannelsStore: createChannel(rid, {name,kind})
    ChannelsStore->>Api: createChannel(rid, req)
    Api->>BE: POST /api/v1/rooms/{rid}/channels
    BE-->>Api: 201 ChannelDto
    ChannelsStore->>ChannelsStore: channelsByRoom[rid].push(channel)
```

**Error cases (create):**
| Code | UI |
|---|---|
| `CHANNEL-001` invalid name | inline |
| `CHANNEL-002` invalid kind | bug-баннер (UI должен валидировать enum) |
| `CHANNEL-004` name taken | inline «Имя уже занято» |
| `CHANNEL-007` admin/owner required | toast |

### Use Case 11: Открытие канала и подписка через WS

```mermaid
sequenceDiagram
    actor User
    participant Page as AppShellPage
    participant ChatStore as useChatStore
    participant Api as chatApi.listMessages
    participant Ws as wsClient
    participant BE

    User->>Page: click на text-канал
    Page->>Page: navigate(`/rooms/${rid}/channels/${cid}`)
    Page->>ChatStore: useEffect → openChannel(cid)
    Note over ChatStore: если истории ещё нет:
    ChatStore->>Api: listMessages(cid, {limit:50})
    Api->>BE: GET /api/v1/channels/{cid}/messages?limit=50
    BE-->>Api: 200 { items, nextBefore }
    Api-->>ChatStore: messagesByChannel[cid] = items.reverse()
    ChatStore->>ChatStore: nextBeforeByChannel[cid] = nextBefore
    ChatStore->>Ws: send {type:"subscribe", channel_id:cid}
    Ws->>BE: ws frame
    BE-->>Ws: {type:"subscribed", data:{channel_id:cid}}
    Ws-->>ChatStore: onSubscribed(data)
    ChatStore->>ChatStore: subscribedChannelId = cid
    Page-->>User: ChatPanel готов к отправке
```

**Error cases:**
| Code | Источник | UI | Store |
|---|---|---|---|
| 403 CHAT-004 | listMessages REST | toast «Нет доступа» | удалить из rooms/channels (мы не member) |
| 404 CHAT-002 | listMessages REST | toast «Канал не найден» | удалить channel |
| WS error CHAT-002 | ws | toast «Канал не найден» | — |
| WS error CHAT-004 | ws | toast | — |
| WS error CHAT-005 | ws | bug-баннер | — |

### Use Case 12: Отправка сообщения (optimistic + ack)

```mermaid
sequenceDiagram
    actor User
    participant Comp as MessageComposer
    participant ChatStore
    participant Ws as wsClient
    participant BE

    User->>Comp: вводит текст, Enter
    Comp->>Comp: zod-validate (1..4000)
    Comp->>ChatStore: sendMessage(cid, text)
    ChatStore->>ChatStore: tempId=uuid(); push {id:tempId, status:"pending", ...}
    ChatStore->>Ws: send {type:"message.send", channel_id, text}
    Ws->>BE: ws frame
    BE-->>Ws: {type:"message.sent", data:{id, channel_id, created_at}}
    Ws-->>ChatStore: onMessageSent(data)
    ChatStore->>ChatStore: найти последний pending, обновить id+createdAt, status="committed"
    BE-->>Ws: {type:"message.new", data:{id,channel_id,author_id,text,created_at}}
    Ws-->>ChatStore: onMessageNew(data)
    ChatStore->>ChatStore: dedup по id — игнор (уже committed выше)
```

**Multi-tab:**

```mermaid
sequenceDiagram
    actor UserA as User (Tab A)
    actor UserB as User (Tab B, тот же пользователь)
    participant ChatA as ChatStore (Tab A)
    participant ChatB as ChatStore (Tab B)
    participant BE

    UserA->>ChatA: sendMessage(cid,"hi")
    ChatA->>ChatA: optimistic push tempId
    ChatA->>BE: ws send message.send (Tab A WS)
    BE-->>ChatA: ws message.sent (только Tab A)
    ChatA->>ChatA: commit tempId → real id
    BE-->>ChatA: ws message.new (broadcast канал)
    BE-->>ChatB: ws message.new (broadcast канал)
    ChatA->>ChatA: dedup (id уже committed) — skip
    ChatB->>ChatB: append (новое сообщение) — show
```

**Error cases:**
| WS error code | UI | Store |
|---|---|---|
| `CHAT-001 invalid text` | inline под composer; pending → failed | failMessage(tempId) |
| `CHAT-002/003/004` | toast; pending → failed | failMessage(tempId) |
| WS disconnect перед message.sent | UI: status="pending" → "failed" по таймауту 10s | timer + failMessage |

### Use Case 13: Infinite scroll вверх (история)

```mermaid
sequenceDiagram
    actor User
    participant List as MessageList
    participant ChatStore
    participant Api as chatApi.listMessages
    participant BE

    User->>List: scroll к самому верху
    List->>ChatStore: loadMoreHistory(cid)
    Note over ChatStore: nextBeforeByChannel[cid] не null
    ChatStore->>Api: listMessages(cid, {before, limit:50})
    Api->>BE: GET /messages?before=...&limit=50
    BE-->>Api: 200 { items, nextBefore }
    Api-->>ChatStore: предпендить items в messagesByChannel[cid]
    ChatStore->>ChatStore: nextBeforeByChannel[cid] = nextBefore (может стать null = конец)
    List-->>User: видит подгруженные старые сообщения
```

**Edge case:** если `nextBefore === null` — пользователь у самого начала истории. UI: подпись «Больше сообщений нет».

### Use Case 14: WS reconnect

```mermaid
sequenceDiagram
    participant Ws as wsClient
    participant ChatStore
    participant UI as ConnectionStatusBanner
    participant BE

    BE-->>Ws: close 1006 (network)
    Ws->>ChatStore: setWsStatus("reconnecting")
    UI-->>UI: рендер баннера «Переподключение…»
    Ws->>Ws: ждём 1s (backoff 1)
    Ws->>BE: new WebSocket?token=
    BE-->>Ws: open
    Ws->>ChatStore: setWsStatus("open")
    UI-->>UI: убирает баннер
    Ws->>BE: re-subscribe текущий открытый канал (если был)
    BE-->>Ws: {type:"subscribed"}
    Note over Ws,ChatStore: room:* подписки восстановлены сервером автоматически
    ChatStore->>Api: loadHistory(cid) с before=null до пересечения с локальной
```

**Backoff:** 1s, 2s, 5s, 15s, 15s, 15s, ... (бесконечно). Каждая попытка обновляет `wsStatus="reconnecting"`.

**После logout:** `wsClient.disconnect()` → close с 1000 → reconnect НЕ запускается.

### Use Case 15: Logout

```mermaid
sequenceDiagram
    actor User
    participant Badge as UserBadge
    participant AuthStore as useAuthStore
    participant RoomsStore
    participant ChannelsStore
    participant ChatStore
    participant TokStore
    participant Ws

    User->>Badge: click «Выйти»
    Badge->>AuthStore: logout()
    AuthStore->>TokStore: clear()
    AuthStore->>RoomsStore: clear()
    AuthStore->>ChannelsStore: clear()
    AuthStore->>ChatStore: clear()
    AuthStore->>Ws: disconnect()
    AuthStore->>AuthStore: clear()
    Badge->>Badge: useNavigate("/login", {replace:true})
```

Серверного эндпоинта нет — операция чисто локальная.

### Use Case 16: Восстановление сессии после reload

```mermaid
sequenceDiagram
    actor User
    participant Boot as App.tsx
    participant AuthStore
    participant TokStore
    participant Me as authApi.me
    participant BE

    User->>Boot: F5 / открыл вкладку
    Boot->>TokStore: getAccess()
    alt access есть
        Boot->>AuthStore: loadMe()
        AuthStore->>Me: me()
        Me->>BE: GET /auth/me  [Bearer]
        alt 200 OK
            BE-->>Me: CurrentUser
            AuthStore->>AuthStore: status="authenticated"
            Boot->>Boot: рендерит /rooms или текущий route
        else 401 AUTH-011
            Me->>Fetch: внутренний flow auto-refresh
            Note over Fetch: refresh → retry me → ok
        else 401 AUTH-007/008/009/010
            Note over Boot: logout → navigate("/login")
        end
    else access нет
        Boot->>Boot: navigate("/login")
    end
```

---

## Error cases (сводная таблица по всем UC)

| Условие | Источник | Реакция UI | Реакция Store |
|---|---|---|---|
| 401 `AUTH-011` access expired | apiFetch (любой) | прозрачно (idle spinner) | auto-refresh single-flight → retry |
| 401 `AUTH-007/008/009/010` | apiFetch | redirect `/login` | logout: clear stores+tokens |
| 401 `AUTH-006` invalid creds | login form | form-level error | `status=error` |
| 400 `AUTH-001/002/003` | register form | inline field error | `status=error` |
| 409 `AUTH-004/005` | register | inline field error | `status=error` |
| 400 `ROOM-001` invalid name | createRoom | inline | — |
| 404 `ROOM-002` | getRoom / list | toast + redirect `/rooms` | удалить из стора |
| 403 `ROOM-003` not member | get/list members/channels | toast + redirect | удалить из стора |
| 403 `ROOM-004` admin/owner | regenerate invite, create channel | toast | — |
| 403 `ROOM-005` only owner | delete room | toast | — |
| 409 `ROOM-006` already member | join | toast + navigate туда | — |
| 404 `ROOM-007` invite not found | join | inline | — |
| 400 `ROOM-008` invalid code | join | inline | — |
| 400 `CHANNEL-001` invalid name | create channel | inline | — |
| 409 `CHANNEL-004` name taken | create channel | inline | — |
| 400 `CHAT-005` invalid uuid | list messages | bug-баннер | — |
| 400 `CHAT-006` invalid limit | list messages | bug-баннер | — |
| 403 `CHAT-004` not member | list messages | toast + удалить channel/room | — |
| WS `CHAT-001` invalid text | ws-error frame | inline под composer; pending→failed | failMessage |
| WS `CHAT-002/003` | ws-error frame | toast + close channel | — |
| WS `CHAT-004` | ws-error frame | toast | — |
| WS close 1006/1011 | wsClient | banner «Переподключение…» | wsStatus=reconnecting |
| WS close 1000 | wsClient | банер исчезает | wsStatus=closed; reconnect=no |
| network error / offline | apiFetch | toast «Нет соединения» | — |
| 500 INTERNAL (любой) | apiFetch | toast «Внутренняя ошибка» | — |

---

## Edge cases

**Multi-tab:**
- См. UC-12 (multi-tab диаграмма). Дедупликация по `id` в `onMessageNew`.
- `useAuthStore` персистится в localStorage → smena токенов в одной вкладке видна в другой через `storage` event (опционально слушаем).

**WS reconnect после join:**
- См. UC-8: после `joinByCode` обязателен `wsClient.reconnect()`. Иначе пользователь не увидит `member.joined` в новой комнате и не получит WS-обновлений для будущих каналов в ней (если откроет).

**Race condition: одновременный refresh:**
- Single-flight через `refresh.ts`, см. UC-3.

**Empty states:**
- Нет комнат: «У вас пока нет комнат. Создайте первую или присоединитесь по коду» + кнопки `Create` и `Join`.
- Нет каналов в комнате: «В этой комнате пока нет каналов» (+ кнопка `Create` для admin/owner).
- Нет сообщений в канале: «Будь первым».
- Нет членов (теоретически невозможно — actor сам там): defensive «—».

**Stale data:**
- При возврате на экран после долгого отсутствия — `useEffect` на mount компонента рефетчит данные. Сообщения дополняются через WS, но если был disconnect — после reconnect нужен `loadHistory` с `before=` до пересечения. См. `05-state-model.md`.

**Bug-баннер:**
- Реакция на `AUTH-012`, `ROOM-009`, `CHANNEL-005`, `CHAT-005/006/007`, `CHANNEL-002`. Это коды, которые не должны прилетать от валидного фронта. UI: красный баннер «Внутренняя ошибка клиента. Перезагрузите страницу или сообщите в поддержку».
- Логируем в `console.error` + (опционально) отправляем в Sentry без тела запроса.

**Невалидный JSON в WS-фрейме:**
- `parseFrame` возвращает `null`, dispatcher логирует warn, фрейм игнорируется. WS-соединение НЕ закрываем.

**Длинное сообщение:**
- 4000 рун — лимит бэка. Composer показывает счётчик при >3000 (для UX). На submit — zod-validate. Если каким-то образом ушло >4000 — WS `error` `CHAT-001`.

---

## Дополнительные сценарии

### Удаление текущего канала другим пользователем

Бэк не присылает WS-событие об удалении канала. Если другой админ удалил канал, в котором текущий пользователь — следующий запрос (`listMessages`/`message.send`) даст 404 `CHAT-002`. Реакция: удалить канал из локального `useChannelsStore`, navigate на `/rooms/:rid`, toast «Канал удалён».

### Удаление комнаты другим пользователем

Аналогично: при следующем запросе по этой комнате — 404 `ROOM-002`, redirect `/rooms` + удалить из локального стора + toast.

### Изменение роли (бэк не даёт это менять в PR-2)

Out of scope — UI просто отображает текущую роль из `RoomWithRole.role`.

### Истечение refresh-токена в фоне

При следующем запросе → 401 AUTH-007/008/009 → logout. Пользователь видит redirect на `/login` с toast «Сессия истекла, войдите снова».

### Vite dev: WS-прокси

Прокси Vite поддерживает `ws: true`. WS-апгрейд работает прозрачно. См. `06-api-integration.md`.

### Открытие deep-link на закрытой комнате

`/rooms/:rid/channels/:cid` — гард `requireMembership` проверяет наличие комнаты в `useRoomsStore.rooms`. Если её нет (ещё не загрузились) — wait spinner. Если после загрузки её всё ещё нет — redirect `/rooms` + toast «Комната не найдена».
