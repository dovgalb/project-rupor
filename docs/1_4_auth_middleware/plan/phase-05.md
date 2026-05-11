---
phase: 5
name: интеграция в роутер и упрощение MeHandler
layer: infra + transport
depends_on: [phase-01, phase-02, phase-03, phase-04]
plan: ./README.md
---

# Phase 5: Интеграция middleware-stack

## Цель

Подключить четыре глобальные middleware (RequestID/Recover/Logger/CORS) в `cmd/server/main.go`, разделить `routes.go` на public/protected группы (с `RequireAuth` на protected), упростить `MeHandler` (читает `userID` из ctx), перевести `error_mapper.go:writeError` на `pkg/httpx.WriteJSONError`. Убедиться, что 19 регрессионных тестов 1.3 проходят без изменений в коде тестов.

## Контекст

Готовые компоненты:
- Фаза 1: `pkg/httpx.WriteJSONError`, `pkg/httpx.RequestIDFromContext`, `pkg/httpx/middleware.responseWriter`.
- Фаза 2: `pkg/httpx/middleware.RequestID/Recover/Logger/CORS`.
- Фаза 3: `internal/auth/transport/http/middleware.RequireAuth`, `WithUserID`/`UserIDFromContext`.
- Фаза 4: `cfg.CORSAllowedOrigins() []string`, env `CORS_ALLOWED_ORIGINS`.

Существующий `setup_test.go` (`internal/auth/transport/http/setup_test.go:200-207`) уже передаёт `TokenIssuer` и `Clock` через `httpauth.Deps`. Это значит, что после изменения `routes.go` с подключением `RequireAuth` тесты регрессии 1.3 (`MeHandler_NoAuth`, `_BadScheme`, `_InvalidSignature`, `_TokenExpired`, `_Success`) продолжают работать без правок в коде тестов: middleware читает те же `deps.TokenIssuer` и `deps.Clock` и возвращает те же 401 + AUTH-010/011.

## Файлы для модификации

### `cmd/server/main.go`

**Что меняется:** добавление 4 строк `mux.Use(...)` + composition of `userIDHook` + новые импорты.

**Точка вставки:** после `mux := chi.NewRouter()` (`cmd/server/main.go:100`), до `mux.Route("/api/v1", ...)` (`cmd/server/main.go:101`).

**Новый блок:**
```go
mux := chi.NewRouter()

uuidGen := func() string { return uuid.New().String() }
userIDHook := func(ctx context.Context) []slog.Attr {
    uid, ok := authmw.UserIDFromContext(ctx)
    if !ok {
        return nil
    }
    return []slog.Attr{slog.String("user_id", uid.String())}
}

mux.Use(httpxmw.RequestID(uuidGen))
mux.Use(httpxmw.Recover(logger))
mux.Use(httpxmw.Logger(logger, userIDHook))
mux.Use(httpxmw.CORS(cfg.CORSAllowedOrigins(), false))

mux.Route("/api/v1", func(r chi.Router) {
    r.Get("/health", healthHandler)
    httpauth.RegisterRoutes(r, httpauth.Deps{
        Register:    registerUC,
        Login:       loginUC,
        Refresh:     refreshUC,
        Me:          meUC,
        TokenIssuer: issuer,
        Clock:       clock,
    })
})
```

**Новые импорты (упорядочены `goimports` с `local-prefixes: github.com/dovgalb/project-rupor`):**
- `"context"` (если уже есть — оставить).
- `"log/slog"` (уже есть, `cmd/server/main.go:7`).
- `authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"`.
- `httpxmw "github.com/dovgalb/project-rupor/pkg/httpx/middleware"`.
- `"github.com/google/uuid"` (уже есть).

**Без изменений:**
- Загрузка config (`cmd/server/main.go:40-53`).
- Создание `pgxpool`, репозиториев, hasher, issuer, use cases (`cmd/server/main.go:64-98`).
- `http.Server` setup, signal handling, graceful shutdown (`cmd/server/main.go:113-149`).
- `realClock`, `realUUID`, `cryptoRand` структуры в `runtime.go`.

### `internal/auth/transport/http/routes.go`

**Что меняется:** разделение на public/protected groups + конструктор `RequireAuth`.

