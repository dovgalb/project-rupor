---
phase: 6
name: arch tests + manual_qa
layer: tests
depends_on: [phase-01, phase-02, phase-03, phase-05]
plan: ./README.md
---

# Phase 6: Архитектурные smoke-тесты + manual QA

## Цель

Закрепить структурные правила фичи в `arch_test.go` (две новых проверки импортов) и подготовить набор `.http`-файлов для ручного тестирования всех аспектов middleware-stack: CORS preflight, request-id, RequireAuth-сценарии, smoke через полную цепочку.

## Контекст

Готовые компоненты:
- Существующий `arch_test.go` (`arch_test.go:1-148`) с 4 тестами — архитектурный фундамент. Расширяется ровно двумя проверками.
- Существующий `manual_qa/1_3_auth/` — образец организации `.http`-файлов, который копируется по структуре.
- `pkg/httpx/`, `internal/auth/transport/http/middleware/` — реализованы (фазы 1–3).
- Сервер собирается с middleware-stack (фаза 5).

Эта фаза — финальная. После неё фича закрыта, выполнены все пункты success criteria из `plan/README.md`.

## Файлы для модификации

### `arch_test.go`

**Что меняется:** добавление двух тестов в конец файла. Импорты не меняются (используется тот же `collectImports` helper из фазы 1.3).

**Точка вставки:** после `TestArchitecture_UsecaseUsedAsContract` (`arch_test.go:133-147`).

**Новые тесты:**

```go
func TestArchitecture_PkgHttpxImports(t *testing.T) {
    t.Parallel()

    allowed := map[string]struct{}{
        "github.com/google/uuid": {},
    }

    dirs := []string{"pkg/httpx", "pkg/httpx/middleware"}
    for _, dir := range dirs {
        imports := collectImports(t, dir)
        for file, ims := range imports {
            for _, im := range ims {
                if isStdlib(im) {
                    continue
                }
                if _, ok := allowed[im]; ok {
                    continue
                }
                t.Fatalf("%s/%s imports forbidden %q", dir, file, im)
            }
        }
    }
}

func TestArchitecture_AuthMiddlewareImports(t *testing.T) {
    t.Parallel()

    allowed := map[string]struct{}{
        "github.com/google/uuid":                                            {},
        "github.com/dovgalb/project-rupor/internal/auth/usecase":            {},
        "github.com/dovgalb/project-rupor/internal/auth/domain":             {},
        "github.com/dovgalb/project-rupor/pkg/httpx":                        {},
        "github.com/go-chi/chi/v5":                                          {},
    }

    imports := collectImports(t, "internal/auth/transport/http/middleware")
    for file, ims := range imports {
        for _, im := range ims {
            if isStdlib(im) {
                continue
            }
            if _, ok := allowed[im]; ok {
                continue
            }
            // Запрет на repository
            if strings.HasPrefix(im, "github.com/dovgalb/project-rupor/internal/auth/repository/") {
                t.Fatalf("%s imports adapter %q", file, im)
            }
            t.Fatalf("%s imports forbidden %q", file, im)
        }
    }
}
```

**Замечание:** `chi/v5` в whitelist `RequireAuth` — middleware возвращает `func(http.Handler) http.Handler`, что не требует чёткого chi-импорта; фактически в реализации `RequireAuth` чистый stdlib. Включаем chi в whitelist на случай, если `RequireAuth` использует `chi.Route`-context для логирования (не используется в дизайне 1.4, но оставляем заголовок мягкий — проверяющий запрет `repository/` важнее).

### `.env` и `.env.example`

Без изменений (обновлены в фазе 4).

## Файлы для создания

### `manual_qa/1_4_auth_middleware/README.md`

```markdown
# Manual QA: 1_4 auth middleware

HTTP-запросы для ручного тестирования middleware-stack:
RequestID, Recover, Logger, CORS, RequireAuth.

Формат `.http` — совместим с VS Code REST Client и JetBrains HTTP Client.

## Подготовка стенда

```bash
make dc-up && make migrate-up && make run
```

В `.env` (или ENV) должны быть выставлены:
- `JWT_SECRET=...`
- `DATABASE_URL=...`
- `CORS_ALLOWED_ORIGINS=http://localhost:5173` (опционально, дефолт)

## Файлы

- `00_smoke.http` — общая прогонка через middleware-stack: health-endpoint, проверка `X-Request-ID` в каждом ответе.
- `01_cors_preflight.http` — preflight + cross-origin запросы; whitelisted и denied origins.
- `02_request_id.http` — клиентский `X-Request-ID`: приём валидного, отвержение CRLF/non-ASCII/слишком длинного.
- `03_require_auth.http` — все 401-сценарии RequireAuth: без токена, не-Bearer, пустой токен, невалидная подпись, истёкший токен, lowercase `bearer`.

## Переменные

