---
date: 2026-05-13
feature: 3_5-frontend
source_research: ../../.thoughts/research/2026-05-13-3_5-frontend-baseline.md
---

# Ресерч: бэк-baseline для PR-3.5 (веб-клиент)

## Резюме

Полный baseline бэка для веб-клиента лежит в `.thoughts/research/2026-05-13-3_5-frontend-baseline.md` (исследован 2026-05-13, commit `7d7da4e`). Этот файл — компактный конспект для дизайна PR-3.5: какие эндпоинты и фреймы есть, какие специфики важны для UI, где интегрироваться.

Все нужные домены бэка реализованы: **auth** (без logout-эндпоинта), **room** (CRUD + инвайты + join + members), **channel** (CRUD `text`/`voice`), **chat** (REST-история + WS-протокол с `subscribe`/`message.send` ↔ `subscribed`/`message.sent`/`message.new`/`member.joined`/`error`). Папка `web/` отсутствует — PR-3.5 пишется с нуля. В docker-compose только Postgres; ни Go-сервер, ни фронт не контейнеризированы (Go запускается через `make run`).

Стандарты фронта в `prompts/` свежие (`Frontend Architecture Layers.txt`, `TypeScript Style.txt`, `React Components.txt`, `Zustand Stores.txt`, `API Integration.txt`, `Tests Style (Web).txt`).

## Бэк-контракт (обнаружено)

### REST

| Метод | Путь | Auth | Хендлер | Контракт |
|---|---|---|---|---|
| POST | `/api/v1/auth/register` | public | `internal/auth/transport/http/register_handler.go:17` | `docs/1_3_auth_domen/08-api-contract.md` |
| POST | `/api/v1/auth/login` | public | `login_handler.go:17` | `docs/1_3_auth_domen/08-api-contract.md` |
| POST | `/api/v1/auth/refresh` | public (body) | `refresh_handler.go:17` | `docs/1_3_auth_domen/08-api-contract.md` |
| GET | `/api/v1/auth/me` | Bearer | `me_handler.go:19` | `docs/1_3_auth_domen/08-api-contract.md` |
| POST | `/api/v1/rooms` | Bearer | `create_room_handler.go:19` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| GET | `/api/v1/rooms` | Bearer | `list_rooms_handler.go:19` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| GET | `/api/v1/rooms/{roomID}` | Bearer + member | `get_room_handler.go:23` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| DELETE | `/api/v1/rooms/{roomID}` | Bearer + owner | `delete_room_handler.go:23` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| GET | `/api/v1/rooms/{roomID}/members` | Bearer + member | `list_members_handler.go:23` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| POST | `/api/v1/rooms/{roomID}/invite` | Bearer + admin/owner | `regenerate_invite_handler.go:23` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| POST | `/api/v1/rooms/join/{code}` | Bearer | `join_by_code_handler.go:21` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| POST | `/api/v1/rooms/{roomID}/channels` | Bearer + admin/owner | `internal/channel/transport/http/create_channel_handler.go:23` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| GET | `/api/v1/rooms/{roomID}/channels` | Bearer + member | `list_channels_handler.go:23` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| DELETE | `/api/v1/rooms/{roomID}/channels/{channelID}` | Bearer + admin/owner | `delete_channel_handler.go:23` | `docs/2_1_rooms_and_channels/08-api-contract.md` |
| GET | `/api/v1/channels/{channelID}/messages` | Bearer + member | `internal/chat/transport/http/list_messages_handler.go:26` | `docs/3_1_realtime_chat/08-api-contract.md` |
| GET | `/api/v1/health` | public | `cmd/server/health.go:7` | — |

**Logout-эндпоинта НЕТ.** Фронтовый logout = локальная очистка стора и `localStorage`.

### WebSocket

**Подключение**: `GET /api/v1/ws?token=<jwt>` — `internal/chat/transport/ws/routes.go:23-25`. Токен в query (browser-API не отдаёт `Authorization` для WS).

**Origin check**: `OriginPatterns = stripScheme(cfg.CORSAllowedOrigins())` — `cmd/server/main.go:207`. Несовпадение origin → 403 (не наш JSON).

