---
parent: ./README.md
view: quality
---

# 04 — Testing (Quality View)

Все тесты используют стандартную библиотеку `testing` (`prompts/Tests Style.txt:24-26`), `t.Parallel()` везде, без testify/gomock. Фейки пишутся руками рядом с тестом (`prompts/Tests Style.txt:75-100`).

Black-box по умолчанию: `package <name>_test` (например, `package middleware_test`).

## Coverage Mapping

Каждый код ошибки и каждый сценарий из `02-behavior.md` и `08-api-contract.md` имеет покрытие тестом.

| Use Case / Компонент | Условие | Code | Тест |
|----------------------|---------|------|------|
| `httpx.WriteJSONError` | базовый round-trip | — | `TestWriteJSONError_BasicEnvelope` |
| `httpx.WriteJSONError` | заголовки и тело | — | `TestWriteJSONError_SetsContentTypeAndStatus` |
| `httpx.WithRequestID` / `httpx.RequestIDFromContext` | put + get | — | `TestRequestIDContext_RoundTrip` |
| `httpx.RequestIDFromContext` | пустой ctx | — | `TestRequestIDContext_AbsentReturnsEmpty` |
| `httpxmw.RequestID` | заголовок отсутствует — генерируем | — | `TestRequestID_GeneratesWhenAbsent` |
| `httpxmw.RequestID` | валидный заголовок принимается | — | `TestRequestID_AcceptsValidIncoming` |
| `httpxmw.RequestID` | невалидный заголовок (CRLF / >128 / non-ASCII) — игнорируется | — | `TestRequestID_RejectsInvalidIncoming` (table-driven) |
| `httpxmw.RequestID` | `X-Request-ID` выставлен в response | — | `TestRequestID_SetsResponseHeader` |
| `httpxmw.RequestID` | request_id виден в ctx у downstream | — | `TestRequestID_ProvidesContextValue` |
| `httpxmw.Recover` | panic → 500 INTERNAL | — | `TestRecover_PanicReturnsInternalEnvelope` |
| `httpxmw.Recover` | `http.ErrAbortHandler` пере-paniкует | — | `TestRecover_PassesAbortHandler` |
| `httpxmw.Recover` | panic после `WriteHeader` — лог, без второго WriteHeader | — | `TestRecover_PanicAfterWriteHeader_LogsAndContinues` |
| `httpxmw.Recover` | в slog попадает stack + request_id | — | `TestRecover_LogsStackAndRequestID` |
| `httpxmw.Logger` | пишет access-log с полями | — | `TestLogger_LogsAccessFields` |
| `httpxmw.Logger` | захватывает status через wrapped writer | — | `TestLogger_CapturesResponseStatus` |
| `httpxmw.Logger` | request_id в логах при наличии | — | `TestLogger_IncludesRequestID` |
| `httpxmw.Logger` | hook добавляет user_id (когда RequireAuth прошёл) | — | `TestLogger_HookAddsAttrs` |
| `httpxmw.Logger` | НЕ логирует Authorization header | — | `TestLogger_DoesNotLogAuthorization` |
| `httpxmw.Logger` | НЕ логирует тело запроса/ответа | — | `TestLogger_DoesNotLogBody` |
| `httpxmw.CORS` | preflight match → 204 + headers | — | `TestCORS_PreflightAllowedOrigin` |
| `httpxmw.CORS` | preflight no match → next (chi 405) | — | `TestCORS_PreflightDeniedOrigin` |
| `httpxmw.CORS` | non-OPTIONS match → headers + next | — | `TestCORS_NonOptionsAllowedOrigin` |
| `httpxmw.CORS` | non-OPTIONS no match → next без Allow-Origin | — | `TestCORS_NonOptionsDeniedOrigin` |
| `httpxmw.CORS` | OPTIONS без Origin → next | — | `TestCORS_OptionsWithoutOrigin` |
| `httpxmw.CORS` | заголовки preflight: Methods/Headers/Max-Age/Vary | — | `TestCORS_PreflightHeaders` |
| `httpxmw.CORS` | wildcard origin не поддерживается | — | `TestCORS_WildcardNotSupported` |
| `authmw.WithUserID` / `authmw.UserIDFromContext` | put + get | — | `TestUserIDContext_RoundTrip` |
| `authmw.UserIDFromContext` | пустой ctx | — | `TestUserIDContext_AbsentReturnsFalse` |
| `authmw.RequireAuth` | happy path: ctx содержит UserID, next вызван | — | `TestRequireAuth_Success` |
| `authmw.RequireAuth` | нет Authorization | AUTH-010 | `TestRequireAuth_NoHeader` |
| `authmw.RequireAuth` | Authorization не Bearer (Basic) | AUTH-010 | `TestRequireAuth_BadScheme` |
| `authmw.RequireAuth` | Bearer без токена | AUTH-010 | `TestRequireAuth_EmptyToken` |
| `authmw.RequireAuth` | подпись неправильная | AUTH-010 | `TestRequireAuth_InvalidSignature` |
| `authmw.RequireAuth` | токен истёк | AUTH-011 | `TestRequireAuth_TokenExpired` |
| `authmw.RequireAuth` | bad sub claim | AUTH-010 | `TestRequireAuth_BadSubClaim` |
| `authmw.RequireAuth` | next НЕ вызван при ошибке | — | `TestRequireAuth_NextNotCalledOnError` |
| `MeHandler` (упрощён) | UserID из ctx → use case | — | `TestMeHandler_ReadsUserIDFromContext` |
| `MeHandler` (упрощён) | UserID отсутствует (защитная ветка) | AUTH-010 | `TestMeHandler_FallbackWhenNoContext` |
| `MeHandler` (упрощён) | usecase ErrUserNotFound | AUTH-010 | `TestMeHandler_UseCaseUserNotFound` |
| 1.3 интеграционные тесты `MeHandler` | `_NoAuth`, `_BadScheme`, `_InvalidSignature`, `_TokenExpired`, `_Success` | AUTH-010/011 | сохраняются без изменений (см. ниже «Регрессия 1.3») |
| `config.Load` | `CORS_ALLOWED_ORIGINS` отсутствует → default | — | `TestConfig_Load_DefaultCORS` |
| `config.Load` | `CORS_ALLOWED_ORIGINS` валидный CSV | — | `TestConfig_Load_CSVOrigins` |
| `config.Load` | `CORS_ALLOWED_ORIGINS` пустой / только пробелы | CONFIG-006 | `TestConfig_Load_EmptyCORS` |
| `config.Load` | невалидный URL в одном из origin | CONFIG-006 | `TestConfig_Load_InvalidOrigin` (table-driven) |
| `arch_test.go` | `pkg/httpx/...` импортирует только stdlib + uuid | — | `TestArchitecture_PkgHttpxImports` |
| `arch_test.go` | `internal/auth/transport/http/middleware/` импортирует {usecase,domain,httpx,uuid,stdlib} | — | `TestArchitecture_AuthMiddlewareImports` |

