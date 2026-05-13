---
phase: 4
name: Auth feature
layer: feature
depends_on: [phase-02, phase-03]
plan: ./README.md
---

# Phase 4: Auth feature

## Цель

Реализовать feature-слайс `web/src/features/auth/`: TS-типы, api-функции (`register/login/refresh/me`), `useAuthStore` с persist, формы `<LoginForm />` и `<RegisterForm />` (react-hook-form + zod), `<UserBadge />`. Создать страницы `LoginPage` и `RegisterPage`. После фазы — login/register работают через реальный бэк, токены сохраняются, profile подгружается; но защищённых роутов ещё нет (это Phase 5).

## Контекст

Phase 2 дала `apiFetch`, `tokenStorage`, `refreshAccessOnce`, `logoutFlow`. Phase 3 дала `<Button />`, `<Input />`, `<FormField />`, `<Avatar />`, `<Spinner />`, `toast`. Контракт auth — `../06-api-integration.md` секция «REST: Auth». Поведение — `../02-behavior.md` UC-1, UC-2, UC-15, UC-16. UI — `../07-ui-contract.md` секция «features/auth». State — `../05-state-model.md` `useAuthStore`. Стандарты — `prompts/Zustand Stores.txt`, `prompts/React Components.txt`.

## Файлы для создания

### `web/src/features/auth/types.ts`

```ts
// DTO (как с бэка, camelCase)
export type RegisterRequest = { email: string; username: string; password: string };
export type LoginRequest = { email: string; password: string };
export type RefreshRequest = { refreshToken: string };
export type UserResponse = { id: string; email: string; username: string; createdAt: string };
export type TokensResponse = {
  accessToken: string;
  refreshToken: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
};

// ViewModel
export type CurrentUser = UserResponse;
```

### `web/src/features/auth/api/http.ts` (+ `.test.ts`)

```ts
export function register(req: RegisterRequest): Promise<ApiResult<UserResponse>>;
export function login(req: LoginRequest): Promise<ApiResult<TokensResponse>>;
export function refresh(req: RefreshRequest): Promise<ApiResult<TokensResponse>>;
export function me(): Promise<ApiResult<CurrentUser>>;
```

- Каждая функция — однострочный вызов `apiFetch(...)`.
- Пути: `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/me` (префикс `/api/v1` подставляется в `apiFetch`).
- Tests (~4): каждая api-функция мокается через `vi.stubGlobal("fetch", ...)`, проверяется URL, method, body, headers.

### `web/src/features/auth/store/mapErrors.ts` (+ `.test.ts`)

```ts
export function mapAuthErrorToMessage(code: string): string;
// AUTH-006 → "Неверный email или пароль"
// AUTH-001 → "Введите корректный email"
// AUTH-002 → "Имя пользователя: 3..32 символа, ASCII alnum/_/-"
// AUTH-003 → "Пароль: 8..72 символа"
// AUTH-004 → "Email уже занят"
// AUTH-005 → "Имя пользователя уже занято"
// INTERNAL → "Внутренняя ошибка, повторите"
// NETWORK → "Нет соединения"
// default → "Неизвестная ошибка"
```

Сообщения берутся из `i18n/ru.ts` (раздел `auth`).

Tests (~3): несколько кодов, дефолтная ветка.

### `web/src/features/auth/store/index.ts` (+ `.test.ts`)

См. `../05-state-model.md` `useAuthStore` для полной структуры и actions.

**Структура:**
```ts
type AuthStatus = "idle" | "loading" | "authenticated" | "error";
type AuthState = { status; currentUser; error; lastErrorCode?: string };
type AuthActions = { register; login; loadMe; logout; clear };
```

**Особенности:**
- `persist` middleware (`zustand/middleware`):
  ```ts
  persist(initializer, {
    name: "rupor.auth",
    version: 1,
    partialize: (s) => ({ currentUser: s.currentUser }),
  })
  ```
