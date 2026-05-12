---
parent: ./README.md
view: api
---

# 08 — API Contract

REST под префиксом `/api/v1`. Все эндпоинты защищены `RequireAuth` (JWT в `Authorization: Bearer <jwt>`). Формат envelope-ошибок единый: `{"error":{"code":"DOMAIN-NNN","message":"human readable"}}` через `pkg/httpx.WriteJSONError`. Content-Type ответов — `application/json; charset=utf-8`. Камелкейс полей JSON.

## Карта эндпоинтов

| Метод | Путь | Кто может | Use case |
|-------|------|-----------|----------|
| POST | `/api/v1/rooms` | любой авторизованный | UC-R1 CreateRoom |
| GET | `/api/v1/rooms` | любой авторизованный | UC-R3 ListUserRooms |
| GET | `/api/v1/rooms/{roomID}` | member, admin, owner | UC-R2 GetRoom |
| DELETE | `/api/v1/rooms/{roomID}` | owner only | UC-R4 DeleteRoom |
| GET | `/api/v1/rooms/{roomID}/members` | member, admin, owner | UC-R5 ListMembers |
| POST | `/api/v1/rooms/{roomID}/invite` | admin, owner | UC-R6 RegenerateInvite |
| POST | `/api/v1/rooms/join/{code}` | любой авторизованный | UC-R7 JoinByCode |
| POST | `/api/v1/rooms/{roomID}/channels` | admin, owner | UC-C1 CreateChannel |
| GET | `/api/v1/rooms/{roomID}/channels` | member, admin, owner | UC-C2 ListChannels |
| DELETE | `/api/v1/rooms/{roomID}/channels/{channelID}` | admin, owner | UC-C3 DeleteChannel |

---

## POST /api/v1/rooms

Создать комнату. Создатель автоматически становится `owner`.

**Request:**

```
Headers:
  Authorization: Bearer <jwt>
  Content-Type: application/json

Body:
  name: string (required, 1..64 chars after trim, no control chars)
```

**Response 201:**

```json
{
  "id": "uuid-string",
  "ownerId": "uuid-string",
  "name": "My Room",
  "createdAt": "2026-05-11T12:00:00Z"
}
```

**Error responses:**

| Status | Error Code | Body | Когда |
|--------|------------|------|-------|
| 400 | `ROOM-009` | `{"error":{"code":"ROOM-009","message":"invalid request body"}}` | JSON не парсится |
| 400 | `ROOM-001` | `{"error":{"code":"ROOM-001","message":"invalid room name"}}` | name пустое / > 64 / control chars |
| 401 | `AUTH-010` | `{"error":{"code":"AUTH-010","message":"access token invalid"}}` | Нет/невалидный JWT |
| 401 | `AUTH-011` | `{"error":{"code":"AUTH-011","message":"access token expired"}}` | Истёк JWT |
| 500 | `INTERNAL` | `{"error":{"code":"INTERNAL","message":"internal"}}` | Транзакция не удалась |

---

## GET /api/v1/rooms

Список комнат, в которых состоит текущий пользователь (любая роль).

**Request:**

```
Headers:
  Authorization: Bearer <jwt>
```

**Response 200:**

```json
{
  "items": [
    {
      "id": "uuid-string",
      "ownerId": "uuid-string",
      "name": "My Room",
      "createdAt": "2026-05-11T12:00:00Z",
      "role": "owner"
    },
    {
      "id": "uuid-string",
      "ownerId": "uuid-string",
      "name": "Other Room",
      "createdAt": "2026-05-10T09:30:00Z",
      "role": "member"
    }
  ]
}
```

Пустой ответ:

```json
{ "items": [] }
```

`role` — роль пользователя в этой комнате (`"owner"` | `"admin"` | `"member"`).

**Error responses:**

| Status | Error Code | Когда |
|--------|------------|-------|
| 401 | `AUTH-010`/`AUTH-011` | JWT |
| 500 | `INTERNAL` | БД |

---

## GET /api/v1/rooms/{roomID}

Получить детальный объект комнаты. Доступно member/admin/owner.

**Request:**

```
Path:
  roomID: uuid (required)
Headers:
  Authorization: Bearer <jwt>
```

**Response 200:**

```json
{
  "id": "uuid-string",
  "ownerId": "uuid-string",
  "name": "My Room",
  "createdAt": "2026-05-11T12:00:00Z"
}
```

**Error responses:**

| Status | Error Code | Body | Когда |
|--------|------------|------|-------|
| 404 | `ROOM-002` | `{"error":{"code":"ROOM-002","message":"room not found"}}` | roomID не UUID / комнаты нет |
| 403 | `ROOM-003` | `{"error":{"code":"ROOM-003","message":"not a member"}}` | Не состоит в комнате |
| 401 | `AUTH-010`/`AUTH-011` | … | JWT |

