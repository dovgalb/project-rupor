---
phase: 7
name: internal/auth/transport/http
layer: transport
depends_on: phase-03, phase-06
plan: ./README.md
---

# Phase 7: `internal/auth/transport/http/`

## Цель

Реализовать HTTP-транспорт: 4 хендлера, DTO, error-mapper, регистрацию роутов на chi. После фазы — все 4 эндпоинта работают через `httptest.NewServer` (с фейковыми репозиториями + реальными use case'ами + реальными jwt/bcrypt-адаптерами с минимальным cost'ом). 19 unit-тестов зелёные (`../04-testing.md:256-298`).

## Контекст

- Контракт — `../08-api-contract.md`. Тела запросов/ответов и JSON-формы — точно зафиксированы.
- Маппинг ошибок — `../08-api-contract.md:32-46` + sequence diagrams в `../02-behavior.md`.
- В фазе 1.3 нет JWT-middleware (фаза 1.4) — `MeHandler` валидирует токен inline (`../03-decisions.md` ADR-009).

## Файлы для создания

Все — в `internal/auth/transport/http/`. Имя пакета: `httpauth` (не `http`, чтобы избежать коллизии с stdlib `net/http`).

### `dto.go`

Все request/response/error структуры. JSON-теги — точно как в `../08-api-contract.md`.

```go
package httpauth

import "time"

// Register
type registerRequest struct {
    Email    string `json:"email"`
    Username string `json:"username"`
    Password string `json:"password"`
}

type registerResponse struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Username  string    `json:"username"`
    CreatedAt time.Time `json:"createdAt"`
}

// Login + Refresh (общий ответ)
type loginRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type refreshRequest struct {
    RefreshToken string `json:"refreshToken"`
}

type tokensResponse struct {
    AccessToken      string    `json:"accessToken"`
    RefreshToken     string    `json:"refreshToken"`
    AccessExpiresAt  time.Time `json:"accessExpiresAt"`
    RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
}

// Me
type meResponse struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Username  string    `json:"username"`
    CreatedAt time.Time `json:"createdAt"`
}

// Errors
type errorEnvelope struct {
    Error errorBody `json:"error"`
}

type errorBody struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

DTO — приватные (lowercase). Они не утекают за пределы пакета — handler принимает JSON, конвертит в `usecase.*Input`, получает `usecase.*Output` и формирует response.

`time.Time` сериализуется в RFC3339 с UTC `Z`-суффиксом по умолчанию — соответствует `../08-api-contract.md:10`.

### `error_mapper.go`

```go
package httpauth

import (
    "errors"
    "net/http"

    "github.com/dovgalb/project-rupor/internal/auth/domain"
)

type httpError struct {
    status  int
    code    string
    message string
}

func mapError(err error) httpError {
    switch {
    case errors.Is(err, domain.ErrInvalidEmail):
        return httpError{http.StatusBadRequest, "AUTH-001", "invalid email"}
    case errors.Is(err, domain.ErrInvalidUsername):
        return httpError{http.StatusBadRequest, "AUTH-002", "invalid username"}
    case errors.Is(err, domain.ErrInvalidPassword):
        return httpError{http.StatusBadRequest, "AUTH-003", "invalid password"}
    case errors.Is(err, domain.ErrEmailAlreadyTaken):
        return httpError{http.StatusConflict, "AUTH-004", "email already taken"}
    case errors.Is(err, domain.ErrUsernameAlreadyTaken):
        return httpError{http.StatusConflict, "AUTH-005", "username already taken"}
    case errors.Is(err, domain.ErrInvalidCredentials):
        return httpError{http.StatusUnauthorized, "AUTH-006", "invalid credentials"}
    case errors.Is(err, domain.ErrRefreshTokenNotFound):
        return httpError{http.StatusUnauthorized, "AUTH-007", "refresh token not found"}
    case errors.Is(err, domain.ErrRefreshTokenRevoked):
        return httpError{http.StatusUnauthorized, "AUTH-008", "refresh token revoked"}
    case errors.Is(err, domain.ErrRefreshTokenExpired):
        return httpError{http.StatusUnauthorized, "AUTH-009", "refresh token expired"}
    case errors.Is(err, domain.ErrAccessTokenInvalid),
         errors.Is(err, domain.ErrUserNotFound),
         errors.Is(err, domain.ErrInvalidUserID):
        return httpError{http.StatusUnauthorized, "AUTH-010", "access token invalid"}
    case errors.Is(err, domain.ErrAccessTokenExpired):
        return httpError{http.StatusUnauthorized, "AUTH-011", "access token expired"}
    default:
        return httpError{http.StatusInternalServerError, "INTERNAL", "internal"}
    }
}

func writeError(w http.ResponseWriter, he httpError) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(he.status)
    _ = jsonEncode(w, errorEnvelope{Error: errorBody{Code: he.code, Message: he.message}})
}

func writeBadBody(w http.ResponseWriter) {
    writeError(w, httpError{http.StatusBadRequest, "AUTH-012", "malformed request body"})
}
```

`ErrUserNotFound` мапится на AUTH-010 (а не на 404), потому что у нас единственное место, где он возникает в HTTP-цепочке — `MeHandler.GetCurrentUser` после успешной валидации токена; «токен валиден, но user удалён» = «токен не валиден на этот момент» = AUTH-010 (`../08-api-contract.md:255`).

`ErrInvalidUserID` — внутренняя ошибка use case (sub в JWT не парсится); тоже AUTH-010 для `MeHandler`.

### `register_handler.go`

```go
type RegisterHandler struct {
    uc *usecase.RegisterUser
}

