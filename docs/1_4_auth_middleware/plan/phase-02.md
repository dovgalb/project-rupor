---
phase: 2
name: pkg/httpx/middleware (cross-cutting)
layer: infra
depends_on: [phase-01]
plan: ./README.md
---

# Phase 2: `pkg/httpx/middleware/` — cross-cutting middleware

## Цель

Реализовать четыре общих HTTP-middleware: `RequestID`, `Recover`, `Logger`, `CORS`. Все они независимы от auth-домена и переиспользуются в фазе 2/3 другими модулями. Без новых зависимостей в `go.mod`.

## Контекст

Фаза 1 произвела:
- `pkg/httpx.WriteJSONError(w, status, code, message)` — единый JSON-envelope для ошибок.
- `pkg/httpx.WithRequestID(ctx, rid)` / `RequestIDFromContext(ctx)` — context helpers.
- `pkg/httpx/middleware.responseWriter` (приватный) — wrapper для отслеживания status и headerWritten.

Все четыре middleware ниже — функции-фабрики `func(...) func(http.Handler) http.Handler`, совместимые с `chi.Router.Use(...)`.

Цепочка использования (зафиксирована в `../01-architecture.md:225-260`):
```
RequestID → Recover → Logger → CORS → [chi route]
```

Подключение в `cmd/server/main.go` — фаза 5.

## Файлы для создания

### `pkg/httpx/middleware/requestid.go`

**Назначение:** обеспечивает `request_id` в каждом запросе. Принимает клиентский `X-Request-ID` при валидности, иначе генерирует UUID v4.

**Детали реализации:**
- Сигнатура: `func RequestID(uuidGen func() string) func(http.Handler) http.Handler`.
- Внутри `next.ServeHTTP`:
  1. `incoming := r.Header.Get("X-Request-ID")`.
  2. `rid := normalize(incoming, uuidGen)`. Логика `normalize`:
     - Если `incoming` непустая И `len(incoming) <= 128` И каждый байт — ASCII печатный (`0x20..0x7E` исключая управляющие) — используем `incoming`.
     - Иначе — `uuidGen()`.
  3. `ctx := httpx.WithRequestID(r.Context(), rid)`.
  4. `w.Header().Set("X-Request-ID", rid)` — выставляем ДО `next.ServeHTTP`, чтобы клиент видел, даже если handler паникует и Recover пишет 500.
  5. `next.ServeHTTP(w, r.WithContext(ctx))`.
- Helper `func isValidRequestID(s string) bool` — приватный, проверяет ASCII-printable + длину.
- `uuidGen` — параметр функции; в проде — `func() string { return uuid.New().String() }`. В тестах — детерминированный.

### `pkg/httpx/middleware/recover.go`

**Назначение:** ловит panic в нижестоящих middleware/хендлерах, логирует с request_id и stack trace, отвечает 500 INTERNAL.

**Детали реализации (`../01-architecture.md:96-108`, `../03-decisions.md`, ADR-008):**
- Сигнатура: `func Recover(logger *slog.Logger) func(http.Handler) http.Handler`.
- Внутри `next.ServeHTTP`:
  1. `sw := wrap(w)` — обернуть для отслеживания headerWritten.
  2. `defer func() { ... }()` блок:
     - `v := recover()`.
     - Если `v == nil` — выход.
     - Если `v == http.ErrAbortHandler` (через `errors.Is`, но `recover()` возвращает `interface{}`, поэтому `if e, ok := v.(error); ok && errors.Is(e, http.ErrAbortHandler) { panic(v) }`) — re-panic.
     - Извлечь `requestID, _ := httpx.RequestIDFromContext(r.Context())`.
     - `logger.Error("panic recovered", slog.Any("err", v), slog.String("stack", string(debug.Stack())), slog.String("request_id", requestID), slog.String("method", r.Method), slog.String("path", r.URL.Path))`.
     - Если `!sw.headerWritten` — `httpx.WriteJSONError(sw, http.StatusInternalServerError, "INTERNAL", "internal")`.
     - Иначе — `logger.Warn("panic after WriteHeader, response truncated", slog.String("request_id", requestID))`.
  3. `next.ServeHTTP(sw, r)`.
