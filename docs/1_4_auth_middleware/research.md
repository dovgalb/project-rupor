---
date: 2026-05-10
researcher: Claude
commit: 92c8852
branch: feature/PR-1_4-auth_middleware
research_question: |
  Baseline кодовой базы перед реализацией фичи 1.4 «Middleware» из general_plan.md:
  JWT-middleware (извлечение Authorization: Bearer, валидация, проброс userID
  в контекст), логирование запросов, recover middleware, CORS (для будущего
  фронта).
---

# Исследование: auth_middleware (baseline)

## Резюме

Текущий HTTP-стек проекта — `chi v5.2.1` без единой подключённой middleware.
В `cmd/server/main.go:100` создаётся один корневой роутер, всё дерево висит на
`mux.Route("/api/v1", ...)`. Ни вызовов `Use(...)`, ни импорта `chi/middleware`,
ни кастомных middleware-пакетов в репозитории нет (ни в `internal/`, ни в `pkg/`).
Зависимостей для CORS/recover/логирования запросов в `go.mod` тоже нет.

Авторизация по access-токену сегодня живёт инлайново внутри одного хендлера —
`internal/auth/transport/http/me_handler.go:23-37`: парсинг префикса `Bearer `,
вызов `usecase.TokenIssuer.VerifyAccess(token, now)`, передача `uid.String()`
в `usecase.GetCurrentUser`. В `request.Context()` userID не кладётся, общего
key/helper для извлечения userID из контекста в коде нет.

JWT-инфраструктура: HS256, claims = `jwtv5.RegisteredClaims` (`sub` = UUID-строка,
`iat`, `exp`), TTL берётся из `cfg.JWTAccessTTL()`. Refresh-токены —
не JWT (32 байта rand + sha256 в БД). Порт `usecase.TokenIssuer`
(`internal/auth/usecase/ports.go:29-32`) предоставляет ровно два метода:
`IssueAccess` и `VerifyAccess`. Все ошибки валидации схлопываются в
`domain.ErrAccessTokenInvalid` / `domain.ErrAccessTokenExpired`; маппер
(`internal/auth/transport/http/error_mapper.go:36-41`) переводит их в
`401 AUTH-010` и `401 AUTH-011` соответственно с envelope
`{"error":{"code":"...","message":"..."}}`.

