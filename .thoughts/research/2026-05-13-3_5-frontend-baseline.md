---
date: 2026-05-13
researcher: Claude
commit: 7d7da4e
branch: feature/PR-3_5_frontend
research_question: |
  PR-3.5 — веб-клиент для уже реализованного бэкенда. Задокументировать всё,
  что фронту нужно знать о контрактах бэка: HTTP API (auth/room/channel/chat),
  WebSocket-протокол, общие middleware и формат ошибок, инфраструктуру и
  текущее состояние папки web/.
---

# Исследование: бэкенд-baseline для веб-клиента PR-3.5

## Резюме

Бэкенд для всех пунктов чек-листа PR-3.5 готов и работоспособен в текущей сборке (`7d7da4e`, ветка `feature/PR-3_5_frontend`). Реализованы четыре домена под `/api/v1`: **auth** (register/login/refresh/me — logout-эндпоинта нет, отрабатывает локально), **room** (CRUD + инвайты + join по коду + список членов), **channel** (CRUD `text`/`voice` каналов внутри комнаты) и **chat** (REST-история сообщений с курсорной пагинацией + WebSocket-протокол `/api/v1/ws?token=` с командами `subscribe`/`message.send` и событиями `message.new`/`message.sent`/`subscribed`/`error`/`member.joined`). Общий формат ошибок — `{"error":{"code":"DOMAIN-NNN","message":"..."}}`, JSON везде в camelCase для REST и snake_case для WS-payload. CORS-whitelist по умолчанию — `http://localhost:5173` (Vite dev), `allowCredentials=false`, тот же список после `stripScheme` используется как `OriginPatterns` для WS-апгрейда.

Папка `web/` **отсутствует**: фронт нужно создавать с нуля. В `docker-compose.yml` поднимается только `postgres:16-alpine`; ни Go-сервер, ни фронт не контейнеризированы (`make run` запускает сервер локально). В `docs/` нет папки про фазу 3.5/frontend, нет и MVP-спецификации фронта в `tasks/` (файл `Functional and non-functional requirements.md` пуст). Домены `internal/voice/` и `internal/user/` — пустые заглушки (`.gitkeep`), и это влияет на UI: DTO участников комнаты содержит только `userId/role/joinedAt`, без `username`/`displayName` (зафиксировано в ADR D-09, `docs/2_1_rooms_and_channels/08-api-contract.md:207`); в WS-payload `message.new` тоже приходит только `author_id`, без ника.

Автоподписка на топик `room:<id>` происходит **один раз — в момент WS-connect** для всех комнат, где пользователь уже состоит. Если пользователь вступит в новую комнату через `POST /api/v1/rooms/join/{code}` — для появления её событий потребуется reconnect WS. Подписка на канал — всегда явная (`{"type":"subscribe","channel_id":"..."}`), членство и подписка — независимые сущности.

---

## Детальные результаты

### 1. Auth HTTP API + cross-cutting

#### 1.1 Эндпоинты

Префикс `/api/v1/auth`, регистрация — `internal/auth/transport/http/routes.go:19-31`. Группа `register/login/refresh` публичная, `me` — за `authmw.RequireAuth` (`routes.go:27-28`). Все ответы `application/json; charset=utf-8`, timestamps RFC3339 UTC, UUID 36-символьная форма. JSON-поля — **camelCase**.

| Метод | Путь | Хендлер | Auth |
|---|---|---|---|
| `POST` | `/api/v1/auth/register` | `register_handler.go:17` | публичный |
| `POST` | `/api/v1/auth/login` | `login_handler.go:17` | публичный |
| `POST` | `/api/v1/auth/refresh` | `refresh_handler.go:17` | публичный (валидируется body) |
| `GET`  | `/api/v1/auth/me` | `me_handler.go:19` | `Bearer` |

**POST /auth/register** — `register_handler.go:17`, DTO `dto.go:9-21`.
- Request: `{"email":"...","username":"...","password":"..."}`.
- Domain VO: email RFC-like ≤254 (`AUTH-001`); username ASCII alnum/`_`/`-`, 3..32 (`AUTH-002`); password 8..72 байт (`AUTH-003`).
- Success: `201 Created`, `{"id","email","username","createdAt"}`.
- Errors: `400 AUTH-001/002/003/012`, `409 AUTH-004 email already taken`, `409 AUTH-005 username already taken`, `500 INTERNAL`. Маппинг — `error_mapper.go:17-54`.