---

## DELETE /api/v1/rooms/{roomID}

Удалить комнату. Только owner. Каскадно сносит room_members, invites, channels.

**Request:**

```
Path:
  roomID: uuid (required)
Headers:
  Authorization: Bearer <jwt>
```

**Response 204:** пустое тело.

**Error responses:**

| Status | Error Code | Body | Когда |
|--------|------------|------|-------|
| 404 | `ROOM-002` | `{"error":{"code":"ROOM-002","message":"room not found"}}` | UUID не парсится / уже удалена |
| 403 | `ROOM-003` | `{"error":{"code":"ROOM-003","message":"not a member"}}` | Не состоит |
| 403 | `ROOM-005` | `{"error":{"code":"ROOM-005","message":"only owner can delete room"}}` | admin или member пытается удалить |

---

## GET /api/v1/rooms/{roomID}/members

Список участников комнаты. Доступно member/admin/owner.

**Request:**

```
Path:
  roomID: uuid (required)
Headers:
  Authorization: Bearer <jwt>
```

**Response 200:**

```json
{
  "items": [
    {
      "userId": "uuid-string",
      "role": "owner",
      "joinedAt": "2026-05-11T12:00:00Z"
    },
    {
      "userId": "uuid-string",
      "role": "member",
      "joinedAt": "2026-05-11T13:14:15Z"
    }
  ]
}
```

В PR-2 ответ **не содержит** `username` или `email` — см. ADR D-09.

**Error responses:**

| Status | Error Code | Когда |
|--------|------------|-------|
| 404 | `ROOM-002` | UUID не парсится |
| 403 | `ROOM-003` | Не состоит |

---

## POST /api/v1/rooms/{roomID}/invite

Сгенерировать новый инвайт-код. Старый активный код (если был) автоматически отзывается. Доступно admin/owner.

**Request:**

```
Path:
  roomID: uuid (required)
Headers:
  Authorization: Bearer <jwt>
Body: пусто или {} (игнорируется)
```

**Response 200:**

```json
{
  "code": "K3M9PX2A",
  "createdBy": "uuid-string",
  "createdAt": "2026-05-11T12:00:00Z"
}
```

`code` — 8 символов из Crockford base32 (`0123456789ABCDEFGHJKMNPQRSTVWXYZ`), upper-case.

**Error responses:**

| Status | Error Code | Body | Когда |
|--------|------------|------|-------|
| 404 | `ROOM-002` | `{"error":{"code":"ROOM-002","message":"room not found"}}` | UUID не парсится |
| 403 | `ROOM-003` | `{"error":{"code":"ROOM-003","message":"not a member"}}` | Не состоит |
| 403 | `ROOM-004` | `{"error":{"code":"ROOM-004","message":"insufficient role: admin or owner required"}}` | member пытается создать |
| 500 | `INTERNAL` | `…` | После 3 ретраев — все коды коллизировали (крайне маловероятно) |

---

## POST /api/v1/rooms/join/{code}

Вступить в комнату по активному инвайт-коду. Создатель кода также может вступить (но он, скорее всего, уже member как owner — получит 409).

**Request:**

```
Path:
  code: string (8 chars, Crockford base32; case-insensitive в URL,
                нормализуется в upper-case на сервере)
Headers:
  Authorization: Bearer <jwt>
Body: пусто или {} (игнорируется)
```

**Response 200:**

```json
{
  "id": "uuid-string",
  "ownerId": "uuid-string",
  "name": "My Room",
  "createdAt": "2026-05-11T12:00:00Z"
}
```

После успешного вступления пользователь становится `member`.

**Error responses:**

| Status | Error Code | Body | Когда |
|--------|------------|------|-------|
| 400 | `ROOM-008` | `{"error":{"code":"ROOM-008","message":"invalid invite code"}}` | Длина не 8 / символ вне Crockford-алфавита |
| 404 | `ROOM-007` | `{"error":{"code":"ROOM-007","message":"invite not found or revoked"}}` | Активного инвайта с таким кодом нет |
| 409 | `ROOM-006` | `{"error":{"code":"ROOM-006","message":"already a member"}}` | Уже состоит в этой комнате |
| 401 | `AUTH-010`/`AUTH-011` | … | JWT |

---

## POST /api/v1/rooms/{roomID}/channels

Создать канал внутри комнаты. Доступно admin/owner.

**Request:**