## Модуль `pkg/httpx/` — Test Cases

Файлы тестов: `pkg/httpx/jsonerror_test.go`, `pkg/httpx/contextkeys_test.go`. Пакет: `httpx_test`.

### `WriteJSONError` (2 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestWriteJSONError_BasicEnvelope` | `WriteJSONError(rec, 401, "AUTH-010", "access token invalid")` → status 401, body содержит `"code":"AUTH-010"` и `"message":"access token invalid"`. |
| `TestWriteJSONError_SetsContentTypeAndStatus` | `Content-Type: application/json; charset=utf-8`, тело завершается `\n`. Двойной вызов не падает (второй раз WriteHeader игнорируется net/http). |

### `RequestID` context helpers (2 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestRequestIDContext_RoundTrip` | `WithRequestID(ctx, "rid-1")` → `RequestIDFromContext(ctx)` возвращает `("rid-1", true)`. |
| `TestRequestIDContext_AbsentReturnsEmpty` | На пустом ctx — `("", false)`. |

## Модуль `pkg/httpx/middleware/` — Test Cases

Файлы тестов: `pkg/httpx/middleware/{recover,logger,cors,requestid}_test.go`. Пакет: `middleware_test`.

Тесты используют минимальный mini-router: `chi.NewRouter()` с одним handler-stub'ом и нужной middleware. Запросы — через `httptest.NewRecorder()` (не реальный TCP), потому что cross-cutting middleware не зависят от listener-а.