**POST /auth/login** — `login_handler.go:17`, DTO `dto.go:22-37`.
- Request: `{"email","password"}`. Любая ошибка валидации формата сводится к `401 AUTH-006 invalid credentials` (защита от user-enumeration, dummy-hash в `usecase/login_user.go:79`).
- Success: `200 OK`, `{"accessToken","refreshToken","accessExpiresAt","refreshExpiresAt"}`.
- `accessToken` — JWT HS256, claims `sub` (UserID UUID), `iat`, `exp`. Выпуск — `repository/jwt/token_issuer.go`.
- `refreshToken` — `base64url(32 random bytes)` без padding, 43 символа (`usecase/login_user.go:126`). На стороне БД хранится sha256 от него.
- Errors: `400 AUTH-012`, `401 AUTH-006`, `500 INTERNAL`.

**POST /auth/refresh** — `refresh_handler.go:17`.
- Request: `{"refreshToken":"..."}`.
- Семантика: старый refresh отзывается, выдаётся новая пара access+refresh (ротация, `usecase/refresh_access.go:104`).
- Success: тело идентично login-response.
- Errors: `400 AUTH-012 malformed body`, `401 AUTH-007 not found` (невалидный base64url / длина ≠ 32 байт / нет в БД), `401 AUTH-008 revoked`, `401 AUTH-009 expired`, `500 INTERNAL`.

**GET /auth/me** — `me_handler.go:19`.
- Request: только `Authorization: Bearer <accessToken>` (CamelCase, case-sensitive).
- Success: `200 OK`, `{"id","email","username","createdAt"}`.
- Errors из middleware (см. §1.2): `401 AUTH-010`, `401 AUTH-011`.

**Logout-эндпоинта нет** — в `routes.go` его нет, в use-cases — нет. Фронт делает logout локально: стереть `accessToken`/`refreshToken` из стора.

`405 Method Not Allowed` отвечает chi plain-text без JSON-envelope (`docs/1_4_auth_middleware/08-api-contract.md:91-97`).

#### 1.2 Auth middleware

Файл — `internal/auth/transport/http/middleware/auth.go:17` (`RequireAuth(issuer, clock)`).

- Источник токена — заголовок `Authorization: Bearer <jwt>` (`auth.go:13-22`). Любое отклонение → 401 + `AUTH-010`.
- Mapping ошибок верификации (`auth.go:40`): `domain.ErrAccessTokenExpired` → `401 AUTH-011 access token expired`; всё остальное (битая подпись, не тот алгоритм, `sub` не UUID, и т.п.) → `401 AUTH-010 access token invalid`.
- Context: `middleware.WithUserID(ctx, uid)` (`auth.go:34`, `contextkeys.go:12`); чтение — `middleware.UserIDFromContext(ctx)` (`contextkeys.go:17`). Импортируется как `authmw` (`cmd/server/main.go:27`).

**Что это значит для фронта**: на `401 AUTH-011` — попытка `POST /auth/refresh`, на ответ `200 OK` — перезапуск запроса с новым `accessToken`. На `401 AUTH-007/008/009/010` — полный logout (refresh либо невалиден, либо access ни при каком refresh не починится).

#### 1.3 Cross-cutting middleware

Подключаются глобально в `cmd/server/main.go:162-165` в порядке: `RequestID` → `Recover` → `Logger` → `CORS`.

**Единый формат ошибок** — `pkg/httpx/jsonerror.go:18` (`WriteJSONError(w, status, code, message)`):
```json
{ "error": { "code": "AUTH-XXX", "message": "human-readable description" } }
```
Поле `details` отсутствует — только `code` и `message`. Префиксы кодов: `AUTH-001..AUTH-012`, `ROOM-001..ROOM-009`, `CHANNEL-001..CHANNEL-007`, `CHAT-001..CHAT-007`, `INTERNAL`.

**CORS** — `pkg/httpx/middleware/cors.go:13`.
- Источник whitelist — env `CORS_ALLOWED_ORIGINS` (CSV), default `http://localhost:5173`.
- Сравнение строгое (wildcard `*` не поддерживается), нужны схема + host, без path/query/fragment (`config/config.go:263-281`).
- На cross-origin при match: `Access-Control-Allow-Origin: <origin>` (точное эхо, не `*`), `Vary: Origin`. **`Access-Control-Allow-Credentials` НЕ выставляется** — `allowCredentials=false` (`main.go:165`).
- Preflight `OPTIONS`: `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`; `Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID`; `Access-Control-Max-Age: 600` (`cors.go:5-9`).

**RequestID** — `pkg/httpx/middleware/requestid.go:16`.
- Заголовок `X-Request-ID`. Клиент может прислать своё значение (ASCII-printable, ≤128 симв.). Иначе — `uuid.New().String()` (`main.go:153`).
- Результат всегда возвращается в response-header `X-Request-ID` и кладётся в контекст.

**Logger** — `pkg/httpx/middleware/logger.go:30`. Пишет `slog.Info` `"http"` с `method`, `path`, `status`, `duration`, `request_id`, `remote_ip`, `user_agent`, `user_id`. В query-параметре `token` значение заменяется на `REDACTED` (защита WS-токена).

