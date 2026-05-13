---
phase: 2
name: Shared API foundation
layer: shared
depends_on: [phase-01]
plan: ./README.md
---

# Phase 2: Shared API foundation

## Цель

Реализовать сетевой фундамент в `web/src/shared/api/`: типы (`ApiResult`, `ErrorEnvelope`, WS-фреймы), хранилище токенов в localStorage, single-flight refresh и fetch-обёртку с auto-refresh + logout flow. Юнит-тесты для всех модулей. После фазы — любой компонент/store может вызывать `apiFetch<T>(path, init)` и получать `Promise<ApiResult<T>>` с корректной обработкой 401.

## Контекст

Phase 1 создала каркас `web/`, ESLint/TS-конфиги, пустую папку `web/src/shared/api/`. Используем стандарты из `prompts/API Integration.txt`, `prompts/TypeScript Style.txt`. Базовый контракт — `../06-api-integration.md`, специфики — ADR `../03-decisions.md` D-04 (localStorage), D-05 (single-flight), D-06 (locgout-flow без эндпоинта).

## Файлы для создания

### `web/src/shared/api/errors.ts`

**Назначение:** общие TS-типы, без runtime-кода.

**Детали реализации:**
- `ErrorEnvelope` — `{ error: { code: string; message: string } }`.
- `ApiOk<T>`, `ApiErr`, `ApiResult<T>` — см. `../06-api-integration.md`.
- Литералы кодов: `AuthErrorCode`, `RoomErrorCode`, `ChannelErrorCode`, `ChatErrorCode`, `ClientErrorCode = "NETWORK" | "CLIENT" | "INTERNAL"`.
- TS-типы WS-фреймов: `SubscribeFrame`, `MessageSendFrame`, `OutgoingFrame`, `SubscribedFrame`, `MessageNewFrame`, `MessageSentFrame`, `MemberJoinedFrame`, `ErrorFrame`, `IncomingFrame`. **snake_case в payload — strict**.

Точные определения — `../06-api-integration.md` секции «Общие типы», «Outbound фреймы», «Inbound фреймы».

### `web/src/shared/api/token-storage.ts`

**Назначение:** обёртка над localStorage для access/refresh + expires.

**Сигнатура:**
```ts
export type StoredTokens = {
  access: string;
  refresh: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
};

export const tokenStorage: {
  getAccess(): string | null;
  getRefresh(): string | null;
  getAccessExpiresAt(): string | null;
  getRefreshExpiresAt(): string | null;
  set(tokens: StoredTokens): void;
  clear(): void;
};
```

**Детали реализации:**
- Ключи: `rupor.access`, `rupor.refresh`, `rupor.accessExpiresAt`, `rupor.refreshExpiresAt`.
- Все методы синхронные.
- `set` — атомарность не гарантирована (localStorage не транзакционен), но мы пишем в неизменном порядке и читаем индивидуально (не нужна атомарность).
- НЕ логировать значения (logger.debug запрещён для содержимого).

### `web/src/shared/api/refresh.ts`

**Назначение:** single-flight refresh access-токена.

**Сигнатура:**
```ts
export function refreshAccessOnce(): Promise<ApiResult<TokensResponse>>;
```

**Детали реализации:**
- Глобальный модульный `let currentRefresh: Promise<ApiResult<TokensResponse>> | null = null;`.
- Если `currentRefresh != null` — возвращаем тот же promise.
- Если `tokenStorage.getRefresh()` отсутствует — резолвим как `ok: false, status: 401, error: { code: "AUTH-007", message: "no refresh token" }` (имитируем поведение бэка).
- Иначе зовём `doRefresh(refreshToken)` (внутренняя функция, делающая `fetch("/api/v1/auth/refresh", {...})` БЕЗ зависимости от `apiFetch` чтобы избежать рекурсии).
- При ok — `tokenStorage.set(...)`.
- В `.finally(() => { currentRefresh = null; })`.
- НЕ логировать тело refresh.

### `web/src/shared/api/fetch.ts`

**Назначение:** единая fetch-обёртка с auto-refresh.

**Сигнатура:**
```ts
export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<ApiResult<T>>;
```

