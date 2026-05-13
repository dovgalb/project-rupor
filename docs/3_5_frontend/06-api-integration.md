---
parent: ./README.md
feature: 3_5-frontend
---

# 3.5 Frontend — API Integration

## Общий формат

### REST

- Базовый URL: `/api/v1` (dev: проксируется Vite на `http://localhost:8080`; prod: nginx).
- JSON, поля **camelCase**.
- Заголовки: `Content-Type: application/json` (при body), `Authorization: Bearer <accessToken>` (для приватных).
- Без `credentials: "include"` — CORS `allowCredentials=false`.

### WebSocket

- URL: `/api/v1/ws?token=<jwt>` (актуальный access).
- Текстовые JSON-фреймы.
- Outbound: плоский JSON, snake_case (`{ type, channel_id, text }`).
- Inbound: обёртка `{ type, data }`, snake_case в `data`.

### Общие типы (`web/src/shared/api/errors.ts`)

```ts
export type ErrorEnvelope = {
  error: { code: string; message: string };
};

export type ApiOk<T>  = { ok: true;  data: T };
export type ApiErr    = { ok: false; status: number; error: ErrorEnvelope["error"] };
export type ApiResult<T> = ApiOk<T> | ApiErr;

// Литералы кодов, которые UI визуализирует
export type AuthErrorCode =
  | "AUTH-001" | "AUTH-002" | "AUTH-003"
  | "AUTH-004" | "AUTH-005" | "AUTH-006"
  | "AUTH-007" | "AUTH-008" | "AUTH-009"
  | "AUTH-010" | "AUTH-011" | "AUTH-012"
  | "INTERNAL" | "NETWORK" | "CLIENT";

export type RoomErrorCode =
  | "ROOM-001" | "ROOM-002" | "ROOM-003" | "ROOM-004" | "ROOM-005"
  | "ROOM-006" | "ROOM-007" | "ROOM-008" | "ROOM-009";

export type ChannelErrorCode =
  | "CHANNEL-001" | "CHANNEL-002" | "CHANNEL-003" | "CHANNEL-004"
  | "CHANNEL-005" | "CHANNEL-006" | "CHANNEL-007";

export type ChatErrorCode =
  | "CHAT-001" | "CHAT-002" | "CHAT-003" | "CHAT-004"
  | "CHAT-005" | "CHAT-006" | "CHAT-007";
```

`NETWORK` и `CLIENT` — псевдо-коды, которые ставит fetch-обёртка при сетевой ошибке / невалидном JSON.

---

## REST: Auth (источник: `docs/1_3_auth_domen/08-api-contract.md`)

### POST /api/v1/auth/register

```ts
export type RegisterRequest = {
  email: string;     // RFC-like, ≤254
  username: string;  // ASCII alnum / _ / -, 3..32
  password: string;  // 8..72 байт
};

export type UserResponse = {
  id: string;        // uuid
  email: string;
  username: string;
  createdAt: string; // RFC3339
};

export function register(req: RegisterRequest): Promise<ApiResult<UserResponse>>;
```

**Success:** `201 Created` → `UserResponse`. **НЕ** возвращает токены — после register идёт автоматический login.

**Errors (UI mapping):**
| HTTP | Code | Реакция |
|---|---|---|
| 400 | `AUTH-001` invalid email | inline field error на `email` |
| 400 | `AUTH-002` invalid username | inline field error на `username` |
| 400 | `AUTH-003` invalid password | inline field error на `password` |
| 400 | `AUTH-012` malformed body | bug-баннер (не должно случаться) |
| 409 | `AUTH-004` email already taken | inline field error на `email` |
| 409 | `AUTH-005` username already taken | inline field error на `username` |
| 500 | `INTERNAL` | toast «Внутренняя ошибка, повторите» |

### POST /api/v1/auth/login

```ts
export type LoginRequest = {
  email: string;
  password: string;
};

export type TokensResponse = {
  accessToken: string;     // JWT HS256
  refreshToken: string;    // base64url 43 char
  accessExpiresAt: string; // RFC3339 UTC
  refreshExpiresAt: string;
};

export function login(req: LoginRequest): Promise<ApiResult<TokensResponse>>;
```

