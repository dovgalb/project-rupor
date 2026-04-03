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

## Фазы MVP

### Фаза 1 — Фундамент + Авторизация

- Структура проекта, Docker Compose (PostgreSQL), миграции
- Регистрация, логин, JWT (access + refresh)
- Middleware авторизации
- Результат: рабочий API с авторизацией, можно тестить через curl

### Фаза 2 — Комнаты и каналы

- CRUD комнат (создание, получение, удаление)
- CRUD каналов внутри комнат
- Роли и права (owner создаёт, admin управляет каналами, member читает)
- Инвайт-ссылки (генерация + вступление по коду)
- Список участников комнаты

### Фаза 3 — Текстовый чат (реалтайм)

- WebSocket подключение с авторизацией по JWT
- Hub: подписка на каналы, рассылка сообщений подписчикам
- Сохранение сообщений в БД
- REST-эндпоинт для истории сообщений с пагинацией
- Результат: полноценный чат, можно общаться в реалтайме

### Фаза 4 — Голосовые звонки

- Сигнальный сервер поверх WebSocket (обмен SDP/ICE)
- WebRTC P2P для аудио
- STUN-сервер (публичный Google STUN на старте)
- Управление состоянием: кто в голосовом канале, mute/unmute
- Результат: можно созваниваться в голосовых каналах

### Фаза 5 — Веб-клиент (React)

- Авторизация (формы логина/регистрации)
- Список комнат и каналов (сайдбар)
- Текстовый чат (WebSocket)
- Голосовой интерфейс (кнопки подключения/мута, WebRTC)

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
