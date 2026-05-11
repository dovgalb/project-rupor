---
phase: 3
name: internal/auth/transport/http/middleware (RequireAuth)
layer: adapter
depends_on: [phase-01]
plan: ./README.md
---

# Phase 3: `internal/auth/transport/http/middleware/` — `RequireAuth`

## Цель

Реализовать auth-specific middleware: typed-key и хелперы для `domain.UserID` в контексте, плюс `RequireAuth` фабрику, которая валидирует `Authorization: Bearer <jwt>` через существующий порт `usecase.TokenIssuer.VerifyAccess` и кладёт `domain.UserID` в контекст.

## Контекст

Готовые компоненты:
- `pkg/httpx.WriteJSONError` (фаза 1) — для 401-ответов.
- `pkg/httpx.RequestIDFromContext` (фаза 1) — не используется напрямую RequireAuth, но полезен в тестах для интеграции с Logger.
- `usecase.TokenIssuer` (фаза 1.3) — порт с методом `VerifyAccess(token, now) (domain.UserID, error)`. Возвращает `domain.ErrAccessTokenInvalid` или `domain.ErrAccessTokenExpired`.
- `usecase.Clock` (фаза 1.3) — порт `Now() time.Time`.
- `domain.UserID` (фаза 1.3) — value object с приватным `value uuid.UUID`. Конструктор `NewUserID` отвергает `uuid.Nil`.
- `domain.ErrAccessTokenInvalid`, `domain.ErrAccessTokenExpired` (фаза 1.3) — sentinel-ошибки.

Фаза не требует изменений в существующих пакетах. Подключение в `routes.go` — фаза 5.

## Файлы для создания

### `internal/auth/transport/http/middleware/contextkeys.go`

**Назначение:** типизированный ключ + хелперы для `domain.UserID` в `context.Context`.

**Детали реализации (`../01-architecture.md:140-148`, `../03-decisions.md`, ADR-006/013):**
- Пакет: `package middleware` (короткое имя по `prompts/Go style.txt:14-18`; полный путь `internal/auth/transport/http/middleware/`).
- Приватный тип: `type userIDKey struct{}`.
- Публичные функции:
  ```go
  func WithUserID(ctx context.Context, id domain.UserID) context.Context {
      return context.WithValue(ctx, userIDKey{}, id)
  }

  func UserIDFromContext(ctx context.Context) (domain.UserID, bool) {
      v := ctx.Value(userIDKey{})
      if v == nil {
          return domain.UserID{}, false
      }
      id, ok := v.(domain.UserID)
      return id, ok
  }
  ```
- `WithUserID` принимает `domain.UserID` — value object. Не принимает строку или `uuid.UUID`. Это инвариант: попавшая в ctx UserID — гарантированно валидная (zero-uuid невозможен).
- `UserIDFromContext` возвращает `(zero-VO, false)` при отсутствии. Соответствует idiom `value, ok :=`.
- Импорты: `context`, `github.com/dovgalb/project-rupor/internal/auth/domain`.

### `internal/auth/transport/http/middleware/auth.go`

**Назначение:** middleware-фабрика `RequireAuth`. Извлекает Bearer-токен, валидирует через TokenIssuer, кладёт `domain.UserID` в ctx, делегирует обработку handler-у.

**Детали реализации (`../01-architecture.md:130-138`, `../02-behavior.md` Use Case 2/3, `../03-decisions.md`, ADR-007/010):**
- Пакет: `package middleware`.
- Сигнатура:
  ```go
  func RequireAuth(issuer usecase.TokenIssuer, clock usecase.Clock) func(http.Handler) http.Handler
  ```
