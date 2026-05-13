---
parent: ./README.md
feature: 3_5-frontend
---

# 3.5 Frontend — Standards Compliance

Compliance-матрица по 6 файлам стандартов из `prompts/`. Для каждого — статус и ключевые точки compliance в этой фиче.

| Стандарт | Статус | Ключевые точки compliance |
|----------|--------|---------------------------|
| `Frontend Architecture Layers.txt` | ✅ | Корневая структура `web/src/{app,pages,features,shared,routes.tsx}` (`01-architecture.md` L2). Зависимости `app → pages → features → shared`, нет cross-feature импортов. Единый fetch/WS в `shared/api/` (`06-api-integration.md`). |
| `TypeScript Style.txt` | ✅ | strict-tsconfig + `noUncheckedIndexedAccess` + `exactOptionalPropertyTypes` (`03-decisions.md` D-04 неявно требует). DTO с суффиксами `*Dto/*Request/*Response/*Frame` (`06-api-integration.md`). ViewModel — отдельно (`01-architecture.md` L3). Discriminated union для `IncomingFrame` (`06-api-integration.md`). Запрет `any` (см. правила реализации). |
| `React Components.txt` | ✅ | Все 5 состояний (idle/loading/empty/error/success/disabled) перечислены для каждого компонента (`07-ui-contract.md`). a11y: `aria-label`, `role`, keyboard (Enter/Esc/Tab). Запрет `dangerouslySetInnerHTML` — текст рендерится как text-node, ссылки через `linkify` с валидацией протокола. |
| `Zustand Stores.txt` | ✅ | По одному store на фичу: `useAuthStore`, `useRoomsStore`, `useChannelsStore`, `useChatStore` (`05-state-model.md`). Узкие селекторы. Side-effects только в actions. `clear()` для user-data во всех сторах. `persist` только для `useAuthStore` (`03-decisions.md` D-19). WS-handlers → store actions, не `set` напрямую. |
| `API Integration.txt` | ✅ | Единственные `shared/api/fetch.ts` и `shared/api/ws.ts` (`01-architecture.md` L3, `06-api-integration.md`). `ApiResult<T>` для всех api-функций. Single-flight refresh (`06-api-integration.md` Auto-refresh). WS reconnect восстанавливает подписки (`06-api-integration.md`). Без `credentials: "include"`. Токены в `Authorization: Bearer` через `token-storage.ts`. |
| `Tests Style (Web).txt` | ✅ | AAA-паттерн в шаблонах тестов (`04-testing.md`). Селекторы по role/label. Coverage mapping для каждого backend error code. Моки через `vi.spyOn`. WS — подмена фабрики `createWsClient`. E2E против реального бэка. |

## Уточнения и отклонения

### Persist в `useAuthStore` хранит только `currentUser`

Стандарт `Zustand Stores.txt` рекомендует осторожность с persist чувствительных данных. Решение:
- Токены кладём в **отдельный** модуль `tokenStorage` (localStorage с явными ключами `rupor.access`, `rupor.refresh`, `rupor.accessExpiresAt`, `rupor.refreshExpiresAt`).
- В Zustand persist'им только `currentUser` (UserResponse — id/email/username/createdAt).
- При reload — `loadMe()` обновляет `currentUser` из бэка (так свежая инфа важнее закэшированной).

Это соответствует духу стандарта: токены — отдельная инфраструктура, store — бизнес-state.

### Логирование токенов

`API Integration.txt` запрещает логирование токенов. Реализация:
- Единый `logger` в `shared/lib/logger.ts`.
- В dev — `console.*`, в prod — только `error`.
- `Authorization` header не логируется (в fetch-обёртке).
- `?token=` в WS-URL: значение маскируется (`logger.debug("ws connect ?token=REDACTED")`).

### Voice-каналы

CRUD на бэке есть, сигналинга нет. UI рендерит `voice` как disabled item (см. `07-ui-contract.md` `<ChannelListItem />`). Это **расширение** к стандарту — стандарты не покрывают эту специфику Rupor. Зафиксировано в `03-decisions.md` D-12.

### UUID-placeholder для никнеймов

Бэк не отдаёт `username` в `members/message.new`. Стандарты не покрывают эту специфику. Решение в `03-decisions.md` D-11 + `<Avatar />` в `07-ui-contract.md`. После MVP — добавить `/users/{id}` на бэке.

### Cascade logout

`useAuthStore.logout()` явно вызывает `clear()` у `useRoomsStore`, `useChannelsStore`, `useChatStore`. Это **прямая зависимость между сторами** (cross-store), что слегка расходится с правилом изоляции из `Zustand Stores.txt`. Митигация: импорт делается лениво через `useXxxStore.getState()` (нет циклов на уровне модулей), и это **только** в одном action — logout. Альтернатива (shared `logoutFlow` функция) — будет рассмотрена в `plan/`.

### `tabIndex` в модалках

Стандарт `React Components.txt` запрещает `tabIndex={-1}` на видимых интерактивных. В focus-trap внутри `<Modal />` мы используем библиотеку (`focus-trap-react` или ручной) — она ставит `tabIndex` на overlay временно для управления фокусом. Это соответствует ARIA-практикам и **не нарушает дух** правила (запрет был про скрытие интерактивных от Tab; здесь — наоборот, удерживание фокуса в диалоге).

---

## Соответствие спецификам Rupor

Из `.thoughts/research/2026-05-13-3_5-frontend-baseline.md`:

| Специфика | Где реализована |
|-----------|------------------|
| REST = camelCase, WS = snake_case | TS-типы в `06-api-integration.md` строго следуют |
| `Authorization: Bearer`, `allowCredentials=false` | `06-api-integration.md`, без `credentials: "include"` |
| Auto-refresh single-flight | `06-api-integration.md` + `02-behavior.md` UC-3 |
| Logout локальный, без эндпоинта | `02-behavior.md` UC-15 |
| WS auto-room-subscribe только при connect | `02-behavior.md` UC-8 (reconnect после join) |
| Reconnect WS после `/rooms/join/{code}` | `03-decisions.md` D-10, `05-state-model.md` `useRoomsStore.joinByCode` |
| DTO members без `username` | `03-decisions.md` D-11, `<MemberListItem />` и `<Avatar />` |
| Voice — заглушка | `03-decisions.md` D-12, `<ChannelListItem />` disabled |
| Multi-tab дедупликация по `id` | `03-decisions.md` D-08, `useChatStore.onMessageNew` |
| Курсор истории `before=<message-id>` UUID | `06-api-integration.md` + `useChatStore.loadMoreHistory` |
| Empty members не бывает | `<MembersList />` defensive «—» |
| CORS default `http://localhost:5173` | Vite dev на этом порту, прокси настроен |

---

## Открытые точки (фиксируются перед началом реализации)

Из `03-decisions.md` Open Questions:

- Темизация: подразумевается тёмная Discord-like (см. `07-ui-contract.md` CSS-переменные). Финальный голос — за пользователем.
- i18n: строки выносятся в `web/src/shared/lib/i18n/ru.ts` для будущего перевода.
- Toaster: собственная реализация в `shared/ui/Toaster.tsx`.
- Avatar palette: 8 цветов с AA-контрастом, фиксируется в `shared/lib/uuidToColor.ts`.

Эти решения принимает Lead implementer'а при первой phase плана (`Bootstrap`), либо подтверждаются пользователем перед утверждением плана.