**Детали реализации:**
- Базовый URL: префикс `/api/v1` подставляется ЕСЛИ `path` начинается с `/` И НЕ начинается с `/api`. Иначе используем path как есть. (В тестах — мокаем fetch, базовый URL не важен.)
- Подставлять `Authorization: Bearer <access>` если `tokenStorage.getAccess()` есть.
- Если `init.body` и нет `Content-Type` — выставляем `application/json`.
- Парсим JSON ответа всегда (через `await res.text(); JSON.parse(text)`), чтобы обработать пустое тело (`204 No Content` → `data: null as T`).
- Если `!res.ok`:
  - Если status === 401:
    - Если `path === "/auth/refresh"` — НЕ запускаем refresh (loop protection). Если код `AUTH-007/008/009` — `logoutFlow()`, возвращаем как есть.
    - Иначе парсим code:
      - `AUTH-011`: `refreshAccessOnce()`. На fail refresh → logoutFlow + возвращаем исходный 401. На ok — `doFetch` повторно. Если повторный тоже 401 AUTH-011 — logoutFlow + возвращаем второй ответ.
      - `AUTH-007/008/009/010`: logoutFlow + возвращаем как есть.
      - Другие коды на 401 — возвращаем без специальной обработки.
  - Возвращаем `{ ok: false, status, error: parsed.error ?? { code: "CLIENT", message: "unknown" } }`.
- На network error / non-JSON body:
  - `{ ok: false, status: 0, error: { code: "NETWORK", message: "network error" } }`.

**Side-effects:**
- `logoutFlow()` (из `web/src/shared/lib/logoutFlow.ts`) — каскадная очистка. См. ниже.

### `web/src/shared/lib/logoutFlow.ts`

**Назначение:** функция, которую вызывает любой код для logout. Не вызывает `useNavigate` (роутер делает это в компонентах через подписку на authStore).

**Сигнатура:**
```ts
export function logoutFlow(): void;
```

**Детали реализации:**
- `tokenStorage.clear()`.
- В этой фазе сторов ещё нет — функция содержит TODO-комментарий с упоминанием Phase 4-9. Реально стор будут добавлены позже:
  ```ts
  export function logoutFlow() {
    tokenStorage.clear();
    // TODO(phase-04+): useAuthStore.getState().clear()
    // TODO(phase-06+): useRoomsStore.getState().clear()
    // TODO(phase-07+): useChannelsStore.getState().clear()
    // TODO(phase-08+): useChatStore.getState().clear()
    // TODO(phase-09+): wsClient.disconnect()
    logger.info("logout: storage cleared");
  }
  ```
- Это исключение к правилу «без TODO» — оно с явной фазой-владельцем и будет убрано в последующих фазах.

### Тесты (vitest)

#### `web/src/shared/api/token-storage.test.ts` (~3 теста)

См. `../04-testing.md`.

#### `web/src/shared/api/refresh.test.ts` (~3 теста)

- первый вызов делает POST /auth/refresh
- параллельные вызовы получают один и тот же promise
- после resolve `currentRefresh` сбрасывается в null

Использовать `vi.stubGlobal("fetch", ...)`.

#### `web/src/shared/api/fetch.test.ts` (~10 тестов)

См. `../04-testing.md`.

**Setup для тестов с refresh:**
- Перед каждым тестом — `tokenStorage.clear()`, `vi.unstubAllGlobals()`, ресет `currentRefresh` (экспортируем `_resetRefreshForTests()` в refresh.ts, dev-only).

## Файлы для модификации

В этой фазе — никаких модификаций существующих файлов. Phase 1 создала пустые `.gitkeep` в `shared/api/` — их можно оставить или удалить.

## Ключевые решения

- **D-04** localStorage — все 4 ключа `rupor.*`.
- **D-05** Single-flight refresh — реализован в `refresh.ts`, защита от refresh-loop — в `fetch.ts`.
- **D-06** Logout без эндпоинта — функция `logoutFlow()` в `shared/lib`, не зовёт бэк.

## Verification

- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все ~16 тестов в `shared/api/` зелёные.
- [ ] `apiFetch` корректно подставляет `Authorization: Bearer` (test verifies).
- [ ] При 401 AUTH-011 запускается refresh ровно один раз для N параллельных запросов (test verifies).
- [ ] При 401 на сам `/auth/refresh` повторного refresh нет (test verifies).
- [ ] Загрузка/чтение токенов из localStorage работает идемпотентно (test verifies round-trip).
- [ ] Нет логов с содержимым токенов в любых сценариях (ручная проверка `console.log`-выводов).
- [ ] Все типы DTO/WS-фреймов совпадают с `../06-api-integration.md` побайтно (поля и регистр).