Архитектурные тесты `arch_test.go` ограничивают импорты в `domain/`, `usecase/`
и `repository/{postgres,jwt,bcrypt}`, а транспорту запрещают импортировать
`repository/...`. Каркас доменов `internal/{user,room,channel,chat,voice}/`
существует, но пуст. `pkg/websocket/` — единственный подкаталог `pkg/`, в нём
только `.gitkeep`. Конфиг (`config/config.go`) знает про `DATABASE_URL`,
`JWT_SECRET`, `SERVER_PORT`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`; полей про
CORS/origins/log-level нет. Логгер — `slog` JSON на уровне Info, без request-id
и без контекстного логгера. Линтер v2 включает только `govet`, `staticcheck`,
`errcheck`, `ineffassign`, `unused`.

## Детальные результаты

### 1. HTTP-стек и точки внедрения middleware

- **Корневой роутер**: `cmd/server/main.go:100` — `mux := chi.NewRouter()`.
  Никакого `Use(...)`, `Group(...)`, `Mount(...)` на уровне `mux` нет.
- **Единственная группа**: `cmd/server/main.go:101` —
  `mux.Route("/api/v1", func(r chi.Router) { ... })`. Внутри:
  - `cmd/server/main.go:102` — `r.Get("/health", healthHandler)` →
    `/api/v1/health`.
  - `cmd/server/main.go:103-110` — `httpauth.RegisterRoutes(r, httpauth.Deps{...})`.
- **Передача в `http.Server`**: `cmd/server/main.go:114-118`, выставлен
  только `ReadHeaderTimeout: 10s`.
- **Регистрация auth-маршрутов**: `internal/auth/transport/http/routes.go:18-25` —
  `r.Route("/auth", func(r chi.Router) { ... })`, внутри подряд четыре
  маршрута без `Group`/`Use`:
  - `POST /register` (`routes.go:20`) — публичный.
  - `POST /login`    (`routes.go:21`) — публичный.
  - `POST /refresh`  (`routes.go:22`) — публичный.
  - `GET  /me`       (`routes.go:23`) — приватный, авторизация инлайново
    в хендлере.
- **Структура `Deps`**: `internal/auth/transport/http/routes.go:9-16` —
  `Register *usecase.RegisterUser`, `Login *usecase.LoginUser`,
  `Refresh *usecase.RefreshAccess`, `Me *usecase.GetCurrentUser`,
  `TokenIssuer usecase.TokenIssuer`, `Clock usecase.Clock`.
- **Health endpoint**: `cmd/server/health.go:7-11` — пишет
  `Content-Type: application/json; charset=utf-8`, статус 200, тело
  `{"status":"ok"}\n`. Тест: `cmd/server/health_test.go`.
- **Текущая защита `/api/v1/auth/me`**:
  `internal/auth/transport/http/me_handler.go:23-51` — инлайново:
  - `r.Header.Get("Authorization")` (`:24`).
  - Префикс `bearerPrefix = "Bearer "` (`:11`), проверка через
    `strings.HasPrefix` и непустой хвост (`:25-28`); иначе
    `mapError(domain.ErrAccessTokenInvalid)` → `writeError`.
  - `token := auth[len(bearerPrefix):]` (`:29`).
  - `uid, err := h.issuer.VerifyAccess(token, h.clock.Now())` (`:31`);
    при ошибке — `writeError(w, mapError(err))` (`:33`).
  - Передача в usecase: `h.uc.Execute(r.Context(),
    usecase.GetCurrentUserInput{UserID: uid.String()})` (`:37`).
  - В `r.Context()` userID **не кладётся**.

### 2. JWT-инфраструктура

- **TokenIssuer**: `internal/auth/repository/jwt/token_issuer.go:14-21` —
  `struct { secret []byte; accessTTL time.Duration }`. Конструктор
  `NewTokenIssuer(secret []byte, accessTTL time.Duration)` (`:17-21`).
  Никаких refreshTTL/iss/aud/kid.
- **Выпуск (`IssueAccess`)**: `token_issuer.go:23-35` — алгоритм HS256
  (`SigningMethodHS256`, `:30`), claims = `jwtv5.RegisteredClaims`
  (`:25-29`): `Subject = userID.String()`, `IssuedAt = now`,
  `ExpiresAt = now.Add(i.accessTTL)`. Подпись — `token.SignedString(i.secret)`
  (`:31`).
- **Верификация (`VerifyAccess`)**: `token_issuer.go:38-62`. Каждый раз
  создаётся `jwtv5.NewParser(WithValidMethods([]string{"HS256"}),
  WithTimeFunc(...))` (`:39-42`). `now` приходит снаружи, `time.Now()`
  внутри не вызывается. Маппинг ошибок (`:47-60`):
  - `errors.Is(err, jwtv5.ErrTokenExpired)` → `domain.ErrAccessTokenExpired`.
  - Прочие ошибки парсера → `fmt.Errorf("%w: %v",
    domain.ErrAccessTokenInvalid, err)`.
  - Невалидный `claims.Subject` (`uuid.Parse`) →
    `domain.ErrAccessTokenInvalid`.
  - Ошибка `domain.NewUserID` (zero-uuid) →
    `domain.ErrAccessTokenInvalid`.
- **Refresh ≠ JWT**: refresh-токены — 32 байта `crypto/rand` + sha256, хранятся
  в БД через `RefreshTokenRepository` (`internal/auth/usecase/login_user.go:14`,
  `internal/auth/usecase/refresh_access.go:48-58, 83-91`). На уровне claim'ов
  «тип токена» не различается, потому что отдельного claim'а нет.
- **Порт на стороне usecase**: `internal/auth/usecase/ports.go:29-32` —
  ```go
  type TokenIssuer interface {
      IssueAccess(userID domain.UserID, now time.Time) (string, time.Time, error)
      VerifyAccess(token string, now time.Time) (domain.UserID, error)
  }
  ```
  Compile-time assertion: `internal/auth/repository/jwt/compile_check_test.go:8`.
- **Подключение в main**: `cmd/server/main.go:75` —
  `issuer := jwtadapter.NewTokenIssuer([]byte(cfg.JWTSecret()), cfg.JWTAccessTTL())`.
  Один и тот же `issuer` инжектится в `LoginUser`, `RefreshAccess` и
  `httpauth.Deps.TokenIssuer` (`main.go:91-110`).
- **`domain.UserID`**: `internal/auth/domain/user_id.go:5-16` — value object
  поверх `uuid.UUID`, конструктор отвергает `uuid.Nil`. В JWT кладётся как
  каноническая UUID-строка (`uuid.UUID.String()`).
- **Тесты `token_issuer_test.go`**:
  1. `TestTokenIssuer_RoundTrip` (`:40-62`).
  2. `TestTokenIssuer_VerifyAccess_RejectsAlgNone` (`:64-78`) — отсечение
     через `WithValidMethods`.
  3. `TestTokenIssuer_VerifyAccess_RejectsWrongSecret` (`:80-101`).
  4. `TestTokenIssuer_VerifyAccess_RejectsExpired` (`:103-119`).
  5. `TestTokenIssuer_VerifyAccess_RejectsBadSub` (`:121-142`).
  6. `TestTokenIssuer_VerifyAccess_RejectsTampered` (`:144-169`).
  Тестов на пустую строку / отсутствие `exp` / `nbf` в будущем нет.
- **Тесты me-хендлера** (`me_handler_test.go`): `TestMeHandler_Success`
  (`:30-56`), `TestMeHandler_NoAuth` (`:58-73`), `TestMeHandler_BadScheme`
  (`:75-90`), `TestMeHandler_InvalidSignature` (`:92-115`),
  `TestMeHandler_TokenExpired` (`:117-144`).

### 3. Транспорт auth и контракт ошибок

- **DTO** (`internal/auth/transport/http/dto.go`):
  - `registerRequest` (`:9-13`), `registerResponse` (`:15-20`).
  - `loginRequest` (`:22-25`), `tokensResponse` (`:31-36`).
  - `refreshRequest` (`:27-29`).
  - `meResponse` (`:38-43`).
  - JSON-имена — camelCase, поля времени — `time.Time`.
  - Хелперы: `jsonDecode` (`:54-56`), `jsonEncode` (`:58-60`).
- **Envelope ошибок** (`dto.go:45-52`):
  ```json
  { "error": { "code": "AUTH-XXX", "message": "..." } }
  ```
  Никаких `details`/`error` (как строки)/HTTP-кода в теле нет. Полей `traceId`
  тоже нет.
- **Маппер ошибок** (`internal/auth/transport/http/error_mapper.go:16-45`):

  | Доменная ошибка | HTTP | code | message |
  |---|---|---|---|
  | `ErrInvalidEmail` | 400 | `AUTH-001` | `invalid email` |
  | `ErrInvalidUsername` | 400 | `AUTH-002` | `invalid username` |
  | `ErrInvalidPassword` | 400 | `AUTH-003` | `invalid password` |
  | `ErrEmailAlreadyTaken` | 409 | `AUTH-004` | `email already taken` |
  | `ErrUsernameAlreadyTaken` | 409 | `AUTH-005` | `username already taken` |
  | `ErrInvalidCredentials` | 401 | `AUTH-006` | `invalid credentials` |
  | `ErrRefreshTokenNotFound` | 401 | `AUTH-007` | `refresh token not found` |
  | `ErrRefreshTokenRevoked` | 401 | `AUTH-008` | `refresh token revoked` |
  | `ErrRefreshTokenExpired` | 401 | `AUTH-009` | `refresh token expired` |
  | `ErrAccessTokenInvalid` / `ErrUserNotFound` / `ErrInvalidUserID` | 401 | `AUTH-010` | `access token invalid` |
  | `ErrAccessTokenExpired` | 401 | `AUTH-011` | `access token expired` |
  | (default) | 500 | `INTERNAL` | `internal` |

  Дополнительно: `writeBadBody(w)` (`error_mapper.go:53-55`) — 400 +
  `AUTH-012` + `malformed request body`.

- **`writeError`** (`error_mapper.go:47-51`): `Content-Type:
  application/json; charset=utf-8`, `WriteHeader(status)`, `jsonEncode`
  envelope; ошибка записи проглатывается (`_ =`).
- **Хендлеры**:

  | Путь | Метод | Request | Success | Статус | Ошибки |
  |---|---|---|---|---|---|
  | `/auth/register` | POST | `registerRequest` | `registerResponse` | 201 | `writeBadBody` / `mapError` |
  | `/auth/login` | POST | `loginRequest` | `tokensResponse` | 200 | то же |
  | `/auth/refresh` | POST | `refreshRequest` | `tokensResponse` | 200 | то же |
  | `/auth/me` | GET | `Authorization: Bearer <jwt>` | `meResponse` | 200 | inline-401 для bad bearer / `mapError` |

- **Тесты** (`setup_test.go`, `*_handler_test.go`):
  - Пакет внешний — `httpauth_test`.
  - `httptest.NewServer(chi.NewRouter())` + `httpauth.RegisterRoutes`
    (`setup_test.go:199-210`); `t.Cleanup(srv.Close)`.
  - Запросы — через `http.DefaultClient.Do(...)` (реальный TCP-листенер),
    не `httptest.NewRecorder`.
  - Параллелизм — `t.Parallel()` в каждом тесте.
  - Реальные prod-зависимости в setup: bcrypt (`xbcrypt.MinCost`), JWT
    (`testJWTSecret`, `testAccessTTL`).
  - Фейки usecase-репозиториев — `fakeUserRepo` (`setup_test.go:31-74`),
    `fakeRefreshRepo` (`:76-125`).
  - Детерминизм: `realClock{now}` (`:127-129`), `seqUUID` (`:131-141`),
    `seqRand` (`:143-161`); фиксированное `now = 2026-05-10 12:00:00 UTC`.
  - Утверждения — без сторонних библиотек, только `t.Fatalf`.
  - Проверяется только `error.code`, не `error.message`.
- **Заголовки ответов**: только `Content-Type: application/json; charset=utf-8`.
  Нет `Vary`, `Cache-Control`, `WWW-Authenticate`, никаких
  `Access-Control-*`, нет `X-Request-ID`. Тело завершается `\n` (следствие
  `json.Encoder.Encode`).

### 4. Архитектурные правила, конфиг, логирование, линтер

- **`prompts/Architecture Layers.txt`**:
  - Слои: domain → usecase → adapters (transport, repository) → infrastructure
    (`cmd`, `pkg`, `config`, `migrations`).
  - `domain/` — только stdlib; никаких JSON/sqlc-тегов.
  - `usecase/` — только `domain/` + stdlib (+ `uuid` де-факто).
  - `transport/...` и `repository/...` импортируют `usecase/` и `domain/`.
  - `cmd/server/` — composition root, может импортировать всё.
  - Запреты: SQL/HTTP в usecase/domain; `*http.Request`/`http.ResponseWriter`
    глубже хендлера; глобальные переменные для зависимостей; пакеты
    `utils`/`helpers`/`common`/`models`; ORM (только sqlc); `panic` вне
    composition root; новые библиотеки без согласования.
  - Терминология: `usecase` (не `service`); зависимости — по роли
    (`UserRepository`, `TokenIssuer`, `Clock`, `UUIDGenerator`); usecase-DTO —
    `XxxInput/XxxOutput`, транспортные DTO — `XxxRequest/XxxResponse`.
- **`arch_test.go`**:
  1. `TestArchitecture_DomainImports` (`:61-80`) — `internal/auth/domain/`:
     stdlib + `github.com/google/uuid`.
  2. `TestArchitecture_UseCaseImports` (`:82-102`) — `internal/auth/usecase/`:
     stdlib + `uuid` + `domainPath`.
  3. `TestArchitecture_RepositoriesIsolated` (`:104-130`) — `postgres`/`jwt`/
     `bcrypt` не импортируют друг друга и не импортируют `transport/...`.
  4. `TestArchitecture_UsecaseUsedAsContract` (`:133-147`) —
     `internal/auth/transport/http/` не импортирует
     `internal/auth/repository/...`.

  Тесты сейчас ничего не говорят про новые подкаталоги: middleware можно
  поместить в существующий `internal/auth/transport/http/` (импорт
  `usecase.TokenIssuer` через usecase-порт там разрешён; `repository/...`
  по-прежнему запрещён) либо в новый `internal/auth/transport/middleware/`,
  `pkg/...` или `cmd/server/`. Запрещено: `domain/`, `usecase/`, любые
  `repository/...` подпапки.

- **Конфиг (`config/config.go`)**:
  - `Config` (`:36-42`): `serverPort`, `databaseURL`, `jwtSecret`,
    `jwtAccessTTL`, `jwtRefreshTTL`. Геттеры (`:44-48`).
  - Полей про CORS/origins/log-level **нет**.
  - `Lookuper` (`:50-52`), `OsLookuper` (`:54-58`).
  - `ValidationError{Code, Field, Reason}` + `ErrConfigInvalid`
    (`:20-34`).
  - Валидация:
    | Env | Default | Правило | Код |
    |---|---|---|---|
    | `JWT_SECRET` | — | required | `CONFIG-001` |
    | `DATABASE_URL` | — | required | `CONFIG-002` |
    | `SERVER_PORT` | `8080` | `Atoi`, `1..65535` | `CONFIG-003` |
    | `JWT_ACCESS_TTL` | `15m` | `>0`, `<= 1h` | `CONFIG-004` |
    | `JWT_REFRESH_TTL` | `720h` | `>0`, `> access`, `<= 90*24h` | `CONFIG-005` |
- **`.env` и `.env.example`** — идентичны, содержат только `DATABASE_URL`,
  `JWT_SECRET`, `SERVER_PORT`. `JWT_ACCESS_TTL`/`JWT_REFRESH_TTL` берут defaults.
- **slog в `cmd/server/main.go`**:
  - `slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level:
    slog.LevelInfo}))` (`main.go:35-38`).
  - `slog.SetDefault(logger)` (`:38`).
  - Логи на старте/остановке: `slog.String`, `slog.Any`. Ключи: `code`,
    `field`, `reason`, `err`, `addr`.
  - Request-ID и контекстный логгер **не используются** (нет ни вызовов
    `Use(...)`, ни `slog.With(ctx)`).
- **`.golangci.yml`** v2: `default: none`. Включены `govet` (`enable-all:
  true`, отключён только `fieldalignment`), `staticcheck`, `errcheck`,
  `ineffassign`, `unused`. Форматтеры — `gofmt`, `goimports` с
  `local-prefixes: github.com/dovgalb/project-rupor`. Запретов на `panic`,
  `interface{}`, `_ = err`, использование `log` вместо `slog`, ограничений
  cyclomatic-complexity — нет.
- **Структура `internal/`**: `auth/` — реализован, остальные домены
  (`channel`, `chat`, `room`, `user`, `voice`) — пустые каркасы папок.
  `pkg/websocket/` — единственная подпапка `pkg/`, содержит только
  `.gitkeep`.
- **`go.mod` (require)**: `chi/v5 v5.2.1`, `golang-jwt/jwt/v5 v5.3.1`,
  `google/uuid v1.6.0`, `pgx/v5 v5.9.2`, `golang.org/x/crypto v0.51.0`.
  CORS-библиотек, recover-/logger-middleware пакетов нет; `chi/middleware`
  не импортируется. `testify` присутствует в `go.sum`, но не объявлен в
  `require`.
- **`manual_qa/1_3_auth/`** — формат `.http` (REST Client / JetBrains HTTP
  Client). Файлы: `README.md`, `00_flow.http`, `01_register.http`,
  `02_login.http`, `03_refresh.http`, `04_me.http`, `99_health.http`.
  Захват токенов через `# @name`, переменные `@host`/`email`/`username`/
  `password`.

