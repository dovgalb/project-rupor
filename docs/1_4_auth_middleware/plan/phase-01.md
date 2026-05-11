---
phase: 1
name: pkg/httpx primitives
layer: infra
depends_on: none
plan: ./README.md
---

# Phase 1: `pkg/httpx/` — примитивы

## Цель

Заложить фундамент для всех middleware: единый JSON-error helper, типизированный context-key для request-id, общий response-writer wrapper. Новых middleware пока не добавляется.

## Контекст

`pkg/httpx/` — новый верхний пакет в репо. До этого момента в `pkg/` лежал только пустой каталог `pkg/websocket/.gitkeep`. Никаких зависимостей за пределами stdlib и `github.com/google/uuid` (уже есть в go.mod) пакет не получает.

Эти примитивы будут переиспользованы:
- `WriteJSONError` — фаза 2 (Recover), фаза 3 (RequireAuth), фаза 5 (`error_mapper.go:writeError`).
- `RequestID context helpers` — фаза 2 (Logger читает request_id), будущие домены.
- `responseWriter` — фаза 2 (Recover, Logger).

## Файлы для создания

### `pkg/httpx/jsonerror.go`

**Назначение:** единая сериализация envelope ошибок проекта.

**Детали реализации:**
- Сигнатура: `func WriteJSONError(w http.ResponseWriter, status int, code string, message string)`.
- Заголовок: `Content-Type: application/json; charset=utf-8`.
- `WriteHeader(status)`.
- Тело — `json.Encoder.Encode` структуры:
  ```go
  type errorBody struct {
      Code    string `json:"code"`
      Message string `json:"message"`
  }
  type errorEnvelope struct {
      Error errorBody `json:"error"`
  }
  ```
  Структуры — приватные (`errorBody`/`errorEnvelope`) или встроенные в функцию через анонимные struct literal — на выбор реализатора. Главное — JSON-форма `{"error":{"code":"...","message":"..."}}` совпадает с 1.3 (`docs/1_3_auth_domen/08-api-contract.md:18-26`).
- Ошибка `json.Encoder.Encode` проглатывается через `_ =` (ResponseWriter уже не примет другую). Это допустимо для финального write — соответствует поведению `internal/auth/transport/http/error_mapper.go:50` (1.3).
- `\n` в конце — следствие `json.Encoder.Encode`, не вручную.

### `pkg/httpx/contextkeys.go`

**Назначение:** типизированный ключ + хелперы для request-id в `context.Context` (`prompts/Go style.txt:51`).

**Детали реализации:**
- Приватный тип: `type requestIDKey struct{}`.
- Публичные функции:
  ```go
  func WithRequestID(ctx context.Context, requestID string) context.Context
  func RequestIDFromContext(ctx context.Context) (string, bool)
  ```
- `WithRequestID` — простой `context.WithValue(ctx, requestIDKey{}, requestID)`. Не валидирует значение (валидация — на уровне middleware-фабрики `RequestID`, фаза 2).
- `RequestIDFromContext` — `value, ok := ctx.Value(requestIDKey{}).(string)`. Возвращает `("", false)` при отсутствии.

### `pkg/httpx/middleware/responsewriter.go`

**Назначение:** общий wrapper, который Recover (фаза 2) и Logger (фаза 2) будут использовать для отслеживания статуса ответа и факта `WriteHeader`.

**Детали реализации:**
- Тип:
  ```go
  type responseWriter struct {
      http.ResponseWriter
      status        int
      headerWritten bool
  }
  ```
- Конструктор-обёртка `wrap(w http.ResponseWriter) *responseWriter`. Идемпотентна: если `w` уже `*responseWriter`, возвращает его как есть (`if rw, ok := w.(*responseWriter); ok { return rw }`). Это нужно, потому что Recover оборачивает первым, потом Logger — оба вызвать `wrap` независимо.
- `WriteHeader(code int)`:
  - Если `headerWritten` — игнорируем (повторный вызов недопустим в HTTP).
  - Иначе — `r.status = code; r.headerWritten = true; r.ResponseWriter.WriteHeader(code)`.
- `Write(b []byte) (int, error)`:
  - Если `!headerWritten` — `r.WriteHeader(http.StatusOK)`.
  - `return r.ResponseWriter.Write(b)`.
- Геттеры/публичные поля: НЕТ. Тип приватный (`type responseWriter`), но используется внутри пакета `middleware`. Recover и Logger обращаются к `sw.status` и `sw.headerWritten` напрямую — это допустимо в одном пакете.
- Реализация интерфейсов кроме `http.ResponseWriter` (`http.Flusher`, `http.Hijacker`) — пока **не нужна** (ни WS, ни SSE в фазе 1.4 нет). Если потом понадобится — отдельной правкой через type assertion в delegating-методах.