**Recover** — `pkg/httpx/middleware/recover.go:14`. На panic: лог + 500 `INTERNAL`.

#### 1.4 JWT TTL

- `JWT_ACCESS_TTL` — env, default `15m` (`config/config.go:15, :146-178`), max `1h`.
- `JWT_REFRESH_TTL` — env, default `720h` (30 дней) (`config/config.go:16, :180-227`), max `90 дней`; должен быть строго > access TTL.
- В ответах login/refresh клиент получает абсолютные `accessExpiresAt`/`refreshExpiresAt` — фронт сам решает, когда вызвать refresh.

---

### 2. Room + Channel HTTP API

Базовый префикс `/api/v1`. На всю группу `/rooms` и `/rooms/{roomID}/channels` навешан `authmw.RequireAuth`. `userID` из контекста — `authmw.UserIDFromContext`. Envelope ошибок и Content-Type — общие. JSON-поля — **camelCase**.

Маршруты: `internal/room/transport/http/routes.go:23-35`, `internal/channel/transport/http/routes.go:19-27`.

#### 2.1 Room endpoints

| Метод | Путь | Хендлер | Authz |
|---|---|---|---|
| `POST` | `/api/v1/rooms` | `create_room_handler.go:19` | Bearer |
| `GET` | `/api/v1/rooms` | `list_rooms_handler.go:19` | Bearer |
| `GET` | `/api/v1/rooms/{roomID}` | `get_room_handler.go:23` | Bearer + member |
| `DELETE` | `/api/v1/rooms/{roomID}` | `delete_room_handler.go:23` | Bearer + **owner** |
| `GET` | `/api/v1/rooms/{roomID}/members` | `list_members_handler.go:23` | Bearer + member |
| `POST` | `/api/v1/rooms/{roomID}/invite` | `regenerate_invite_handler.go:23` | Bearer + admin/owner |
| `POST` | `/api/v1/rooms/join/{code}` | `join_by_code_handler.go:21` | Bearer (без существующего членства) |

**POST /rooms** — `{"name":"1..64 рун, без Cc/Cf"}` → `201 {"id","ownerId","name","createdAt"}`. Создатель автоматически получает роль `owner` (`usecase/create_room.go:49-53`). Errors: `400 ROOM-001 invalid room name`, `400 ROOM-009 invalid body`.

**GET /rooms** — `200 {"items":[{"id","ownerId","name","createdAt","role"}]}` — список комнат, где actor — member любой роли. Поле `role`: `"owner"|"admin"|"member"`. Пусто — `{"items":[]}` (slice, не null).

**GET /rooms/{roomID}** — `200 {"id","ownerId","name","createdAt"}`. Errors: `404 ROOM-002` (UUID невалиден или комнаты нет), `403 ROOM-003 not a member`.

**DELETE /rooms/{roomID}** — `204 No Content`. Errors: `404 ROOM-002`, `403 ROOM-003`, **`403 ROOM-005 only owner can delete room`** (admin/member). Маппинг сделан в `mapDeleteRoomError` (`error_mapper.go:53`).

**GET /rooms/{roomID}/members** — `200 {"items":[{"userId","role","joinedAt"}]}`. **Важно**: в DTO нет `username`/`email`/`displayName` (ADR D-09, `docs/2_1_rooms_and_channels/08-api-contract.md:207`). DTO — `internal/room/transport/http/dto.go:35-39`.

**POST /rooms/{roomID}/invite** — атомарно отзывает все активные инвайты и создаёт новый. До 3 ретраев на коллизию (`maxInviteCodeRetries=3`). Отдельного GET для текущего активного кода **нет**: ответ этого POST — единственный способ получить код. Success: `200 {"code","createdBy","createdAt"}`. `code` — 8 символов Crockford base32 (uppercase, алфавит `0123456789ABCDEFGHJKMNPQRSTVWXYZ`). Errors: `403 ROOM-004 insufficient role: admin or owner required`.

**POST /rooms/join/{code}** — `code` нормализуется в upper-case в `domain.NewInviteCode` (`invite_code.go:12`). Новый член получает роль `member` (`usecase/join_by_code.go:66`). Best-effort публикует `member.joined` в hub. Success: `200 {"id","ownerId","name","createdAt"}` (комната, в которую вступили). Errors: `400 ROOM-008 invalid invite code`, `404 ROOM-007 invite not found or revoked`, `409 ROOM-006 already a member`.

#### 2.2 Channel endpoints

| Метод | Путь | Хендлер | Authz |
|---|---|---|---|
| `POST` | `/api/v1/rooms/{roomID}/channels` | `create_channel_handler.go:23` | Bearer + admin/owner |
| `GET` | `/api/v1/rooms/{roomID}/channels` | `list_channels_handler.go:23` | Bearer + member |
| `DELETE` | `/api/v1/rooms/{roomID}/channels/{channelID}` | `delete_channel_handler.go:23` | Bearer + admin/owner |

