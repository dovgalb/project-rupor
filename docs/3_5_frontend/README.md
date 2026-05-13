---
date: 2026-05-13
feature: 3_5-frontend
status: draft
research: ./research.md
---

# PR-3.5: Веб-клиент Rupor — Документы дизайна

## Бизнес-контекст

Rupor — коммуникационная платформа (аналог Discord). Бэкенд на Go реализован: реализованы домены `auth` (регистрация/логин/refresh/me), `room` (CRUD + инвайты + members), `channel` (CRUD `text`/`voice`), `chat` (REST-история + WebSocket-протокол с message.send/subscribe и `message.new`/`message.sent`/`subscribed`/`member.joined`/`error`).

PR-3.5 закрывает MVP-фронт: создаёт веб-клиент с нуля (папка `web/` отсутствует), подключает его ко всем уже работающим бэк-эндпоинтам, упаковывает в docker-compose и сдаёт работоспособный сценарий: пользователь регистрируется → создаёт комнату → приглашает второго → они переписываются в реалтайме.

Это **первый клиент** — после MVP к проекту присоединятся внешние пользователи для сбора обратной связи. Поэтому фокус: стабильность core-flow, простота добавления новых фич, прозрачность ошибок, грамотные edge cases (multi-tab, reconnect, истечение токенов).

## Критерии приёмки

1. Пользователь может зарегистрировать новый аккаунт по email/username/password.
2. Пользователь может залогиниться существующим аккаунтом, на reload-страницы сессия восстанавливается.
3. Пользователь видит свой профиль (username, email) в верхней панели и может выйти (локальный logout, токены и стор очищаются).
4. Пользователь видит список своих комнат в левом сайдбаре. Может создать новую и удалить свою (только owner).
5. Owner/admin может сгенерировать новый код приглашения; код 8-символьный Crockford base32. Второй пользователь может вступить по коду.
6. После `join` пользователь видит новую комнату в списке и получает realtime-события по ней (`member.joined`).
7. Внутри комнаты пользователь видит список каналов (text + voice-секция disabled c подписью «coming soon») и список членов с ролями (имя — UUID-placeholder).
8. Admin/owner может создать text-канал и удалить его. Voice-каналы создаются, но недоступны для входа.
9. При выборе text-канала: грузится последние 50 сообщений (REST), фронт подписывается на канал через WS. При скролле вверх — подгружается следующая страница.
10. Пользователь может отправить сообщение через WS. Сообщение появляется немедленно (optimistic), затем коммитится (ack `message.sent`).
11. Сообщения других пользователей приходят в реалтайме (`message.new`).
12. Multi-tab: сообщения дедуплицируются по `id`.
13. При истечении access-токена (401 `AUTH-011`) auto-refresh происходит прозрачно (single-flight). При непригодности refresh-токена — пользователь редиректится на `/login`.
14. При потере WS-связи показывается баннер «Переподключение…»; после восстановления — баннер исчезает, подписки восстанавливаются.
15. Бэк не отдаёт `username` в `members`/`message.new` — UI показывает первые 6 символов UUID + цветной avatar по hash UUID. Для текущего пользователя имя берётся из `/auth/me`.
16. Фронт собирается в docker-compose как сервис `web` (nginx:alpine + статика + прокси `/api/v1` на `api`).
17. Ручной e2e-сценарий проходит: register → create room → invite → second user → realtime-сообщение в обе стороны.
18. Unit/component/e2e тесты зелёные (`04-testing.md`).

## Контракт бэка

| Эндпоинт / Событие | Источник в docs/ | Что от него хотим |
|---|---|---|
| `POST /api/v1/auth/register` | `docs/1_3_auth_domen/08-api-contract.md` | Создать аккаунт; на success делаем auto-login |
| `POST /api/v1/auth/login` | `docs/1_3_auth_domen/08-api-contract.md` | Получить пару access+refresh, сохранить в localStorage |
| `POST /api/v1/auth/refresh` | `docs/1_3_auth_domen/08-api-contract.md` | Обновить access при 401 AUTH-011 |
| `GET /api/v1/auth/me` | `docs/1_3_auth_domen/08-api-contract.md` | Подгрузить currentUser при mount и после login |
| `POST /api/v1/rooms` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Создать комнату (owner) |
| `GET /api/v1/rooms` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Список моих комнат с ролью |
| `GET /api/v1/rooms/{roomId}` | `docs/2_1_rooms_and_channels/08-api-contract.md` | (опционально) одна комната; в MVP — список достаточен |
| `DELETE /api/v1/rooms/{roomId}` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Удалить комнату (только owner) |
| `GET /api/v1/rooms/{roomId}/members` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Показать список членов |
| `POST /api/v1/rooms/{roomId}/invite` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Создать новый код (отзывает старые) |
| `POST /api/v1/rooms/join/{code}` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Вступить по коду + reconnect WS |
| `POST /api/v1/rooms/{roomId}/channels` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Создать text/voice канал |
| `GET /api/v1/rooms/{roomId}/channels` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Список каналов комнаты |
| `DELETE /api/v1/rooms/{roomId}/channels/{channelId}` | `docs/2_1_rooms_and_channels/08-api-contract.md` | Удалить канал |
| `GET /api/v1/channels/{channelId}/messages` | `docs/3_1_realtime_chat/08-api-contract.md` | История сообщений с курсорной пагинацией |
| `WS /api/v1/ws?token=` | `docs/3_1_realtime_chat/05-events.md` + `08-api-contract.md` | Подключение, токен в query |
| WS `subscribe` | `docs/3_1_realtime_chat/05-events.md` | Подписаться на канал |
| WS `message.send` | `docs/3_1_realtime_chat/05-events.md` | Отправить сообщение |
| WS `subscribed` (in) | `docs/3_1_realtime_chat/05-events.md` | ack подписки |
| WS `message.new` (in) | `docs/3_1_realtime_chat/05-events.md` | Получить новое сообщение |
| WS `message.sent` (in) | `docs/3_1_realtime_chat/05-events.md` | ack отправки автору |
| WS `member.joined` (in) | `docs/3_1_realtime_chat/05-events.md` | Кто-то вступил в одну из моих комнат |
| WS `error` (in) | `docs/3_1_realtime_chat/05-events.md` | Адресная ошибка |