### `RequestID` (5 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestRequestID_GeneratesWhenAbsent` | Без `X-Request-ID` в request, middleware вызывает фабрику UUID, выставляет в response header и ctx. |
| `TestRequestID_AcceptsValidIncoming` | `X-Request-ID: my-correlation-1` → middleware использует это значение, не вызывая фабрику. |
| `TestRequestID_RejectsInvalidIncoming` | Table-driven: `>128 chars`, `\r\n` (CRLF), `non-ASCII` (`τ`), пустая после trim → middleware игнорирует, генерирует свой. |
| `TestRequestID_SetsResponseHeader` | Response содержит `X-Request-ID: <value>`. |
| `TestRequestID_ProvidesContextValue` | Downstream handler через `httpx.RequestIDFromContext(r.Context())` получает то же значение. |

### `Recover` (4 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestRecover_PanicReturnsInternalEnvelope` | Handler panic'ит → 500, body `{"error":{"code":"INTERNAL","message":"internal"}}`, `Content-Type: application/json`. |
| `TestRecover_PassesAbortHandler` | Handler `panic(http.ErrAbortHandler)` → recover пропускает дальше (re-panic), test ловит panic в горутине через `recover()`. |
| `TestRecover_PanicAfterWriteHeader_LogsAndContinues` | Handler сделал `w.WriteHeader(200)` потом panic → status в ответе остался 200, тело может быть усечено. Лог содержит `panic after WriteHeader`. |
| `TestRecover_LogsStackAndRequestID` | После panic в logger попало `request_id`, `method`, `path`, `err`, `stack` (через `bytes.Contains`). Используется test-handler `slog.NewJSONHandler` поверх `bytes.Buffer`. |

### `Logger` (6 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestLogger_LogsAccessFields` | После запроса в slog-buffer попадают: `method`, `path`, `status`, `duration` (>0), `request_id`. Уровень — INFO. |
| `TestLogger_CapturesResponseStatus` | Handler вызывает `w.WriteHeader(404)` → лог содержит `status=404`. Дефолт (без явного WriteHeader) → `status=200`. |
| `TestLogger_IncludesRequestID` | RequestID middleware установил `rid-X` в ctx → Logger вытащил его и записал в access-log. Если ctx без request_id — поле отсутствует. |
| `TestLogger_HookAddsAttrs` | `Logger(logger, hook)` где `hook(ctx) → []slog.Attr{slog.String("user_id","abc")}` — в логе появляется `user_id=abc`. На public-запросе hook возвращает nil, поля нет. |
| `TestLogger_DoesNotLogAuthorization` | Запрос с `Authorization: Bearer <jwt>` → в slog-buffer **не содержится** строки токена и не содержится слова `Authorization`. |
| `TestLogger_DoesNotLogBody` | POST с body `password=secret` → в логе нет ни `password`, ни `secret`. |