**POST** — `{"name":"1..64 рун","kind":"text|voice"}` (kind строго lowercase, `domain/channel_kind.go:33`) → `201 {"id","roomId","name","kind","createdAt"}`. Errors: `400 CHANNEL-001 invalid name`, `400 CHANNEL-002 invalid kind`, `400 CHANNEL-005 invalid body`, `403 CHANNEL-006 not a member`, `403 CHANNEL-007 admin or owner required`, `404 CHANNEL-003`, `409 CHANNEL-004 name already taken` (case-sensitive, в пределах комнаты).

**GET** — `200 {"items":[{"id","roomId","name","kind","createdAt"}]}`.

**DELETE** — `204 No Content`. Семантика: удаляется только если `channelID` принадлежит `roomID` (`DeleteInRoom`).

`voice`-каналы поддерживаются на уровне CRUD (можно создавать), но фактического сигналинга в текущем коде нет — это другая фаза. Для PR-3.5 фронт ставит voice-кнопку как заглушку.

#### 2.3 Роли и матрица прав

`domain.Role` — uint8 (`internal/room/domain/role.go:3`):
- `RoleMember` → `"member"`
- `RoleAdmin` → `"admin"`
- `RoleOwner` → `"owner"`

Из `domain/membership.go:65-83`:
- `CanReadRoom/Members/Channels` — любая валидная роль.
- `CanCreateChannel/DeleteChannel/GenerateInvite` — admin или owner.
- `CanDeleteRoom` — только owner.

---

### 3. Chat REST + WebSocket-протокол

#### 3.1 REST: история сообщений

**`GET /api/v1/channels/{channelID}/messages`** — `internal/chat/transport/http/list_messages_handler.go:26`. Регистрация — `routes.go:19-24`. Auth — `RequireAuth`.

Параметры:
- Path: `channelID` (UUID).
- Query `before` — UUID последнего сообщения предыдущей страницы (опционально). Курсор — **UUID существующего message id**, не timestamp. Если опущен — отдаются самые свежие.
- Query `limit` — int. Default `50`, диапазон `1..100` (`usecase/list_messages.go:14-17`). Вне диапазона → `400 CHAT-006`.

Семантика курсора: сообщения возвращаются от новых к старым (`createdAt DESC, id DESC`). `before` означает «строго ДО этого id». `nextBefore` выставляется только если страница полная (`len(items) == limit`); иначе `null`.

Ответ — `200 OK`, `listMessagesResponse` (`dto.go:21-24`):
```json
{
  "items": [
    {
      "id":        "9c5b9a6a-3d27-4f5e-9a1b-2b9b1f9b3e88",
      "channelId": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff",
      "authorId":  "5fa8b3c1-22d4-4e9f-9012-345678901234",
      "text":      "Привет!",
      "createdAt": "2026-05-12T17:32:08.123456789Z"
    }
  ],
  "nextBefore": "9c5b9a6a-3d27-4f5e-9a1b-2b9b1f9b3e88"
}
```

REST-поля **camelCase**: `id`, `channelId`, `authorId`, `text`, `createdAt`. `nextBefore: string | null`. Поля `editedAt` нет — редактирование не поддержано.

Лимит текста сообщения (для понимания): `1..4000` рун после `TrimSpace`, без control/Cf кроме `\n\r\t` (`domain/message_text.go:11-39`).

Коды ошибок (`error_mapper.go:20-41`):

| HTTP | Code | Когда |
|---|---|---|
| `400` | `CHAT-005` | `channelID` или `before` — невалидный UUID |
| `400` | `CHAT-006` | `limit` вне `1..100` или нечисловой |
| `401` | `AUTH-010/011` | middleware |
| `403` | `CHAT-004` | не член комнаты |
| `404` | `CHAT-002` | канал не существует |
| `500` | `INTERNAL` | прочее |

#### 3.2 WebSocket подключение

URL — **`GET /api/v1/ws?token=<jwt>`** (`internal/chat/transport/ws/routes.go:23-25`, монтирование `cmd/server/main.go:200-208`).

- Токен — из query (`handler.go:36`). Header `Authorization` не используется (браузерный WebSocket API не поддерживает custom headers).
- Проверка ДО апгрейда: `TokenIssuer.VerifyAccess(token, clock.Now())` (`handler.go:41`).
  - Нет токена / пустой / невалидный → ответ обычным HTTP `401 AUTH-010`.
  - Истёкший → `401 AUTH-011`.
