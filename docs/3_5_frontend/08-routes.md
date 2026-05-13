---
parent: ./README.md
feature: 3_5-frontend
---

# 3.5 Frontend — Routes & Guards

Все роуты декларируются единственно в `web/src/routes.tsx` (см. `prompts/Frontend Architecture Layers.txt`). Никаких «теневых» роутов внутри фичей.

## Карта роутов

| Path | Page Component | Guards | Описание |
|------|----------------|--------|----------|
| `/` | redirect → `/rooms` (если `authenticated`) или `/login` (иначе) | — | Корневой роут |
| `/login` | `<LoginPage />` | `redirectIfAuthenticated` (если уже залогинен → `/rooms`) | Экран логина |
| `/register` | `<RegisterPage />` | `redirectIfAuthenticated` | Экран регистрации |
| `/rooms` | `<RoomsIndexPage />` | `requireAuth` | Список комнат (empty-state если нет; редирект на первую если есть) |
| `/rooms/:roomId` | `<RoomPage />` | `requireAuth` + `requireMembership(:roomId)` | Просмотр комнаты без выбранного канала (показывает «Выбери канал слева») |
| `/rooms/:roomId/channels/:channelId` | `<ChatRoutePage />` | `requireAuth` + `requireMembership(:roomId)` + `requireChannelInRoom(:roomId,:channelId)` | Основной чат |
| `*` | `<NotFoundPage />` | — | Fallback |

## Структура `routes.tsx`

```tsx
import { createBrowserRouter, Navigate } from "react-router-dom";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <Navigate to="/rooms" replace />,
  },
  {
    path: "/login",
    element: <RedirectIfAuthenticated><LoginPage /></RedirectIfAuthenticated>,
  },
  {
    path: "/register",
    element: <RedirectIfAuthenticated><RegisterPage /></RedirectIfAuthenticated>,
  },
  {
    path: "/rooms",
    element: <RequireAuth><AppShell /></RequireAuth>,
    children: [
      { index: true, element: <RoomsIndexPage /> },
      {
        path: ":roomId",
        element: <RequireMembership><RoomOutlet /></RequireMembership>,
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

Подключение — в `app/App.tsx`:

```tsx
import { RouterProvider } from "react-router-dom";
import { router } from "./routes";