- `register`: вызывает `authApi.register`. На ok — автоматически зовёт `login` с теми же `email/password` (см. `02-behavior.md` UC-1).
- `login`: на ok — `tokenStorage.set(...)`, затем `loadMe()`.
- `loadMe`: вызывает `authApi.me`. На ok — `status="authenticated"`, `currentUser=...`. На fail — `status="idle"`, `currentUser=null`.
- `logout`: только tokenStorage.clear() + clear() этого стора. Каскад очистки других сторов выполняется через `logoutFlow` в Phase 9 (когда сторов больше). Сейчас просто:
  ```ts
  logout() {
    tokenStorage.clear();
    get().clear();
  }
  ```
- `clear`: сброс state до initial.

**Tests (~10):** см. `../04-testing.md` секция `useAuthStore`.

**Setup для тестов:**
- `beforeEach`: `useAuthStore.setState(useAuthStore.getInitialState(), true)`, `tokenStorage.clear()`, `vi.restoreAllMocks()`.

### Update `web/src/shared/lib/logoutFlow.ts`

В Phase 2 был TODO. Теперь добавляем:
```ts
import { useAuthStore } from "@/features/auth/store";

export function logoutFlow() {
  tokenStorage.clear();
  useAuthStore.getState().clear();
  // TODO(phase-06+): useRoomsStore.getState().clear()
  // TODO(phase-07+): useChannelsStore.getState().clear()
  // TODO(phase-08+): useChatStore.getState().clear()
  // TODO(phase-09+): wsClient.disconnect()
  logger.info("logout: storage cleared, auth store reset");
}
```

### `web/src/features/auth/components/LoginForm.schema.ts`

```ts
import { z } from "zod";

export const loginSchema = z.object({
  email:    z.string().email("Введите корректный email"),
  password: z.string().min(8, "Минимум 8 символов").max(72, "Максимум 72 символа"),
});
export type LoginValues = z.infer<typeof loginSchema>;
```

### `web/src/features/auth/components/LoginForm.tsx` (+ `.module.css` + `.test.tsx`)

**Props:** `{ onSuccess: () => void }`.

**Логика:**
- `useForm<LoginValues>({ resolver: zodResolver(loginSchema), defaultValues: {...} })`.
- `onSubmit(values)`:
  ```ts
  const ok = await useAuthStore.getState().login(values);
  if (!ok) {
    const code = useAuthStore.getState().lastErrorCode;
    if (code === "AUTH-006") form.setError("root.serverError", { message: ru.auth.invalidCredentials });
    else form.setError("root.serverError", { message: mapAuthErrorToMessage(code ?? "INTERNAL") });
    return;
  }
  onSuccess();
  ```
- Render:
  - `<FormField label="Email" htmlFor="email" error={errors.email?.message}><Input id="email" type="email" {...register("email")} invalid={!!errors.email} /></FormField>`
  - Аналогично password (`type="password"`).
  - `{errors.root?.serverError && <div role="alert">{...}</div>}`.
  - `<Button type="submit" isLoading={isSubmitting}>{ru.auth.loginBtn}</Button>`.
  - `<Link to="/register">{ru.auth.toRegister}</Link>`.

**Tests (~6):** см. `../04-testing.md` секция `<LoginForm />`.

### `web/src/features/auth/components/RegisterForm.schema.ts`

```ts
export const registerSchema = z.object({
  email: z.string().email("..."),
  username: z.string().min(3, "...").max(32, "...").regex(/^[A-Za-z0-9_-]+$/, "..."),
  password: z.string().min(8, "...").max(72, "..."),
});
```

### `web/src/features/auth/components/RegisterForm.tsx` (+ `.module.css` + `.test.tsx`)

Аналогично `LoginForm`, плюс:
- Поле `username`.
- В `onSubmit` маппим коды на поля:
  ```ts
  if (code === "AUTH-001" || code === "AUTH-004") form.setError("email", { message });
  else if (code === "AUTH-002" || code === "AUTH-005") form.setError("username", { message });
  else if (code === "AUTH-003") form.setError("password", { message });
  else form.setError("root.serverError", { message });
  ```

