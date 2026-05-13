---
phase: 7
name: Channels feature
layer: feature
depends_on: [phase-06]
plan: ./README.md
---

# Phase 7: Channels feature

## Цель

Реализовать feature-слайс `web/src/features/channels/`: types, api, `useChannelsStore`, UI-компоненты (ChannelList, modals). Заполнить guard `<RequireChannelInRoom>`. Подключить ChannelList к `RoomScopedSidebar`. После фазы — пользователь видит каналы выбранной комнаты, может создавать/удалять text-каналы (admin/owner), voice-каналы отображаются как disabled.

## Контекст

Phase 6 дала `useRoomsStore` и Sidebar. UI — `../07-ui-contract.md` секция «features/channels». Behavior — `../02-behavior.md` UC-10. Store — `../05-state-model.md` `useChannelsStore`. API — `../06-api-integration.md` секция «REST: Channels». ADR D-12 — voice как disabled.

## Файлы для создания

### `web/src/features/channels/types.ts`

```ts
export type ChannelId = string;
export type ChannelKind = "text" | "voice";

export type ChannelDto = {
  id: string;
  roomId: string;
  name: string;
  kind: ChannelKind;
  createdAt: string;
};
export type CreateChannelRequest = { name: string; kind: ChannelKind };
export type ListChannelsResponse = { items: ChannelDto[] };

export type Channel = ChannelDto;
```

### `web/src/features/channels/api/http.ts` (+ `.test.ts`)

```ts
export function listChannels(roomId: string): Promise<ApiResult<ListChannelsResponse>>;
export function createChannel(roomId: string, req: CreateChannelRequest): Promise<ApiResult<ChannelDto>>;
export function deleteChannel(roomId: string, channelId: string): Promise<ApiResult<null>>;
```

Пути из `../06-api-integration.md`. Tests (~3).

### `web/src/features/channels/store/mapErrors.ts` (+ `.test.ts`)

```ts
// CHANNEL-001 → "Название канала: 1..64 символа"
// CHANNEL-002 → "Тип канала: text или voice"
// CHANNEL-003 → "Канал не найден"
// CHANNEL-004 → "Имя канала уже занято"
// CHANNEL-006 → "Вы не состоите в этой комнате"
// CHANNEL-007 → "Только админ или владелец могут это делать"
```

### `web/src/features/channels/store/index.ts` (+ `.test.ts`)

```ts
type ChannelsState = {
  channelsByRoom: Record<RoomId, Channel[]>;
  loadingByRoom: Record<RoomId, boolean>;
  errorByRoom: Record<RoomId, string | null>;
  activeChannelId: ChannelId | null;
};

type ChannelsActions = {
  loadChannels: (roomId: RoomId) => Promise<void>;
  createChannel: (roomId: RoomId, req: CreateChannelRequest) => Promise<Channel | null>;
  deleteChannel: (roomId: RoomId, channelId: ChannelId) => Promise<boolean>;
  selectChannel: (channelId: ChannelId | null) => void;
  clear: () => void;
};
```

**Особенности:**
- `loadChannels`: идемпотентный — не делает дубль fetch, если уже loading.
- На ROOM-003/CHANNEL-006 (not member) — каскадно удаляем room из `useRoomsStore.rooms` (через `getState`). Это исключение к правилу изоляции — задокументируем в `09-standards.md` (уже есть «Cascade logout», тут аналогично «Cascade on 403 not member»).
- `selectChannel` НЕ делает WS-subscribe сам — это будет делать `useChatStore.openChannel` в Phase 8/9.

**Tests (~8):** см. `../04-testing.md`.

### Modified: `web/src/app/routes/RequireChannelInRoom.tsx`

Заменить заглушку Phase 5 на полную реализацию:

```tsx
export function RequireChannelInRoom({ children }: Props) {
  const { roomId, channelId } = useParams<{ roomId: string; channelId: string }>();
  const channels = useChannelsStore((s) => s.channelsByRoom[roomId!] ?? EMPTY_CHANNELS);
  const loading = useChannelsStore((s) => s.loadingByRoom[roomId!]);

  if (loading) return <FullScreenSpinner />;
  const channel = channels.find((c) => c.id === channelId);
  if (!channel) {
    toast.error(ru.channels.notFound);
    return <Navigate to={`/rooms/${roomId}`} replace />;
  }
  if (channel.kind === "voice") {
    toast.info(ru.channels.voiceComingSoon);
    return <Navigate to={`/rooms/${roomId}`} replace />;
  }
  return <>{children}</>;
}
```

**Tests (~4):** loading; channel не найден; voice → redirect; text → children.

### `web/src/features/channels/components/ChannelList.tsx` (+ `.module.css` + `.test.tsx`)