**Success:** `200 OK` → `TokensResponse`. Action сохраняет токены через `tokenStorage.set(...)`.

**Errors:**
| HTTP | Code | Реакция |
|---|---|---|
| 400 | `AUTH-012` | bug-баннер |
| 401 | `AUTH-006` invalid credentials | form-level error «Неверный email или пароль» |
| 500 | `INTERNAL` | toast |

### POST /api/v1/auth/refresh

```ts
export type RefreshRequest = { refreshToken: string };

export function refresh(req: RefreshRequest): Promise<ApiResult<TokensResponse>>;
```

**Success:** `200 OK` → `TokensResponse` (новая пара, старый refresh отозван — ротация).

**Errors:**
| HTTP | Code | Реакция |
|---|---|---|
| 400 | `AUTH-012` | logout (refresh битый) |
| 401 | `AUTH-007` not found | logout |
| 401 | `AUTH-008` revoked | logout |
| 401 | `AUTH-009` expired | logout |
| 500 | `INTERNAL` | toast + logout (не возвращаемся в loop) |

### GET /api/v1/auth/me

```ts
export type CurrentUser = UserResponse;  // ViewModel = DTO (без переименований)

export function me(): Promise<ApiResult<CurrentUser>>;
```

**Success:** `200 OK` → `CurrentUser`. Заголовок `Authorization: Bearer` обязателен.

**Errors:** делегируются middleware:
| HTTP | Code | Реакция |
|---|---|---|
| 401 | `AUTH-010` access invalid | logout |
| 401 | `AUTH-011` access expired | auto-refresh + retry (single-flight) |
| 500 | `INTERNAL` | toast |

### POST /api/v1/auth/logout — ОТСУТСТВУЕТ

Logout-эндпоинта НЕТ. Frontend-flow:

```ts
function logout() {
  tokenStorage.clear();
  useAuthStore.getState().clear();
  useRoomsStore.getState().clear();
  useChannelsStore.getState().clear();
  useChatStore.getState().clear();
  wsClient.disconnect();
  navigate("/login", { replace: true });
}
```

---

## REST: Rooms (источник: `docs/2_1_rooms_and_channels/08-api-contract.md`)

### POST /api/v1/rooms — создать

```ts
export type CreateRoomRequest = { name: string };  // 1..64 руны
export type RoomDto = {
  id: string;
  ownerId: string;
  name: string;
  createdAt: string;
};
export function createRoom(req: CreateRoomRequest): Promise<ApiResult<RoomDto>>;
```

`201 Created`. Errors: 400 `ROOM-001 invalid name` → inline, 400 `ROOM-009 invalid body` → bug-баннер.

### GET /api/v1/rooms — список своих

```ts
export type RoomWithRoleDto = RoomDto & { role: "owner" | "admin" | "member" };
export type ListRoomsResponse = { items: RoomWithRoleDto[] };
export function listRooms(): Promise<ApiResult<ListRoomsResponse>>;
```

`200 OK`. Пустой массив — `{"items": []}`.

### GET /api/v1/rooms/{roomId} — одна комната

```ts
export function getRoom(roomId: string): Promise<ApiResult<RoomDto>>;
```

Errors: 404 `ROOM-002` → redirect `/rooms` + toast; 403 `ROOM-003` → удалить комнату из стора + redirect.

### DELETE /api/v1/rooms/{roomId}

```ts
export function deleteRoom(roomId: string): Promise<ApiResult<null>>;
```

`204 No Content` (тело — `null` в ApiResult). Errors: 404 `ROOM-002`, 403 `ROOM-003`, 403 `ROOM-005 only owner`.

### GET /api/v1/rooms/{roomId}/members

```ts
export type MemberDto = {
  userId: string;
  role: "owner" | "admin" | "member";
  joinedAt: string;
};
export type ListMembersResponse = { items: MemberDto[] };
export function listMembers(roomId: string): Promise<ApiResult<ListMembersResponse>>;
```

**Важно:** `username` отсутствует — фронт показывает UUID-placeholder (см. `03-decisions.md` D-11).

