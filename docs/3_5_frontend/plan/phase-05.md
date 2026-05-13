---
phase: 5
name: Routes + AppShell + Guards
layer: route + page
depends_on: [phase-04]
plan: ./README.md
---

# Phase 5: Routes + AppShell + Guards

## Цель

Реализовать полную карту роутов с гардами и трёхпанельный `<AppShell />`. После фазы — пользователь может залогиниться и попасть на защищённую страницу `/rooms` (пока пустая sidebar — заполняется в Phase 6). Deep-links работают: `/rooms/:rid/channels/:cid` без auth → редирект на `/login` с возвратом после login.

## Контекст

Phase 4 дала `useAuthStore`, `LoginPage`, `RegisterPage`, `<UserBadge />`. Routes/Guards описаны в `../08-routes.md`. AppShell — `../07-ui-contract.md` секция «Каркас приложения». Sidebar и его содержимое заполняются в Phase 6 (rooms) и 7 (channels). Стандарты — `prompts/Frontend Architecture Layers.txt` (правило: routes импортируется только из `app/`).

## Файлы для создания

### Guards

#### `web/src/app/routes/RequireAuth.tsx` (+ `.test.tsx`)

```tsx
type Props = { children: ReactNode };
export function RequireAuth({ children }: Props) {
  const status = useAuthStore((s) => s.status);
  const location = useLocation();
  if (status === "loading") return <FullScreenSpinner />;
  if (status !== "authenticated") {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <>{children}</>;
}
```

**Tests (~4):** authenticated → children; idle → redirect /login; loading → spinner; передаёт `from` в state.

#### `web/src/app/routes/RedirectIfAuthenticated.tsx` (+ `.test.tsx`)

```tsx
export function RedirectIfAuthenticated({ children }: Props) {
  const status = useAuthStore((s) => s.status);
  if (status === "authenticated") return <Navigate to="/rooms" replace />;
  return <>{children}</>;
}
```

**Tests (~2).**

#### `web/src/app/routes/RequireMembership.tsx` (+ `.test.tsx`)

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

**Note для Phase 5:** `useRoomsStore` ещё не существует (Phase 6). Решение: создаём гард в Phase 5 со ссылкой на «будущий» стор и добавляем `// TODO(phase-06): typed import once useRoomsStore exists`. ИЛИ — создаём в Phase 6 вместе с самим стором. Принято: создаём в Phase 5 с **временной заглушкой**:

```tsx
// Phase 5 stub — full impl in Phase 6
export function RequireMembership({ children }: Props) {
  return <>{children}</>;
}
```

И отмечаем в Phase 6 `Files for modification`: реализовать guard полностью.

Tests — отложить до Phase 6.

#### `web/src/app/routes/RequireChannelInRoom.tsx` (+ `.test.tsx`)

Аналогично — заглушка в Phase 5, реальная реализация в Phase 7 (когда есть `useChannelsStore`).

### AppShell

#### `web/src/app/AppShell.tsx` (+ `.module.css` + `.test.tsx`)

```tsx
export function AppShell() {
  return (
    <div className={styles.shell}>
      <TopBar />
      <Sidebar />
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  );
}
```

CSS grid: `grid-template-columns: 240px 1fr; grid-template-rows: 48px 1fr;` (примерно).

**Tests (~2):** рендер children через Outlet, наличие TopBar.

#### `web/src/app/TopBar.tsx` (+ `.module.css` + `.test.tsx`)

```tsx
export function TopBar() {
  const currentUser = useAuthStore((s) => s.currentUser);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();
  if (!currentUser) return null;
  return (
    <header className={styles.topbar}>
      <UserBadge user={currentUser} onLogout={() => { logout(); navigate("/login", { replace: true }); }} />
    </header>
  );
}
```

**Tests (~3):** рендер с currentUser; null без currentUser; логаут вызывает navigate.

#### `web/src/app/Sidebar.tsx` (+ `.module.css`)

```tsx
export function Sidebar() {
  return (
    <aside className={styles.sidebar}>
      {/* В Phase 6 здесь будет RoomList и RoomScopedSidebar */}
    </aside>
  );
}
```

Пустая в Phase 5 — наполнится постепенно.

#### `web/src/app/RoomScopedSidebar.tsx`

```tsx
export function RoomScopedSidebar() {
  const { roomId } = useParams<{ roomId: string }>();
  if (!roomId) return null;
  // В Phase 6 добавим MembersList, в Phase 7 — ChannelList
  return <div className={styles.scoped}></div>;
}
```

### Pages

#### `web/src/pages/RoomsIndexPage.tsx`

Заглушка — empty state. Полная реализация в Phase 6.
```tsx
export function RoomsIndexPage() {
  return <EmptyState title={ru.rooms.emptyTitle} description={ru.rooms.emptyDescription} />;
}
```

#### `web/src/pages/RoomPage.tsx`

