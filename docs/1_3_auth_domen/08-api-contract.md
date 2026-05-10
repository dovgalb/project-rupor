---
parent: ./README.md
view: contract
---

# 08 — HTTP API Contract

В фазе 1.3 в `/api/v1/` появляется группа `/auth/*` с четырьмя эндпоинтами. Префикс зарезервирован в `docs/1_1_project-structure/08-api-contract.md:74-82`.

Все запросы и ответы — `application/json; charset=utf-8`. Все timestamps — RFC3339 в UTC (`2026-05-07T15:42:01Z`). Все UUID — каноническая 36-символьная форма (`550e8400-e29b-41d4-a716-446655440000`).

## Общие соглашения

### Формат тела ошибки

Все ошибочные ответы (4xx, 5xx, кроме 405/404 от chi-дефолтов) имеют единое тело:

```json
{
  "error": {
    "code": "AUTH-XXX",
    "message": "human-readable description"
  }
}
```

`code` — стабильный идентификатор для клиента; `message` — человекочитаемое описание (на английском в этой фазе; локализация в фазе 5 — фронт сам делает lookup по `code`). Оба поля обязательны.

### Карта кодов ошибок (диапазон AUTH-001..AUTH-099)

| Code | Описание | HTTP Status |
|------|----------|-------------|
| AUTH-001 | invalid email format | 400 |
| AUTH-002 | invalid username format | 400 |
| AUTH-003 | invalid password format | 400 |
| AUTH-004 | email already taken | 409 |
| AUTH-005 | username already taken | 409 |
| AUTH-006 | invalid credentials | 401 |
| AUTH-007 | refresh token not found | 401 |
| AUTH-008 | refresh token revoked | 401 |
| AUTH-009 | refresh token expired | 401 |
| AUTH-010 | access token invalid | 401 |
| AUTH-011 | access token expired | 401 |
| AUTH-012 | malformed request body | 400 |
| INTERNAL | internal server error | 500 |

`INTERNAL` — единое значение `code` для всех 5xx (без AUTH-префикса); подробности — в логах сервера.

Существующие коды в проекте (`config/config.go:48-103`): `CONFIG-001`, `CONFIG-002`, `CONFIG-003` — диапазон config. Конфликтов нет.

### Авторизация

- `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh` — публичные, токен не требуется.
- `GET /auth/me` — требует `Authorization: Bearer <accessToken>`.

В фазе 1.3 проверка делается inline в `MeHandler` (см. `03-decisions.md`, ADR-009). В фазе 1.4 — переедет в middleware.

---

## POST /api/v1/auth/register

**Назначение:** создать нового пользователя.

### Request

```
Method: POST
Path:   /api/v1/auth/register
Headers:
  Content-Type: application/json
```

**Body:**

```json
{
  "email": "user@example.com",
  "username": "tester",
  "password": "sup3rs3cret"
}
```

| Поле | Тип | Обязательно | Ограничения |
|------|-----|-------------|-------------|
| `email` | string | да | RFC-like email, ≤254 символов; нормализуется (trim + lowercase) |
| `username` | string | да | ASCII alnum/`_`/`-`, 3..32 символов |
| `password` | string | да | 8..72 байт |

### Response 201 Created

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "tester",
  "createdAt": "2026-05-07T15:42:01Z"
}
```

### Error responses

| Status | Code | Body | Когда |
|--------|------|------|-------|
| 400 | AUTH-012 | `{"error":{"code":"AUTH-012","message":"malformed request body"}}` | Не-JSON, отсутствуют обязательные поля, лишние поля игнорируются |
| 400 | AUTH-001 | `{"error":{"code":"AUTH-001","message":"invalid email"}}` | Не похоже на email |
| 400 | AUTH-002 | `{"error":{"code":"AUTH-002","message":"invalid username"}}` | Длина / запрещённые символы |
| 400 | AUTH-003 | `{"error":{"code":"AUTH-003","message":"invalid password"}}` | Длина <8 или >72 |
| 409 | AUTH-004 | `{"error":{"code":"AUTH-004","message":"email already taken"}}` | Уникальность email нарушена |
| 409 | AUTH-005 | `{"error":{"code":"AUTH-005","message":"username already taken"}}` | Уникальность username нарушена |
| 500 | INTERNAL | `{"error":{"code":"INTERNAL","message":"internal"}}` | Сбой БД, сбой bcrypt и т.п. |

---

## POST /api/v1/auth/login

**Назначение:** обменять email+password на пару access/refresh токенов.

### Request

```
Method: POST
Path:   /api/v1/auth/login
Headers:
  Content-Type: application/json