В каждом файле наверху объявлены `@host` и whitelisted `@origin`. Меняйте по необходимости.

В `03_require_auth.http` для happy-path-проверки нужен валидный access-токен — получите его через `manual_qa/1_3_auth/00_flow.http` (register + login) и подставьте в `@access_token`.
```

### `manual_qa/1_4_auth_middleware/00_smoke.http`

```http
@host = http://localhost:8080

### Health через middleware-stack
GET {{host}}/api/v1/health

# Проверить:
# - Status 200
# - Response header "X-Request-ID" присутствует, формат UUID v4
# - Body: {"status":"ok"}
# - В логах сервера: slog INFO с полями method=GET, path=/api/v1/health, status=200, duration, request_id

### Health с клиентским X-Request-ID
GET {{host}}/api/v1/health
X-Request-ID: my-correlation-12345

# Проверить:
# - Response "X-Request-ID" совпадает с переданным "my-correlation-12345"
# - В логах request_id="my-correlation-12345"

### Любой неизвестный путь — 404 от chi (без JSON envelope)
GET {{host}}/api/v1/nonexistent

# Проверить:
# - Status 404
# - X-Request-ID присутствует (middleware подключилось до chi.NotFound)
# - Body: "404 page not found\n" (chi default, plain text — это OK для 1.4)
```

### `manual_qa/1_4_auth_middleware/01_cors_preflight.http`

```http
@host = http://localhost:8080
@allowed_origin = http://localhost:5173
@denied_origin = http://evil.com

### Preflight: whitelisted origin → 204
OPTIONS {{host}}/api/v1/auth/login
Origin: {{allowed_origin}}
Access-Control-Request-Method: POST
Access-Control-Request-Headers: Content-Type

# Проверить:
# - Status 204 No Content
# - Headers:
#   Access-Control-Allow-Origin: http://localhost:5173
#   Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
#   Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID
#   Access-Control-Max-Age: 600
#   Vary: Origin
#   X-Request-ID: <uuid>
# - Body пустой
# - Access-Control-Allow-Credentials НЕ выставлен

### Preflight: denied origin → 405 (без CORS-заголовков)
OPTIONS {{host}}/api/v1/auth/login
Origin: {{denied_origin}}
Access-Control-Request-Method: POST

# Проверить:
# - Status 405 Method Not Allowed (chi default — auth/login не регистрирует OPTIONS)
# - НЕТ Access-Control-Allow-Origin
# - X-Request-ID присутствует

### Cross-origin POST: whitelisted → CORS заголовки в ответе
POST {{host}}/api/v1/auth/login
Origin: {{allowed_origin}}
Content-Type: application/json

{"email": "x@x.com", "password": "wrong"}

# Проверить:
# - Status 401 + AUTH-006 (хендлер обработал — 401 от usecase)
# - Header Access-Control-Allow-Origin: http://localhost:5173
# - Header Vary: Origin
# - Body — стандартный envelope ошибки
# - X-Request-ID присутствует

### Cross-origin POST: denied → нет CORS заголовков
POST {{host}}/api/v1/auth/login
Origin: {{denied_origin}}
Content-Type: application/json

{"email": "x@x.com", "password": "wrong"}

# Проверить:
# - Status 401 + AUTH-006
# - НЕТ Access-Control-Allow-Origin
# - X-Request-ID присутствует
# - (Браузер на стороне клиента заблокировал бы; curl нет)

### Same-origin (без Origin header)
POST {{host}}/api/v1/auth/login
Content-Type: application/json

{"email": "x@x.com", "password": "wrong"}

# Проверить:
# - Status 401 + AUTH-006
# - НЕТ Access-Control-* (это not-CORS запрос)
# - X-Request-ID присутствует
```

### `manual_qa/1_4_auth_middleware/02_request_id.http`

```http
@host = http://localhost:8080

### Без X-Request-ID — middleware генерирует UUID
GET {{host}}/api/v1/health

# Проверить:
# - X-Request-ID присутствует
# - Формат — UUID v4 (8-4-4-4-12 hex с дефисами)
# - В логах сервера request_id == значению из header

### С валидным клиентским
GET {{host}}/api/v1/health
X-Request-ID: my-correlation-1

# Проверить:
# - X-Request-ID == "my-correlation-1" (как передали)
# - В логах request_id == "my-correlation-1"

### С CRLF (попытка инъекции в логи)
GET {{host}}/api/v1/health
X-Request-ID: bad\r\nX-Hack: 1

# Проверить:
# - X-Request-ID — сгенерированный UUID (НЕ переданное значение)
# - В response НЕТ заголовка X-Hack
# - В логах request_id — сгенерированный UUID

### С non-ASCII
GET {{host}}/api/v1/health
X-Request-ID: τεστ

# Проверить:
# - X-Request-ID — сгенерированный UUID