```tsx
export function RoomPage() {
  return <EmptyState title={ru.rooms.selectChannel} />;
}
```

#### `web/src/pages/ChatRoutePage.tsx`

Заглушка — реальная в Phase 8.
```tsx
export function ChatRoutePage() {
  return <div>Чат — coming soon</div>;
}
```

#### `web/src/pages/NotFoundPage.tsx`

```tsx
export function NotFoundPage() {
  return (
    <EmptyState
      title="404 — страница не найдена"
      action={<Link to="/rooms">На главную</Link>}
    />
  );
}
```

### Routes

#### Modified: `web/src/routes.tsx`

Полная карта роутов согласно `../08-routes.md`:

```tsx
import { createBrowserRouter, Navigate, Outlet } from "react-router-dom";
import { RequireAuth } from "@/app/routes/RequireAuth";
import { RedirectIfAuthenticated } from "@/app/routes/RedirectIfAuthenticated";
import { RequireMembership } from "@/app/routes/RequireMembership";
import { RequireChannelInRoom } from "@/app/routes/RequireChannelInRoom";
import { AppShell } from "@/app/AppShell";
import { LoginPage } from "@/pages/LoginPage";
import { RegisterPage } from "@/pages/RegisterPage";
import { RoomsIndexPage } from "@/pages/RoomsIndexPage";
import { RoomPage } from "@/pages/RoomPage";
import { ChatRoutePage } from "@/pages/ChatRoutePage";
import { NotFoundPage } from "@/pages/NotFoundPage";

export const router = createBrowserRouter([
  { path: "/", element: <Navigate to="/rooms" replace /> },
  { path: "/login",    element: <RedirectIfAuthenticated><LoginPage /></RedirectIfAuthenticated> },
  { path: "/register", element: <RedirectIfAuthenticated><RegisterPage /></RedirectIfAuthenticated> },
  {
    path: "/rooms",
    element: <RequireAuth><AppShell /></RequireAuth>,
    children: [
      { index: true, element: <RoomsIndexPage /> },
      {
        path: ":roomId",
        element: <RequireMembership><Outlet /></RequireMembership>,
        children: [
          { index: true, element: <RoomPage /> },
          { path: "channels/:channelId", element: <RequireChannelInRoom><ChatRoutePage /></RequireChannelInRoom> },
        ],
      },
    ],
  },
  { path: "*", element: <NotFoundPage /> },
]);
```

#### Modified: `web/src/app/App.tsx`

Заменяет заглушку Phase 1 на:
```tsx
import { RouterProvider } from "react-router-dom";
import { router } from "@/routes";
import { ErrorBoundary } from "@/shared/ui/ErrorBoundary";
import { Toaster } from "@/shared/ui/Toaster";
import { useEffect } from "react";
import { useAuthStore } from "@/features/auth";
import { tokenStorage } from "@/shared/api/token-storage";

export function App() {
  useEffect(() => {
    // При старте — если токен есть, подгружаем профиль
    if (tokenStorage.getAccess()) {
      useAuthStore.getState().loadMe();
    }
  }, []);
  return (
    <ErrorBoundary>
      <RouterProvider router={router} />
      <Toaster />
    </ErrorBoundary>
  );
}
```

Этот mount-effect реализует UC-16 (восстановление сессии после reload).

## Файлы для модификации

- `web/src/shared/lib/i18n/ru.ts` — добавить ключи `rooms.emptyTitle`, `rooms.emptyDescription`, `rooms.selectChannel`, `rooms.notMember`, `nav.toMain`, etc.

## Ключевые решения

- **D-01** react-router-dom v6 — `createBrowserRouter` с nested routes.
- Гарды `<RequireMembership>` и `<RequireChannelInRoom>` — заглушки в Phase 5, полная реализация в Phase 6/7 соответственно (зависят от сторов rooms/channels).
- Восстановление сессии — в `<App>` через `useEffect` (UC-16).

## Verification

- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все тесты Phase 5 (~10) зелёные.
- [ ] `npm --prefix web run dev` — открыть `http://localhost:5173/`, видим redirect → `/rooms`, видим redirect → `/login`.
- [ ] После login — попадаем на `/rooms` с empty state «У вас нет комнат».
- [ ] Open `/rooms/123` (несуществующая) — гард-заглушка пропускает; `<RoomPage>` рендерится. (В Phase 6 — будет redirect.)
- [ ] Deep-link `/rooms/123/channels/456` без auth → `/login` с `from`-state.
- [ ] После login со `from` → возвращаемся на `from`.
- [ ] Refresh page при залогиненном user-е — `loadMe()` подтягивает профиль, остаёмся на текущем URL.
- [ ] Logout из TopBar — очищает state, redirect на `/login`.
- [ ] Никаких CSS-классов hardcoded в JSX (используем `styles.classname` из `.module.css`).
- [ ] AppShell соответствует layout из `../07-ui-contract.md`.