### `CORS` (7 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestCORS_PreflightAllowedOrigin` | OPTIONS, `Origin: http://localhost:5173` (в whitelist) → 204, `Access-Control-Allow-Origin: http://localhost:5173`, `Vary: Origin`. Body пустой. |
| `TestCORS_PreflightDeniedOrigin` | OPTIONS, `Origin: http://evil.com` (не в whitelist) → middleware вызывает next, который возвращает 405 (chi default). Без `Access-Control-Allow-Origin`. |
| `TestCORS_NonOptionsAllowedOrigin` | POST, `Origin: http://localhost:5173` → ответ от handler + `Access-Control-Allow-Origin: http://localhost:5173`, `Vary: Origin`. |
| `TestCORS_NonOptionsDeniedOrigin` | POST, `Origin: http://evil.com` → ответ от handler без `Access-Control-Allow-Origin`. |
| `TestCORS_OptionsWithoutOrigin` | OPTIONS без `Origin` → middleware вызывает next (preflight без Origin — не CORS). |
| `TestCORS_PreflightHeaders` | OPTIONS match → response содержит: `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`; `Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID`; `Access-Control-Max-Age: 600`; `Access-Control-Allow-Credentials` НЕ установлен (т. к. `allowCredentials = false`). |
| `TestCORS_WildcardNotSupported` | Конструктор `CORS([]string{"*"}, false)` либо паникует на старте, либо считает `*` обычным origin (не wildcard). Тест проверяет, что запрос с `Origin: http://anywhere.com` НЕ получает `Access-Control-Allow-Origin`. |

## Модуль `internal/auth/transport/http/middleware/` — Test Cases

Файлы: `auth_test.go`, `contextkeys_test.go`. Пакет: `middleware_test`.

Тесты используют:
- Реальный `*jwt.TokenIssuer` (из `internal/auth/repository/jwt`) с тестовым секретом и фиксированным `accessTTL`. Это даёт настоящие подписи (`prompts/Tests Style.txt:147-152` — реальные адаптеры в HTTP-тестах).
- Фейковый `Clock` (`fixedClock{now}`).
- Mini-handler `next` через `http.HandlerFunc`, который пишет `200 OK` и проверяет ctx через `UserIDFromContext`.

### `WithUserID` / `UserIDFromContext` (2 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestUserIDContext_RoundTrip` | `WithUserID(ctx, uid)` → `UserIDFromContext(ctx)` возвращает `(uid, true)`. |
| `TestUserIDContext_AbsentReturnsFalse` | Пустой ctx → `(zero, false)`. |

### `RequireAuth` (8 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestRequireAuth_Success` | Валидный `Bearer` token → next вызван, ctx содержит `domain.UserID == issuedUID`. Response 200. |
| `TestRequireAuth_NoHeader` | Без `Authorization` → 401, body `AUTH-010`, next НЕ вызван. |
| `TestRequireAuth_BadScheme` | `Authorization: Basic xxx` → 401 + AUTH-010. |
| `TestRequireAuth_EmptyToken` | `Authorization: Bearer ` (с пробелом, без токена) → 401 + AUTH-010. |
| `TestRequireAuth_InvalidSignature` | Токен подписан другим секретом → 401 + AUTH-010 (через `domain.ErrAccessTokenInvalid`). |
| `TestRequireAuth_TokenExpired` | Токен с `exp < now` (вручную собранный с прошлым ExpiresAt) → 401 + AUTH-011. |
| `TestRequireAuth_BadSubClaim` | Токен валидный по подписи, но `sub = "not-uuid"` → 401 + AUTH-010. |
| `TestRequireAuth_NextNotCalledOnError` | На любом из ошибочных сценариев — флаг `nextCalled` остаётся `false`. Гарантирует, что middleware блокирует выполнение. |

### Stubs / Mocks

```go
// internal/auth/transport/http/middleware/auth_test.go

type fixedClock struct{ now time.Time }
func (c fixedClock) Now() time.Time { return c.now }

func newRealIssuer(t *testing.T) usecase.TokenIssuer {
    t.Helper()
    return jwt.NewTokenIssuer([]byte("test-secret-1234567890"), 15*time.Minute)
}

func newRequest(t *testing.T, authHeader string) *http.Request {
    t.Helper()
    req := httptest.NewRequest(http.MethodGet, "/protected", nil)
    if authHeader != "" {
        req.Header.Set("Authorization", authHeader)
    }
    return req
}
```

