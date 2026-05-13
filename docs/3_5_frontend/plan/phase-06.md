---
phase: 6
name: Rooms feature
layer: feature
depends_on: [phase-05]
plan: ./README.md
---

# Phase 6: Rooms feature

## Цель

Реализовать feature-слайс `web/src/features/rooms/`: types, api (REST-only), `useRoomsStore` (без WS handlers — WS добавляется в Phase 9), все UI-компоненты (RoomList, modals, MembersList). Заполнить guard `<RequireMembership>`. Подключить компоненты к Sidebar и `RoomsIndexPage`. После фазы — пользователь видит свои комнаты, создаёт/удаляет, генерирует инвайт-коды, вступает по коду, видит участников.

## Контекст

Phase 5 дала роуты, AppShell, заглушку `<RequireMembership>`. UI — `../07-ui-contract.md` секция «features/rooms». Behavior — `../02-behavior.md` UC-4..UC-9. Store — `../05-state-model.md` `useRoomsStore`. API — `../06-api-integration.md` секция «REST: Rooms». Стандарты — `prompts/Zustand Stores.txt`, `prompts/React Components.txt`.

WS-handler `member.joined` НЕ реализуется в этой фазе — он добавится в Phase 9 как modification.

`joinByCode` action сейчас НЕ вызывает `wsClient.reconnect()` (его ещё нет). Сейчас просто обновляет state, в Phase 9 добавим вызов.

## Файлы для создания

### `web/src/features/rooms/types.ts`

```ts
export type Role = "owner" | "admin" | "member";
export type RoomId = string;

// DTO
export type RoomDto = { id: string; ownerId: string; name: string; createdAt: string };
export type RoomWithRoleDto = RoomDto & { role: Role };
export type MemberDto = { userId: string; role: Role; joinedAt: string };
export type InviteCodeDto = { code: string; createdBy: string; createdAt: string };
export type CreateRoomRequest = { name: string };
export type ListRoomsResponse = { items: RoomWithRoleDto[] };
export type ListMembersResponse = { items: MemberDto[] };

// ViewModel = DTO (без переименований)
export type Room = RoomDto;
export type RoomWithRole = RoomWithRoleDto;
export type Member = MemberDto;
export type InviteCode = InviteCodeDto;
```

### `web/src/features/rooms/api/http.ts` (+ `.test.ts`)

```ts
export function listRooms(): Promise<ApiResult<ListRoomsResponse>>;
export function createRoom(req: CreateRoomRequest): Promise<ApiResult<RoomDto>>;
export function getRoom(roomId: string): Promise<ApiResult<RoomDto>>;
export function deleteRoom(roomId: string): Promise<ApiResult<null>>;
export function listMembers(roomId: string): Promise<ApiResult<ListMembersResponse>>;
export function regenerateInvite(roomId: string): Promise<ApiResult<InviteCodeDto>>;
export function joinByCode(code: string): Promise<ApiResult<RoomDto>>;
```

Пути из `../06-api-integration.md`. Tests (~7): каждый эндпоинт — URL, method, body.

### `web/src/features/rooms/store/mapErrors.ts` (+ `.test.ts`)

```ts
export function mapRoomErrorToMessage(code: string): string;
// ROOM-001 → "Название: 1..64 символа"
// ROOM-002 → "Комната не найдена"
// ROOM-003 → "Вы не состоите в этой комнате"
// ROOM-004 → "Только админ или владелец могут это делать"
// ROOM-005 → "Удалять комнату может только владелец"
// ROOM-006 → "Вы уже состоите в этой комнате"
// ROOM-007 → "Код приглашения не найден или отозван"
// ROOM-008 → "Неверный формат кода (8 символов)"
// default → "Неизвестная ошибка"
```

### `web/src/features/rooms/store/index.ts` (+ `.test.ts`)

См. `../05-state-model.md` `useRoomsStore`.