func NewRegisterHandler(uc *usecase.RegisterUser) *RegisterHandler {
    return &RegisterHandler{uc: uc}
}

func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var req registerRequest
    if err := jsonDecodeStrict(r.Body, &req); err != nil {
        writeBadBody(w)
        return
    }
    out, err := h.uc.Execute(r.Context(), usecase.RegisterUserInput{
        Email: req.Email, Username: req.Username, Password: req.Password,
    })
    if err != nil {
        writeError(w, mapError(err))
        return
    }
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusCreated)
    _ = jsonEncode(w, registerResponse{
        ID: out.UserID, Email: out.Email, Username: out.Username, CreatedAt: out.CreatedAt,
    })
}
```

`jsonDecodeStrict` — helper в `dto.go` или отдельном `json.go`:

```go
func jsonDecodeStrict(r io.Reader, v any) error {
    dec := json.NewDecoder(r)
    dec.DisallowUnknownFields() // НЕ применяем — по контракту лишние поля игнорируются (../08-api-contract.md:104)
    return dec.Decode(v)
}
```

Действительно — `DisallowUnknownFields` отключаем (контракт допускает). Strict-режим только в смысле «JSON должен распарситься».

### `login_handler.go`, `refresh_handler.go`

По шаблону. Только status 200.

### `me_handler.go`

```go
type MeHandler struct {
    uc     *usecase.GetCurrentUser
    issuer usecase.TokenIssuer
    clock  usecase.Clock
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    auth := r.Header.Get("Authorization")
    const prefix = "Bearer "
    if !strings.HasPrefix(auth, prefix) || len(auth) <= len(prefix) {
        writeError(w, mapError(domain.ErrAccessTokenInvalid))
        return
    }
    token := auth[len(prefix):]
    uid, err := h.issuer.VerifyAccess(token, h.clock.Now())
    if err != nil {
        writeError(w, mapError(err))
        return
    }
    out, err := h.uc.Execute(r.Context(), usecase.GetCurrentUserInput{UserID: uid.String()})
    if err != nil {
        writeError(w, mapError(err))
        return
    }
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    _ = jsonEncode(w, meResponse{ID: out.UserID, Email: out.Email, Username: out.Username, CreatedAt: out.CreatedAt})
}
```

Проверка `Bearer ` (CamelCase) — ровно по `../03-decisions.md` риску «Authorization header регистр».

### `routes.go`

```go
type Deps struct {
    Register   *usecase.RegisterUser
    Login      *usecase.LoginUser
    Refresh    *usecase.RefreshAccess
    Me         *usecase.GetCurrentUser
    TokenIssuer usecase.TokenIssuer
    Clock      usecase.Clock
}

func RegisterRoutes(r chi.Router, deps Deps) {
    r.Route("/auth", func(r chi.Router) {
        r.Post("/register", NewRegisterHandler(deps.Register).ServeHTTP)
        r.Post("/login", NewLoginHandler(deps.Login).ServeHTTP)
        r.Post("/refresh", NewRefreshHandler(deps.Refresh).ServeHTTP)
        r.Get("/me", (&MeHandler{uc: deps.Me, issuer: deps.TokenIssuer, clock: deps.Clock}).ServeHTTP)
    })
}
```

`Deps` — публичная структура для composition root в `cmd/server/main.go`.

### Тесты `*_test.go`

19 тестов из `../04-testing.md:256-298`. Каждый — через `httptest.NewServer` с реальными use case'ами (не моками!), фейковыми репозиториями (из фазы 6), реальными jwt/bcrypt-адаптерами с `bcrypt.MinCost=4` (`prompts/Tests Style.txt:147-162`).

Helper `setupServer(t)` строит граф: `fakeUserRepo` + `fakeRefreshRepo` + `bcrypt.NewPasswordHasher(4)` + `jwt.NewTokenIssuer(testSecret, time.Minute)` + use case'ы + chi-router → `httptest.NewServer`.

## Файлы для модификации

Нет (composition root в `cmd/server/main.go` — фаза 8).

## Опционально

`manual_qa/auth/test-flow.http` — IDE HTTP Client скрипт по `../08-api-contract.md:281-296`. Helpful для ручного QA, не блокирует фазу.

## Ключевые решения

- Имя пакета `httpauth` — избегаем shadow `http` stdlib.
- DTO приватные — не экспортируются. Это упрощает рефакторинг и предотвращает использование DTO за пределами транспорта.
- `mapError` обрабатывает все известные доменные ошибки в одном месте; default = 500. Расширение этого switch — единственный способ добавить новый AUTH-XXX код.
- `MeHandler` инлайн-валидирует токен; в фазе 1.4 — переедет в middleware (минимальный refactor: вынести 6 строк).
- Тесты используют `httptest` + реальный chi — это даёт подлинно HTTP-уровневую проверку, включая роутинг и method-mismatch (405 от chi default).

## Verification

- [ ] `go build ./internal/auth/transport/http/...` — зелёный.
- [ ] `go test ./internal/auth/transport/http/... -race -count=1` — все 19 тестов зелёные.
- [ ] `make lint` зелёный.
- [ ] Каждый эндпоинт из `../08-api-contract.md` имеет хендлер и тест happy-path.
- [ ] Каждый код ошибки `AUTH-001..AUTH-012` имеет ветку в `mapError` и тест, его срабатывающий.
- [ ] `git diff --name-only` ограничен `internal/auth/transport/http/**` (+ опционально `manual_qa/auth/**`).
