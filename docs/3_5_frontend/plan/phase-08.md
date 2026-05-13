---
phase: 8
name: Chat REST
layer: feature
depends_on: [phase-07]
plan: ./README.md
---

# Phase 8: Chat REST

## Цель

Реализовать feature-слайс `web/src/features/chat/` без WS-интеграции: types, api (`listMessages`), `useChatStore` с REST-логикой (load history, load more, базовая структура `wsStatus`), все UI-компоненты (`ChatPanel`, `MessageList`, `MessageItem`, `MessageComposer`, `EmptyChat`, `ConnectionStatusBanner`). После фазы — пользователь видит историю сообщений канала, infinite-scroll вверх работает; `MessageComposer` рендерится disabled (т.к. WS не открыт), отправка ещё не работает (это Phase 9).

## Контекст

Phase 7 дала каналы и роут `/rooms/:rid/channels/:cid`. UI — `../07-ui-contract.md` секция «features/chat». Behavior — `../02-behavior.md` UC-11 (history load), UC-13 (infinite scroll). Store — `../05-state-model.md` `useChatStore`. API — `../06-api-integration.md` секция «REST: Chat history». WS-handlers и `sendMessage` — Phase 9.

## Файлы для создания

### `web/src/features/chat/types.ts`

```ts
export type MessageStatus = "pending" | "committed" | "failed";

export type MessageDto = {
  id: string;
  channelId: string;
  authorId: string;
  text: string;
  createdAt: string;
};

export type ListMessagesQuery = { before?: string; limit?: number };
export type ListMessagesResponse = { items: MessageDto[]; nextBefore: string | null };

// ViewModel — расширяем DTO статусом и tempId
export type Message = MessageDto & {
  status: MessageStatus;
  tempId?: string;  // присутствует только для pending
};

export type WsStatus = "idle" | "connecting" | "open" | "reconnecting" | "closed";
```

### `web/src/features/chat/api/http.ts` (+ `.test.ts`)

```ts
export function listMessages(channelId: string, query?: ListMessagesQuery): Promise<ApiResult<ListMessagesResponse>>;
```

- Query-string: `?before=...&limit=...` (если параметры есть).
- Tests (~3): без параметров, с `before`, с `limit`.

### `web/src/features/chat/store/mapErrors.ts`

```ts
// CHAT-001 → "Сообщение не может быть пустым (1..4000 символов)"
// CHAT-002 → "Канал не найден"
// CHAT-003 → "Голосовой канал недоступен"
// CHAT-004 → "Вы не состоите в этой комнате"
// CHAT-005 → "Внутренняя ошибка клиента"
// CHAT-006 → "Неверный лимит (1..100)"
```

### `web/src/features/chat/store/index.ts` (+ `.test.ts`)

См. `../05-state-model.md` `useChatStore`.

**Phase 8 (без WS):**

```ts
type ChatState = {
  messagesByChannel: Record<ChannelId, Message[]>;     // отсортированы createdAt ASC
  nextBeforeByChannel: Record<ChannelId, string | null>;
  loadingHistoryByChannel: Record<ChannelId, boolean>;
  loadingMoreHistoryByChannel: Record<ChannelId, boolean>;
  subscribedChannelId: ChannelId | null;  // в Phase 8 всегда null
  wsStatus: WsStatus;  // в Phase 8 всегда "idle"
};

type ChatActions = {
  loadHistory: (channelId: ChannelId) => Promise<void>;
  loadMoreHistory: (channelId: ChannelId) => Promise<void>;
  // sendMessage, openChannel, WS-handlers — добавляются в Phase 9
  clear: () => void;
};
```

**Особенности:**
- `loadHistory`: вызывает `listMessages(channelId, { limit: 50 })`. На ok — `messagesByChannel[id] = items.slice().reverse()` (бэк отдаёт DESC, нам в UI ASC удобнее), `nextBeforeByChannel[id] = nextBefore`. Идемпотентен: если `messagesByChannel[id]` уже есть — не грузит.
- `loadMoreHistory`: использует `nextBeforeByChannel[id]` как cursor. Префиксует к `messagesByChannel[id]`. Останавливается если `nextBefore` стал null.
- На CHAT-004/CHAT-002 — каскадно дёргаем `useChannelsStore.getState().loadChannels(roomId)` для пересинхронизации (опционально toast).

**Tests (~6 в Phase 8; ~8 будет добавлено в Phase 9):** loadHistory happy, loadMoreHistory pagination, end of history (nextBefore=null), CHAT-002, CHAT-004, clear.

### `web/src/features/chat/components/ChatPanel.tsx` (+ `.module.css`)

```tsx
type Props = { channelId: string };

export function ChatPanel({ channelId }: Props) {
  return (
    <div className={styles.panel}>
      <ConnectionStatusBanner />
      <MessageList channelId={channelId} />
      <MessageComposer channelId={channelId} />
    </div>
  );
}
```

В Phase 8 нет полноценного header канала, можно добавить простой `<h3>{channelName}</h3>` через получение из `useChannelsStore`. Опционально.

### `web/src/features/chat/components/MessageList.tsx` (+ `.module.css` + `.test.tsx`)

```tsx
type Props = { channelId: string };
```

