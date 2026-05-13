---
parent: ./README.md
view: decision
feature: 3_5-frontend
---

# 3.5 Frontend — Решения и риски (Decision View)

## Решения

| # | Решение | Выбор | Альтернативы | Обоснование |
|---|---------|-------|--------------|-------------|
| D-01 | Router | **react-router-dom v6** | TanStack Router; wouter | Mainstream, привычно команде, простая RTL-интеграция через `MemoryRouter`. Для MVP с 5-6 роутами type-safety TanStack не окупает overhead. |
| D-02 | Управление формами | **react-hook-form + zod** | useState вручную | 4 формы (login, register, create room, join by code) + composer чата. Zod-схема даёт single source of truth для валидации и TS-типов; `setError` по доменному коду — стандартный путь для серверных ошибок. См. `prompts/React Components.txt`. |
| D-03 | Стилизация | **CSS Modules** | Tailwind; vanilla CSS + BEM | Нативно для Vite, scoped-классы, ноль runtime. Для Discord-like с собственными компонентами без дизайн-системы. TS-типизация классов — `vite-plugin-css-modules-types-loader` или vanilla approach. |
| D-04 | Хранение токенов | **localStorage** | in-memory only; sessionStorage; HttpOnly cookies | `allowCredentials=false` на бэке → cookies не вариант. In-memory ломает UX при reload в чат-приложении (постоянный re-login). XSS-риск митигируется общими React-правилами (см. `prompts/React Components.txt`: запрет `dangerouslySetInnerHTML`, экранирование). |
| D-05 | Auto-refresh access-токена | **Single-flight в `shared/api/fetch.ts`** | Refresh при каждом запросе; refresh по таймеру до истечения | Триггер — 401 `AUTH-011`. Глобальный `currentRefresh: Promise \| null` объединяет параллельные 401 в один вызов `/auth/refresh`. После — retry оригинальных запросов с новым access. Защита от loop: если retry-after-refresh снова даёт 401 AUTH-011 → logout. |
| D-06 | Logout-flow | **Локальная очистка** | Вызов `/auth/logout` (НЕТ эндпоинта) | Бэк не имеет logout-эндпоинта (см. `research.md`). Логика: `tokenStorage.clear()` → `useAuthStore.clear()` → cascade `clear()` остальных сторов → `navigate("/login")`. WS-клиент при потере токена сам disconnect-ится. |
| D-07 | Optimistic update сообщений | **Temp-id + реконсиляция по committed-id** | Без optimistic (ждать `message.sent`); only optimistic без реконсиляции | Чат должен ощущаться мгновенным. Сообщение вставляется в `useChatStore` с `tempId: uuid v4 локально`, статус `pending`. На `message.sent` (ack автору) — статус `committed`, привязка к серверному id. На приходящий `message.new` с уже committed id — игнорируем дубль. На ошибку — статус `failed` + UI «попробовать снова». |
| D-08 | Реконсиляция multi-tab дублей | **Дедупликация по `id` в `useChatStore.appendMessage`** | TabId-based фильтр; broadcast channel | Вкладка A отправляет → получает `message.sent` и `message.new` (если подписана). Вкладка B — только `message.new`. Action `appendMessage` проверяет `messages.some(m => m.id === incoming.id)` и не добавляет дубль. Простой и работает в любом числе вкладок. |
| D-09 | WS reconnect | **Exponential backoff `1s, 2s, 5s, 15s, 15s...` с восстановлением подписок** | Бесконечный backoff без верхней границы; reconnect только по таймауту | Backoff сглаживает рестарты сервера. После успешного reconnect — `useChatStore` отправляет `subscribe` для текущего открытого канала. Подписки на `room:<id>` восстанавливаются сервером автоматически (handler делает auto-subscribe при connect). |
| D-10 | Reconnect WS после `POST /rooms/join/{code}` | **Принудительный reconnect WS** | Игнорировать (новые `member.joined` не придут); polling | Бэк делает auto-room-subscribe только в момент WS-connect. Вступление в новую комнату через REST НЕ добавит подписку к открытому WS. Action `useRoomsStore.joinByCode` после успеха вызывает `wsClient.reconnect()`. |
| D-11 | Source-of-truth для никнеймов | **UI placeholder на основе UUID** | Кешировать ники при login; отдельный endpoint `/users/{id}` | Бэк не отдаёт `username` в DTO членов / `message.new` (ADR D-09 бэка). Создавать новый эндпоинт — out of scope PR-3.5. Решение: отображать первые 6 символов UUID (например, `c5f2b3`) с инициалами-аватаром. Текущего пользователя знаем по `/auth/me` → его UUID мапится на свой username в UI. Полноценные ники — отдельная фаза в `after_mvp_plan.md`. |
| D-12 | Voice-каналы в UI | **Disabled list-item с подписью «voice — coming soon»** | Скрывать; рендерить как text | Бэк CRUD умеет, но сигналинга нет (`internal/voice/` — заглушка). Чтобы не дезориентировать пользователя — отображаем, но не кликабельно. |
| D-13 | Структура папок | **Feature-based** (`web/src/features/<feature>/{components,api,store,types}`) | Layers-based; domain-based (зеркало internal/) | Выбор зафиксирован при создании `design_frontend_feature` команды. Изоляция фич, единый паттерн масштабирования. См. `prompts/Frontend Architecture Layers.txt`. |
| D-14 | Тестовый стек | **vitest + RTL + Playwright** | jest+RTL; cypress | Vitest нативен Vite, скорость и совместимость out-of-box. Playwright проще CI-настройки чем Cypress, поддерживает multi-tab сценарии (нужно для теста `message.sent` vs `message.new`). См. `prompts/Tests Style (Web).txt`. |
| D-15 | Моки для unit-тестов | **`vi.spyOn` на api-функции; ручная фабрика для WS** | MSW для всех слоёв | MSW отлично для тестов `shared/api/fetch.ts`, но для тестов сторов overkill — spyOn на конкретную api-функцию читаемее. WS мокается подменой фабрики (`createWsClient`) — в тестах возвращаем in-memory implementation. |
| D-16 | Vite dev proxy | **Прокси `/api/v1` на `http://localhost:8080`** | Прямой fetch с CORS | Vite dev на `5173`, бэк на `8080`. Без прокси нужен CORS preflight каждый раз. Прокси — стандартная практика, в проде nginx делает то же. CORS на бэке всё равно настроен для localhost:5173 — двойная защита. |
| D-17 | Production-сборка фронта | **multi-stage Dockerfile: `node:lts-alpine` builder → `nginx:alpine` runtime** | Vite build на хост-машине; serve через бэк | Реалистичная prod-сборка для деплоя. Nginx раздаёт статику + проксирует `/api/v1` на бэк. В `docker-compose.yml` добавляется сервис `web`. |
| D-18 | State management | **Один store на фичу (`useAuthStore`, `useRoomsStore`, `useChannelsStore`, `useChatStore`)** | Один глобальный store; контексты | Изоляция, ясные границы, легко тестировать. См. `prompts/Zustand Stores.txt`. |
| D-19 | Persist в Zustand | **Только `useAuthStore` (токены + tokens expires)** | Persist всех сторов | Остальные сторы (rooms, channels, chat) рефетчатся при mount и при reconnect. Persist данных сообщений — лишний state, который быстро стухает. |
| D-20 | Selector helper | **Прямые селекторы + `shallow` из `zustand/shallow` при нужде** | `reselect`; кастомный memo-helper | Zustand нативно поддерживает custom equality. Для tuple/object-селекторов используем `shallow`. Нет необходимости в внешней библиотеке. |
| D-21 | Локаль | **Только русский (ru)** | i18n с самого начала | CLAUDE.md фиксирует русский MVP, английский — после MVP. Никаких i18n-библиотек, все строки — literal в коде (но вынесены в файл `web/src/shared/lib/i18n/ru.ts` для будущего перехода). |
| D-22 | Темизация | **Тёмная (Discord-like), CSS-переменные в `app/styles/theme.css`** | Светлая; system-detect | Discord-аналог по UX. Светлая тема — после MVP, через подмену CSS-переменных. |
| D-23 | Размещение строк UI | **`web/src/shared/lib/i18n/ru.ts` с экспортом объекта** | Литералы в JSX | Готовит почву к будущему переводу без переписывания компонентов. Объект — не функция (`t("key")`), это шаг 1 минимализма. |
| D-24 | Toast-библиотека | **Собственный `<Toaster />` в `shared/ui/`** | `sonner` / `react-hot-toast` | Свой ~5KB укладывается в дизайн без зависимости. Минимальный API: `toast.info/success/error/warn`. |
| D-25 | Avatar palette | **8 фиксированных цветов с AA-контрастом в `shared/lib/uuidToColor.ts`** | Палитра из HSL по hash; внешняя библиотека | Детерминированная палитра, проще тестировать и поддерживать. Алгоритм: `hash(uuid) % 8 → palette[i]`. |