### С >128 символов
GET {{host}}/api/v1/health
X-Request-ID: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

# Проверить:
# - X-Request-ID — сгенерированный UUID
```

### `manual_qa/1_4_auth_middleware/03_require_auth.http`

```http
@host = http://localhost:8080
@access_token = <получи из manual_qa/1_3_auth/02_login.http>

### Без Authorization
GET {{host}}/api/v1/auth/me

# Проверить:
# - Status 401
# - Body: {"error":{"code":"AUTH-010","message":"access token invalid"}}
# - X-Request-ID присутствует

### Authorization не Bearer
GET {{host}}/api/v1/auth/me
Authorization: Basic dGVzdDp0ZXN0

# Проверить:
# - Status 401 + AUTH-010

### Bearer пустой
GET {{host}}/api/v1/auth/me
Authorization: Bearer

# Проверить:
# - Status 401 + AUTH-010

### lowercase bearer (case-sensitive)
GET {{host}}/api/v1/auth/me
Authorization: bearer {{access_token}}

# Проверить:
# - Status 401 + AUTH-010 (даже если access_token валиден)

### Невалидная подпись
GET {{host}}/api/v1/auth/me
Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.invalid_payload.fake_signature

# Проверить:
# - Status 401 + AUTH-010

### Happy path
GET {{host}}/api/v1/auth/me
Authorization: Bearer {{access_token}}

# Проверить:
# - Status 200
# - Body: {"id":"<uuid>","email":"...","username":"...","createdAt":"..."}
# - X-Request-ID присутствует
# - В логах сервера user_id="<uid>" (через userIDHook)
```

## Файлы для создания (продолжение)

Если в фазе 5 был создан `cmd/server/main_test.go` — он остаётся как был. Если не был создан, в фазе 6 ничего не добавляется на этот счёт.

## Ключевые решения

- **`arch_test.go` ловит структурные нарушения** — каждое нарушение слоёв падает на CI ещё до запуска кода. Это закрепляет ADR-003 (разделение пакетов) и ADR-001 (без зависимостей в `pkg/httpx/`).
- **Manual QA как контракт** — `.http`-файлы документируют ожидаемое поведение API surface для ручной проверки. Особенно важно для CORS preflight и request-id (тесты не покрывают браузерный реальный preflight).
- **Структура `manual_qa/1_4_auth_middleware/` копирует `manual_qa/1_3_auth/`** — единый стиль документации QA в проекте.
- **Tests НЕ покрывают**: реальный browser preflight (CORS блок на стороне клиента), high-load benchmark middleware-stack, panic в slog (см. «Намеренно НЕ покрытые сценарии» в `../04-testing.md`).

## Verification

- [ ] `go test ./... -run TestArchitecture` проходит — 6 тестов (4 существующих + 2 новых).
- [ ] Phase-specific check: `TestArchitecture_PkgHttpxImports` — обнаруживает добавление любого `internal/...` импорта в `pkg/httpx/...`. Симулировать падение через временный импорт можно вручную.
- [ ] Phase-specific check: `TestArchitecture_AuthMiddlewareImports` — обнаруживает добавление любого `internal/auth/repository/...` импорта в middleware. Симулировать падение можно вручную.
- [ ] `manual_qa/1_4_auth_middleware/` создан, 5 файлов (`README.md`, `00_smoke.http`, `01_cors_preflight.http`, `02_request_id.http`, `03_require_auth.http`).
- [ ] Manual QA запускается на стенде:
  - `make dc-up && make migrate-up && make run`.
  - Прогон `00_smoke.http` — все шаги дают ожидаемые ответы.
  - Прогон `01_cors_preflight.http` — preflight allowed/denied работают, заголовки выставлены.
  - Прогон `02_request_id.http` — все 5 кейсов request-id ведут себя как описано.
  - Прогон `03_require_auth.http` — все 6 кейсов 401-сценариев + happy path работают.
- [ ] `git diff --name-only` ограничен `arch_test.go` + `manual_qa/1_4_auth_middleware/**`.

## Финальная проверка фичи (cumulative success criteria)

После этой фазы — проверить, что **все** пункты Success Criteria из `plan/README.md` выполнены:

- [ ] Все 14 пунктов «Критериев приёмки» из `../README.md:33-46`.
- [ ] `go build ./...` — exit 0.
- [ ] `make lint` — exit 0.
- [ ] `make test` — все ~65 тестов зелёные.
- [ ] `go test ./... -race -count=1` — race-detector чистый.
- [ ] Manual QA — все шаги пройдены.
- [ ] `git status` чистый, изменения только в перечисленных в `plan/README.md` путях.
- [ ] `go.mod` без новых прямых зависимостей.
- [ ] Production-код не содержит `panic(...)`, кроме `pkg/httpx/middleware.Recover` re-panic для `http.ErrAbortHandler`.
