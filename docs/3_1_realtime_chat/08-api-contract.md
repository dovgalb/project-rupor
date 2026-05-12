---
parent: ./README.md
---

# 08 — API Contract

Точный контракт всех новых эндпоинтов фазы 3.1. REST — JSON с camelCase (как существующие домены). WebSocket — JSON-конверт `{type, ...}` со snake_case (см. `05-events.md` § Кодирование payload).

## REST

### `GET /api/v1/channels/{channelID}/messages`

Возвращает страницу истории сообщений канала, отсортированную от новых к старым. Курсорная пагинация.

**Auth:** обязательно `Authorization: Bearer <access-token>`.

**Request:**

```
Path params:
  channelID: string (uuid)         — id канала; обязательно

Query params:
  before:  string (uuid, optional) — id сообщения, после которого возвращать;
                                     если не задан — возвращаются самые свежие
  limit:   int    (optional)       — размер страницы; default 50; min 1; max 100

Headers:
  Authorization: Bearer {access-token}
```

**Response 200 OK:**

```json
{
  "items": [
    {
      "id":        "9c5b9a6a-3d27-4f5e-9a1b-2b9b1f9b3e88",
      "channelId": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff",
      "authorId":  "5fa8b3c1-22d4-4e9f-9012-345678901234",
      "text":      "Привет!",
      "createdAt": "2026-05-12T17:32:08.123456789Z"
    },
    {
      "id":        "...",
      "channelId": "...",
      "authorId":  "...",
      "text":      "...",
      "createdAt": "2026-05-12T17:31:00.000000000Z"
    }
  ],
  "nextBefore": "9c5b9a6a-3d27-4f5e-9a1b-2b9b1f9b3e88"
}
```

- `items` — массив сообщений, отсортированный по `createdAt DESC, id DESC`.
- `nextBefore` — id ПОСЛЕДНЕГО элемента в `items`, чтобы клиент мог запросить следующую страницу как `?before=<nextBefore>`. Если `len(items) < limit` (это была последняя страница), `nextBefore` — `null`.
- При пустом `items` (`len == 0`) — `nextBefore = null`.

**Error responses:**

| Status | Error code | Body | Когда |
|---|---|---|---|
| 400 | `CHAT-005` | `{"error":{"code":"CHAT-005","message":"invalid channel or cursor uuid"}}` | `channelID` или `before` не uuid |
| 400 | `CHAT-006` | `{"error":{"code":"CHAT-006","message":"limit must be 1..100"}}` | `limit` вне диапазона |
| 401 | `AUTH-010` | `{"error":{"code":"AUTH-010","message":"access token invalid"}}` | Нет токена / невалидный |
| 401 | `AUTH-011` | `{"error":{"code":"AUTH-011","message":"access token expired"}}` | Истёкший токен |
| 403 | `CHAT-004` | `{"error":{"code":"CHAT-004","message":"access denied: not a room member"}}` | Не член комнаты канала |
| 404 | `CHAT-002` | `{"error":{"code":"CHAT-002","message":"channel not found"}}` | Канал не существует |
| 500 | `INTERNAL` | `{"error":{"code":"INTERNAL","message":"internal"}}` | Сбой БД |

---

## WebSocket

### `GET /api/v1/ws?token=<jwt>`

Apgrade соединения в WebSocket. После апгрейда — двусторонний обмен JSON-фреймами.

**Auth:** обязательно `token` в query-параметре. `Authorization`-заголовок не используется (браузерный WS API не поддерживает custom headers при handshake).

**Handshake-request:**

```
GET /api/v1/ws?token=<jwt> HTTP/1.1
Host:                example.com
Upgrade:             websocket
Connection:          Upgrade
Sec-WebSocket-Key:   ...
Sec-WebSocket-Version: 13
Origin:              https://app.example.com
```

**Handshake-response (успех): 101 Switching Protocols** — стандартный WS-апгрейд.

**Handshake-response (ошибка, до апгрейда):**

| Status | Error code | Body | Когда |
|---|---|---|---|
| 401 | `AUTH-010` | `{"error":{"code":"AUTH-010","message":"access token invalid"}}` | Нет токена / невалидный |
| 401 | `AUTH-011` | `{"error":{"code":"AUTH-011","message":"access token expired"}}` | Истёкший токен |
| 403 | — | (nhooyr-default) | Origin не в whitelist |
| 400 | — | (nhooyr-default) | Не запрошен `Upgrade: websocket` |

После апгрейда ошибки представляются:
- **WS-фрейм** `{type:"error",...}` для прикладных ошибок (см. ниже).
- **Close frame** с close-кодом и причиной для протокольных ошибок (см. `02-behavior.md` § UC-3 error cases).

---

### Конверт фрейма

Все фреймы — text JSON со схемой:

```json
{ "type": "<event>", "data": { ...payload } }
```

или (для команд клиента, не требующих body):

```json
{ "type": "<event>", "channel_id": "<uuid>" }
```

Поля верхнего уровня — snake_case. Сериализация UUID — каноническая строка. Время — RFC3339Nano UTC.

---

### Inbound (client → server)

#### `subscribe`

Подписка на канал. Сервер начинает доставлять `message.new` этому коннекту для указанного канала.

```json
{
  "type": "subscribe",
  "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff"
}
```

**Ответ при успехе:**