**Автоподписка**: при connect handler сам подписывает conn на `RoomTopic(roomID)` для всех комнат пользователя (`handler.go:72-74`). На каналы — только явная команда.

**Inbound фреймы (server → client)** — обёртка `{type, data}`, snake_case в payload:

| `type` | Payload | Источник |
|---|---|---|
| `subscribed` | `{ channel_id }` | ack `subscribe` |
| `message.new` | `{ id, channel_id, author_id, text, created_at }` | broadcast в `channel:<id>` |
| `message.sent` | `{ id, channel_id, created_at }` | ack `message.send` (только отправителю) |
| `member.joined` | `{ room_id, user_id, joined_at }` | broadcast в `room:<id>` |
| `error` | `{ code, message }` | адресная ошибка инициатору |

**Outbound фреймы (client → server)** — плоский JSON, snake_case:

| `type` | Payload | Семантика |
|---|---|---|
| `subscribe` | `{ channel_id }` | подписаться на канал |
| `message.send` | `{ channel_id, text }` | отправить сообщение |

`unsubscribe`, `ping`/`pong`, `member.left`, `message.edit/delete`, `typing`, `presence` — **НЕ реализованы**.

### Коды ошибок (для UI mapping)

| Префикс | Источник | Где визуализируем |
|---|---|---|
| `AUTH-001..AUTH-003` | валидация полей формы | inline field error на login/register |
| `AUTH-004 email taken` | 409 | inline `email` на register |
| `AUTH-005 username taken` | 409 | inline `username` на register |
| `AUTH-006 invalid credentials` | 401 на login | form-level error |
| `AUTH-007/008/009` | 401 на refresh | logout |
| `AUTH-010 access invalid` | 401 от middleware | logout |
| `AUTH-011 access expired` | 401 от middleware | trigger refresh (single-flight) |
| `AUTH-012 malformed body` | 400 | bug-баннер (не должно случаться у клиента) |
| `ROOM-001 invalid name` | 400 | inline на create room |
| `ROOM-002 not found` | 404 | redirect `/rooms` + toast |
| `ROOM-003 not a member` | 403 | удалить комнату из стора + redirect |
| `ROOM-004 admin/owner required` | 403 | toast «недостаточно прав» |
| `ROOM-005 only owner can delete` | 403 | toast |
| `ROOM-006 already a member` | 409 на join | inline на форме join |
| `ROOM-007 invite not found` | 404 на join | inline |
| `ROOM-008 invalid invite code` | 400 на join | inline |
| `ROOM-009 invalid body` | 400 | bug-баннер |
| `CHANNEL-001..007` | различные | inline / toast / redirect (см. 02-behavior) |
| `CHAT-001..006` | различные | inline / toast (см. 02-behavior) |
| `CHAT-007 unsupported event` | WS error-frame | log warning, не визуализируем |

## Специфики Rupor (критичны для дизайна)

1. **REST = camelCase, WS payload = snake_case** — маппинг в api-слое, не в компонентах.
2. **CORS `allowCredentials=false`** → токены только в `Authorization: Bearer`, в `localStorage`. Нет `credentials: "include"`.
3. **JWT TTL**: access `15m` (default), refresh `720h` (30 дней). Сервер не делает auto-refresh — клиент сам.
4. **Auto-refresh**: 401 `AUTH-011` → `POST /auth/refresh` → retry. Single-flight (параллельные 401 ждут один refresh-promise). Защита от refresh-loop: если оригинальный запрос — это сам `/auth/refresh`, не повторяем; если retry после refresh снова даёт AUTH-011 — logout.
5. **Logout** = очистка localStorage + clear() всех сторов + redirect на `/login`. Серверного отзыва нет.
6. **WS auto-room-subscribe только при connect**. После `POST /rooms/join/{code}` подписки на новый `room:<id>` НЕ появится — нужен **reconnect WS**.
7. **WS reconnect**: backoff (1s/2s/5s/15s/15s/...). После reconnect восстанавливаем явные `subscribe` на каналы из стора.
8. **DTO членов содержит только `userId/role/joinedAt`** (ADR D-09 `docs/2_1_rooms_and_channels/08-api-contract.md:207`). Никаких `username/displayName`. То же в `message.new`: только `author_id`. Решение фронта — UI placeholder с кратким UUID, см. `03-decisions.md`.
9. **Voice-каналы — CRUD-заглушка**. Можно создавать/листать, никакого сигналинга нет. UI рендерит voice-каналы в списке отдельной секцией с подписью «coming soon», нельзя «войти в voice».
10. **Multi-tab**: каждая вкладка — свой WS-конект. `message.sent` приходит только в исходную вкладку. `message.new` — во все подписанные. Реконсиляция temp-id ↔ committed-id — в `useChatStore`.
11. **Empty members не бывает**: actor сам всегда в списке как минимум.
12. **Курсор истории**: `before=<message-id>` (UUID), `limit=1..100`, default `50`. `nextBefore` — UUID или `null` (если страница неполная). Сортировка `createdAt DESC, id DESC`.
13. **CORS default**: `http://localhost:5173`. Vite dev-сервер по умолчанию слушает там же — конфликта нет.