```

**Body:**

```json
{
  "email": "user@example.com",
  "password": "sup3rs3cret"
}
```

| Поле | Тип | Обязательно | Ограничения |
|------|-----|-------------|-------------|
| `email` | string | да | свободный формат; валидация мягкая, ошибки → AUTH-006 |
| `password` | string | да | свободная строка; валидация мягкая, ошибки → AUTH-006 |

### Response 200 OK

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAiLCJpYXQiOjE3NTUwMDAwMDAsImV4cCI6MTc1NTAwMDkwMH0.xxx",
  "refreshToken": "VGhpcy1pcy1hLWZha2UtcmVmcmVzaC10b2tlbi1mb3ItZG9jcw",
  "accessExpiresAt": "2026-05-07T15:57:01Z",
  "refreshExpiresAt": "2026-06-06T15:42:01Z"
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `accessToken` | string | JWT (HS256), claims: `sub` (UserID), `iat`, `exp` |
| `refreshToken` | string | base64url(32 случайных байт), без padding (длина 43) |
| `accessExpiresAt` | string (RFC3339) | момент истечения access (в UTC) |
| `refreshExpiresAt` | string (RFC3339) | момент истечения refresh (в UTC) |

### Error responses

| Status | Code | Body | Когда |
|--------|------|------|-------|
| 400 | AUTH-012 | стандарт | malformed body |
| 401 | AUTH-006 | `{"error":{"code":"AUTH-006","message":"invalid credentials"}}` | Любая из: невалидный формат email, email не найден, пароль не совпал. **Намеренно** одинаковый ответ — защита от user-enumeration (`03-decisions.md`, ADR-008) |
| 500 | INTERNAL | стандарт | Сбой БД / bcrypt / crypto/rand |

---

## POST /api/v1/auth/refresh

**Назначение:** обменять `refreshToken` на новую пару access+refresh (с ротацией).

### Request

```
Method: POST
Path:   /api/v1/auth/refresh
Headers:
  Content-Type: application/json
```

**Body:**

```json
{
  "refreshToken": "VGhpcy1pcy1hLWZha2UtcmVmcmVzaC10b2tlbi1mb3ItZG9jcw"
}
```

| Поле | Тип | Обязательно | Ограничения |
|------|-----|-------------|-------------|
| `refreshToken` | string | да | base64url, после декода — 32 байт |

### Response 200 OK

Идентичен телу ответа `POST /auth/login`:

```json
{
  "accessToken": "eyJhbGc...",
  "refreshToken": "QW5vdGhlci1mYWtlLXJlZnJlc2gtdG9rZW4tZm9yLWRvY3M",
  "accessExpiresAt": "2026-05-07T16:00:01Z",
  "refreshExpiresAt": "2026-06-06T15:45:01Z"
}
```

После успешного refresh старый `refreshToken` помечается отозванным и не может быть использован повторно. Клиент **обязан** заменить хранимое значение `refreshToken` на новое.

### Error responses

| Status | Code | Body | Когда |
|--------|------|------|-------|
| 400 | AUTH-012 | стандарт | malformed body |
| 401 | AUTH-007 | `{"error":{"code":"AUTH-007","message":"refresh token not found"}}` | Токен не декодируется из base64url; длина после декода ≠ 32; хеш не найден в БД |
| 401 | AUTH-008 | `{"error":{"code":"AUTH-008","message":"refresh token revoked"}}` | Токен в БД, но `revoked_at != NULL` (либо ранее отозван, либо race) |
| 401 | AUTH-009 | `{"error":{"code":"AUTH-009","message":"refresh token expired"}}` | Токен в БД, но `now ≥ expires_at` |
| 500 | INTERNAL | стандарт | Сбой БД / crypto/rand / транзакции |

---

## GET /api/v1/auth/me

**Назначение:** вернуть данные текущего залогиненного пользователя.

### Request

```
Method: GET
Path:   /api/v1/auth/me
Headers:
  Authorization: Bearer <accessToken>