- Origin check: `pkg/websocket.Upgrade` использует `OriginPatterns = stripScheme(cfg.CORSAllowedOrigins())` (`upgrade.go:18-25`, `ws_adapters.go:69-79`, `main.go:207`). `InsecureSkipVerify: false`. При несовпадении origin — `403` от библиотеки (без нашего JSON-envelope).
- `SetReadLimit = 64 KB` (`upgrade.go:9, 26-31`).

После апгрейда:
1. Генерируется `connID = uuid.New()`, привязывается к `userID` (`pkg/websocket/conn.go:25-27`).
2. `Hub.Register(conn)` — добавляет в реестр, возвращает `cleanup` (`hub.go:36-56`).
3. **Автоподписка на все `room:<id>`** пользователя: `MembershipForRooms.ListRoomIDsByUser(userID)` → для каждого `Hub.Subscribe(conn, RoomTopic(roomID))` (`handler.go:53, 72-74`).
4. Запуск `readLoop` (`handler.go:80-95`).

Close-codes:
- Нормальное закрытие / read error → `1000 NormalClosure` "bye" (`hub.go:53`).
- Graceful shutdown сервера → `1001 GoingAway` всем conn'ам (`hub.go:153`, очерёдность в `main.go:244-248`: сначала hub, потом HTTP-сервер).
- Прикладные ошибки доставляются JSON-фреймом `type:"error"` (см. §3.4), соединение **не закрывается**.

#### 3.3 WS: клиент → сервер

Общий конверт — плоский JSON, поля **snake_case** (`internal/chat/transport/ws/event_dto.go:21-25`):
```json
{ "type": "...", "channel_id": "...", "text": "..." }
```

**`subscribe`** — подписаться на топик канала:
```json
{ "type": "subscribe", "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff" }
```
Хендлер — `handler.go:98-121`. Проверка: парс UUID → `MembershipForChat.Require(...RoleAnyMember)` → `Hub.Subscribe(conn, ChannelTopic(channelID))`. Ответ только подписавшемуся: `{"type":"subscribed","data":{"channel_id":"..."}}`. Ошибки: `CHAT-002/004/005`.

**`message.send`** — отправить сообщение:
```json
{
  "type": "message.send",
  "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff",
  "text": "Привет!"
}
```
Хендлер — `handler.go:124-145`. Делегирует `SendMessage.Execute(actorID, channelID, text)`. После успеха:
- Отправителю — ack `message.sent` (см. §3.4).
- Всем подписчикам канала — broadcast `message.new` через `Hub.Publish(ChannelTopic, "message.new", payload)` (`usecase/send_message.go:101-104`). Broadcast best-effort.

Ошибки `message.send`: `CHAT-001 invalid text`, `CHAT-002 channel not found`, `CHAT-003 channel is not text`, `CHAT-004 access denied: not a room member`, `CHAT-005 invalid uuid`, `INTERNAL`.

**Неизвестный `type`** → error-фрейм с `CHAT-007 unsupported event type`, соединение не закрывается (`handler.go:91-93`).

Отсутствуют (НЕ реализованы): `unsubscribe`, `ping`/`pong` (используется встроенный механизм `coder/websocket`), `message.edit`, `message.delete`, typing, presence, protocol-version handshake.

#### 3.4 WS: сервер → клиент

Общий конверт outbound — `{"type":"<event>","data":{...}}` (формируется в `Hub.Publish`, `hub.go:100`). Timestamps — RFC3339Nano UTC, UUID — каноническая строка, поля payload — **snake_case**.

**`message.new`** — broadcast в `channel:<id>` (`usecase/send_message.go:117-125`, `ws_adapters.go:23-25`):
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

**`message.sent`** — ack отправителю (только тот conn, через который пришёл `message.send`):
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
`author_id` в `message.sent` не передаётся — отправитель знает сам себя.

**`subscribed`** — ack подписки:
```json
{ "type": "subscribed", "data": { "channel_id": "..." } }
```

**`error`** — адресная ошибка инициатору (`event_dto.go:28-36`):
```json
{ "type": "error", "data": { "code": "CHAT-004", "message": "access denied: not a room member" } }
```
Коды — `internal/chat/transport/ws/error_mapper.go:11-29`: `CHAT-001`, `CHAT-002`, `CHAT-003`, `CHAT-004`, `CHAT-005`, `CHAT-007`, `INTERNAL`.

**`member.joined`** — broadcast в `room:<id>` (`internal/room/usecase/join_by_code.go` через `cmd/server/ws_adapters.go:38-45`):
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
Тонкость: сам новый член **не получит** своё `member.joined` через свой текущий WS — на момент publish он ещё не подписан на этот `room:`. Подписка появится только при reconnect.

Других событий нет. `member.left`, `presence.*`, `typing.*`, `message.edit/delete` — не реализованы.

#### 3.5 Hub: жизненный цикл подписок

Файлы — `pkg/websocket/hub.go`, `conn.go`, `topic.go`, `upgrade.go`.