### `pkg/httpx/jsonerror_test.go`

**Назначение:** verify сериализации.

**Детали реализации (2 теста):**
- `TestWriteJSONError_BasicEnvelope` (`../04-testing.md:43`):
  - Создать `httptest.NewRecorder()`.
  - Вызвать `WriteJSONError(rec, 401, "AUTH-010", "access token invalid")`.
  - Assert: `rec.Code == 401`.
  - Assert: тело содержит `"code":"AUTH-010"` и `"message":"access token invalid"` (через `strings.Contains` или `json.Unmarshal` в локальный struct).
- `TestWriteJSONError_SetsContentTypeAndStatus` (`../04-testing.md:44`):
  - Assert: `rec.Header().Get("Content-Type") == "application/json; charset=utf-8"`.
  - Assert: тело завершается `\n`.
  - Двойной вызов на один recorder не должен приводить к panic (второй `WriteHeader` net/http логирует warning и игнорирует — это OK).

Пакет — `httpx_test` (черный ящик).

### `pkg/httpx/contextkeys_test.go`

**Назначение:** verify round-trip context helpers.

**Детали реализации (2 теста):**
- `TestRequestIDContext_RoundTrip` (`../04-testing.md:46`):
  - `ctx := context.Background()`.
  - `ctx = WithRequestID(ctx, "rid-1")`.
  - `got, ok := RequestIDFromContext(ctx)`.
  - Assert: `ok == true`, `got == "rid-1"`.
- `TestRequestIDContext_AbsentReturnsEmpty` (`../04-testing.md:47`):
  - `got, ok := RequestIDFromContext(context.Background())`.
  - Assert: `ok == false`, `got == ""`.

Пакет — `httpx_test` (черный ящик).

### `pkg/httpx/middleware/responsewriter_test.go`

**Назначение:** verify семантику обёртки.

**Детали реализации (2 теста):**
- `TestResponseWriter_CapturesStatus`:
  - `rec := httptest.NewRecorder()`.
  - `sw := wrap(rec)`.
  - `sw.WriteHeader(404)`.
  - Assert: `sw.status == 404`, `sw.headerWritten == true`, `rec.Code == 404`.
- `TestResponseWriter_DoubleWriteHeaderIsNoOp`:
  - `sw.WriteHeader(404); sw.WriteHeader(500)`.
  - Assert: `sw.status == 404` (первый победил), `rec.Code == 404` (тоже не перезаписан повторно).
- `TestResponseWriter_WrapIdempotent`:
  - `sw1 := wrap(rec); sw2 := wrap(sw1)`.
  - Assert: `sw1 == sw2` (тот же указатель).

Пакет — `middleware` (белый ящик, потому что нужен доступ к приватному `wrap`). Альтернатива — экспортировать `Wrap`, но это излишне (используется только внутри пакета `middleware`).

## Файлы для модификации

Никаких. Это «зелёное поле».

## Ключевые решения

- **`WriteJSONError` экспортирована, а `errorEnvelope`/`errorBody` — нет.** Public API пакета `pkg/httpx/` ограничен функциями. Структуры — детали реализации (ADR-011 в `../03-decisions.md`).
- **`responseWriter` — приватный тип.** Не экспортируется, не нужен вне `pkg/httpx/middleware/`. Recover и Logger в фазе 2 пользуются им как пакетным types.
- **`wrap` идемпотентен.** Это позволяет нескольким middleware независимо обернуть `w`, не дублируя обёртку (`../03-decisions.md`, ADR-002).

## Verification

- [ ] `go build ./pkg/httpx/...` проходит.
- [ ] `go test ./pkg/httpx/...` проходит — 6 тестов (2 jsonerror + 2 contextkeys + 3 responsewriter; считаем `WrapIdempotent` отдельно).
- [ ] `gofmt -l pkg/httpx/` — пустой вывод.
- [ ] `go vet ./pkg/httpx/...` — exit 0.
- [ ] Phase-specific check: `pkg/httpx/jsonerror.go` импортирует только stdlib (`encoding/json`, `net/http`).
- [ ] Phase-specific check: `pkg/httpx/contextkeys.go` импортирует только stdlib (`context`).
- [ ] Phase-specific check: `pkg/httpx/middleware/responsewriter.go` импортирует только stdlib (`net/http`).
- [ ] `git diff --name-only` ограничен `pkg/httpx/**`. Никаких других файлов.