**Tests (~7):** см. `../04-testing.md`.

### `web/src/features/auth/components/UserBadge.tsx` (+ `.module.css` + `.test.tsx`)

**Props:** `{ user: CurrentUser; onLogout: () => void }`.

**Render:**
```tsx
<div className={styles.badge}>
  <Avatar userId={user.id} size="sm" />
  <span className={styles.username} title={user.email}>{user.username}</span>
  <Button variant="ghost" size="sm" onClick={onLogout} aria-label={ru.auth.logoutBtn}>
    {ru.auth.logoutBtn}
  </Button>
</div>
```

**Tests (~3):** см. `../04-testing.md`.

### `web/src/features/auth/index.ts`

```ts
export { useAuthStore } from "./store";
export { LoginForm } from "./components/LoginForm";
export { RegisterForm } from "./components/RegisterForm";
export { UserBadge } from "./components/UserBadge";
export type { CurrentUser, UserResponse } from "./types";
```

### `web/src/pages/LoginPage.tsx` (+ `.module.css`)

```tsx
export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const from = (location.state as { from?: Location })?.from?.pathname ?? "/rooms";

  return (
    <div className={styles.page}>
      <h1>{ru.auth.loginTitle}</h1>
      <LoginForm onSuccess={() => navigate(from, { replace: true })} />
    </div>
  );
}
```

### `web/src/pages/RegisterPage.tsx` (+ `.module.css`)

Аналогично:
```tsx
export function RegisterPage() {
  const navigate = useNavigate();
  return (
    <div className={styles.page}>
      <h1>{ru.auth.registerTitle}</h1>
      <RegisterForm onSuccess={() => navigate("/rooms")} />
    </div>
  );
}
```

## Файлы для модификации

- `web/src/shared/lib/i18n/ru.ts` — расширить раздел `auth` (тексты ошибок, labels полей, заголовки страниц).
- `web/src/shared/lib/logoutFlow.ts` — заменить TODO(phase-04) на реальный вызов `useAuthStore.getState().clear()`.
- `web/src/routes.tsx` — добавить роуты `/login` и `/register` (полная карта — в Phase 5, но эти два сразу нужны для тестов):
  ```ts
  // временно, до Phase 5:
  export const router = createBrowserRouter([
    { path: "/login",    element: <LoginPage /> },
    { path: "/register", element: <RegisterPage /> },
    { path: "*",         element: <div>...</div> },
  ]);
  ```

## Ключевые решения

- **D-02** react-hook-form + zod — используется во всех формах фичи.
- **D-04** localStorage — токены через `tokenStorage` (Phase 2).
- **D-18** Один store на фичу — `useAuthStore` единственный для auth.
- **D-19** Persist только в `useAuthStore.currentUser`.
- **D-11** UUID-placeholder — `<UserBadge />` использует `<Avatar />` (от Phase 3), но имя — реальный `user.username`.

## Verification

- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все тесты auth-фичи (~30) зелёные.
- [ ] LoginForm покрывает: render, Tab navigation, Enter submits, form-level error при AUTH-006.
- [ ] RegisterForm: каждый из AUTH-001..005 маппится на конкретное поле.
- [ ] `useAuthStore.persist` восстанавливает `currentUser` после reload (mock localStorage).
- [ ] `login` сохраняет токены через `tokenStorage.set` (test verifies).
- [ ] `logout` чистит `tokenStorage` и `useAuthStore.currentUser` (test verifies).
- [ ] Сообщения ошибок берутся из `i18n/ru.ts`, не литералы в JSX.
- [ ] Нет `any`, нет `dangerouslySetInnerHTML`.
- [ ] LoginPage и RegisterPage доступны через `/login`, `/register` в Vite dev.
- [ ] Ручная проверка: `npm run dev`, открыть `http://localhost:5173/register`, попробовать зарегистрироваться (требует поднятый `make run`).