**Полный новый текст файла:**
```go
package httpauth

import (
    "github.com/go-chi/chi/v5"

    authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
    "github.com/dovgalb/project-rupor/internal/auth/usecase"
)

type Deps struct {
    Register    *usecase.RegisterUser
    Login       *usecase.LoginUser
    Refresh     *usecase.RefreshAccess
    Me          *usecase.GetCurrentUser
    TokenIssuer usecase.TokenIssuer
    Clock       usecase.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
    r.Route("/auth", func(r chi.Router) {
        r.Group(func(r chi.Router) {
            r.Post("/register", NewRegisterHandler(deps.Register).ServeHTTP)
            r.Post("/login", NewLoginHandler(deps.Login).ServeHTTP)
            r.Post("/refresh", NewRefreshHandler(deps.Refresh).ServeHTTP)
        })
        r.Group(func(r chi.Router) {
            r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock))
            r.Get("/me", NewMeHandler(deps.Me).ServeHTTP)
        })
    })
}
```

**Изменения по строкам vs текущая версия (`internal/auth/transport/http/routes.go:1-26`):**
- `import` — добавить `authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"`.
- `Deps` — без изменений в полях.
- `RegisterRoutes` — две `r.Group(...)` внутри `r.Route("/auth", ...)`. Public-группа содержит 3 хендлера, protected-группа — 1 (`/me`) с `r.Use(RequireAuth)`.
- `NewMeHandler(deps.Me)` — теперь принимает только usecase, без `TokenIssuer` и `Clock` (см. изменения `me_handler.go` ниже).

### `internal/auth/transport/http/me_handler.go`

**Что меняется:** упрощение — убирается inline-парсинг `Authorization` и вызов `VerifyAccess`.

**Полный новый текст файла:**
```go
package httpauth

import (
    "net/http"

    "github.com/dovgalb/project-rupor/internal/auth/domain"
    authmw "github.com/dovgalb/project-rupor/internal/auth/transport/http/middleware"
    "github.com/dovgalb/project-rupor/internal/auth/usecase"
)

type MeHandler struct {
    uc *usecase.GetCurrentUser
}

func NewMeHandler(uc *usecase.GetCurrentUser) *MeHandler {
    return &MeHandler{uc: uc}
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    uid, ok := authmw.UserIDFromContext(r.Context())
    if !ok {
        // Защитная ветка: handler оказался в public-группе по ошибке маршрутизации.
        writeError(w, mapError(domain.ErrAccessTokenInvalid))
        return
    }

    out, err := h.uc.Execute(r.Context(), usecase.GetCurrentUserInput{UserID: uid.String()})
    if err != nil {
        writeError(w, mapError(err))
        return
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    _ = jsonEncode(w, meResponse{
        ID:        out.UserID,
        Email:     out.Email,
        Username:  out.Username,
        CreatedAt: out.CreatedAt,
    })
}
```

**Изменения vs текущая версия (`internal/auth/transport/http/me_handler.go:1-52`):**
- Удалить `const bearerPrefix = "Bearer "` — больше не используется.
- Убрать поля `issuer usecase.TokenIssuer` и `clock usecase.Clock` из `MeHandler`.
- Конструктор `NewMeHandler(uc *usecase.GetCurrentUser)` — только usecase.
- В `ServeHTTP`:
  - удалить парсинг `r.Header.Get("Authorization")` и проверку `bearerPrefix`;
  - удалить вызов `h.issuer.VerifyAccess(token, h.clock.Now())`;
  - читать `uid, ok := authmw.UserIDFromContext(r.Context())` (новое);
  - защитная ветка `if !ok { writeError(...) }` — на случай ошибки маршрутизации;
  - `h.uc.Execute(...)` — без изменений.
- Импорты — добавить `authmw`, удалить `strings` (не используется), оставить `domain`, `usecase`, `net/http`.

### `internal/auth/transport/http/error_mapper.go`

**Что меняется:** `writeError` делегирует сериализацию envelope в `pkg/httpx.WriteJSONError`.

**Точечные изменения:**

`error_mapper.go:47-51`:
```go
func writeError(w http.ResponseWriter, he httpError) {
    httpx.WriteJSONError(w, he.status, he.code, he.message)
}
```

`error_mapper.go:53-55`:
```go
func writeBadBody(w http.ResponseWriter) {
    httpx.WriteJSONError(w, http.StatusBadRequest, "AUTH-012", "malformed request body")
}
```

**Импорты:**
- Добавить `"github.com/dovgalb/project-rupor/pkg/httpx"`.
- Убрать `jsonEncode`-вызов из `writeError` — `jsonEncode` остаётся в `dto.go` для success-ответов в хендлерах.
- Структуры `errorEnvelope` и `errorBody` (`dto.go:45-52`) можно удалить — `pkg/httpx.WriteJSONError` использует свои внутренние. **Решение:** оставить как есть в `dto.go` (нет вреда), не трогать DTO в этой фазе. Это уменьшает diff и не требует обновления `register_handler.go`/`login_handler.go`/`refresh_handler.go`.