- Внутри замыкания:
  1. `auth := r.Header.Get("Authorization")`.
  2. Проверка префикса:
     ```go
     const bearerPrefix = "Bearer "
     if !strings.HasPrefix(auth, bearerPrefix) || len(auth) <= len(bearerPrefix) {
         httpx.WriteJSONError(w, http.StatusUnauthorized, "AUTH-010", "access token invalid")
         return
     }
     ```
     Case-sensitive (`../03-decisions.md`, ADR-010). Пустой токен после префикса (`len(auth) <= len(bearerPrefix)`) → AUTH-010.
  3. `token := auth[len(bearerPrefix):]`.
  4. `uid, err := issuer.VerifyAccess(token, clock.Now())`.
  5. На ошибке — маппинг через локальный helper:
     ```go
     func mapAuthError(err error) (status int, code, message string) {
         if errors.Is(err, domain.ErrAccessTokenExpired) {
             return http.StatusUnauthorized, "AUTH-011", "access token expired"
         }
         return http.StatusUnauthorized, "AUTH-010", "access token invalid"
     }
     ```
     Это ровно та же таблица, что в `error_mapper.go:36-41` (1.3) для ErrAccessTokenInvalid/Expired.
  6. Успех:
     ```go
     ctx := WithUserID(r.Context(), uid)
     next.ServeHTTP(w, r.WithContext(ctx))
     ```
- Импорты: `context`, `errors`, `net/http`, `strings`, `github.com/dovgalb/project-rupor/internal/auth/domain`, `github.com/dovgalb/project-rupor/internal/auth/usecase`, `github.com/dovgalb/project-rupor/pkg/httpx`.

### `internal/auth/transport/http/middleware/contextkeys_test.go`

**Назначение:** verify round-trip context helpers.

**Детали (2 теста из `../04-testing.md:170-173`):**
- `TestUserIDContext_RoundTrip`:
  - Создать валидную `uid := mustUserID(t, "550e8400-e29b-41d4-a716-446655440000")` через `domain.NewUserID(uuid.MustParse(...))`.
  - `ctx := WithUserID(context.Background(), uid)`.
  - `got, ok := UserIDFromContext(ctx)`.
  - Assert: `ok == true`, `got.String() == uid.String()`.
- `TestUserIDContext_AbsentReturnsFalse`:
  - `got, ok := UserIDFromContext(context.Background())`.
  - Assert: `ok == false`. `got` — zero-VO, доступ к `String()` тоже допустим (uuid.Nil.String() == `"00000000-0000-0000-0000-000000000000"`), но это деталь реализации; тест проверяет только `ok`.

Пакет — `middleware_test` (черный ящик).

Helper:
```go
func mustUserID(t *testing.T, raw string) domain.UserID {
    t.Helper()
    parsed, err := uuid.Parse(raw)
    if err != nil { t.Fatalf("uuid.Parse: %v", err) }
    uid, err := domain.NewUserID(parsed)
    if err != nil { t.Fatalf("domain.NewUserID: %v", err) }
    return uid
}
```

### `internal/auth/transport/http/middleware/auth_test.go`

**Назначение:** verify все сценарии RequireAuth.

**Детали (8 тестов из `../04-testing.md:175-200`, `../03-decisions.md`, ADR-007 + ADR-010):**

**Setup:**
- Реальный `*jwt.TokenIssuer` через `jwt.NewTokenIssuer([]byte("test-secret-1234567890"), 15*time.Minute)`.
- `fixedClock{now: time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)}`.
- Mini-router:
  ```go
  func newTestRouter(t *testing.T, issuer usecase.TokenIssuer, clock usecase.Clock) (chi.Router, *bool) {
      called := false
      r := chi.NewRouter()
      r.Use(middleware.RequireAuth(issuer, clock))
      r.Get("/protected", func(w http.ResponseWriter, req *http.Request) {
          uid, ok := middleware.UserIDFromContext(req.Context())
          if !ok { t.Errorf("UserIDFromContext: ok = false") }
          called = true
          fmt.Fprintf(w, "ok: %s", uid.String())
      })
      return r, &called
  }
  ```

**Tests:**

- `TestRequireAuth_Success`:
  - Issue access token: `tok, _, _ := issuer.IssueAccess(uid, clock.Now())`.
  - Запрос `GET /protected` + `Authorization: Bearer <tok>`.
  - Assert: 200, body содержит `"ok: <uid>"`, флаг `called == true`.