- Импорты: `errors`, `log/slog`, `net/http`, `runtime/debug`, `pkg/httpx`.

### `pkg/httpx/middleware/logger.go`

**Назначение:** access-log через slog с метаданными запроса и опциональным hook'ом для дополнительных attrs (например, `user_id`).

**Детали реализации (`../01-architecture.md:113-130`, `../03-decisions.md`, ADR-012):**
- Сигнатура:
  ```go
  func Logger(
      logger *slog.Logger,
      hook func(ctx context.Context) []slog.Attr,
  ) func(http.Handler) http.Handler
  ```
- Если `hook == nil` — заменяем на `func(context.Context) []slog.Attr { return nil }` для упрощения логики.
- Внутри `next.ServeHTTP`:
  1. `start := time.Now()`.
  2. `sw := wrap(w)`.
  3. `next.ServeHTTP(sw, r)` — выполняем основной handler.
  4. После — собираем attrs:
     ```go
     requestID, _ := httpx.RequestIDFromContext(r.Context())
     attrs := []slog.Attr{
         slog.String("method", r.Method),
         slog.String("path", r.URL.Path),
         slog.Int("status", sw.status),
         slog.Duration("duration", time.Since(start)),
         slog.String("request_id", requestID),
         slog.String("remote_ip", r.RemoteAddr),
         slog.String("user_agent", r.UserAgent()),
     }
     if extra := hook(r.Context()); len(extra) > 0 {
         attrs = append(attrs, extra...)
     }
     logger.LogAttrs(r.Context(), slog.LevelInfo, "http", attrs...)
     ```
- **Не логируется**: `Authorization` header, тело request/response, любые user-controlled headers кроме whitelisted (`User-Agent`, `Remote-IP`).
- Если `sw.status` всё ещё `0` после next.ServeHTTP (handler не вызвал WriteHeader явно) — Go стандарт `200 OK`, статус устанавливается обёрткой при первом `Write`. В `wrap.WriteHeader` дефолт `status = http.StatusOK` гарантирует корректное значение.

### `pkg/httpx/middleware/cors.go`

**Назначение:** обработка CORS preflight `OPTIONS` и cross-origin запросов с whitelisted origins.

**Детали реализации (`../01-architecture.md:132-148`, `../03-decisions.md`, ADR-004):**
- Сигнатура: `func CORS(allowedOrigins []string, allowCredentials bool) func(http.Handler) http.Handler`.
- В замыкании сохранить `set := makeOriginSet(allowedOrigins)` (локальная map[string]struct{} для O(1) lookup).
- Внутри `next.ServeHTTP`:
  1. `origin := r.Header.Get("Origin")`.
  2. Если `origin == ""` — `next.ServeHTTP` без CORS-заголовков (это не cross-origin).
  3. Если `_, ok := set[origin]; !ok` — `next.ServeHTTP` без CORS-заголовков (origin не в whitelist).
  4. Если match:
     - `w.Header().Set("Access-Control-Allow-Origin", origin)`.
     - `w.Header().Add("Vary", "Origin")` (через `Add`, не `Set`, чтобы не затереть другие Vary-заголовки от downstream).
     - Если `allowCredentials` — `w.Header().Set("Access-Control-Allow-Credentials", "true")`.
  5. Если `r.Method == http.MethodOptions`:
     - Это preflight. Дополнительно выставить:
       - `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`.
       - `Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID`.
       - `Access-Control-Max-Age: 600`.
     - `w.WriteHeader(http.StatusNoContent)` — 204.
     - **НЕ** вызывать `next.ServeHTTP` — preflight завершается тут.
     - `return`.
  6. Иначе (не OPTIONS) — `next.ServeHTTP(w, r)`.
- Wildcard `*` НЕ поддерживается (`../03-decisions.md`, ADR-004). Если `*` присутствует в whitelist — обрабатывается как обычный origin (то есть запросы с `Origin: *` совпадут, что бессмысленно). Тест `TestCORS_WildcardNotSupported` проверяет.
- Helper `makeOriginSet([]string) map[string]struct{}` — приватный.

### `pkg/httpx/middleware/requestid_test.go`