```

| Header | Тип | Обязательно | Ограничения |
|--------|-----|-------------|-------------|
| `Authorization` | string | да | Точный формат `Bearer ` + JWT (CamelCase scheme) |

Тело запроса не передаётся.

### Response 200 OK

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "username": "tester",
  "createdAt": "2026-05-07T15:42:01Z"
}
```

Структура совпадает с `RegisterUserResponse` (намеренно — `tester` после регистрации сразу видит ту же форму, что и `me` спустя месяц).

### Error responses

| Status | Code | Body | Когда |
|--------|------|------|-------|
| 401 | AUTH-010 | `{"error":{"code":"AUTH-010","message":"access token invalid"}}` | Нет заголовка; не Bearer; не парсится; неправильная подпись; неправильный алгоритм; sub не парсится в UUID; пользователь по sub не найден |
| 401 | AUTH-011 | `{"error":{"code":"AUTH-011","message":"access token expired"}}` | `exp < now` |
| 500 | INTERNAL | стандарт | Сбой БД |

---

## Поведенческие гарантии

- Все эндпоинты — синхронные. Никаких long-poll/SSE/WS в фазе 1.3.
- Логирование: каждый запрос пишется через `slog` access-log middleware (см. `docs/1_1_project-structure/08-api-contract.md:57`, OQ-9). На уровне INFO логируется method/path/status/duration; password/token не логируются никогда.
- Метрики: не вводим в фазе 1.3 (отдельная фича «observability»).
- Rate-limiting: не вводим в фазе 1.3. Для login/refresh — open question (отдельная фича в фазе 2/3).
- CORS: не настраивается в фазе 1.3. Будет добавлен в фазе 1.4 вместе с middleware (`general_plan.md:118`).

## Резерв префиксов (повтор для целостности)

- `/api/v1/health` — фаза 1.1 ✅
- `/api/v1/auth/*` — **эта фаза 1.3** ✅
- `/api/v1/rooms/*`, `/api/v1/rooms/:id/channels/*` — фаза 2
- `/api/v1/channels/:id/messages` — фаза 3 (REST)
- `/api/v1/ws` — фаза 3 (WebSocket)

## Test Plan (HTTP-уровень, manual)

Файл `manual_qa/auth/test-flow.http` (создаётся при реализации, опциональный артефакт; формат IDE HTTP Client). Шаги:

| Шаг | Команда | Ожидание |
|-----|---------|----------|
| 1. Регистрация | `POST /auth/register` со свежим email | 201 + JSON, `id` — uuid |
| 2. Повтор регистрации | `POST /auth/register` с тем же email | 409 + AUTH-004 |
| 3. Login | `POST /auth/login` с правильным паролем | 200 + accessToken + refreshToken |
| 4. Login с неверным паролем | `POST /auth/login` | 401 + AUTH-006 |
| 5. /me с access | `GET /auth/me` Bearer accessToken | 200 + JSON совпадает с шагом 1 |
| 6. /me без auth | `GET /auth/me` | 401 + AUTH-010 |
| 7. Refresh | `POST /auth/refresh` с refreshToken из шага 3 | 200 + новые токены |
| 8. Refresh с ротированным | `POST /auth/refresh` со старым refreshToken | 401 + AUTH-008 |
| 9. /me с истёкшим access | подождать `JWT_ACCESS_TTL` или подделать exp | 401 + AUTH-011 |
| 10. Сборка под нагрузкой | 100 параллельных register с разными email | все 201 (либо 409 при коллизии username) |

(Шаг 10 — smoke на race; в integration-тестах задачи 1.5 будет проверка с настоящим Postgres-уровнем.)
