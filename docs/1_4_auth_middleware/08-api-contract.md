---
parent: ./README.md
view: contract
---

# 08 — HTTP API Contract (изменения surface)

Фаза 1.4 **не вводит новых эндпоинтов**. Все четыре эндпоинта auth (`/api/v1/auth/{register,login,refresh,me}`) и `/api/v1/health` остаются с теми же сигнатурами и кодами ошибок. Меняется HTTP-поведение **на каждом** эндпоинте за счёт middleware-stack:

1. На каждый ответ выставляется новый заголовок `X-Request-ID`.
2. На запросы с `Origin` (cross-origin) выставляются `Access-Control-Allow-Origin` и `Vary: Origin`.
3. Появляется поддержка preflight `OPTIONS` для всех путей (включая `/api/v1/auth/*` и `/api/v1/health`).
4. На `GET /auth/me` источник 401-ответа меняется (с inline в хендлере на middleware), но контракт ответа сохраняется.
5. На любой panic в нижестоящем коде сервер возвращает 500 + `INTERNAL` envelope (вместо краха процесса).

Все запросы и ответы — `application/json; charset=utf-8`. Все timestamps — RFC3339 в UTC. Все UUID — каноническая 36-символьная форма.

## Общие соглашения (актуализация)

### Заголовки ответов (новое)

| Заголовок | Когда выставляется | Значение |
|-----------|--------------------|----------|
| `X-Request-ID` | **на каждом ответе** (включая 4xx/5xx, OPTIONS) | UUID v4, либо повторение клиентского значения, если оно валидно (1 ≤ len ≤ 128, ASCII-печатные) |
| `Access-Control-Allow-Origin` | если `Origin` ∈ whitelist `CORS_ALLOWED_ORIGINS` | значение `Origin` запроса (точное, не wildcard) |
| `Vary: Origin` | если `Origin` ∈ whitelist | `Origin` (для корректного кеширования) |
| `Access-Control-Allow-Methods` | preflight (OPTIONS) match | `GET, POST, PUT, DELETE, OPTIONS` |
| `Access-Control-Allow-Headers` | preflight match | `Authorization, Content-Type, X-Request-ID` |
| `Access-Control-Max-Age` | preflight match | `600` (10 минут) |
| `Access-Control-Allow-Credentials` | **не выставляется** в фазе 1.4 | (см. `03-decisions.md`, ADR-004) |

Все заголовки выставляются ДО `WriteHeader(...)` в middleware. На уровне хендлеров заголовки не меняются.

### Заголовки запроса (опциональные)

| Заголовок | Назначение | Поведение |
|-----------|------------|-----------|
| `X-Request-ID` | клиентский коррелятор для трассировки | принимается, если валиден; иначе генерируется свой |
| `Origin` | cross-origin запрос | проверяется по whitelist; если match — выставляются CORS-заголовки в ответ |
| `Authorization` | Bearer для protected-маршрутов | требуется на `/auth/me`; формат `Bearer <jwt>` (CamelCase scheme) |

### Карта кодов ошибок (без изменений по диапазону `AUTH-*`)

Карта 1.3 (`docs/1_3_auth_domen/08-api-contract.md:29-46`) сохраняется. В 1.4 новых кодов **не вводится**.

Поведение `AUTH-010` и `AUTH-011` теперь генерируется в `RequireAuth` (а не в `MeHandler`):

| Code | Описание | HTTP Status | Источник в 1.3 | Источник в 1.4 |
|------|----------|-------------|----------------|----------------|
| AUTH-010 | access token invalid | 401 | inline в `me_handler.go:23-37` | `RequireAuth` middleware (любые проблемы с Bearer/подписью/sub) |
| AUTH-011 | access token expired | 401 | inline в `me_handler.go:31` | `RequireAuth` middleware (`domain.ErrAccessTokenExpired`) |
| INTERNAL | internal server error | 500 | через `error_mapper` default branch | + `Recover` middleware (panic-recovery) |

**Конфиг-коды**: добавляется `CONFIG-006` для `CORS_ALLOWED_ORIGINS`-валидации (это код запуска сервера, не runtime API).

| Code | Field | HTTP / поведение |
|------|-------|------------------|
| CONFIG-006 | `CORS_ALLOWED_ORIGINS` | сервер не стартует, exit code != 0; в логах `slog.Error("config invalid", code=CONFIG-006, field=CORS_ALLOWED_ORIGINS, reason=<...>)`. |

## Preflight `OPTIONS` (новое поведение для всех путей)

### Request

```
Method: OPTIONS
Path:   /api/v1/auth/login   (или любой другой путь под /api/v1/...)
Headers:
  Origin: http://localhost:5173
  Access-Control-Request-Method: POST
  Access-Control-Request-Headers: Content-Type
```

### Response 204 No Content (Origin в whitelist)

```
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: http://localhost:5173
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID
Access-Control-Max-Age: 600
Vary: Origin
X-Request-ID: <uuid>
```