## Риски и митигация

| # | Риск | Влияние | Митигация |
|---|------|---------|-----------|
| R-01 | XSS через сообщения чата (теоретически: bad-actor вставляет HTML) | Medium | React по умолчанию экранирует text-node. Запрет `dangerouslySetInnerHTML` в `prompts/React Components.txt`. Линковка URL — отдельный проход с протоколом-валидацией (только `http(s):`/`mailto:`). |
| R-02 | Утечка access-токена через XSS из localStorage | Medium | Принимаемый риск для MVP. Митигация: запрет inline-скриптов, CSP-заголовок при деплое, ESLint-правило `react/no-danger`. После MVP — рассмотреть `Refresh in HttpOnly cookie + access in memory` с поддержкой бэка. |
| R-03 | Refresh-loop при «вечно битом» refresh-токене | High | Single-flight + circuit-breaker (D-05): если retry после refresh даёт 401 AUTH-011, logout. Если сам `/auth/refresh` 401 → logout без повтора. |
| R-04 | WS-disconnect незаметен пользователю | Medium | `ConnectionStatusBanner` в TopBar показывает `wsStatus`. При `reconnecting` — жёлтый баннер. При `closed` (после max-attempts; впрочем backoff бесконечный) — логичное сообщение. |
| R-05 | Дублирование сообщений в multi-tab | Low | Дедупликация по `id` в `useChatStore.appendMessage` (D-08). |
| R-06 | Пользователь после `join` не видит новых событий в комнате | Medium | Принудительный reconnect WS в `useRoomsStore.joinByCode` (D-10). |
| R-07 | Длинная история чата (10k+ сообщений) тормозит рендер | Medium | MVP-стратегия: рендерим только подгруженный кусок (по умолчанию `limit=50` на страницу). После MVP — virtualization (`react-window` или `@tanstack/react-virtual`). |
| R-08 | Невозможно показать `username` для других пользователей | Low | UI placeholder с кратким UUID (D-11). После MVP — добавить `/users/{id}` на бэке. |
| R-09 | Утечка токена в логи / Sentry / console | High | ESLint-правило `no-console` + единый `shared/lib/logger.ts`. В `shared/api/fetch.ts` `Authorization`-header не логируется. `?token=` в WS-URL уже маскируется бэком (`pkg/httpx/middleware/logger.go`) — клиент тоже не должен его эхить в console. |
| R-10 | CORS-преflight отказ при деплое (origin не в whitelist) | Medium | В `.env` бэка добавить prod-origin в `CORS_ALLOWED_ORIGINS`. Документировать в `08-routes.md` / README. |
| R-11 | Voice-каналы создаются и засоряют list, юзер не понимает почему «нельзя войти» | Low | D-12: disabled list-item с явной подписью. |
| R-12 | Bundle size растёт неконтролируемо | Low | Vite + tree-shaking. Запрет тяжёлых зависимостей без согласования (`TypeScript Style.txt`). Bundle-size-budget — отслеживается в final review. |

## Open Questions

- [x] **Хранение токенов**: localStorage. *Ответ: D-04, принимаем XSS-риск с митигациями R-02.*
- [x] **Routing**: react-router-dom v6. *Ответ: D-01.*
- [x] **Формы**: react-hook-form + zod. *Ответ: D-02.*
- [x] **Стилизация**: CSS Modules. *Ответ: D-03.*
- [x] **Никнеймы**: UI placeholder с UUID. *Ответ: D-11, полноценная фича — после MVP.*
- [x] **Voice-каналы**: disabled list-item. *Ответ: D-12.*
- [x] **Темизация**: тёмная Discord-like, CSS-переменные. *Ответ: D-22.*
- [x] **i18n-файл**: `shared/lib/i18n/ru.ts`. *Ответ: D-23.*
- [x] **Toast-библиотека**: собственный `<Toaster />`. *Ответ: D-24.*
- [x] **Avatar palette**: 8 фиксированных цветов AA-контраст. *Ответ: D-25.*