## Ссылки на код

### Точка входа и роутер
- `cmd/server/main.go:34-59` — main: загрузка конфига, инициализация
  логгера, вызов `run`.
- `cmd/server/main.go:61-150` — `run`: инициализация pgx, репозиториев,
  адаптеров, usecase, роутера, сервера, graceful shutdown.
- `cmd/server/main.go:75` — создание `TokenIssuer`.
- `cmd/server/main.go:100-111` — chi-роутер и подключение
  `/api/v1/health` и `httpauth.RegisterRoutes`.
- `cmd/server/main.go:114-118` — `http.Server` (`ReadHeaderTimeout: 10s`).

### Auth transport
- `internal/auth/transport/http/routes.go:9-25` — `Deps` и
  `RegisterRoutes`.
- `internal/auth/transport/http/me_handler.go:11, 13-21, 23-51` —
  bearer-prefix, структура хендлера, ServeHTTP.
- `internal/auth/transport/http/error_mapper.go:10-14, 16-45, 47-55` —
  `httpError`, `mapError`, `writeError`/`writeBadBody`.
- `internal/auth/transport/http/dto.go:9-60` — DTO + envelope ошибок +
  json-helpers.
- `internal/auth/transport/http/setup_test.go:31-210` — тестовое
  окружение и фейки.