Тело — пустое.

### Response 405 Method Not Allowed (Origin вне whitelist или нет Origin)

`OPTIONS`-запрос проходит дальше в chi-router, который для большинства путей не зарегистрировал OPTIONS-handler → возвращает 405 plain-text (chi default).

```
HTTP/1.1 405 Method Not Allowed
X-Request-ID: <uuid>
Content-Type: text/plain; charset=utf-8

Method Not Allowed
```

Браузер интерпретирует отсутствие `Access-Control-Allow-Origin` как отказ в CORS и блокирует основной запрос на стороне клиента.

## Изменения по конкретным эндпоинтам

### POST /api/v1/auth/register, /login, /refresh

**Изменения:**
- Все ответы (success и error) теперь содержат заголовок `X-Request-ID`.
- При cross-origin запросе — добавляются CORS-заголовки.
- При panic в хендлере (нештатная ситуация) — 500 + `INTERNAL` envelope от Recover middleware (вместо краха процесса).

**Без изменений:**
- Тело request/response (DTO).
- Статусы 200/201/400/401/409/500.
- Карта кодов ошибок `AUTH-001..AUTH-009`, `AUTH-012`.
- Поведение валидации (на уровне use case и domain).

### GET /api/v1/auth/me

**Изменения:**
- 401-ответы при отсутствии токена / неправильной схеме / истечении / неверной подписи теперь генерируются `RequireAuth` middleware (см. `02-behavior.md`, Use Case 3).
- В случае panic в `MeHandler` (например, баг будущего кода) — 500 + `INTERNAL` от Recover.
- Ответы содержат `X-Request-ID`.
- При cross-origin — CORS-заголовки.

**Без изменений:**
- Структура запроса (`Authorization: Bearer <jwt>`).
- Структура успешного ответа (`meResponse`).
- HTTP-коды (401 для AUTH-010/011, 200 для success, 500 для INTERNAL).
- Тело ошибок (envelope `{"error":{"code","message"}}`).

#### Регрессия 1.3 — гарантии для существующих клиентов

| Сценарий | До 1.4 | После 1.4 |
|----------|--------|-----------|
| `GET /auth/me` без `Authorization` | 401 + AUTH-010 (от `me_handler.go:25-28`) | 401 + AUTH-010 (от `RequireAuth`) |
| `GET /auth/me` с `Authorization: Basic xxx` | 401 + AUTH-010 (от `me_handler.go:25`) | 401 + AUTH-010 (от `RequireAuth`) |
| `GET /auth/me` с `Authorization: Bearer ` (пустой токен) | 401 + AUTH-010 (от `me_handler.go:26`) | 401 + AUTH-010 (от `RequireAuth`) |
| `GET /auth/me` с истёкшим токеном | 401 + AUTH-011 (от `me_handler.go:31` через `mapError`) | 401 + AUTH-011 (от `RequireAuth`) |
| `GET /auth/me` с невалидной подписью | 401 + AUTH-010 | 401 + AUTH-010 (от `RequireAuth`) |
| `GET /auth/me` с `sub` не в UUID | 401 + AUTH-010 | 401 + AUTH-010 (от `RequireAuth`) |
| `GET /auth/me` с `sub` user'а, которого нет в БД | 401 + AUTH-010 (через `mapError(domain.ErrUserNotFound)`) | 401 + AUTH-010 (через `mapError(domain.ErrUserNotFound)` — usecase, не middleware) |
| `GET /auth/me` happy path | 200 + JSON | 200 + JSON + `X-Request-ID` |

Все 5 существующих тестов 1.3 (`TestMeHandler_Success`, `_NoAuth`, `_BadScheme`, `_InvalidSignature`, `_TokenExpired`) проходят без изменений (см. `04-testing.md`, раздел «Регрессия 1.3»).

### GET /api/v1/health

**Изменения:**
- Ответ содержит `X-Request-ID`.
- Логируется через access-log (метод/путь/статус/duration/request_id).
- При cross-origin — CORS-заголовки.

**Без изменений:**
- Тело (`{"status":"ok"}`).
- Статус 200.

## Поведение `Access-Control-*` на разных эндпоинтах

Все эндпоинты ведут себя одинаково в части CORS — middleware висит глобально. Различий по путям нет.

| Запрос | Поведение |
|--------|-----------|
| `OPTIONS /api/v1/<любой путь>` + `Origin` в whitelist | 204 + полный набор Access-Control-* + Vary + X-Request-ID |
| `OPTIONS /api/v1/<любой путь>` + `Origin` НЕ в whitelist | 405 (chi default) + X-Request-ID (без Access-Control-*) |
| Любой не-OPTIONS + `Origin` в whitelist | хендлер обрабатывает обычно + `Access-Control-Allow-Origin` + `Vary: Origin` + X-Request-ID |
| Любой не-OPTIONS + `Origin` НЕ в whitelist | хендлер обрабатывает обычно (без Access-Control-*) + X-Request-ID |
| Любой запрос без `Origin` (например, curl) | хендлер обрабатывает обычно (без Access-Control-*) + X-Request-ID |