- `TestRequireAuth_NoHeader`:
  - Запрос без `Authorization`.
  - Assert: 401, body содержит `"AUTH-010"`, флаг `called == false`.
- `TestRequireAuth_BadScheme`:
  - `Authorization: Basic dGVzdA==`.
  - Assert: 401 + AUTH-010, `called == false`.
- `TestRequireAuth_EmptyToken`:
  - `Authorization: Bearer ` (с пробелом, без токена).
  - Assert: 401 + AUTH-010, `called == false`.
- `TestRequireAuth_InvalidSignature`:
  - Токен подписан другим секретом: создать второй `jwt.NewTokenIssuer([]byte("wrong-secret-..."))`, выпустить токен → проверять основным issuer-ом.
  - Assert: 401 + AUTH-010, `called == false`.
- `TestRequireAuth_TokenExpired`:
  - Вручную собрать JWT: `claims := RegisteredClaims{Subject: uid.String(), IssuedAt: NumericDate(now-2h), ExpiresAt: NumericDate(now-1h)}`. Подписать main-секретом.
  - Запрос с этим токеном на `clock.Now() = now`.
  - Assert: 401 + AUTH-011, `called == false`.
- `TestRequireAuth_BadSubClaim`:
  - Вручную собрать JWT с `Subject: "not-a-uuid"`. Подписать.
  - Assert: 401 + AUTH-010, `called == false`.
- `TestRequireAuth_NextNotCalledOnError`:
  - Через table-driven по предыдущим 6 ошибочным сценариям.
  - В каждом — assert `called == false`.

Используется `httptest.NewRecorder()` (не `httptest.NewServer()` — без TCP-listener'а, потому что middleware и handler в одном процессе).

## Файлы для модификации

Никаких. Это «зелёное поле».

## Ключевые решения

- **`RequireAuth` принимает интерфейс `usecase.TokenIssuer`, не конкретный `*jwt.TokenIssuer`.** Соответствует правилу слоёв — middleware зависит от usecase-порта, не от адаптера (`prompts/Architecture Layers.txt:71`).
- **Тесты используют реальный `*jwt.TokenIssuer`, не fake-issuer.** `prompts/Tests Style.txt:147-152` — реальные адаптеры в HTTP-тестах. Проверяет интеграцию middleware + jwt-валидация.
- **Маппинг ошибок дублирует логику `error_mapper.go:36-41`** для `ErrAccessTokenInvalid`/`ErrAccessTokenExpired`. Альтернатива (импорт `error_mapper.go` напрямую) создаст циклическую зависимость `middleware → http → middleware`. Дублирование 4 строк допустимо.
- **Case-sensitive `Bearer ` префикс** (`../03-decisions.md`, ADR-010). Совпадает с поведением 1.3.
- **`UserIDFromContext` возвращает `(VO, bool)`, не `(VO, error)`** (`../03-decisions.md`, ADR-013).

## Verification

- [ ] `go build ./internal/auth/transport/http/middleware/...` проходит.
- [ ] `go test ./internal/auth/transport/http/middleware/...` проходит — 10 тестов.
- [ ] `gofmt -l internal/auth/transport/http/middleware/` — пустой вывод.
- [ ] `go vet ./internal/auth/transport/http/middleware/...` — exit 0.
- [ ] Phase-specific check: пакет импортирует только stdlib, `uuid`, `internal/auth/{usecase,domain}`, `pkg/httpx`. НЕ импортирует `internal/auth/repository/...` ни в продовом коде, ни в тестах кроме реального `jwt.NewTokenIssuer` для setup (это допустимо в `_test.go`-файле — `arch_test.go` проверяет только non-test импорты).
- [ ] Phase-specific check: коды ошибок RequireAuth — только `AUTH-010` и `AUTH-011`. Никаких других. Совпадает с `../08-api-contract.md`.
- [ ] `go test ./internal/auth/transport/http/middleware/... -race -count=1` — race-detector чистый.
- [ ] `git diff --name-only` ограничен `internal/auth/transport/http/middleware/**`.