**Структура:**
```ts
type RoomsState = {
  rooms: RoomWithRole[];
  status: "idle" | "loading" | "error" | "ready";
  error: string | null;
  membersByRoom: Record<RoomId, Member[]>;
  membersLoadingByRoom: Record<RoomId, boolean>;
  activeInviteByRoom: Record<RoomId, InviteCode>;
  activeRoomId: RoomId | null;
};

type RoomsActions = {
  loadRooms: () => Promise<void>;
  createRoom: (name: string) => Promise<RoomWithRole | null>;
  deleteRoom: (roomId: RoomId) => Promise<boolean>;
  loadMembers: (roomId: RoomId) => Promise<void>;
  regenerateInvite: (roomId: RoomId) => Promise<InviteCode | null>;
  joinByCode: (code: string) => Promise<Room | null>;
  selectRoom: (roomId: RoomId | null) => void;
  // appendMember — добавится в Phase 9 как WS handler
  clear: () => void;
};
```

**Без persist.**

**Особенности actions:**
- `createRoom`: на ok — `rooms.push({...room, role: "owner"})` (создатель сразу owner).
- `deleteRoom`: на ok — `rooms.filter`, **каскадно** очищает `membersByRoom[roomId]`, `activeInviteByRoom[roomId]`. На `ROOM-005` — toast, return false.
- `joinByCode`: на ok — `rooms.push({...room, role: "member"})`. **TODO для Phase 9:** `wsClient.reconnect()`.
- `selectRoom(id)`: меняет `activeRoomId`. Не грузит данные — это делают компоненты через `useEffect`.
- `clear`: сброс state до initial.

**Tests (~12):** см. `../04-testing.md` `useRoomsStore`.

### Modified: `web/src/app/routes/RequireMembership.tsx`

Заменить заглушку Phase 5 на полную реализацию (см. `../08-routes.md`):

```tsx
export function RequireMembership({ children }: Props) {
  const { roomId } = useParams<{ roomId: string }>();
  const room = useRoomsStore((s) => s.rooms.find((r) => r.id === roomId));
  const status = useRoomsStore((s) => s.status);
  if (status === "idle" || status === "loading") return <FullScreenSpinner />;
  if (!room) {
    toast.error(ru.rooms.notMember);
    return <Navigate to="/rooms" replace />;
  }
  return <>{children}</>;
}
```

**Tests (~3):** loading → spinner; room не найдена → redirect + toast; room найдена → children.

### `web/src/features/rooms/components/RoomList.tsx` (+ `.module.css` + `.test.tsx`)

См. `../07-ui-contract.md` `<RoomList />`.

```tsx
export function RoomList() {
  const navigate = useNavigate();
  const rooms = useRoomsStore((s) => s.rooms);
  const status = useRoomsStore((s) => s.status);
  const { roomId: activeRoomId } = useParams();
  const [createOpen, setCreateOpen] = useState(false);
  const [joinOpen, setJoinOpen] = useState(false);

  useEffect(() => {
    if (status === "idle") useRoomsStore.getState().loadRooms();
  }, [status]);

  if (status === "loading") return /* skeletons */;
  if (status === "error") return /* inline error + retry */;
  if (rooms.length === 0) return <EmptyRoomState onCreate={...} onJoin={...} />;

  return (
    <nav className={styles.list}>
      <header>
        <h2>{ru.rooms.myRooms}</h2>
        <PopoverMenu items={[{label: ru.rooms.create, onClick: () => setCreateOpen(true)}, {label: ru.rooms.join, onClick: () => setJoinOpen(true)}]}>+</PopoverMenu>
      </header>
      <ul role="list">
        {rooms.map((r) => <RoomListItem key={r.id} room={r} active={r.id === activeRoomId} onClick={() => navigate(`/rooms/${r.id}`)} />)}
      </ul>
      <CreateRoomModal open={createOpen} onClose={() => setCreateOpen(false)} onCreated={(room) => navigate(`/rooms/${room.id}`)} />
      <JoinByCodeModal open={joinOpen} onClose={() => setJoinOpen(false)} onJoined={(room) => navigate(`/rooms/${room.id}`)} />
    </nav>
  );
}
```