**Логика:**
- `useEffect` → `useChatStore.getState().loadHistory(channelId)`.
- Селекторы: `messages = useChatStore(s => s.messagesByChannel[channelId] ?? EMPTY)`, `loading`, `nextBefore`.
- Состояния:
  - loading первичный — spinner;
  - empty — `<EmptyChat />`;
  - ready — `<ul role="log" aria-live="polite">` с `<MessageItem>` для каждого.
- Infinite scroll up через `IntersectionObserver` на «sentinel» в начале списка — если sentinel visible и `nextBefore != null` → `loadMoreHistory(channelId)`.
- Auto-scroll к низу при mount.
- При новых сообщениях (через WS в Phase 9) — auto-scroll если пользователь у низа; иначе показать pill «↓ новые (N)».

**Tests (~8):** см. `../04-testing.md`.

### `web/src/features/chat/components/MessageItem.tsx` (+ `.module.css` + `.test.tsx`)

```tsx
type Props = { message: Message; isAuthor: boolean; authorPlaceholder: string };
```

- Avatar по `message.authorId`.
- `authorPlaceholder` — `isAuthor ? meUsername : first6chars(authorId)`.
- Время через `formatTime(message.createdAt)`.
- Текст: `linkify(message.text)` → массив частей → рендер `<span>` для text + `<a target="_blank" rel="noopener noreferrer">` для link.
- Статус (только для `isAuthor`):
  - `pending`: серый текст + иконка «часы».
  - `committed`: обычный.
  - `failed`: красная иконка ⚠ + кнопка `<Button variant="ghost" size="sm" onClick={retry}>Повторить</Button>` (retry будет работать только после Phase 9, сейчас просто визуальный плейсхолдер; `disabled` если wsStatus !== "open").

**Tests (~6):** layout, статусы (3), linkify работает, HTML не рендерится.

### `web/src/features/chat/components/MessageComposer.tsx` (+ `.module.css` + `.schema.ts` + `.test.tsx`)

`.schema.ts`:
```ts
export const messageSchema = z.object({
  text: z.string().trim().min(1).max(4000),
});
```

`.tsx`:
```tsx
type Props = { channelId: string };
```

- `<Textarea />` (autosize) + кнопка send.
- Enter — submit; Shift+Enter — newline.
- В Phase 8 — `disabled` всегда (потому что `wsStatus !== "open"`).
- Phase 9 будет вызывать `useChatStore.sendMessage(channelId, text)` на submit.
- Counter показывается при > 3000 рун.

**Tests (~6 в Phase 8 — без отправки; в Phase 9 добавятся тесты на submit).**

### `web/src/features/chat/components/EmptyChat.tsx`

Простой `<EmptyState />` с подписью «Здесь пока пусто, напиши первое сообщение».

### `web/src/features/chat/components/ConnectionStatusBanner.tsx` (+ `.module.css` + `.test.tsx`)

```tsx
export function ConnectionStatusBanner() {
  const status = useChatStore((s) => s.wsStatus);
  if (status === "open" || status === "idle" || status === "closed") return null;
  return (
    <div className={styles.banner} role="status" aria-live="polite">
      {status === "connecting" ? "Подключение…" : "Переподключение…"}
    </div>
  );
}
```

**Tests (~3):** скрыт при open, виден при reconnecting, role=status.

### `web/src/features/chat/index.ts`

```ts
export { useChatStore } from "./store";
export { ChatPanel } from "./components/ChatPanel";
export type { Message, MessageStatus, WsStatus } from "./types";
```

## Файлы для модификации

- `web/src/pages/ChatRoutePage.tsx` — заменить заглушку Phase 5 на:
  ```tsx
  export function ChatRoutePage() {
    const { channelId } = useParams<{ channelId: string }>();
    return <ChatPanel channelId={channelId!} />;
  }
  ```
- `web/src/shared/lib/i18n/ru.ts` — раздел `chat`.
- `web/src/shared/lib/logoutFlow.ts` — снять TODO(phase-08), добавить `useChatStore.getState().clear()`.

## Ключевые решения

- **Курсор истории**: UUID существующего сообщения, не timestamp. См. `../06-api-integration.md`.
- **Storage порядка**: бэк отдаёт DESC, в `messagesByChannel` храним ASC для удобства рендера. Reverse на load.
- **Linkify**: shared utility, валидирует протоколы (см. `prompts/React Components.txt`).
- **Composer disabled** в Phase 8 — `wsStatus="idle"` всегда. В Phase 9 включится.

## Verification

- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все ~30 тестов chat-фичи (без WS) зелёные.
- [ ] Ручная проверка: открываем text-канал → видим спиннер, потом сообщения (если есть на бэке) → composer виден, но disabled.
- [ ] Scroll вверх до начала — подгружается следующая страница; при `nextBefore=null` — подпись «Начало канала».
- [ ] MessageItem рендерит linkify корректно (URL → `<a>`, `javascript:` — текст).
- [ ] Тег `<a>` имеет `target="_blank" rel="noopener noreferrer"`.
- [ ] Никаких `dangerouslySetInnerHTML` (lint enforce).
- [ ] `logoutFlow` чистит `useChatStore`.
- [ ] `ChatPanel` рендерит ConnectionBanner, MessageList, MessageComposer.