```tsx
type Props = { roomId: string };

export function ChannelList({ roomId }: Props) {
  const navigate = useNavigate();
  const { channelId: activeChannelId } = useParams();
  const channels = useChannelsStore((s) => s.channelsByRoom[roomId] ?? EMPTY_CHANNELS);
  const loading = useChannelsStore((s) => s.loadingByRoom[roomId]);
  const myRole = useRoomsStore((s) => s.rooms.find((r) => r.id === roomId)?.role);
  const canManage = myRole === "owner" || myRole === "admin";
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => { useChannelsStore.getState().loadChannels(roomId); }, [roomId]);

  if (loading) return <Spinner />;
  const textChannels = channels.filter((c) => c.kind === "text");
  const voiceChannels = channels.filter((c) => c.kind === "voice");
  return (
    <section>
      <header>
        <h3>{ru.channels.channels}</h3>
        {canManage && <button onClick={() => setCreateOpen(true)} aria-label={ru.channels.createBtn}>+</button>}
      </header>
      {textChannels.length === 0 && voiceChannels.length === 0 ? (
        <EmptyState title={ru.channels.empty} action={canManage ? <Button onClick={...}>...</Button> : undefined} />
      ) : (
        <>
          <h4>{ru.channels.textChannels}</h4>
          <ul role="list">
            {textChannels.map((c) => <ChannelListItem key={c.id} channel={c} active={c.id === activeChannelId} onSelect={() => navigate(`/rooms/${roomId}/channels/${c.id}`)} />)}
          </ul>
          {voiceChannels.length > 0 && (
            <>
              <h4>{ru.channels.voiceChannels} <span className={styles.comingSoon}>(coming soon)</span></h4>
              <ul role="list">
                {voiceChannels.map((c) => <ChannelListItem key={c.id} channel={c} disabled onSelect={noop} active={false} />)}
              </ul>
            </>
          )}
        </>
      )}
      <CreateChannelModal open={createOpen} roomId={roomId} onClose={() => setCreateOpen(false)} />
    </section>
  );
}
```

**Tests (~6):** см. `../04-testing.md`.

### `web/src/features/channels/components/ChannelListItem.tsx` (+ `.test.tsx`)

```tsx
type Props = { channel: Channel; active: boolean; disabled?: boolean; onSelect: () => void };
```

- Если `disabled` — `aria-disabled="true"`, не реагирует на click, tooltip «coming soon».
- Если `active` — `aria-current="page"`.
- Префикс `#` для text-каналов, иконка voice для voice.

**Tests (~3).**

### `web/src/features/channels/components/CreateChannelModal.tsx` (+ `.schema.ts` + `.test.tsx`)

`.schema.ts`:
```ts
export const createChannelSchema = z.object({
  name: z.string().trim().min(1, "...").max(64, "..."),
  kind: z.enum(["text", "voice"]),
});
```

`.tsx`:
- `<Modal />`.
- react-hook-form + zod.
- `onSubmit` → `useChannelsStore.createChannel(roomId, values)`.
- Если `kind="voice"` — после создания toast `Канал создан, но голосовая часть появится позже`.

**Tests (~5).**

### `web/src/features/channels/components/DeleteChannelConfirm.tsx` (+ `.test.tsx`)

Аналог `<DeleteRoomConfirm>`. Видим только admin/owner.

**Tests (~3).**

### `web/src/features/channels/index.ts`

```ts
export { useChannelsStore } from "./store";
export { ChannelList } from "./components/ChannelList";
export type { Channel, ChannelKind } from "./types";
```

## Файлы для модификации

- `web/src/app/RoomScopedSidebar.tsx` — добавить `<ChannelList roomId={roomId} />` перед `<MembersList />`.
- `web/src/pages/RoomPage.tsx` — empty state «Выберите канал» (если нет channelId) + меню с DeleteRoom (owner) + DeleteChannel (для текущего, admin/owner) кнопками.
- `web/src/shared/lib/i18n/ru.ts` — раздел `channels`.
- `web/src/shared/lib/logoutFlow.ts` — снять TODO(phase-07), добавить `useChannelsStore.getState().clear()`.

## Ключевые решения

- **D-12** Voice — disabled list-item. Реализовано в `<ChannelListItem disabled />`.
- `<RequireChannelInRoom>` — полная реализация, voice также редиректит.
- Cascade на 403 не member — аналогично logout-cascade, документируется в `09-standards.md`.

## Verification

- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все ~30 тестов channels-фичи зелёные.
- [ ] Ручная проверка: открываем комнату → ChannelList пуст → admin/owner кликает «+» → создаём text-канал → редирект на `/rooms/:rid/channels/:cid`.
- [ ] Voice-каналы отображаются disabled, не кликабельны, tooltip «coming soon».
- [ ] `<RequireChannelInRoom>` редиректит на `/rooms/:rid` для несуществующего канала и для voice.
- [ ] DeleteChannel: admin/owner может удалить text-канал.
- [ ] `logoutFlow` чистит `useChannelsStore`.