**Назначение:** verify RequestID поведение.

**Детали (5 тестов из `../04-testing.md:108-116`):**
- `TestRequestID_GeneratesWhenAbsent`: запрос без `X-Request-ID` → middleware вызывает `uuidGen` (счётчик в фейке), выставляет в response header. Downstream-handler через `httpx.RequestIDFromContext` видит то же значение.
- `TestRequestID_AcceptsValidIncoming`: `X-Request-ID: my-correlation-1` → middleware использует это значение, `uuidGen` НЕ вызывается. Response содержит `my-correlation-1`.
- `TestRequestID_RejectsInvalidIncoming` (table-driven):
  - кейсы: длина 129, `\r\n` в значении, `\x00`, `τ` (non-ASCII), `"   "` (пробелы), пустая.
  - Для каждого — middleware игнорирует и генерирует свой. `uuidGen` вызывается.
- `TestRequestID_SetsResponseHeader`: response содержит `X-Request-ID` ДО того, как handler начнёт писать body (важно для panic-сценария — Recover ловит позже).
- `TestRequestID_ProvidesContextValue`: handler-stub читает `RequestIDFromContext(r.Context())` → значение совпадает с response header.

Используется `httptest.NewRecorder()` + `chi.NewRouter()` mini-router с одним handler-ом.

### `pkg/httpx/middleware/recover_test.go`

**Детали (4 теста из `../04-testing.md:118-127`):**
- `TestRecover_PanicReturnsInternalEnvelope`:
  - Router: `Use(Recover(logger)) → handler{ panic("nil deref") }`.
  - Запрос → 500, body `{"error":{"code":"INTERNAL","message":"internal"}}`, `Content-Type: application/json`.
- `TestRecover_PassesAbortHandler`:
  - Handler: `panic(http.ErrAbortHandler)`.
  - Используется `defer func() { recover() }()` в самом тесте, чтобы поймать re-panic.
  - Assert: статус не 500 (значит middleware пропустил, не маскировал).
- `TestRecover_PanicAfterWriteHeader_LogsAndContinues`:
  - Handler: `w.WriteHeader(200); panic("oops")`.
  - Assert: `rec.Code == 200` (не перезаписан в 500). Лог-buffer содержит `"panic after WriteHeader"`.
- `TestRecover_LogsStackAndRequestID`:
  - Lifecycle: `Use(RequestID(...)) → Use(Recover(logger))`. Внутри handler — panic.
  - Logger создаётся через `slog.New(slog.NewJSONHandler(buf, ...))` поверх `bytes.Buffer`.
  - Assert: лог содержит `"panic recovered"`, `"err"`, `"stack"`, `"request_id"`, `"method"`, `"path"`.

### `pkg/httpx/middleware/logger_test.go`

**Детали (6 тестов из `../04-testing.md:129-138`):**
- `TestLogger_LogsAccessFields`:
  - Запрос → лог-buffer содержит JSON-event с полями `method`, `path`, `status`, `duration` (>0), `request_id`. Уровень — INFO.
- `TestLogger_CapturesResponseStatus`:
  - Handler `w.WriteHeader(404)` → лог содержит `"status":404`.
  - Default (без явного WriteHeader) → `"status":200`.
- `TestLogger_IncludesRequestID`:
  - Запуск через `Use(RequestID(...)) → Use(Logger(...))`.
  - Лог содержит request_id, совпадающий с response-header'ом.
  - Без RequestID middleware: лог содержит пустой `"request_id":""`.
- `TestLogger_HookAddsAttrs`:
  - Hook возвращает `[]slog.Attr{slog.String("user_id", "abc-123")}`.
  - Лог содержит `"user_id":"abc-123"`.
  - При hook возвращающем nil — поле отсутствует в логе.
- `TestLogger_DoesNotLogAuthorization`:
  - Запрос с `Authorization: Bearer secret-token-xxx`.
  - Лог-buffer как строка НЕ содержит `"secret-token-xxx"` и НЕ содержит ключ `"authorization"` (case-insensitive проверка).
- `TestLogger_DoesNotLogBody`:
  - POST с body `{"password":"hunter2"}`.
  - Лог-buffer НЕ содержит `"hunter2"`.