## Точки интеграции (где новая фича подключается)

Нет существующего кода `web/` — все слои создаются заново. План интеграции:

- `web/src/shared/api/fetch.ts` — единственное место `fetch()`. Подставляет `Authorization: Bearer`, разбирает `ErrorEnvelope`, делает auto-refresh single-flight.
- `web/src/shared/api/ws.ts` — единственное место `new WebSocket()`. Reconnect, восстановление подписок, dispatch фреймов в фичевые сторы.
- `web/src/shared/api/token-storage.ts` — localStorage-обёртка (получить/сохранить/очистить access+refresh+expires).
- `web/src/features/auth/` — `register`, `login`, `refresh`, `me`, `logout(local)`. Включает экраны `/login`, `/register`.
- `web/src/features/rooms/` — список своих комнат, создание, удаление, инвайт, join. Подписки на `member.joined` в стор.
- `web/src/features/channels/` — список каналов комнаты, создание/удаление `text` (voice — заглушка).
- `web/src/features/chat/` — REST-история + WS subscribe + send + receive. Optimistic update.
- `web/src/pages/` — `LoginPage`, `RegisterPage`, `AppPage` (главный экран с sidebar + chat).
- `web/src/app/` — Router, провайдеры, ErrorBoundary, toaster.
- `web/src/routes.tsx` — карта роутов + гарды.

Docker: новый `web/Dockerfile` (multi-stage: `node:lts-alpine` builder → `nginx:alpine` runtime) + сервис `web` в `docker-compose.yml`, проксирующий `/api/v1` на сервер.

## UX-разведка (из user-story PR-3.5)

Сценарий приёмки: «регистрация → создание комнаты → инвайт → второй пользователь → реалтайм-чат». Экраны:

- **Auth flow**: `/login` (email + password), `/register` (email + username + password). Ошибки валидации — inline под полями.
- **Главный экран** `/rooms/:roomId/channels/:channelId`:
  - Левый sidebar: список моих комнат (компактный) + внутри активной комнаты — список каналов + список участников.
  - Центральная панель: чат активного канала (история + composer).
  - Верхняя панель: текущий пользователь (`/auth/me`), кнопка «выйти».
- **Модальные действия**: создать комнату, удалить комнату (только owner), показать инвайт-код, вступить по коду, создать text-канал, удалить канал (admin/owner).
- **Состояния сети**: баннер «Переподключение…» при потере WS, тосты на сетевые ошибки REST.
- **Empty states**: нет комнат → подсказка «Создай первую комнату или вступи по коду»; нет каналов в комнате → подсказка для admin/owner; нет сообщений в канале → «Будь первым».
- **Voice-канал**: в списке каналов отображается, но не кликабелен (или с подписью «voice — coming soon»).

## Что НЕ покрывается в PR-3.5

- Редактирование/удаление сообщений (бэк не поддерживает в фазе 3.1).
- Прочитанность, typing-indicators, presence (бэк не поддерживает).
- Voice/WebRTC (не реализовано).
- Личные сообщения (нет в бэке).
- Уведомления / push.
- Темизация (только default тема, тёмная — после MVP).
- Поиск по сообщениям.

Эти пункты — из `tasks/after_mvp_plan.md`.
