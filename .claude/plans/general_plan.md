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
- [x] `internal/auth/domain/`: сущности `User`, value objects (`Email`, `Password`), доменные ошибки
- [x] `internal/auth/usecase/`: интерфейсы репозиториев + сценарии Register / Login / Refresh
- [x] Хеширование паролей (bcrypt)
- [x] Генерация и валидация JWT (access + refresh)
- [x] `internal/auth/repository/postgres/`: sqlc-запросы и реализация репозитория
- [x] `internal/auth/transport/http/`: хендлеры `POST /auth/register`, `/auth/login`, `/auth/refresh`, `GET /auth/me` + DTO
- [x] Подключение роутов в `cmd/server/main.go`

### 1.4 Middleware
- [x] JWT-middleware (извлечение `Authorization: Bearer`, валидация, проброс userID в контекст)
- [x] Логирование запросов
- [x] Recover middleware
- [x] CORS (для будущего фронта)

### 1.5 Тесты
- [ ] Unit-тесты usecase auth (с моками репо)
- [ ] Интеграционные тесты HTTP-хендлеров auth
- [ ] e2e через curl/http-файл

---

## Фаза 2 — Комнаты и каналы
- [x] Миграции: `rooms`, `room_members` (с ролями owner/admin/member), `invites`, `channels` (тип text/voice)
- [x] Домен `room`: сущности, роли, инварианты прав
- [x] Usecase: создать/получить/удалить комнату, список комнат пользователя
- [x] Usecase: генерация инвайта, вступление по коду
- [x] Usecase: список участников
- [x] Домен `channel`: CRUD каналов внутри комнаты, проверка прав
- [x] HTTP: `POST/GET/DELETE /rooms`, `/rooms/:id/invite`, `/rooms/join/:code`, `/rooms/:id/members`
- [x] HTTP: `POST/GET/DELETE /rooms/:id/channels`
- [x] Репозитории postgres (sqlc) для room/channel
- [x] Тесты

---

## Фаза 3 — Текстовый чат (реалтайм)
- [x] Миграция: `messages` (id, channel_id, author_id, text, created_at) + индексы
- [x] `pkg/websocket/`: hub, регистрация подключений, подписка на каналы, broadcast
- [x] WebSocket-эндпоинт `/api/v1/ws?token=<jwt>` с авторизацией по JWT
- [x] Обработчики событий: `subscribe`, `message.send`, `message.new`, `member.joined`
- [x] Домен `chat`: сущности, usecase отправки/чтения сообщений
- [x] Сохранение сообщений в БД (sqlc)
- [x] REST `GET /channels/:id/messages?before=&limit=` (курсорная пагинация)
- [x] Проверка прав (только член комнаты может писать/читать)
- [x] Тесты hub'а и usecase

---

## Фаза 3.5 — Веб-клиент для уже реализованного бэка
- [ ] `web/` — каркас Vite + React + TypeScript + Zustand
- [ ] HTTP-клиент: базовый fetch-обёртка, проброс `Authorization: Bearer`, обработка 401
- [ ] Авторизация: формы регистрации/логина, хранение access/refresh, авто-refresh при 401, logout
- [ ] Профиль: `GET /auth/me`, отображение текущего пользователя
- [ ] Комнаты: список своих комнат, создание, удаление, приглашение по коду, вступление по инвайту
- [ ] Участники комнаты: список членов с ролями
- [ ] Каналы: список каналов внутри комнаты, создание/удаление text-каналов (voice — заглушка)
- [ ] Чат: WebSocket-клиент (`/api/v1/ws?token=`), `subscribe` на текущий канал
- [ ] Чат: загрузка истории через REST с курсорной пагинацией (`before=`, `limit=`), скролл вверх
- [ ] Чат: отправка сообщений через WS (`message.send`), отображение `message.new` в реалтайме
- [ ] UI-каркас: сайдбар (комнаты + каналы), основная панель (чат), верхняя панель (профиль)
- [ ] Базовые состояния: загрузка, пустой список, ошибки сети/авторизации
- [ ] Сборка фронта через Docker, интеграция в `docker-compose.yml`
- [ ] Ручное e2e: регистрация → создание комнаты → инвайт → второй пользователь → реалтайм-чат