Никаких mocks внешнего API — реальный jwt-issuer и fixedClock. Это интеграция middleware + jwt, она быстрее любых моков.

## Модуль `internal/auth/transport/http/` — Test Cases

### Регрессия 1.3 — что сохраняется без изменений

19 тестов из `docs/1_3_auth_domen/04-testing.md` запускаются с обновлённым `setup_test.go`:

| Тест | Файл | Изменения |
|------|------|-----------|
| `TestRegisterHandler_*` (5 тестов) | `register_handler_test.go` | без изменений |
| `TestLoginHandler_*` (4 теста) | `login_handler_test.go` | без изменений |
| `TestRefreshHandler_*` (5 тестов) | `refresh_handler_test.go` | без изменений |
| `TestMeHandler_Success` | `me_handler_test.go` | без изменений; теперь request проходит через `RequireAuth` |
| `TestMeHandler_NoAuth` | `me_handler_test.go` | без изменений; 401 теперь от `RequireAuth`, не от `MeHandler` |
| `TestMeHandler_BadScheme` | `me_handler_test.go` | без изменений |
| `TestMeHandler_InvalidSignature` | `me_handler_test.go` | без изменений |
| `TestMeHandler_TokenExpired` | `me_handler_test.go` | без изменений |

`setup_test.go` обновляется минимально: один и тот же `httpauth.RegisterRoutes(r, deps)` теперь подключает `RequireAuth` на protected-группу. Тестам не нужны новые helper'ы.

### Новые тесты (`MeHandler` — упрощённый)

Файл: `me_handler_test.go` (расширение).

| Тест | Что проверяет |
|------|---------------|
| `TestMeHandler_ReadsUserIDFromContext` | Ctx содержит `domain.UserID` (через `WithUserID`) → handler передаёт `uid.String()` в use case. Без вызова `Authorization` parsing. |
| `TestMeHandler_FallbackWhenNoContext` | Hand-crafted request минуя `RequireAuth` (помещение `MeHandler` в public-группу) → handler возвращает 401 + AUTH-010 через защитную ветку. |
| `TestMeHandler_UseCaseUserNotFound` | UserID в ctx есть, но `repo.FindByID` вернул `ErrUserNotFound` (например, user удалён) → 401 + AUTH-010 через `mapError`. |

## Модуль `config/` — Test Cases (расширение)

Новые тесты к существующим (`config/config_test.go`):

| Тест | Что проверяет |
|------|---------------|
| `TestConfig_Load_DefaultCORS` | `CORS_ALLOWED_ORIGINS` отсутствует → `cfg.CORSAllowedOrigins() == []string{"http://localhost:5173"}`. |
| `TestConfig_Load_CSVOrigins` | `CORS_ALLOWED_ORIGINS=http://a.com,https://b.com:8080` → `[]string{"http://a.com", "https://b.com:8080"}`. Trim пробелов в каждом. |
| `TestConfig_Load_EmptyCORS` | Table-driven: `""`, `"   "`, `","` → `CONFIG-006`/`CORS_ALLOWED_ORIGINS`/`required`. |
| `TestConfig_Load_InvalidOrigin` | Table-driven: `not-a-url`, `http://`, `https//missing-colon.com`, `http://x.com/path`, `ftp://x.com`, `http://x.com,not-url` → `CONFIG-006` с указанием конкретного невалидного значения. |

## Модуль `cmd/server/` — обновление тестов

`cmd/server/health_test.go` сейчас тестирует только `healthHandler` (`cmd/server/health_test.go:11-37`). С appearance middleware-stack полезно добавить:

| Тест | Что проверяет |
|------|---------------|
| `TestHealthHandler_ThroughMiddlewareStack` (новый) | Запрос `GET /api/v1/health` через полный chi-router с middleware → 200, `X-Request-ID` в ответе, тело `{"status":"ok"}`. Smoke на корректную сборку всех middleware в `cmd/server/main.go`. |

