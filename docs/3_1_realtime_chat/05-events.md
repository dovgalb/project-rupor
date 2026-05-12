---
parent: ./README.md
---

# 05 — Domain Events

Список событий, генерируемых сервером и рассылаемых через WebSocket. Все события — server → client. Формат конверта стабилен:

```json
{
  "type": "<event-name>",
  "data": { ... event-specific payload ... }
}
```

Для исходящих от клиента «команд» используется тот же конверт (`subscribe`, `message.send`), но они описаны в `08-api-contract.md` как часть WS-контракта, не как доменные события.

## `message.new`

Сигнал о новом сообщении в text-канале. Триггерится после успешного `chat/usecase.SendMessage`. Рассылается всем активным WS-коннектам, подписанным на ChannelTopic канала-получателя.

**Где публикуется:** `internal/chat/usecase/send_message.go::SendMessage.Execute`, после `MessageRepository.Save(...)` возвращает `nil`.

**Кто реализует publisher:** `pkg/websocket/Hub` через метод `Publish(topic websocket.Topic, frame []byte)`. Адаптер `Broadcaster` живёт в `cmd/server/main.go` (тонкий враппер на hub'е) и предоставляется в `chat/usecase.SendMessage` как зависимость.

**Topic подписки:** `channel:<channelID>` (тип `TopicKindChannel`). Клиент подписывается явно через event `subscribe`.

**Payload:**

| Поле | Тип | Описание |
|---|---|---|
| `id` | string (uuid) | Идентификатор сообщения |
| `channel_id` | string (uuid) | Канал, в который пришло сообщение |
| `author_id` | string (uuid) | Автор сообщения |
| `text` | string | Текст сообщения (1..4000 рун) |
| `created_at` | string (RFC3339 nano, UTC) | Время создания (соответствует БД) |

**Полный пример фрейма:**

```json
{
  "type": "message.new",
  "data": {
    "id": "9c5b9a6a-3d27-4f5e-9a1b-2b9b1f9b3e88",
    "channel_id": "1aab12cd-3344-4d5e-9f7a-aabbccddeeff",
    "author_id": "5fa8b3c1-22d4-4e9f-9012-345678901234",
    "text": "Привет всем!",
    "created_at": "2026-05-12T17:32:08.123456789Z"
  }
}
```

**Гарантии:**
- Доставка best-effort. Если сообщение в БД сохранено, но publish упал (slow consumer, panic, hub closed) — отправитель получает успех. Получатель восстановит сообщение через `GET /channels/:id/messages` (см. `02-behavior.md` UC-2, `03-decisions.md` § D-08).
- Порядок доставки сообщений в одном канале **не гарантируется** на уровне WS из-за конкурентного `Publish` из разных горутин. Клиент сортирует по `created_at`, при равных — по `id`. (Уникальность `(created_at, id)` обеспечивается генератором UUID + sub-millisecond `time.Now().UTC()`.)
- Дублей не бывает в рамках одной подписки (hub отслеживает множество подписчиков как set).
- Доставка только подписанным. Клиент, не подписанный на `channel:<id>`, не получает событий, даже если состоит в комнате канала.

## `member.joined`

Сигнал о новом участнике комнаты. Триггерится после успешного `room/usecase.JoinByCode`. Рассылается активным WS-коннектам, подписанным на RoomTopic.

**Где публикуется:** `internal/room/usecase/join_by_code.go::JoinByCode.Execute`, после `MembershipRepository.Add(...)` возвращает `nil`.

**Кто реализует publisher:** новый порт `room/usecase.RoomEventsPublisher` (объявлен в `room/usecase/ports.go`), реализация — `pkg/websocket/Hub`. Биндинг в `cmd/server/main.go`.

**Topic подписки:** `room:<roomID>` (тип `TopicKindRoom`). Подписка автоматическая при connect для всех `Membership` пользователя.

**Payload:**

| Поле | Тип | Описание |
|---|---|---|
| `room_id` | string (uuid) | Комната, в которую вступили |
| `user_id` | string (uuid) | Новый член |
| `joined_at` | string (RFC3339 nano, UTC) | Время вступления |

**Полный пример фрейма:**

```json
{
  "type": "member.joined",
  "data": {
    "room_id": "1234aaaa-bbbb-cccc-dddd-555566667777",
    "user_id": "5fa8b3c1-22d4-4e9f-9012-345678901234",
    "joined_at": "2026-05-12T17:35:00.000000000Z"
  }
}
```

**Гарантии:**
- Best-effort, как и `message.new`.
- Сам новый член НЕ получает событие через свой текущий WS (если он был подключен): подписка на новую комнату активируется только при reconnect (см. `03-decisions.md` § D-09).
- Если у пользователя нет активных WS-коннектов в момент publish'а — событие теряется. Восстановление: при reconnect клиент видит обновлённый список членов через `GET /rooms/:id/members` (фаза 2 уже реализовала эндпоинт).

---

## Сводная таблица портов и реализаций

| Порт (где объявлен) | Адаптер (где реализован) | Биндинг | Topic | Источник события |
|---|---|---|---|---|
| `chat/usecase.Broadcaster.PublishToChannel(channelID, "message.new", payload)` | `pkg/websocket.Hub` через `cmd/server/main.go` адаптер | `chat/usecase.SendMessage` deps | `channel:<id>` | `SendMessage.Execute` after `Save` |
| `room/usecase.RoomEventsPublisher.PublishMemberJoined(roomID, userID, joinedAt)` | `pkg/websocket.Hub` через `cmd/server/main.go` адаптер | `room/usecase.JoinByCode` deps | `room:<id>` | `JoinByCode.Execute` after `Add(membership)` |

---

## Кодирование payload

Сериализация payload — стандартный `encoding/json` со следующими нормализациями:

- Все timestamp'ы — UTC, RFC3339Nano (`time.Time.Format(time.RFC3339Nano)`).
- UUID — каноническая строка из `uuid.UUID.String()`.
- Идентификаторы в JSON-payload — snake_case (`channel_id`, `author_id`, `created_at`, `joined_at`) — отличается от REST (camelCase, см. `08-api-contract.md`). Обоснование: WebSocket-протокол ближе по стилю к Discord/Slack (snake_case), REST — к internal проекту. Если согласовать единый стиль — можно унифицировать в фазе 3.2.

## Logging

Каждый publish логируется на уровне INFO:

```
INFO chat.event_published type=message.new channel_id=... message_id=... subscribers=N
INFO room.event_published type=member.joined room_id=... user_id=... subscribers=N
```

Если publish упал (panic в writer'е, hub.WriteJSON error) — WARN с stack trace.