### Domain & usecase
- `internal/auth/usecase/ports.go:29-32` — порт `TokenIssuer`.
- `internal/auth/usecase/get_current_user.go:12-50` — usecase Me на вход
  принимает строку userID.
- `internal/auth/domain/user_id.go:5-16` — value object UserID на UUID.
- `internal/auth/domain/errors.go:28-29` — `ErrAccessTokenInvalid`,
  `ErrAccessTokenExpired`.

### JWT
- `internal/auth/repository/jwt/token_issuer.go:14-62` — IssueAccess +
  VerifyAccess.
- `internal/auth/repository/jwt/compile_check_test.go:8` — assertion
  соответствия порту.

### Архитектура и конфиг
- `arch_test.go:61-147` — четыре архитектурных теста.
- `prompts/Architecture Layers.txt:16-123` — правила слоёв.
- `config/config.go:36-210` — Config, Load, валидация.
- `.env`, `.env.example` — идентичные, 3 переменные.
- `.golangci.yml:1-28` — линтер v2.
- `go.mod:1-19` — зависимости.

## Архитектурные наблюдения

- **Паттерн**: Clean Architecture с явным портом `TokenIssuer` в usecase
  и адаптером `internal/auth/repository/jwt/`. Транспорт зависит только
  от usecase/domain.