### POST /api/v1/rooms/{roomId}/invite — (ре)генерация кода

```ts
export type InviteCodeDto = {
  code: string;       // 8 символов Crockford base32 uppercase
  createdBy: string;  // uuid
  createdAt: string;
};
export function regenerateInvite(roomId: string): Promise<ApiResult<InviteCodeDto>>;
```

`200 OK`. **Эндпоинта GET для текущего активного кода нет** — только этот POST с побочным эффектом ротации.

Errors: 403 `ROOM-004 admin/owner required` → toast.

### POST /api/v1/rooms/join/{code} — войти по коду

```ts
export function joinByCode(code: string): Promise<ApiResult<RoomDto>>;
```

`200 OK` → `RoomDto` комнаты, в которую вступили.

**Side-effect фронта:** после `ok` — `wsClient.reconnect()` (см. D-10), чтобы получить подписку на `room:<id>`.

Errors:
| HTTP | Code | Реакция |
|---|---|---|
| 400 | `ROOM-008 invalid invite code` | inline на форме join |
| 404 | `ROOM-007 invite not found` | inline «Код не найден или отозван» |
| 409 | `ROOM-006 already a member` | toast «Вы уже в этой комнате» + navigate к ней |

---

## REST: Channels (источник: `docs/2_1_rooms_and_channels/08-api-contract.md`)

### POST /api/v1/rooms/{roomId}/channels

```ts
export type ChannelKind = "text" | "voice";
export type CreateChannelRequest = { name: string; kind: ChannelKind };
export type ChannelDto = {
  id: string;
  roomId: string;
  name: string;
  kind: ChannelKind;
  createdAt: string;
};
export function createChannel(roomId: string, req: CreateChannelRequest): Promise<ApiResult<ChannelDto>>;
```

`201 Created`. Errors: 400 `CHANNEL-001/002/005`, 403 `CHANNEL-006/007`, 404 `CHANNEL-003`, 409 `CHANNEL-004`.

### GET /api/v1/rooms/{roomId}/channels

```ts
export type ListChannelsResponse = { items: ChannelDto[] };
export function listChannels(roomId: string): Promise<ApiResult<ListChannelsResponse>>;
```

### DELETE /api/v1/rooms/{roomId}/channels/{channelId}

```ts
export function deleteChannel(roomId: string, channelId: string): Promise<ApiResult<null>>;
```

`204 No Content`.

---

## REST: Chat history (источник: `docs/3_1_realtime_chat/08-api-contract.md`)

### GET /api/v1/channels/{channelId}/messages

```ts
export type MessageDto = {
  id: string;
  channelId: string;
  authorId: string;
  text: string;
  createdAt: string; // RFC3339Nano UTC
};
export type ListMessagesQuery = {
  before?: string;  // UUID последнего на предыдущей странице
  limit?: number;   // 1..100, default 50
};
export type ListMessagesResponse = {
  items: MessageDto[];
  nextBefore: string | null;
};
export function listMessages(channelId: string, query?: ListMessagesQuery): Promise<ApiResult<ListMessagesResponse>>;
```

**Курсор:** UUID существующего сообщения (не timestamp). Сортировка `createdAt DESC, id DESC`. `nextBefore` ставится только если страница полная.

Errors: 400 `CHAT-005` (UUID невалиден), 400 `CHAT-006` (limit вне диапазона), 403 `CHAT-004` (не член), 404 `CHAT-002`.

---

## WebSocket-протокол (источник: `docs/3_1_realtime_chat/05-events.md` + `08-api-contract.md`)

### Подключение

```ts
const url = `${wsBase}/api/v1/ws?token=${encodeURIComponent(accessToken)}`;
const ws = new WebSocket(url);
```

- Токен в query (browser WS API не отдаёт `Authorization`).
- Без `Sec-WebSocket-Protocol`.
- Origin-check на сервере (см. `cmd/server/main.go:207`) — должен совпадать с `CORS_ALLOWED_ORIGINS` (без схемы).

### Outbound фреймы (client → server)