---

## Фаза 4 — Голосовые звонки (бэкенд-сигнализация)

### 4.1 Протокол сигнализации
- [ ] Спецификация WS-событий клиент→сервер: `voice.join`, `voice.leave`, `voice.signal`, `voice.mute`
- [ ] Спецификация WS-событий сервер→клиент: `voice.user-joined`, `voice.user-left`, `voice.signal`, `voice.mute-changed`, `voice.participants` (снапшот при входе)
- [ ] Формат `voice.signal`: `{from_user_id, to_user_id, channel_id, payload: {sdp|ice}}` — адресная маршрутизация по `to_user_id`
- [ ] Описание формата в `docs/voice/protocol.md`

### 4.2 Домен voice
- [ ] `internal/voice/domain/`: `VoiceRoom` (состояние voice-канала), `Participant` (user_id, mute), доменные ошибки
- [ ] Инварианты: один пользователь в одном voice-канале одновременно, дубль-join обрабатывается как rejoin
- [ ] Чистые функции изменения состояния (add/remove/setMute), отдельно от конкурентного доступа

### 4.3 Usecase voice
- [ ] Интерфейсы зависимостей: проверка членства в room, рассылка событий через WS-hub
- [ ] Usecase `JoinVoice`: проверка прав → добавление участника → broadcast `voice.user-joined` остальным + ответ `voice.participants` инициатору
- [ ] Usecase `LeaveVoice`: удаление → broadcast `voice.user-left`
- [ ] Usecase `RelaySignal`: проверка, что отправитель и получатель в одном voice-канале → доставка `voice.signal` целевому пиру
- [ ] Usecase `SetMute`: обновление состояния → broadcast `voice.mute-changed`
- [ ] Авто-leave при разрыве WS (через хук в hub'е)

### 4.4 Хранение состояния
- [ ] In-memory реестр voice-каналов с потокобезопасным доступом (RWMutex или sharded map)
- [ ] НЕ персистится в БД — состояние эфемерное, при рестарте сервера сбрасывается
- [ ] Очистка пустых voice-каналов

### 4.5 Интеграция с WS-hub
- [ ] Расширение роутинга WS-событий в `internal/chat/transport/ws/handler.go` (или общий диспетчер)
- [ ] Адресная отправка сообщения конкретному `user_id` через hub (а не только broadcast на канал)
- [ ] Хук on-disconnect для авто-leave из voice

### 4.6 Конфигурация ICE
- [ ] Эндпоинт `GET /api/v1/voice/ice-servers` — отдаёт список STUN/TURN серверов
- [ ] Дефолт: публичный Google STUN (`stun:stun.l.google.com:19302`)
- [ ] Конфигурация через env (`VOICE_STUN_URLS`, `VOICE_TURN_URL`, `VOICE_TURN_USERNAME`, `VOICE_TURN_CREDENTIAL`)

### 4.7 Тесты
- [ ] Unit-тесты доменных функций состояния voice-канала
- [ ] Unit-тесты usecase с моками hub'а и репо членства
- [ ] Интеграционный тест: два фейковых WS-клиента, join → обмен `voice.signal` → leave

---

## Фаза 5 — Веб-клиент: голос и финализация
- [ ] WebRTC: получение `RTCConfiguration` через `GET /voice/ice-servers`
- [ ] WebRTC: подключение к voice-каналу (`voice.join`), создание `RTCPeerConnection` на каждого участника (mesh)
- [ ] WebRTC: обмен offer/answer/ICE через `voice.signal`, прикрепление аудио-потоков к `<audio>`
- [ ] UI: кнопка «зайти/выйти», индикатор активного voice-канала
- [ ] UI: список участников голоса с индикатором mute и говорящего (volume meter)
- [ ] Кнопка mute/unmute с отправкой `voice.mute`
- [ ] Корректный teardown при выходе/смене канала/разрыве WS
- [ ] Обработка отказа в доступе к микрофону, отсутствия устройств
- [ ] Полировка UX: тёмная тема, адаптивная вёрстка, тосты ошибок
- [ ] Ручное e2e: два браузера, голосовое соединение через NAT

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