**Logout-эндпоинта НЕТ.** Реализуется локально (`02-behavior.md` UC-15).

Полная сводка контрактов и специфик — в `research.md` (он же ссылается на `.thoughts/research/2026-05-13-3_5-frontend-baseline.md`).

## Документы

| Файл | Разрез | Описание |
|------|--------|----------|
| [research.md](./research.md) | — | Сводный конспект бэк-baseline для PR-3.5 |
| [01-architecture.md](./01-architecture.md) | Logical | C4 L1 → L2 → L3 (4 фичевых слайса) + граф зависимостей |
| [02-behavior.md](./02-behavior.md) | Process | DFD + 16 sequence-диаграмм + error mapping + edge cases (multi-tab, reconnect) |
| [03-decisions.md](./03-decisions.md) | Decision | 21 ADR + 12 рисков + open questions |
| [04-testing.md](./04-testing.md) | Quality | Coverage mapping + ~151 теста (vitest + RTL + Playwright) |
| [05-state-model.md](./05-state-model.md) | — | 4 Zustand-стора + form state + optimistic + state machines |
| [06-api-integration.md](./06-api-integration.md) | — | TS-типы DTO/WS-фреймов + auto-refresh single-flight + WS reconnect |
| [07-ui-contract.md](./07-ui-contract.md) | — | Спецификация ~20 компонентов: props, состояния, a11y, ASCII-мокапы |
| [08-routes.md](./08-routes.md) | — | Карта роутов + гарды + deep-links + nginx-конфиг |
| [09-standards.md](./09-standards.md) | — | Compliance-матрица по 6 файлам prompts/ + уточнения |

## Архитектурные хайлайты

- **Foundation-first стратегия фаз** — `web/` создаётся с нуля, поэтому план кода будет начинаться с bootstrap (Vite + TS + ESLint + структура).
- **Feature-based структура** `web/src/features/{auth,rooms,channels,chat}/{components,api,store,types}` (см. `prompts/Frontend Architecture Layers.txt`).
- **Single source of truth для сети** — `web/src/shared/api/{fetch.ts, ws.ts, refresh.ts, token-storage.ts}`. Все api-функции возвращают `ApiResult<T>`. Auto-refresh single-flight, WS reconnect восстанавливает подписки.
- **Zustand** — четыре изолированных стора, по одному на фичу. Persist только в `useAuthStore.currentUser`, токены — в отдельном `tokenStorage`.
- **react-router-dom v6** с гардами `<RequireAuth>`, `<RedirectIfAuthenticated>`, `<RequireMembership>`, `<RequireChannelInRoom>`.
- **react-hook-form + zod** для всех 5+ форм.
- **CSS Modules**, тёмная тема (Discord-like), CSS-переменные в `app/styles/theme.css`.
- **Optimistic update сообщений** через temp-id + реконсиляция с server-id из `message.sent`. Multi-tab дедупликация по `id` в `message.new`.
- **Воссоздание сессии** при reload через `loadMe()` + auto-refresh.
- **Voice-каналы** в UI — disabled list-item с подписью «coming soon».
- **Имена пользователей** для других — UUID-placeholder + цветной avatar; для себя — реальный username из `/auth/me`.

## Состояние документов

- ✏️ Draft — все 11 файлов готовы.
- Ожидается architect review (внутренний — выполнен в этой сессии) + утверждение пользователем.
- Открытые вопросы из `03-decisions.md` (темизация, i18n, toaster, avatar palette) — приняты с разумными defaults; финал — за пользователем.

## Следующие шаги

1. Прочитать дизайн и принять / запросить правки.
2. После approval — сгенерировать план кода (`plan/README.md` + `plan/phase-NN.md`).
3. После approval плана — `/implement_frontend docs/3_5_frontend/plan/README.md`.
