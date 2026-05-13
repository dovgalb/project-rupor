---
phase: 9
name: Shared WS Client + integration
layer: shared + feature
depends_on: [phase-08]
plan: ./README.md
---

# Phase 9: Shared WS Client + integration

## Цель

Реализовать `web/src/shared/api/ws.ts` (WS-клиент с reconnect, восстановление подписок) и интегрировать его в `useChatStore` (sendMessage с optimistic, onMessageNew/Sent/Subscribed/Error handlers, openChannel + subscribe) и `useRoomsStore` (appendMember handler, joinByCode вызывает reconnect). После фазы — чат полностью работает в реалтайме: send/receive/optimistic/multi-tab dedup/reconnect.

## Контекст

Phase 8 дала Chat-фичу с REST-историей и UI-заглушками composer/banner. Phase 6 — `useRoomsStore.joinByCode` без `wsClient.reconnect()` (TODO). Phase 4 — `logoutFlow` без `wsClient.disconnect()` (TODO).

Все detail из `../06-api-integration.md` секция «WebSocket». State — `../05-state-model.md` `useChatStore` (полная версия). Behavior — `../02-behavior.md` UC-11, UC-12, UC-14 (reconnect), multi-tab диаграмма.

## Файлы для создания

### `web/src/shared/api/ws.ts` (+ `.test.ts`)

**API:**
```ts
export type WsClient = {
  connect: () => void;
  disconnect: () => void;
  reconnect: () => void;
  send: (frame: OutgoingFrame) => void;
  status: WsStatus;
  onFrame: (handler: (frame: IncomingFrame) => void) => () => void;
  onStatus: (handler: (s: WsStatus) => void) => () => void;
};

export type CreateWsClientDeps = {
  getToken: () => string | null;
  baseUrl?: string;  // default — derive из window.location
};

export function createWsClient(deps: CreateWsClientDeps): WsClient;
```

**Логика:**

1. **connect():**
   - Если уже `open` или `connecting` — no-op.
   - Получаем токен через `deps.getToken()`. Если null — статус `closed`, не подключаемся.
   - URL: `${baseUrl}/api/v1/ws?token=${encodeURIComponent(token)}` (логгер маскирует token).
   - `new WebSocket(url)`.
   - `onopen` → status `open`, сбросить `backoffAttempt = 0`, emit `onStatus("open")`.
   - `onmessage` → `parseFrame(event.data)` → если ok, emit `onFrame(frame)`; если nil — `logger.warn`.
   - `onerror` → log.
   - `onclose` → если `code !== 1000` и не было вызова `disconnect()` — запустить reconnect (см. ниже).

2. **disconnect():**
   - Сетим внутренний флаг `intentionalClose = true`.
   - Если socket open — `socket.close(1000, "client logout")`.
   - status `closed`, emit `onStatus("closed")`. Внутренние таймеры reconnect — отменяем.

3. **reconnect():** (D-10: после joinByCode)
   - `disconnect()` (close 1000), затем сразу `connect()` без backoff (intentionalClose=false).

4. **Auto-reconnect (при onclose code != 1000):**
   - status `reconnecting`, emit.
   - Backoff: `BACKOFF = [1000, 2000, 5000, 15000]`; attempt `n` → delay `BACKOFF[Math.min(n, 3)]`.
   - `setTimeout(() => { attempt++; connect(); }, delay)`.
   - Бесконечно (до явного `disconnect()`).

5. **parseFrame:**
   - JSON.parse → проверка структуры по `type`:
     - `"subscribed"`: проверить `data.channel_id`.
     - `"message.new"`: проверить полный набор полей.
     - `"message.sent"`: id+channel_id+created_at.
     - `"member.joined"`: room_id+user_id+joined_at.
     - `"error"`: code+message.
     - other → return null + log warn.

6. **send(frame):**
   - Если status !== "open" — TODO для MVP: throw / log warn (или буферизировать). Минимальный MVP: log warn и игнорируем (`useChatStore.sendMessage` сам обработает через timeout → failed).
   - Иначе `socket.send(JSON.stringify(frame))`.

**Tests (~6):** см. `../04-testing.md` секция `shared/api/ws.ts`.

### `web/src/shared/api/wsClient.singleton.ts`

```ts
import { createWsClient } from "./ws";
import { tokenStorage } from "./token-storage";

export const wsClient = createWsClient({
  getToken: () => tokenStorage.getAccess(),
});
```

Единый экспортируемый instance, который импортируют сторы и `<App />`.

## Файлы для модификации

### `web/src/features/chat/store/index.ts`