- **Поток данных при авторизации Me**:
  1. HTTP-запрос → `me_handler.go:23` (chi роутит `/api/v1/auth/me`).
  2. Хендлер парсит `Authorization`, режет префикс `Bearer ` (`:24-29`).
  3. `usecase.TokenIssuer.VerifyAccess(token, clock.Now())`
     (`:31`) → `domain.UserID` либо доменная ошибка.
  4. `usecase.GetCurrentUser.Execute(ctx, {UserID: uid.String()})`
     (`:37`) → `domain.User` либо доменная ошибка.
  5. Успех → `meResponse` JSON-сериализован, 200. Ошибка → `mapError` →
     `writeError` → envelope c `code`/`message`.
- **Ключевые зависимости**: `chi v5.2.1`, `golang-jwt/jwt/v5`, `pgx/v5`,
  `google/uuid`, `golang.org/x/crypto/bcrypt`. `slog` из stdlib.
- **Контракт ошибок**: HTTP-код + envelope `{"error":{"code","message"}}`,
  `code` = `AUTH-XXX` или `INTERNAL`/`AUTH-012`. `WWW-Authenticate` не
  используется.
- **Что зависит от факта «middleware пока нет»**:
  - userID не лежит в `request.Context()` ни на одном маршруте.
  - Логирование запросов не ведётся (логи только в main).
  - `panic` в любом хендлере уронит процесс (нет recover).
  - CORS-заголовков сервер не выставляет ни в одном ответе.
  - `/api/v1/auth/me` — единственное место, парсящее Bearer-токен.