```json
{
  "type": "subscribed",
  "data": { "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff" }
}
```

**Ответ при ошибке:**

```json
{
  "type": "error",
  "data": { "code": "CHAT-004", "message": "access denied: not a room member" }
}
```

**Ошибки:**

| Code | Описание |
|---|---|
| `CHAT-005` | `channel_id` не uuid |
| `CHAT-002` | Канал не существует |
| `CHAT-004` | Не член комнаты |

#### `message.send`

Отправка нового сообщения в text-канал.

```json
{
  "type": "message.send",
  "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff",
  "text": "Привет!"
}
```

**Ответ при успехе (только отправителю):**

```json
{
  "type": "message.sent",
  "data": {
    "id":         "9c5b9a6a-3d27-4f5e-9a1b-2b9b1f9b3e88",
    "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff",
    "created_at": "2026-05-12T17:32:08.123456789Z"
  }
}
```

Параллельно сервер рассылает `message.new` всем подписчикам канала (включая отправителя — для подтверждения и для синхронизации между его вкладками).

**Ответ при ошибке (только отправителю):**

```json
{
  "type": "error",
  "data": { "code": "CHAT-001", "message": "invalid message text" }
}
```

**Ошибки:**

| Code | Описание |
|---|---|
| `CHAT-001` | Пустой / слишком длинный / control-char текст |
| `CHAT-002` | Канал не существует |
| `CHAT-003` | Канал не текстовый (voice) |
| `CHAT-004` | Не член комнаты |
| `CHAT-005` | `channel_id` не uuid |
| `INTERNAL` | Сбой БД |

---

### Outbound (server → client)

#### `message.new`

См. `05-events.md` § `message.new`. Доставляется всем подписчикам канала.

```json
{
  "type": "message.new",
  "data": {
    "id":         "9c5b9a6a-3d27-4f5e-9a1b-2b9b1f9b3e88",
    "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff",
    "author_id":  "5fa8b3c1-22d4-4e9f-9012-345678901234",
    "text":       "Привет!",
    "created_at": "2026-05-12T17:32:08.123456789Z"
  }
}
```

#### `member.joined`

См. `05-events.md` § `member.joined`. Доставляется всем подписчикам комнаты (авто-подписка при connect).

```json
{
  "type": "member.joined",
  "data": {
    "room_id":   "1234aaaa-bbbb-cccc-dddd-555566667777",
    "user_id":   "5fa8b3c1-22d4-4e9f-9012-345678901234",
    "joined_at": "2026-05-12T17:35:00.000000000Z"
  }
}
```

#### `error`

Адресный ответ на конкретную клиентскую команду. Доставляется только отправителю.

```json
{
  "type": "error",
  "data": {
    "code":    "CHAT-004",
    "message": "access denied: not a room member"
  }
}
```

#### `subscribed`

Подтверждение успешной подписки.

```json
{
  "type": "subscribed",
  "data": { "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff" }
}
```

#### `message.sent`

Подтверждение приёма команды `message.send`. Доставляется только отправителю. Содержит `id` и `created_at` — клиент может сразу пометить сообщение как доставленное и подсветить.

(Тело — см. выше под `message.send`.)

---

### Полная таблица кодов ошибок CHAT-*

Используемые префиксы в проекте: `AUTH`, `ROOM`, `CHANNEL`, `CONFIG`, `INTERNAL`. Префикс `CHAT-*` — свободный.

| Code | HTTP Status | WS-фрейм | Описание |
|---|---|---|---|
| `CHAT-001` | 400 | `error` | Невалидный текст сообщения (пустой, слишком длинный, control-char) |
| `CHAT-002` | 404 | `error` | Канал не найден |
| `CHAT-003` | 400 | `error` | Канал не текстовый (попытка послать в voice) |
| `CHAT-004` | 403 | `error` | Пользователь не член комнаты канала |
| `CHAT-005` | 400 | `error` | Невалидный UUID в path/query/payload |
| `CHAT-006` | 400 | — | `limit` вне диапазона `1..100` (REST-only) |
| `CHAT-007` | 400 | `error` | Неподдерживаемый WS-фрейм `type` (зарезервировано) |

`CHAT-007` зарезервирован для возможной валидации `type` (если клиент пришлёт неизвестное событие). В текущей реализации фазы — игнорируется (закрытие коннекта с 1003), но код выделен на будущее.

---

### Размеры и таймауты

| Параметр | Значение | Где зашит |
|---|---|---|
| Max message size (inbound WS frame) | 64 KB | `pkg/websocket/Upgrade` через `*websocket.Conn.SetReadLimit(65536)` |
| Read deadline | дефолт nhooyr | — |
| Write deadline | 5s | передаётся через `context.WithTimeout` в `Conn.WriteJSON` |
| Ping interval | дефолт nhooyr | — |
| Idle close timeout | дефолт nhooyr | — |

Параметризация через env — фаза 3.2, не блокирует текущую фазу (см. `03-decisions.md` Open Questions).

---

### Версионирование

Префикс `/api/v1/` сохраняется как сейчас. WS-эндпоинт также живёт под `/api/v1/`. Будущее версионирование протокола WS — через поле `protocol_version` в инцидентном `subscribe` или через путь `/api/v2/ws`. Решение откладывается до фактической миграции.