Расширить state и actions согласно `../05-state-model.md` (полная версия).

**Добавить:**

```ts
type ChatActions = {
  // существующие из Phase 8:
  loadHistory: ...;
  loadMoreHistory: ...;
  clear: ...;

  // новые в Phase 9:
  openChannel: (channelId: ChannelId) => Promise<void>;
  sendMessage: (channelId: ChannelId, text: string) => Promise<void>;
  retryMessage: (channelId: ChannelId, tempId: string) => Promise<void>;
  onSubscribed: (data: SubscribedFrame["data"]) => void;
  onMessageNew: (data: MessageNewFrame["data"]) => void;
  onMessageSent: (data: MessageSentFrame["data"]) => void;
  onWsError: (data: ErrorFrame["data"]) => void;
  setWsStatus: (status: WsStatus) => void;
};
```

**Logic:**

- `openChannel(channelId)`:
  1. Установить `subscribedChannelId = channelId`.
  2. Если history ещё нет — `loadHistory`.
  3. `wsClient.send({ type: "subscribe", channel_id: channelId })` если `wsStatus === "open"`.

- `sendMessage(channelId, text)`:
  1. Если `text.trim() === ""` — no-op (защита, валидация уже в composer'е).
  2. `tempId = crypto.randomUUID()`.
  3. `currentUserId = useAuthStore.getState().currentUser!.id`.
  4. Push optimistic message с `status: "pending"`, `tempId`, `id: tempId`, `createdAt: new Date().toISOString()`.
  5. `wsClient.send({ type: "message.send", channel_id: channelId, text })`.
  6. Timer 10s: если status всё ещё `pending` — `failMessage(channelId, tempId)`.

- `onMessageSent(data)`:
  - Найти последний `pending` в `messagesByChannel[data.channel_id]` (or первый — порядок мог не совпасть; берём последний по `createdAt`).
  - Обновить: `id = data.id`, `createdAt = data.created_at`, `status = "committed"`, `tempId = undefined`. Resort по `createdAt`.

- `onMessageNew(data)`:
  - Если уже есть message с `id === data.id` — return (dedup).
  - Push: `{ id, channelId, authorId, text, createdAt, status: "committed" }`. Resort по `createdAt`.

- `onSubscribed(data)`:
  - Если `data.channel_id === subscribedChannelId` — отметить готовность (можно отдельный флаг `subscribedReady`, опционально).

- `onWsError(data)`:
  - Map `data.code`:
    - `CHAT-001` → найти последний pending, `failMessage`. Toast inline под composer (через ToastInChannel-механизм — можно отдельный state `composerError`).
    - `CHAT-002/003/004` → toast + navigate `/rooms/:rid` (нужно прокинуть navigate — либо event-bus, либо store держит callback).
    - `INTERNAL` → toast.
    - другие → log.
- `setWsStatus(status)`:
  - Обновить `wsStatus`.
  - При переходе `reconnecting → open` — если `subscribedChannelId` != null, отправить `subscribe`.

- `retryMessage(channelId, tempId)`:
  - Найти message с `tempId`, проверить `status === "failed"`.
  - Установить `status = "pending"`, обновить `createdAt = now()`.
  - `wsClient.send({ type: "message.send", channel_id, text })`.

**Tests (~10 новых):** см. `../04-testing.md` `useChatStore`.

### `web/src/features/rooms/store/index.ts`

Расширить:

```ts
type RoomsActions = {
  // существующие:
  loadRooms; createRoom; deleteRoom; loadMembers; regenerateInvite; joinByCode; selectRoom; clear;

  // новое:
  appendMember: (data: MemberJoinedFrame["data"]) => void;
};
```

- `appendMember`:
  - Если `data.room_id` в `membersByRoom` и нет такого `user_id` — push `{ userId: data.user_id, role: "member", joinedAt: data.joined_at }`.
  - Если нет — игнор (мы не открывали members этой комнаты).

- `joinByCode` — снять TODO(phase-09): после `ok` — `wsClient.reconnect()`.

### `web/src/shared/lib/logoutFlow.ts`

Снять последний TODO:
```ts
import { wsClient } from "@/shared/api/wsClient.singleton";

export function logoutFlow() {
  tokenStorage.clear();
  wsClient.disconnect();
  useAuthStore.getState().clear();
  useRoomsStore.getState().clear();
  useChannelsStore.getState().clear();
  useChatStore.getState().clear();
  logger.info("logout: cleared all");
}
```

### `web/src/app/App.tsx`

Расширить — управление WS-connect:

```tsx
export function App() {
  useEffect(() => {
    if (tokenStorage.getAccess()) {
      useAuthStore.getState().loadMe();
    }
  }, []);

  // Подписка на authenticated → connect; idle → disconnect
  useEffect(() => {
    const unsub = useAuthStore.subscribe((s, prev) => {
      if (s.status === "authenticated" && prev.status !== "authenticated") {
        wsClient.connect();
      } else if (prev.status === "authenticated" && s.status !== "authenticated") {
        wsClient.disconnect();
      }
    });
    return unsub;
  }, []);

  // Routing WS-фреймов в сторы
  useEffect(() => {
    const unsubFrame = wsClient.onFrame((frame) => {
      switch (frame.type) {
        case "subscribed":    useChatStore.getState().onSubscribed(frame.data); break;
        case "message.new":   useChatStore.getState().onMessageNew(frame.data); break;
        case "message.sent":  useChatStore.getState().onMessageSent(frame.data); break;
        case "error":         useChatStore.getState().onWsError(frame.data); break;
        case "member.joined": useRoomsStore.getState().appendMember(frame.data); break;
      }
    });
    const unsubStatus = wsClient.onStatus((status) => {
      useChatStore.getState().setWsStatus(status);
    });
    return () => { unsubFrame(); unsubStatus(); };
  }, []);

  return (
    <ErrorBoundary>
      <RouterProvider router={router} />
      <Toaster />
    </ErrorBoundary>
  );
}
```

### `web/src/features/chat/components/MessageList.tsx`

`useEffect` теперь вызывает `useChatStore.openChannel(channelId)` вместо `loadHistory` напрямую (внутри `openChannel` всё сделается).

### `web/src/features/chat/components/MessageComposer.tsx`

- Снять disabled-флаг — теперь зависит от `wsStatus === "open"`.
- onSubmit: вызвать `useChatStore.getState().sendMessage(channelId, text)` + form.reset().

### Mocking infrastructure: `web/src/shared/api/__tests__/ws.fake.ts`

```ts
export function createFakeWsClient(): WsClient & { _emit(frame: IncomingFrame): void; _setStatus(s: WsStatus): void } {
  // ...in-memory implementation как в `../04-testing.md`...
}
```

В тестах сторов подменяем `wsClient` через `vi.mock("@/shared/api/wsClient.singleton", () => ({ wsClient: createFakeWsClient() }))`.

## Файлы для модификации (свод)

- `web/src/features/chat/store/index.ts` — добавить actions + WS-handlers.
- `web/src/features/rooms/store/index.ts` — добавить `appendMember`, в `joinByCode` добавить `wsClient.reconnect()`.
- `web/src/shared/lib/logoutFlow.ts` — добавить `wsClient.disconnect()`.
- `web/src/app/App.tsx` — auth-state-driven WS connect/disconnect + frame routing.
- `web/src/features/chat/components/MessageList.tsx` — заменить `loadHistory` на `openChannel`.
- `web/src/features/chat/components/MessageComposer.tsx` — enabled при wsStatus="open", submit вызывает sendMessage.
- `web/src/features/chat/components/MessageItem.tsx` — retry-кнопка для failed (enabled при wsStatus="open"), вызывает `retryMessage`.

## Ключевые решения

- **D-05** single-flight refresh — не пересекается с WS, fetch-обёртка работает независимо.
- **D-07** Optimistic update — реализовано через tempId + reconciliation в `onMessageSent`.
- **D-08** Multi-tab dedup — `onMessageNew` проверяет `existing.id === data.id`.
- **D-09** WS reconnect — exponential backoff в `ws.ts`.
- **D-10** Reconnect WS после `joinByCode` — реализовано.
- **D-15** Моки WS через подменную фабрику — `ws.fake.ts`.

## Verification

- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все ~151 unit/component тестов зелёные.
- [ ] Ручная проверка happy-path: открыть text-канал → composer enabled → отправить «hi» → видим pending → видим committed.
- [ ] Multi-tab: открыть две вкладки одного юзера на одном канале → отправить из A → видеть в обеих, без дубля.
- [ ] WS disconnect (kill бэк) → видим banner «Переподключение…» → restart бэка → banner исчезает, composer работает.
- [ ] Logout — closing WS (close-code 1000).
- [ ] Никаких TODO/FIXME в `logoutFlow.ts`, `joinByCode`, `useChatStore`.
- [ ] Никаких голых `new WebSocket()` вне `shared/api/ws.ts`.
- [ ] Никаких голых `fetch()` вне `shared/api/fetch.ts`.
- [ ] WS-токен не утекает в console.log.