**Tests (~5):** см. `../04-testing.md` `<RoomList />`.

### `web/src/features/rooms/components/RoomListItem.tsx` (+ `.module.css` + `.test.tsx`)

```tsx
type Props = { room: RoomWithRole; active: boolean; onClick: () => void };
```

- Avatar по `room.id` (детерминированный цвет).
- При active — `aria-current="page"` и подсветка.

**Tests (~3).**

### `web/src/features/rooms/components/CreateRoomModal.tsx` (+ `.schema.ts` + `.test.tsx`)

`.schema.ts`:
```ts
export const createRoomSchema = z.object({
  name: z.string().trim().min(1, "...").max(64, "..."),
});
```

`.tsx`:
- Использует `<Modal />` из shared/ui.
- react-hook-form + zod.
- `onSubmit` → `useRoomsStore.createRoom(name)` → если ok, `onCreated(room)` + `onClose()`. На fail — `setError` с mapError'ом.

**Tests (~6):** см. `../04-testing.md`.

### `web/src/features/rooms/components/DeleteRoomConfirm.tsx` (+ `.test.tsx`)

**Props:** `{ open: boolean; roomId: string; roomName: string; onCancel: () => void; onDeleted: () => void }`.

- `<Modal />` с warning-стилем.
- Кнопка «Удалить» — `variant="danger"`, `aria-label="Удалить комнату {name}"`.
- onClick «Удалить»: `useRoomsStore.deleteRoom(roomId)` → если ok → `onDeleted()` (компонент-родитель сделает navigate); если fail — toast (уже сделан в стор).

**Tests (~3).**

### `web/src/features/rooms/components/InviteCodeModal.tsx` (+ `.test.tsx`)

См. `../07-ui-contract.md`. Состояния: no-code / generating / has-code / error.

**Logic:**
- При open — НЕ генерируем автоматически. Только при click на кнопку.
- Click «Сгенерировать» → `useRoomsStore.regenerateInvite(roomId)` → отображаем `activeInviteByRoom[roomId]`.
- Кнопка «Скопировать» → `navigator.clipboard.writeText(code)` → toast `Скопировано`. Fallback на `document.execCommand("copy")`.

**Tests (~4).**

### `web/src/features/rooms/components/JoinByCodeModal.tsx` (+ `.schema.ts` + `.test.tsx`)

`.schema.ts`:
```ts
export const joinByCodeSchema = z.object({
  code: z.string().length(8).regex(/^[0-9A-HJKMNP-TV-Z]{8}$/, "..."),
});
```

`.tsx`:
- Input с auto upper-case (через onChange `setValue("code", value.toUpperCase())`).
- onSubmit → `useRoomsStore.joinByCode(code)` → если ok → `onJoined(room)`. На fail — `setError("code", ...)`.
- При `ROOM-006` (already member) — `toast.info` + найти комнату в state и navigate туда.

**Tests (~5).**

### `web/src/features/rooms/components/MembersList.tsx` (+ `.module.css` + `.test.tsx`)

```tsx
type Props = { roomId: string };
export function MembersList({ roomId }: Props) {
  const members = useRoomsStore((s) => s.membersByRoom[roomId] ?? EMPTY_MEMBERS);
  const loading = useRoomsStore((s) => s.membersLoadingByRoom[roomId]);
  const currentUserId = useAuthStore((s) => s.currentUser?.id);
  const currentUsername = useAuthStore((s) => s.currentUser?.username);

  useEffect(() => {
    useRoomsStore.getState().loadMembers(roomId);
  }, [roomId]);

  if (loading && members.length === 0) return <Spinner />;
  return (
    <section>
      <h3>{ru.rooms.members} ({members.length})</h3>
      <ul role="list">
        {members.map((m) => (
          <MemberListItem
            key={m.userId}
            member={m}
            isMe={m.userId === currentUserId}
            meUsername={currentUsername}
          />
        ))}
      </ul>
    </section>
  );
}
```