**Изменения по поведению:**
- Тело ответа — идентично 1.3 (`{"error":{"code":"...","message":"..."}}`).
- Заголовок — идентичный (`Content-Type: application/json; charset=utf-8`).
- Регрессия 1.3 (тесты `_MalformedBody`, `_InvalidEmail`, `_NoAuth`, …) — без правок в коде тестов.

### `internal/auth/transport/http/setup_test.go`

**Что меняется:** ничего в существующем коде. Файл проверяется как pass-through — текущий `RegisterRoutes` теперь сам подключит `RequireAuth` к protected-группе, и тесты регрессии работают без изменений.

**Verification step**:
- Запустить `go test ./internal/auth/transport/http/... -count=1` ДО любых правок setup_test.go. Если все 19 тестов 1.3 зелёные — изменения в setup_test.go НЕ нужны.
- Если какой-то тест упал — это сигнал, что обновлённый `routes.go` ломает контракт. Чинить `routes.go`/`me_handler.go`, не setup.

## Файлы для создания

### `cmd/server/main_test.go` (опционально)

**Назначение:** smoke на полную сборку router'а через middleware-stack.

**Детали (1 тест из `../04-testing.md:243-249`):**
- `TestHealthHandler_ThroughMiddlewareStack`:
  - Сборка mini-router'а: `chi.NewRouter()` + `mux.Use(RequestID/Recover/Logger/CORS)` + `mux.Get("/api/v1/health", healthHandler)`.
  - Запрос `GET /api/v1/health` через `httptest.NewServer`.
  - Assert: 200, body `{"status":"ok"}`, `X-Request-ID` присутствует и валиден (UUID-формат).
  - Логгер — `slog.New(slog.NewJSONHandler(io.Discard, ...))` (заглушка, чтобы тест не загрязнял stdout).

**Альтернатива:** не создавать `main_test.go`, оставить smoke на manual_qa (фаза 6). Это упрощает фазу 5. **Решение реализатора:** если рефактор `cmd/server/main.go` сложен (выделение `buildRouter` helper), отложить тест на фазу 6 и покрыть только manual_qa.

## Ключевые решения

- **`Deps` не меняется.** Поля те же (`Register/Login/Refresh/Me/TokenIssuer/Clock`). `RequireAuth` конструируется внутри `RegisterRoutes`, не передаётся через `Deps`. Это сохраняет публичный API пакета `httpauth`.
- **Защитная ветка `if !ok` в `MeHandler`** — на случай ошибки маршрутизации. Должна быть unreachable в проде (chi-роутер вызывает `RequireAuth` ДО `MeHandler` в protected-группе), но защищает от регрессии при будущих правках routes.go.
- **`error_mapper.go` сохраняет внутренний тип `httpError`** — это локальная структура, удобная для маппинга. `pkg/httpx.WriteJSONError` принимает разложенные параметры, не структуру.
- **`setup_test.go` НЕ меняется.** Регрессия 1.3 — главный гарант, что middleware подключилось правильно.

## Verification

- [ ] `go build ./...` проходит после всех изменений.
- [ ] `go test ./...` проходит — все ~65 тестов (19 регрессии 1.3 + 46 новых).
- [ ] `go test ./internal/auth/transport/http/... -count=1` — 19 тестов 1.3 + 3 новых `MeHandler` зелёные.
- [ ] `make lint` проходит.
- [ ] `gofmt -l .` — пустой вывод.
- [ ] `go vet ./...` — exit 0.
- [ ] Phase-specific check: ручной запуск сервера (`make dc-up && make migrate-up && make run`):
  - `curl -i http://localhost:8080/api/v1/health` → 200 + `X-Request-ID: <uuid>`.
  - `curl -i http://localhost:8080/api/v1/auth/me` (без токена) → 401 + AUTH-010 + `X-Request-ID`.
  - `curl -i -X OPTIONS -H "Origin: http://localhost:5173" -H "Access-Control-Request-Method: POST" http://localhost:8080/api/v1/auth/login` → 204 + Access-Control-Allow-* headers.
- [ ] Phase-specific check: log-output содержит `slog.Info("http", method, path, status, duration, request_id)` на каждый запрос.
- [ ] Phase-specific check: `me_handler.go` НЕ содержит `r.Header.Get("Authorization")` и НЕ вызывает `issuer.VerifyAccess`.
- [ ] `git diff --name-only` ограничен: `cmd/server/main.go`, `internal/auth/transport/http/{routes,me_handler,error_mapper}.go`, опц. `cmd/server/main_test.go`.