```ts
export type SubscribeFrame = {
  type: "subscribe";
  channel_id: string;
};

export type MessageSendFrame = {
  type: "message.send";
  channel_id: string;
  text: string;     // 1..4000 рун, без control/Cf кроме \n\r\t
};

export type OutgoingFrame = SubscribeFrame | MessageSendFrame;
```

`unsubscribe` НЕ существует. Закрыть подписку — только закрыть WS.

### Inbound фреймы (server → client)

Все имеют форму `{ type: string, data: { ... } }`, snake_case в payload:

```ts
export type SubscribedFrame = {
  type: "subscribed";
  data: { channel_id: string };
};

export type MessageNewFrame = {
  type: "message.new";
  data: {
    id: string;
    channel_id: string;
    author_id: string;
    text: string;
    created_at: string;
  };
};

export type MessageSentFrame = {
  type: "message.sent";
  data: {
    id: string;
    channel_id: string;
    created_at: string;
  };
};

export type MemberJoinedFrame = {
  type: "member.joined";
  data: {
    room_id: string;
    user_id: string;
    joined_at: string;
  };
};

export type ErrorFrame = {
  type: "error";
  data: { code: string; message: string };
};

export type IncomingFrame =
  | SubscribedFrame
  | MessageNewFrame
  | MessageSentFrame
  | MemberJoinedFrame
  | ErrorFrame;
```

Парсер `parseFrame(raw: string): IncomingFrame | null` — JSON.parse + `switch (type)` + валидация полей; неизвестный type → null + warning в logger.

### Подписки

- На `room:<id>` для всех своих комнат — **автоматически** при connect (бэк делает auto-subscribe в handler).
- На `channel:<id>` — **явная команда** `{type:"subscribe", channel_id}` после открытия канала в UI.
- При reconnect — клиент сам ре-отправляет `subscribe` для текущего открытого канала.
- После `joinByCode` — `wsClient.reconnect()` (см. D-10), чтобы бэк сделал auto-subscribe на новый room.

### Маппинг кодов ошибок WS-`error` → UI

| Code | Реакция |
|---|---|
| `CHAT-001 invalid message text` | inline под composer (валидация не пропустила) |
| `CHAT-002 channel not found` | toast «Канал не найден» + закрытие канала |
| `CHAT-003 channel is not text` | toast «Голосовой канал недоступен» (defensive — UI не должен отправлять в voice) |
| `CHAT-004 access denied: not a room member` | toast + reload rooms |
| `CHAT-005 invalid uuid in payload` | bug-баннер (фронт-баг) |
| `CHAT-007 unsupported event type` | log warning, не показываем (фронт-баг) |
| `INTERNAL` | toast «Внутренняя ошибка» |

### Close-codes

| Code | Что значит | Реакция |
|---|---|---|
| `1000 NormalClosure` | сервер закрыл нормально (например, при `wsClient.disconnect()`) | НЕ реконнектим |
| `1001 GoingAway` | сервер shutdown | реконнект с backoff |
| `1006` (abnormal) / `1011` (internal) / network error | разрыв | реконнект с backoff |

---

## Auto-refresh access-токена

### Расположение

`web/src/shared/api/refresh.ts` — модуль с глобальным single-flight promise.

```ts
let currentRefresh: Promise<ApiResult<TokensResponse>> | null = null;

export function refreshAccessOnce(): Promise<ApiResult<TokensResponse>> {
  if (currentRefresh) return currentRefresh;
  const refreshToken = tokenStorage.getRefresh();
  if (!refreshToken) {
    return Promise.resolve({ ok: false, status: 401, error: { code: "AUTH-007", message: "no refresh token" }});
  }
  currentRefresh = authApi
    .refresh({ refreshToken })
    .then((result) => {
      if (result.ok) tokenStorage.set({
        access:           result.data.accessToken,
        refresh:          result.data.refreshToken,
        accessExpiresAt:  result.data.accessExpiresAt,
        refreshExpiresAt: result.data.refreshExpiresAt,
      });
      return result;
    })
    .finally(() => { currentRefresh = null; });
  return currentRefresh;
}
```

### Логика в `fetch.ts`