```
Path:
  roomID: uuid (required)
Headers:
  Authorization: Bearer <jwt>
  Content-Type: application/json

Body:
  name: string (required, 1..64 chars after trim, no control chars,
                unique within room — case-sensitive)
  kind: string (required, "text" | "voice")
```

**Response 201:**

```json
{
  "id": "uuid-string",
  "roomId": "uuid-string",
  "name": "general",
  "kind": "text",
  "createdAt": "2026-05-11T12:00:00Z"
}
```

**Error responses:**

| Status | Error Code | Body | Когда |
|--------|------------|------|-------|
| 400 | `CHANNEL-005` | `{"error":{"code":"CHANNEL-005","message":"invalid request body"}}` | JSON не парсится |
| 400 | `CHANNEL-001` | `{"error":{"code":"CHANNEL-001","message":"invalid channel name"}}` | name пустое / > 64 / control chars |
| 400 | `CHANNEL-002` | `{"error":{"code":"CHANNEL-002","message":"invalid channel kind"}}` | kind не text/voice |
| 404 | `CHANNEL-003` | `{"error":{"code":"CHANNEL-003","message":"channel or room not found"}}` | roomID не UUID |
| 403 | `CHANNEL-006` | `{"error":{"code":"CHANNEL-006","message":"access denied: not a member"}}` | Не состоит в комнате |
| 403 | `CHANNEL-007` | `{"error":{"code":"CHANNEL-007","message":"insufficient role: admin or owner required"}}` | member пытается создать |
| 409 | `CHANNEL-004` | `{"error":{"code":"CHANNEL-004","message":"channel name already taken"}}` | Имя занято в этой комнате |

---

## GET /api/v1/rooms/{roomID}/channels

Список каналов комнаты. Доступно member/admin/owner.

**Request:**

```
Path:
  roomID: uuid (required)
Headers:
  Authorization: Bearer <jwt>
```

**Response 200:**

```json
{
  "items": [
    {
      "id": "uuid-string",
      "roomId": "uuid-string",
      "name": "general",
      "kind": "text",
      "createdAt": "2026-05-11T12:00:00Z"
    },
    {
      "id": "uuid-string",
      "roomId": "uuid-string",
      "name": "Voice room 1",
      "kind": "voice",
      "createdAt": "2026-05-11T13:00:00Z"
    }
  ]
}
```

Пустой ответ: `{ "items": [] }`.

**Error responses:**

| Status | Error Code | Когда |
|--------|------------|-------|
| 404 | `CHANNEL-003` | roomID не UUID |
| 403 | `CHANNEL-006` | Не состоит |

---

## DELETE /api/v1/rooms/{roomID}/channels/{channelID}

Удалить канал. Доступно admin/owner.

**Request:**

```
Path:
  roomID: uuid (required)
  channelID: uuid (required)
Headers:
  Authorization: Bearer <jwt>
```

**Response 204:** пустое тело.

**Error responses:**

| Status | Error Code | Body | Когда |
|--------|------------|------|-------|
| 404 | `CHANNEL-003` | `{"error":{"code":"CHANNEL-003","message":"channel or room not found"}}` | roomID/channelID не UUID, или канал не принадлежит указанной комнате |
| 403 | `CHANNEL-006` | `{"error":{"code":"CHANNEL-006","message":"access denied: not a member"}}` | Не состоит в комнате |
| 403 | `CHANNEL-007` | `{"error":{"code":"CHANNEL-007","message":"insufficient role: admin or owner required"}}` | member пытается удалить |

---

## Реестр кодов ошибок (PR-2)

Префикс `ROOM-`:

| Код | Описание | HTTP |
|-----|----------|------|
| `ROOM-001` | invalid room name | 400 |
| `ROOM-002` | room not found | 404 |
| `ROOM-003` | not a member | 403 |
| `ROOM-004` | insufficient role: admin or owner required | 403 |
| `ROOM-005` | only owner can delete room | 403 |
| `ROOM-006` | already a member | 409 |
| `ROOM-007` | invite not found or revoked | 404 |
| `ROOM-008` | invalid invite code | 400 |
| `ROOM-009` | invalid request body | 400 |

Префикс `CHANNEL-`:

| Код | Описание | HTTP |
|-----|----------|------|
| `CHANNEL-001` | invalid channel name | 400 |
| `CHANNEL-002` | invalid channel kind | 400 |
| `CHANNEL-003` | channel or room not found | 404 |
| `CHANNEL-004` | channel name already taken | 409 |
| `CHANNEL-005` | invalid request body | 400 |
| `CHANNEL-006` | access denied: not a member | 403 |
| `CHANNEL-007` | insufficient role: admin or owner required | 403 |

Конфликтов с существующими `AUTH-001..012` нет.