## Поведенческие гарантии (актуализация для 1.4)

- Все эндпоинты — синхронные. Никаких long-poll/SSE/WS в фазе 1.4.
- Логирование: каждый запрос пишется через `Logger` middleware на уровне INFO; password/token не логируются никогда.
- Recover: panic в любом хендлере не уронит процесс; ответ — 500 + `INTERNAL` envelope; stack trace — в `slog.Error`.
- Метрики: не вводим в фазе 1.4 (отдельная фича «observability»).
- Rate-limiting: не вводим в фазе 1.4. Открытый вопрос (`03-decisions.md`, OQ-6).
- CORS: настраивается через env `CORS_ALLOWED_ORIGINS`; default `http://localhost:5173`; wildcard `*` НЕ поддерживается.
- Request-ID: принимается от клиента (валидное значение) или генерируется UUID v4; всегда выставляется в response.

## Резерв префиксов (повтор для целостности)

- `/api/v1/health` — фаза 1.1 ✅; в 1.4 проходит через middleware-stack
- `/api/v1/auth/*` — фаза 1.3 ✅; в 1.4 protected-группа за RequireAuth
- `/api/v1/rooms/*` — фаза 2 (за RequireAuth)
- `/api/v1/channels/:id/messages` — фаза 3 (за RequireAuth)
- `/api/v1/ws` — фаза 3 (своя авторизация через query-string `?token=<jwt>`, не Bearer)

## Test Plan (HTTP-уровень, manual)

Файл `manual_qa/1_4_auth_middleware/` (создаётся при реализации, формат IDE HTTP Client).

### Файлы

- `00_smoke.http` — общая прогонка через middleware-stack (RequestID + Logger + CORS + Recover, без auth).
- `01_cors_preflight.http` — preflight + cross-origin запросы.
- `02_request_id.http` — приём клиентского `X-Request-ID`, отвержение невалидного.
- `03_require_auth.http` — все 401-сценарии RequireAuth.
- `04_recover.http` — endpoint, который panic'ит (только для тестов; вынести в build-tag или отдельный handler в `cmd/server/`?). Альтернатива: e2e-тест без `.http`.

### Шаги для `01_cors_preflight.http`

| Шаг | Команда | Ожидание |
|-----|---------|----------|
| 1. Preflight allowed | `OPTIONS /api/v1/auth/login` + `Origin: http://localhost:5173` | 204 + Access-Control-Allow-Origin + Methods + Headers + Max-Age |
| 2. Preflight denied | `OPTIONS /api/v1/auth/login` + `Origin: http://evil.com` | 405 (chi default) без Access-Control-* |
| 3. Cross-origin success | `POST /api/v1/auth/login` + `Origin: http://localhost:5173` + body | 200 + Access-Control-Allow-Origin + Vary: Origin |
| 4. Same-origin (без Origin) | `POST /api/v1/auth/login` без `Origin` | 200 без Access-Control-* |

### Шаги для `02_request_id.http`

| Шаг | Команда | Ожидание |
|-----|---------|----------|
| 1. Без X-Request-ID | `GET /api/v1/health` | 200 + `X-Request-ID: <uuid>` (формат UUID v4) |
| 2. С валидным | `GET /api/v1/health` + `X-Request-ID: my-correlation-1` | 200 + `X-Request-ID: my-correlation-1` |
| 3. С CRLF (попытка инъекции) | `GET /api/v1/health` + `X-Request-ID: bad\r\nX-Hack: 1` | 200 + сгенерированный UUID (НЕ переданное значение); `X-Hack` не появляется |
| 4. С не-ASCII | `GET /api/v1/health` + `X-Request-ID: τ` | 200 + сгенерированный UUID |

### Шаги для `03_require_auth.http`

| Шаг | Команда | Ожидание |
|-----|---------|----------|
| 1. Без Authorization | `GET /api/v1/auth/me` | 401 + AUTH-010 + X-Request-ID |
| 2. Bearer пустой | `GET /api/v1/auth/me` + `Authorization: Bearer ` | 401 + AUTH-010 |
| 3. Basic | `GET /api/v1/auth/me` + `Authorization: Basic xxx` | 401 + AUTH-010 |
| 4. lowercase bearer | `GET /api/v1/auth/me` + `Authorization: bearer xxx` | 401 + AUTH-010 (case-sensitive) |
| 5. Невалидная подпись | `GET /api/v1/auth/me` + поломанный JWT | 401 + AUTH-010 |
| 6. Истёкший | `GET /api/v1/auth/me` + JWT с `exp < now` | 401 + AUTH-011 |
| 7. Happy path | `GET /api/v1/auth/me` + валидный Bearer | 200 + meResponse |