### `pkg/httpx/middleware/cors_test.go`

**Детали (7 тестов из `../04-testing.md:140-150`):**
- `TestCORS_PreflightAllowedOrigin`:
  - OPTIONS, `Origin: http://localhost:5173` (в whitelist).
  - Assert: 204, `Access-Control-Allow-Origin: http://localhost:5173`, `Vary: Origin`. Body пустой.
- `TestCORS_PreflightDeniedOrigin`:
  - OPTIONS, `Origin: http://evil.com`.
  - Mini-router: `Use(CORS(...)) → handler{ w.WriteHeader(405) }` (имитация chi default 405).
  - Assert: 405, БЕЗ `Access-Control-Allow-Origin`.
- `TestCORS_NonOptionsAllowedOrigin`:
  - POST, `Origin: http://localhost:5173`.
  - Assert: handler вызван (next), response содержит `Access-Control-Allow-Origin: http://localhost:5173`, `Vary: Origin`.
- `TestCORS_NonOptionsDeniedOrigin`:
  - POST, `Origin: http://evil.com`.
  - Assert: handler вызван, response БЕЗ `Access-Control-Allow-Origin`.
- `TestCORS_OptionsWithoutOrigin`:
  - OPTIONS без `Origin`.
  - Mini-router: `Use(CORS(...)) → handler{ w.WriteHeader(405) }`.
  - Assert: handler вызван (CORS пропускает дальше), 405.
- `TestCORS_PreflightHeaders`:
  - OPTIONS match → response содержит:
    - `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`.
    - `Access-Control-Allow-Headers: Authorization, Content-Type, X-Request-ID`.
    - `Access-Control-Max-Age: 600`.
    - **НЕТ** `Access-Control-Allow-Credentials` (т. к. `allowCredentials = false`).
- `TestCORS_WildcardNotSupported`:
  - `CORS([]string{"*"}, false)`. Запрос с `Origin: http://anywhere.com`.
  - Assert: response БЕЗ `Access-Control-Allow-Origin` (wildcard не интерпретируется как «любой»; `"*"` сравнивается как обычная строка с `"http://anywhere.com"` — не match).

## Файлы для модификации

Никаких. Это «зелёное поле» поверх фазы 1.

## Ключевые решения

- **Все middleware принимают параметры через сигнатуру функции, не через интерфейс.** Никаких `LoggerConfig`/`CORSOptions` структур (`../03-decisions.md`, ADR-001 — без зависимостей; ADR-009 — отказ от chi/middleware).
- **`Logger` принимает `hook` функцию для расширений.** Это позволяет в фазе 5 (`cmd/server/main.go`) добавить `user_id` через `authmw.UserIDFromContext` без импорта auth в `pkg/httpx/`.
- **`RequestID` валидирует клиентский header строго** (`../03-decisions.md`, ADR-005). ASCII-печатный + длина ≤128 — защита от header injection в логи.
- **`Recover` пере-paniкует на `http.ErrAbortHandler`** (`../03-decisions.md`, ADR-008). Совместимость с net/http stdlib.
- **`CORS` не поддерживает wildcard `*`** (`../03-decisions.md`, ADR-004). Явный whitelist.

## Verification

- [ ] `go build ./pkg/httpx/middleware/...` проходит.
- [ ] `go test ./pkg/httpx/middleware/...` проходит — 22 теста.
- [ ] `gofmt -l pkg/httpx/middleware/` — пустой вывод.
- [ ] `go vet ./pkg/httpx/middleware/...` — exit 0.
- [ ] Phase-specific check: middleware-файлы импортируют только stdlib + `github.com/dovgalb/project-rupor/pkg/httpx` (для `WriteJSONError`, `RequestIDFromContext`).
- [ ] Phase-specific check: `Recover` — единственное место в проекте с вызовом `recover()`.
- [ ] Phase-specific check: ни один middleware не импортирует `internal/...`.
- [ ] `go test ./pkg/httpx/middleware/... -race -count=1` — race-detector чистый.
- [ ] `git diff --name-only` ограничен `pkg/httpx/middleware/**`.