**Tests (~4):** см. `../04-testing.md` `<MembersList />`.

### `web/src/features/rooms/components/MemberListItem.tsx` (+ `.test.tsx`)

```tsx
type Props = { member: Member; isMe: boolean; meUsername?: string };
```

- Avatar по `member.userId`.
- Имя: если `isMe` — `meUsername`; иначе — first 6 chars of UUID without dashes uppercase.
- Role badge (только если != "member"): `[owner]` / `[admin]`.

**Tests (~3):** placeholder для других, реальный username для me, role badge только для admin/owner.

### `web/src/features/rooms/index.ts`

```ts
export { useRoomsStore } from "./store";
export { RoomList } from "./components/RoomList";
export { MembersList } from "./components/MembersList";
export { CreateRoomModal } from "./components/CreateRoomModal";
export { DeleteRoomConfirm } from "./components/DeleteRoomConfirm";
export { InviteCodeModal } from "./components/InviteCodeModal";
export { JoinByCodeModal } from "./components/JoinByCodeModal";
export type { Room, RoomWithRole, Member, InviteCode, Role } from "./types";
```

## Файлы для модификации

- `web/src/app/Sidebar.tsx` — добавить `<RoomList />` в верхнюю часть.
- `web/src/app/RoomScopedSidebar.tsx` — добавить `<MembersList roomId={roomId} />` в нижнюю часть.
- `web/src/pages/RoomsIndexPage.tsx` — EmptyState либо redirect на первую комнату (если есть). Заменяем заглушку Phase 5:
  ```tsx
  export function RoomsIndexPage() {
    const rooms = useRoomsStore((s) => s.rooms);
    const status = useRoomsStore((s) => s.status);
    if (status === "loading") return <FullScreenSpinner />;
    if (rooms.length > 0) return <Navigate to={`/rooms/${rooms[0].id}`} replace />;
    return <EmptyRoomState />;
  }
  ```
- `web/src/pages/RoomPage.tsx` — добавить меню действий (Owner: Delete; Admin/Owner: Invite, CreateChannel — но CreateChannel в Phase 7).
- `web/src/shared/lib/i18n/ru.ts` — расширить раздел `rooms`.
- `web/src/shared/lib/logoutFlow.ts` — снять TODO(phase-06), добавить вызов `useRoomsStore.getState().clear()`.

## Ключевые решения

- **D-10** Reconnect WS после `joinByCode` — TODO для Phase 9 (сейчас зафиксировано комментарием в action).
- **D-11** UUID-placeholder для members — реализован в `<MemberListItem />`.
- **D-18** Один store на фичу — `useRoomsStore`.
- **D-19** Без persist для rooms.
- `<RequireMembership>` гард — полная реализация.

## Verification

- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все ~45 тестов rooms-фичи зелёные.
- [ ] Ручная проверка: login → видим Sidebar с empty state → создаём комнату через модалку → редирект в новую комнату → видим MembersList с одним участником (me).
- [ ] DeleteRoom: owner — может удалить; admin/member — нет (`ROOM-005` toast).
- [ ] InviteCodeModal: кнопка «Скопировать» работает через clipboard API.
- [ ] JoinByCode: 8-символьный Crockford code валидируется на клиенте.
- [ ] `<RequireMembership>` редиректит на /rooms при отсутствии room.
- [ ] `logoutFlow` чистит `useRoomsStore`.
- [ ] Никаких WS-вызовов в этой фазе (`grep -r "wsClient" web/src/features/rooms/` — пусто; есть только TODO-комментарий в `joinByCode`).