Реализация: помощник `buildRouter(t)` в `cmd/server/main_test.go` (новый файл) собирает router тем же способом, что и `run`.

## Архитектурный smoke-тест (`arch_test.go`)

Существующие 4 теста (`arch_test.go:61-147`) сохраняются без изменений. Добавляется два:

| Тест | Проверка |
|------|----------|
| `TestArchitecture_PkgHttpxImports` | `pkg/httpx/`, `pkg/httpx/middleware/` — импорты только stdlib + `github.com/google/uuid`. Запрещены любые `internal/...`. |
| `TestArchitecture_AuthMiddlewareImports` | `internal/auth/transport/http/middleware/*.go` — импорты только: stdlib, `github.com/google/uuid`, `github.com/dovgalb/project-rupor/internal/auth/{usecase,domain}`, `github.com/dovgalb/project-rupor/pkg/httpx`. Запрещён любой `internal/auth/repository/...`. |

Реализация — расширение функций `collectImports` и whitelist-наборов в `arch_test.go` без новых зависимостей.

## Repo Model Round-Trip Tests

Не применимо — middleware не имеют сущностей в БД.

## Integration Tests (репозитории)

Не применимо — middleware не используют репозитории.

## Test Count Summary

| Модуль | UseCase / Logic | VO / Helpers | Repo | HTTP | Architecture | Total |
|--------|------|------|------|------|------|------|
| `pkg/httpx/` (jsonerror + contextkeys) | — | 4 | — | — | — | 4 |
| `pkg/httpx/middleware/` (Recover/Logger/CORS/RequestID) | 22 | — | — | — | — | 22 |
| `internal/auth/transport/http/middleware/` (RequireAuth + ctx) | 8 | 2 | — | — | — | 10 |
| `internal/auth/transport/http/` (новые `MeHandler`) | — | — | — | 3 | — | 3 |
| `internal/auth/transport/http/` (регрессия 1.3) | — | — | — | 19 | — | 19 |
| `config/` (расширение) | 4 | — | — | — | — | 4 |
| `cmd/server/` (smoke через middleware-stack) | — | — | — | 1 | — | 1 |
| Архитектурный smoke | — | — | — | — | 2 | 2 |
| **ИТОГО** | **34** | **6** | — | **23** | **2** | **65** |

(Из них 19 — регрессия 1.3 без правок в коде тестов; 46 — новые тесты этой фичи.)

## Стратегия запуска

- `make test` — запускает unit-тесты middleware + регрессию 1.3 + config + arch. **Default**.
- `go test ./pkg/httpx/... -count=1` — независимый запуск только cross-cutting middleware (без auth-домена).
- `go test ./internal/auth/transport/http/middleware/... -count=1` — auth-middleware с реальным jwt-issuer.
- `go test ./... -race -count=1` — race detector (особенно важен для middleware: ctx propagation, status capture).
- `go test ./arch_test.go -run TestArchitecture_ -count=1` — архитектурный smoke.

В CI (`.github/workflows/ci.yml`) — все вышеперечисленные команды запускаются как сейчас (`go test ./...`); никаких build-тегов для middleware-тестов не вводится.

## Намеренно НЕ покрытые тестами сценарии

Следующие сценарии описаны в `02-behavior.md`, но осознанно не покрыты автотестами в фазе 1.4:

1. **Двойная panic в `Recover` (panic внутри `slog.Error`)** — требует мок-логгер с panic-инъекцией; крайне маловероятно в продакшене (`slog` JSON-handler panic-free). Если случится — net/http прокидывает на уровень `http.Server.ErrorLog`. Не покрываем.
2. **Реальный browser preflight через WebKit/Chrome** — это E2E; для CORS достаточно проверки заголовков. Покрытие в `manual_qa/1_4_auth_middleware/` через `.http`-файлы.
3. **High-load benchmark middleware-stack** — отдельная задача (фаза «observability»). Сейчас закладываем структурное соответствие, не perf-цифры.