```ts
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<ApiResult<T>> {
  const isRefreshCall = path === "/auth/refresh";
  const result = await doFetch<T>(path, init);

  if (!result.ok && result.status === 401 && result.error.code === "AUTH-011" && !isRefreshCall) {
    const refreshed = await refreshAccessOnce();
    if (!refreshed.ok) {
      triggerLogout();
      return result;  // original error
    }
    // retry оригинального запроса с новым access
    const retry = await doFetch<T>(path, init);
    if (!retry.ok && retry.status === 401 && retry.error.code === "AUTH-011") {
      // refresh-loop защита: что-то очень странно, выходим
      triggerLogout();
      return retry;
    }
    return retry;
  }

  if (!result.ok && result.status === 401 &&
      ["AUTH-007", "AUTH-008", "AUTH-009", "AUTH-010"].includes(result.error.code)) {
    triggerLogout();
  }

  return result;
}
```

### Что НЕ триггерит refresh

- `AUTH-007/008/009/010` — refresh не поможет, logout.
- Ошибки на сам `/auth/refresh` — logout (refresh-loop защита).
- Любые не-401 ошибки.

---

## WS reconnect

### Стратегия

- Backoff (`03-decisions.md` D-09): `1000ms, 2000ms, 5000ms, 15000ms, 15000ms, ...` (последний уровень бесконечно).
- Триггеры: close-code != 1000, network error.
- После каждой попытки — обновить `wsStatus = "reconnecting"` в `useChatStore`.
- При успешном open — `wsStatus = "open"`, ре-отправить `subscribe` для текущего открытого канала из `useChannelsStore.activeChannelId`.
- При `wsClient.disconnect()` (logout) — `wsStatus = "closed"`, reconnect не делаем.

### Что делаем с pending optimistic messages

При disconnect — `useChatStore` помечает все `pending` сообщения как `failed` после max-attempts или сразу (по решению — сразу `failed` с UI-возможностью retry). После reconnect и subscribed — пользователь может нажать «отправить снова» в UI.

### Token expired во время WS

Если `accessExpiresAt` истёк за время сессии — WS-соединение может прерваться сервером, либо новый `subscribe`/`message.send` придёт с ошибкой. Триггер: при reconnect берётся свежий `accessToken` из storage. Если перед reconnect стор обнаружил, что accessToken истёк — сначала refresh, потом connect.

---

## Token Storage (`shared/api/token-storage.ts`)

```ts
const KEY_ACCESS  = "rupor.access";
const KEY_REFRESH = "rupor.refresh";
const KEY_ACCESS_EXP  = "rupor.accessExpiresAt";
const KEY_REFRESH_EXP = "rupor.refreshExpiresAt";

export type StoredTokens = {
  access: string;
  refresh: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
};

export const tokenStorage = {
  getAccess():  string | null { /* localStorage.getItem(KEY_ACCESS) */ },
  getRefresh(): string | null { /* ... */ },
  getAccessExpiresAt():  string | null { /* ... */ },
  getRefreshExpiresAt(): string | null { /* ... */ },
  set(tokens: StoredTokens): void { /* localStorage.setItem(...) для всех 4 */ },
  clear(): void { /* localStorage.removeItem(...) для всех 4 */ },
};
```

- НЕ логировать значения токенов.
- НЕ отправлять токены в сторонние сервисы.
- Чтение/запись синхронны (localStorage блокирующий).

---

## Логирование

`web/src/shared/lib/logger.ts` — единый logger. В dev: `console.log/warn/error`. В prod: только `error`, через console.

**ЗАПРЕЩЕНО** логировать:
- `Authorization` header
- `?token=` параметр WS-URL (если случайно попадает в URL — log_redact)
- Тела `/auth/login`, `/auth/register`, `/auth/refresh` (содержат `password`, `refreshToken`)
- Значения `tokenStorage.*`

---

## Прокси Vite (dev)

`web/vite.config.ts`:

```ts
export default defineConfig({
  // ...
  server: {
    port: 5173,
    proxy: {
      "/api/v1": {
        target: "http://localhost:8080",
        changeOrigin: true,
        ws: true,  // важно для WebSocket-апгрейда
      },
    },
  },
});
```

В production прокси выполняется nginx-ом (см. `01-architecture.md` C4 L2).