export function App() {
  return (
    <ErrorBoundary>
      <RouterProvider router={router} />
      <Toaster />
    </ErrorBoundary>
  );
}
```

## Гарды

### `<RequireAuth>`

**Назначение:** пускает только authenticated пользователей. Иначе redirect на `/login` с сохранением `from` в state.

```tsx
function RequireAuth({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  const location = useLocation();
  if (status === "idle" || status === "error") {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  if (status === "loading") return <FullScreenSpinner />;
  return <>{children}</>;
}
```

При начальной загрузке (`status === "loading"`) — показываем spinner, пока `loadMe` не завершится.

### `<RedirectIfAuthenticated>`

**Назначение:** на `/login` и `/register` не пускаем уже залогиненных.

```tsx
function RedirectIfAuthenticated({ children }: { children: ReactNode }) {
  const status = useAuthStore((s) => s.status);
  if (status === "authenticated") {
    return <Navigate to="/rooms" replace />;
  }
  return <>{children}</>;
}
```

### `<RequireMembership>`

**Назначение:** проверяет, что `:roomId` есть в `useRoomsStore.rooms`. Иначе — redirect `/rooms` + toast.

```tsx
function RequireMembership({ children }: { children: ReactNode }) {
  const { roomId } = useParams<{ roomId: string }>();
  const room = useRoomsStore((s) => s.rooms.find((r) => r.id === roomId));
  const status = useRoomsStore((s) => s.status);

  // ещё не загрузились
  if (status === "idle" || status === "loading") return <FullScreenSpinner />;

  if (!room) {
    toast.error("Комната не найдена или вы в ней не состоите");
    return <Navigate to="/rooms" replace />;
  }
  return <>{children}</>;
}
```

### `<RequireChannelInRoom>`

**Назначение:** проверяет, что `:channelId` есть в `useChannelsStore.channelsByRoom[:roomId]`. Иначе — redirect `/rooms/:roomId`.

```tsx
function RequireChannelInRoom({ children }: { children: ReactNode }) {
  const { roomId, channelId } = useParams<{ roomId: string; channelId: string }>();
  const channels = useChannelsStore((s) => s.channelsByRoom[roomId!] ?? []);
  const loading = useChannelsStore((s) => s.loadingByRoom[roomId!]);

  if (loading) return <FullScreenSpinner />;

  const channel = channels.find((c) => c.id === channelId);
  if (!channel) {
    toast.error("Канал не найден");
    return <Navigate to={`/rooms/${roomId}`} replace />;
  }
  if (channel.kind === "voice") {
    toast.info("Голосовые каналы пока недоступны");
    return <Navigate to={`/rooms/${roomId}`} replace />;
  }
  return <>{children}</>;
}
```

## Layout-композиция

`<AppShell />` — родительский layout для всех авторизованных роутов под `/rooms/...`. Содержит:

- `<TopBar />` сверху (с `<UserBadge />`).
- Левый sidebar с `<RoomList />` + (когда выбран roomId) `<ChannelList />` + `<MembersList />`.
- Центральная панель — `<Outlet />` из react-router (рендерит вложенный роут).

```tsx
function AppShell() {
  return (
    <div className={styles.shell}>
      <TopBar />
      <Sidebar>
        <RoomList />
        <RoomScopedSidebar />  {/* виден только когда :roomId */}
      </Sidebar>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  );
}
```

`<RoomScopedSidebar />` — вложенный sidebar, виден только когда `:roomId` есть в URL. Содержит `<ChannelList />` сверху и `<MembersList />` снизу. Полная спецификация — `07-ui-contract.md`.

## Глубокие ссылки

### Сценарий: пользователь открывает `/rooms/:rid/channels/:cid` напрямую

1. `<RequireAuth>` — если не залогинен → `/login` с `from = original URL`.
2. После логина — `navigate(from)` восстанавливает URL.
3. `<RequireMembership>` ждёт `loadRooms` и проверяет membership.
4. `<RequireChannelInRoom>` ждёт `loadChannels(roomId)` и проверяет наличие.
5. `<ChatRoutePage>` рендерит `<ChatPanel channelId={cid} />`.
6. `<ChatPanel>` через `useEffect` вызывает `useChatStore.openChannel(cid)`.

### Сценарий: deep-link на удалённую комнату/канал

- `<RequireMembership>` или `<RequireChannelInRoom>` — redirect на безопасный уровень + toast.

### Сценарий: deep-link до завершения `loadMe`

- `<RequireAuth>` показывает `<FullScreenSpinner />`, пока `status === "loading"`.
- После `loadMe()`: если ok → продолжаем, если fail → `/login`.

## Фоллбэк навигация

- При logout — `navigate("/login", { replace: true })`. Не сохраняем `from`.
- При 404 от бэка на нечто доступное в пути — соответствующий гард чистит стор и редиректит на уровень выше.
- При полностью неизвестном URL — `<NotFoundPage />` со ссылкой «На главную» → `/rooms`.

## URL как persistent state экрана

Открытые `:roomId` и `:channelId` в URL — единственное «persisted состояние UI» (см. `05-state-model.md`). Это значит:
- Reload страницы сохраняет текущий экран.
- Bookmark/share URL — работает.
- Открытие в новой вкладке — попадает в тот же канал.

## `useNavigate` discipline

- Гарды и компоненты — единственные места, где импортируется `useNavigate`/`<Navigate>`.
- Zustand-сторы НЕ импортируют `useNavigate` (компонент-обёртка вызывает navigate после store-action).
- Это согласуется с `prompts/Zustand Stores.txt`: side-effects router'а — снаружи стора.

## Прокси и nginx (для production deploy)

В prod nginx-конфиге настраиваем SPA-fallback для всех путей (кроме `/api/v1/*`):

```nginx
location / {
  try_files $uri $uri/ /index.html;
}

location /api/v1/ {
  proxy_pass http://api:8080;
  proxy_http_version 1.1;
  proxy_set_header Upgrade $http_upgrade;
  proxy_set_header Connection "upgrade";
  proxy_set_header Host $host;
}
```

`Upgrade`/`Connection` нужны для WebSocket-апгрейда. См. `docker-compose.yml` (модификация в плане кода).
