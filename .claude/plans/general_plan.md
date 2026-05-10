# MVP Project Rupor — общий план

## Сущности и их связи

```
User
 ├── принадлежит многим Room (через Membership)
 └── отправляет Message

Room (аналог "сервера" в Discord)
 ├── имеет Owner (User)
 ├── имеет Members (User[]) с ролями: owner / admin / member
 ├── имеет Channels[]
 └── имеет InviteCode

Channel (принадлежит Room)
 ├── тип: text | voice
 └── имеет Messages[] (только text)

Message
 ├── автор: User
 ├── канал: Channel
 ├── текст + timestamp
 └── (в MVP без редактирования/удаления — добавим позже)
```

## Стек

| Компонент | Технология |
|-----------|-----------|
| API | Go, `net/http` (Go 1.22+), `chi` для роутинга |
| БД | PostgreSQL |
| Миграции | `golang-migrate` |
| SQL | `sqlc` (генерация Go-кода из SQL) |
| WebSocket | `nhooyr.io/websocket` |
| WebRTC | Browser API + сигнализация через WS |
| Конфиг | env-переменные |
| Фронтенд | React + Vite + Zustand |
| Деплой | Docker Compose |

## API-контракт (основные эндпоинты)

```
# Авторизация
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
GET    /api/v1/auth/me

# Комнаты
POST   /api/v1/rooms
GET    /api/v1/rooms
GET    /api/v1/rooms/:id
DELETE /api/v1/rooms/:id
POST   /api/v1/rooms/:id/invite
POST   /api/v1/rooms/join/:code
GET    /api/v1/rooms/:id/members

# Каналы
POST   /api/v1/rooms/:id/channels
GET    /api/v1/rooms/:id/channels
DELETE /api/v1/rooms/:id/channels/:channel_id

# Сообщения (REST)
GET    /api/v1/channels/:id/messages?before=&limit=

# WebSocket
WS     /api/v1/ws?token=<jwt>
```

## WebSocket-события

```
→ сервер: { "type": "subscribe", "channel_id": "..." }
→ сервер: { "type": "message.send", "channel_id": "...", "text": "..." }
← клиент: { "type": "message.new", "data": { ... } }
← клиент: { "type": "member.joined", "data": { ... } }
← клиент: { "type": "voice.signal", "data": { sdp/ice } }
```

---

# Декомпозиция задач

Отметки: `[x]` — сделано, `[~]` — частично, `[ ]` — не начато.

## Фаза 1 — Фундамент + Авторизация

### 1.1 Инфраструктура проекта
- [x] Структура папок по Clean Architecture (`internal/{auth,user,room,channel,chat,voice}`)
- [x] `go.mod`, базовые зависимости (`chi`)
- [x] `Makefile` (run/test/lint/build/migrate/dc-up)
- [x] `docker-compose.yml` (PostgreSQL + Redis)
- [x] `.env.example`, загрузка конфига из env (`config/config.go` + тесты)
- [x] `.golangci.yml` v2, pre-commit хук, CI (`.github/workflows/ci.yml`)
- [x] `cmd/server/main.go` + health endpoint + тест
- [x] Каркас миграций (`golang-migrate`), `sqlc.yaml`

### 1.2 Схема БД
- [x] Подключён `pgcrypto` (миграция `0001_init`)
- [x] Миграция: таблица `users` (id, email, password_hash, username, created_at)
- [x] Миграция: таблица `refresh_tokens` (если хранятся в БД)
- [x] Индексы на email/username

### 1.3 Домен auth
- [ ] `internal/auth/domain/`: сущности `User`, value objects (`Email`, `Password`), доменные ошибки
- [ ] `internal/auth/usecase/`: интерфейсы репозиториев + сценарии Register / Login / Refresh
- [ ] Хеширование паролей (bcrypt)
- [ ] Генерация и валидация JWT (access + refresh)
- [ ] `internal/auth/repository/postgres/`: sqlc-запросы и реализация репозитория
- [ ] `internal/auth/transport/http/`: хендлеры `POST /auth/register`, `/auth/login`, `/auth/refresh`, `GET /auth/me` + DTO
- [ ] Подключение роутов в `cmd/server/main.go`

### 1.4 Middleware
- [ ] JWT-middleware (извлечение `Authorization: Bearer`, валидация, проброс userID в контекст)
- [ ] Логирование запросов
- [ ] Recover middleware
- [ ] CORS (для будущего фронта)

### 1.5 Тесты
- [ ] Unit-тесты usecase auth (с моками репо)
- [ ] Интеграционные тесты HTTP-хендлеров auth
- [ ] e2e через curl/http-файл

---

## Фаза 2 — Комнаты и каналы
- [ ] Миграции: `rooms`, `room_members` (с ролями owner/admin/member), `invites`, `channels` (тип text/voice)
- [ ] Домен `room`: сущности, роли, инварианты прав
- [ ] Usecase: создать/получить/удалить комнату, список комнат пользователя
- [ ] Usecase: генерация инвайта, вступление по коду
- [ ] Usecase: список участников
- [ ] Домен `channel`: CRUD каналов внутри комнаты, проверка прав
- [ ] HTTP: `POST/GET/DELETE /rooms`, `/rooms/:id/invite`, `/rooms/join/:code`, `/rooms/:id/members`
- [ ] HTTP: `POST/GET/DELETE /rooms/:id/channels`
- [ ] Репозитории postgres (sqlc) для room/channel
- [ ] Тесты

---

## Фаза 3 — Текстовый чат (реалтайм)
- [ ] Миграция: `messages` (id, channel_id, author_id, text, created_at) + индексы
- [ ] `pkg/websocket/`: hub, регистрация подключений, подписка на каналы, broadcast
- [ ] WebSocket-эндпоинт `/api/v1/ws?token=<jwt>` с авторизацией по JWT
- [ ] Обработчики событий: `subscribe`, `message.send`, `message.new`, `member.joined`
- [ ] Домен `chat`: сущности, usecase отправки/чтения сообщений
- [ ] Сохранение сообщений в БД (sqlc)
- [ ] REST `GET /channels/:id/messages?before=&limit=` (курсорная пагинация)
- [ ] Проверка прав (только член комнаты может писать/читать)
- [ ] Тесты hub'а и usecase

---

## Фаза 4 — Голосовые звонки
- [ ] Расширение WS-протокола: `voice.signal` (SDP/ICE)
- [ ] Сигнальный сервер в `internal/voice/`: маршрутизация SDP/ICE между пирами в одном voice-канале
- [ ] Состояние voice-канала: список участников, mute/unmute
- [ ] События: `voice.user-joined`, `voice.user-left`, `voice.mute-changed`
- [ ] Конфиг STUN (Google публичный)
- [ ] Тесты сигнализации

---

## Фаза 5 — Веб-клиент (React)
- [ ] `web/` — Vite + React + Zustand скелет
- [ ] Формы регистрации/логина, хранение токенов, refresh-флоу
- [ ] Сайдбар: список комнат и каналов
- [ ] Чат: WS-клиент, отображение истории, отправка сообщений
- [ ] Голос: подключение/мут, WebRTC P2P, отображение участников
- [ ] Сборка через Docker

---

## Сводка прогресса

| Фаза | Готовность |
|------|------------|
| 1. Фундамент + Авторизация | ~20% (только инфраструктура и health) |
| 2. Комнаты и каналы | 0% |
| 3. Текстовый чат | 0% |
| 4. Голос | 0% |
| 5. Фронтенд | 0% |

Следующий шаг — **1.2 + 1.3**: миграция `users`, домен `auth`, регистрация/логин/JWT.