`Topic` — VO с двумя видами (`topic.go:17-37`):
- `ChannelTopic(uuid)` → ключ `"channel:<uuid>"`.
- `RoomTopic(uuid)` → ключ `"room:<uuid>"`.

- На WS-connect handler **сам** подписывает conn на `RoomTopic(roomID)` для всех комнат, где user — member.
- На канал подписка — только явной командой `subscribe`. **Членство ≠ подписка.**
- `Hub.Subscribe` идемпотентен (`hub.go:59-69`).
- На disconnect — `cleanup` от `Register` снимает все подписки сразу (`hub.go:42-55`).
- `Hub.Publish` сериализует payload один раз, под RLock копирует подписчиков и пишет всем последовательно с per-write timeout 5 s (`conn.go:13, 58-64`). Ошибки доставки — WARN, не прерывает рассылку.
- `unsubscribe` не реализован: единственный способ снять подписку — закрыть WS.
- При shutdown сервера `hub.Shutdown` параллельно закрывает все conn'ы (1001 "going away"), потом `srv.Shutdown` (`main.go:244-248`).

---

### 4. Инфраструктура, конфиг, состояние `web/`

#### 4.1 Composition root

Точка входа — `cmd/server/main.go`. `main()` (`main.go:49-74`) загружает конфиг и зовёт `run()`. `run()` (`main.go:76-252`):

1. `pgxpool.New(ctx, cfg.DatabaseURL())` (`main.go:79`).
2. `db.New(pool)` — sqlc queries.
3. Auth-репо, bcrypt-хешер (`bcryptProductionCost = 10`, `main.go:45`), JWT-issuer.
4. `realClock`, `realUUID`, `cryptoRand` (`cmd/server/runtime.go:10-26`).
5. Dummy-хеш для timing-safety (`main.go:96-103`).
6. Auth use-cases (`main.go:105-113`).
7. Room composition (`main.go:115-126`): репо, `Base32CodeGen(cryptorand.Reader)`, все use-cases.
8. Channel composition (`main.go:128-134`): репо, `MembershipQueryAdapter`, use-cases.
9. `hub := pws.NewHub(logger)` (`main.go:137`).
10. Chat composition (`main.go:139-146`) + `JoinByCode` после hub (`main.go:149`).
11. Chi-router + middleware (`main.go:151-166`): `RequestID` → `Recover` → `Logger` → `CORS`.
12. Маршруты под `/api/v1` (`main.go:167-209`): `GET /health`, `RegisterRoutes` для auth/room/channel/chat + `RegisterWSRoute` для WS.
13. `http.Server` с `ReadHeaderTimeout: 10s` (`main.go:213-216`). Других таймаутов нет.
14. Graceful shutdown по `SIGINT`/`SIGTERM`, `shutdownTimeout = 5s` (`main.go:218-251`). Порядок: `hub.Shutdown` → `srv.Shutdown`.

Адаптеры для WS — `cmd/server/ws_adapters.go`:
- `hubChatBroadcaster` — `PublishToChannel/PublishToRoom`.
- `hubRoomEventsPublisher` — `PublishMemberJoined` (`member.joined`).
- `roomIDsAdapter` — `ListRoomIDsByUser` для авто-подписки.
- `stripScheme` — из CORS origins делает host'ы для `OriginPatterns`.

`GET /api/v1/health` → `{"status":"ok"}` (`cmd/server/health.go:7-11`).

#### 4.2 Конфигурация

Все env читаются через `Lookuper` интерфейс (`config/config.go:61-69`). Полная таблица:

| Имя | Тип | Дефолт | Что делает | Код ошибки |
|---|---|---|---|---|
| `JWT_SECRET` | string | — (required) | HMAC-ключ JWT | `CONFIG-001` |
| `DATABASE_URL` | string | — (required) | DSN PostgreSQL | `CONFIG-002` |
| `SERVER_PORT` | int | `8080` | Порт HTTP | `CONFIG-003` (1..65535) |
| `JWT_ACCESS_TTL` | duration | `15m` | TTL access | `CONFIG-004` (≤1h) |
| `JWT_REFRESH_TTL` | duration | `720h` (30 дней) | TTL refresh | `CONFIG-005` (>access, ≤90 дней) |
| `CORS_ALLOWED_ORIGINS` | CSV строк | `http://localhost:5173` | Whitelist origin (и WS) | `CONFIG-006` |

`.env.example` определяет 4 переменных: `DATABASE_URL`, `JWT_SECRET`, `SERVER_PORT`, `CORS_ALLOWED_ORIGINS`. `JWT_ACCESS_TTL` и `JWT_REFRESH_TTL` в `.env.example` отсутствуют — берутся дефолты.

#### 4.3 Docker

`docker-compose.yml` (21 строка) поднимает **только Postgres**:
- `postgres:16-alpine`, container `rupor-postgres`.
- env `POSTGRES_USER/PASSWORD/DB` — default `rupor`.
- порт `${POSTGRES_PORT:-5432}:5432`.
- volume `rupor-pg-data`.
- healthcheck `pg_isready`, 5s/3s/5retries.

**Не контейнеризованы**: Go-сервер (нет `Dockerfile` в репозитории; запускается через `make run` → `go run ./cmd/server`), `web/` (нет папки). Сетей кастомных нет.

#### 4.4 `web/` — состояние

Папки `web/` **не существует** (`ls web/` → `No such file or directory`). В корне нет `package.json`, `vite.config.*`, `frontend/`. Единственные упоминания фронта:
- `.env.example:10-11` — `CORS_ALLOWED_ORIGINS=http://localhost:5173`.
- `CLAUDE.md` — описание стека `React + Vite + Zustand`.

#### 4.5 Состояние доменов

| Домен | Статус | Где |
|---|---|---|
| `auth` | готов | `internal/auth/{domain,usecase,transport,repository}/...` |
| `user` | пустая заглушка (`.gitkeep`) | `internal/user/...` |
| `room` | готов | `internal/room/...` |
| `channel` | готов | `internal/channel/...` |
| `chat` | готов (REST + WS) | `internal/chat/...` |
| `voice` | пустая заглушка (`.gitkeep`) | `internal/voice/...` |

#### 4.6 Документы и tasks

`docs/`:
- `1_1_project-structure`, `1_2_db_schema`, `1_3_auth_domen`, `1_4_auth_middleware`, `2_1_rooms_and_channels`, `3_1_realtime_chat`.
- **Папки про PR-3.5 / frontend нет.**

`tasks/`:
- `Functional and non-functional requirements.md` — **пустой** (0 байт).
- `after_mvp_plan.md` — post-MVP роудмап (observability, видеозвонки, редактирование сообщений, личные сообщения, настройки аккаунта, трансляции экрана, запись звонков, API ботов, нативные приложения). К PR-3.5 прямо не относится.
- `poject_structure.txt` — 2 строки, общая заметка.

`prompts/` — про Go-бэкенд (Clean Architecture, Domain Model, Tests Style, Go style). Правил для фронта нет.

`manual_qa/` — эталонные `.http`-сценарии и в случае 3_1 готовый `ws_test.html`:
- `1_3_auth/01_register.http..04_me.http`
- `1_4_auth_middleware/01_cors_preflight.http`, `02_request_id.http`, `03_require_auth.http`
- `2_1_rooms_and_channels/00_flow.http..07_delete_room.http`
- `3_1_realtime_chat/00_flow.http`, `01_messages_rest.http`, `02_ws_browser_test.md`, `ws_test.html`

---

## Архитектурные наблюдения

- **Стиль JSON**: REST — camelCase; WS-payload — snake_case (плоский для inbound, в обёртке `{type,data}` для outbound).
- **Единый envelope ошибок** `{"error":{"code","message"}}` — без `details`. Коды доменно-префиксованы: `AUTH-NNN`, `ROOM-NNN`, `CHANNEL-NNN`, `CHAT-NNN`, `INTERNAL`.
- **Курсорная пагинация** — UUID-курсор (`before=<message-id>`), `nextBefore` ставится только когда страница полная.
- **WS-аутентификация — токен в query** (`?token=<jwt>`), потому что браузер не умеет ставить `Authorization` на WebSocket. Логгер маскирует значение как `REDACTED`.
- **Авто-подписка room** при WS-connect, **явная подписка** на канал. Подписка ≠ членство.
- **Reconnect-требование**: вступление в новую комнату через `POST /rooms/join/{code}` НЕ добавляет подписку на её `room:` к уже открытому WS — нужен reconnect.
- **Авторские имена в чате не приходят с бэка** — ни в `messageResponse` (`authorId`), ни в `message.new` (`author_id`). `user`-домен пуст; в DTO `member` тоже только `userId`. Это фактическое состояние, не предложение менять.
- **Logout-эндпоинта нет** — клиентский logout = локальная очистка токенов. Серверная инвалидация refresh-токенов умеет работать только через ротацию (старый отзывается при refresh).
- **CORS `allowCredentials=false`** — нельзя использовать cookie-based auth; access/refresh нужно держать на клиенте (localStorage / in-memory + Authorization header).
- **Voice — заглушка** на CRUD-уровне: каналы `kind:"voice"` создаются и видны, но никакого сигналинга в бэке пока нет.

## Открытые вопросы для фазы дизайна

Зафиксированы как факты текущего состояния, не как предложения:
1. Авторские имена и имена участников: ни `users.username` в API чата, ни `members.username` в API комнат нет. Решение, как отображать «Кто написал?», — за фронтом (по `author_id` фоллбэк к UUID или дизайн пред-loading map'а).
2. Получение текущего активного инвайт-кода: отдельного GET нет, единственный путь — `POST /rooms/{roomID}/invite` (с побочным эффектом ротации).
3. Reconnect WS после `join`: нет уведомления о появлении новой комнаты в текущем WS.
4. WS не отдаёт `member.left`, `member.role.changed`, `room.deleted` и т.п. — UI для «другой пользователь меня кикнул из комнаты» сейчас опирается только на REST-ответы (`403/404`).
5. Подтверждения доставки сообщений отправителю — `message.sent` приходит только на тот conn, через который отправили. Multi-tab синхронизация: у того же пользователя в другой вкладке `message.new` придёт только если та вкладка подписана на канал.

## Ссылки на код (сводка)

**Composition root**
- `cmd/server/main.go:49-74` — `main`
- `cmd/server/main.go:76-252` — `run`
- `cmd/server/main.go:151-166` — chi + middleware chain
- `cmd/server/main.go:167-209` — регистрация всех маршрутов `/api/v1`
- `cmd/server/main.go:200-208` — WS-роут
- `cmd/server/main.go:244-248` — shutdown order
- `cmd/server/runtime.go:10-26`, `cmd/server/health.go:7-11`, `cmd/server/ws_adapters.go:17-79`

**Auth**
- `internal/auth/transport/http/routes.go:19-31`
- `internal/auth/transport/http/dto.go:9-43`
- `internal/auth/transport/http/error_mapper.go:17-54`
- `internal/auth/transport/http/middleware/auth.go:13-45`
- `internal/auth/transport/http/middleware/contextkeys.go:9-24`
- `internal/auth/usecase/ports.go:12-44`
- `internal/auth/usecase/login_user.go:64-130`
- `internal/auth/usecase/refresh_access.go:47-117`
- `internal/auth/domain/errors.go:7-30`

**Room**
- `internal/room/transport/http/routes.go:23-35`
- `internal/room/transport/http/dto.go:12-92`
- `internal/room/transport/http/error_mapper.go:18-66`
- `internal/room/transport/http/{create,list,get,delete,list_members,regenerate_invite,join_by_code}_handler.go`
- `internal/room/usecase/{create,get,list,delete,list_members,regenerate_invite,join_by_code}.go`
- `internal/room/domain/role.go:3-46`, `membership.go:5-97`, `invite_code.go:5-30`, `room_name.go:9-35`, `errors.go:6-25`

**Channel**
- `internal/channel/transport/http/routes.go:19-27`
- `internal/channel/transport/http/dto.go:11-36`
- `internal/channel/transport/http/error_mapper.go:18-55`
- `internal/channel/usecase/{create,list,delete}.go`
- `internal/channel/domain/{channel,channel_kind,channel_name,errors}.go`

**Chat REST**
- `internal/chat/transport/http/routes.go:19-24`
- `internal/chat/transport/http/list_messages_handler.go:26-82`
- `internal/chat/transport/http/dto.go:11-35`
- `internal/chat/transport/http/error_mapper.go:20-56`
- `internal/chat/usecase/list_messages.go:13-86`
- `internal/chat/domain/{message,message_text,errors}.go`

**Chat WS**
- `internal/chat/transport/ws/routes.go:23-25`
- `internal/chat/transport/ws/handler.go:31-145`
- `internal/chat/transport/ws/event_dto.go:7-56`
- `internal/chat/transport/ws/error_mapper.go:11-29`
- `internal/chat/usecase/send_message.go:56-125`
- `pkg/websocket/hub.go:12-169`, `conn.go:13-64`, `topic.go:17-37`, `upgrade.go:9-32`

**Cross-cutting**
- `pkg/httpx/jsonerror.go:8-22` (envelope)
- `pkg/httpx/contextkeys.go:5-16`
- `pkg/httpx/middleware/{cors,requestid,recover,logger,responsewriter}.go`

**Config**
- `config/config.go:13-21` — дефолты/максимумы
- `config/config.go:39-69` — `Config`, `Lookuper`
- `config/config.go:71-281` — `Load` + валидаторы (port, TTL, CORS)
- `.env.example:1-11`

**Эталонные сценарии (читать для точных тел)**
- `docs/1_3_auth_domen/08-api-contract.md`
- `docs/1_4_auth_middleware/08-api-contract.md`
- `docs/2_1_rooms_and_channels/08-api-contract.md`
- `docs/3_1_realtime_chat/{02-behavior,05-events,08-api-contract}.md`
- `manual_qa/1_3_auth/`, `manual_qa/1_4_auth_middleware/`, `manual_qa/2_1_rooms_and_channels/`, `manual_qa/3_1_realtime_chat/`
