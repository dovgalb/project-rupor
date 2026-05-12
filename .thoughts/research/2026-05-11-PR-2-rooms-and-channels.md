---
date: 2026-05-11
researcher: Claude
commit: 93d8a0e
branch: feature/PR-2_rooms_and_channels
research_question: "PR-2 'Комнаты и каналы' — как устроен текущий проект и что нужно учесть, чтобы по эталону `auth` реализовать домены `room`/`channel`: миграции, sqlc, домен/usecase/transport/repository, HTTP-маршруты, middleware, тесты, документация"
---

# Исследование: подготовка к PR-2 «Комнаты и каналы»

## Резюме

Проект `project_rupor` — Go-бэкенд на чистой архитектуре (chi + sqlc + pgx, без ORM). Один домен `auth` уже полностью реализован и служит эталоном. Каталоги `internal/room/`, `internal/channel/`, `internal/user/`, `internal/chat/`, `internal/voice/` существуют только в виде пустых каркасов из четырёх `.gitkeep` (`domain/`, `usecase/`, `transport/http/`, `repository/postgres/`) — Go-кода в них нет.

Composition root — `cmd/server/main.go`: вручную собирает зависимости и Mount-ит роутеры доменов внутри `chi.Router.Route("/api/v1", ...)`. Auth-домен экспортирует `httpauth.RegisterRoutes(r chi.Router, deps Deps)` — этот паттерн надо повторить для room/channel. JWT-защита — через `internal/auth/transport/http/middleware/auth.go::RequireAuth`, кладёт `domain.UserID` в контекст; читать в хендлерах через `authmw.UserIDFromContext`.

Миграции — `golang-migrate`, парные up/down в `migrations/` с 4-значными префиксами; sqlc один общий конфиг (`sqlc.yaml`) с `queries: "internal/auth/repository/postgres/queries"` — сейчас под один домен. Для добавления `rooms/room_members/invites/channels` потребуется три новых миграции (0004–0006 или объединённые) и расширение `sqlc.yaml` (массив `sql:` либо общий `queries/` под несколько папок). Автоматический `arch_test.go` enforced только для `auth`-веток — для `room`/`channel` правила надо дописывать по образцу.

Стиль тестов — стандартный `testing` без testify, ручные фейки в `package <pkg>_test`, `t.Parallel()`, табличные тесты, инжекция Clock/UUID/Rand. Интеграционные тесты с реальным Postgres — под build-tag `integration` и env `TEST_DATABASE_URL`. Документация фичи — обязательная пачка `01-architecture.md … 08-api-contract.md` в `docs/<N_имя>/`, плюс ручные QA-сценарии `.http` в `manual_qa/<N_имя>/`.

## Детальные результаты

### 1. Эталонный паттерн домена `auth`

**Расположение:** `internal/auth/`

#### Domain layer (`internal/auth/domain/`)
- Entities имеют только приватные поля; конструкторы `NewXxx` проверяют все инварианты, `ReconstructXxx` — отдельный конструктор для репозитория, тоже проверяет инварианты (`user.go:13-31`, `refresh_token.go:14-40`).
- Value objects (`Email`, `Username`, `Password`, `PasswordHash`, `TokenHash`, `UserID`, `RefreshTokenID`) валидируют формат и нормализуют (trim, lowercase) в конструкторе.
- Бизнес-методы — мутаторы с проверкой инварианта (например `RefreshToken.Revoke(now)` отказывает уже отозванному, `refresh_token.go:42-48`).
- Доменные ошибки — sentinel `var ErrXxx = errors.New("auth: ...")` в `domain/errors.go:6-30`, разделены на «валидация VO/Entity» и «бизнес-правила».
- В `domain/` **нет интерфейсов** — все интерфейсы зависимостей объявляются в `usecase/`.

#### Usecase layer (`internal/auth/usecase/`)
- `ports.go:12-44` — все интерфейсы зависимостей (`UserRepository`, `RefreshTokenRepository`, `PasswordHasher`, `TokenIssuer`, `Clock`, `UUIDGenerator`, `RandomBytes`).
- Один use case — отдельный файл с типами `XxxInput`/`XxxOutput`, конструктором `NewXxx(...)` и единственным методом `Execute(ctx, in) (out, error)`.
- Доменные ошибки от VO-конструкторов пробрасываются «как есть»; технические — оборачиваются `fmt.Errorf("usecase: ...: %w", err)`.
- Транзакции инкапсулированы в репозитории (`RefreshTokenRepository.Rotate` — атомарная операция). Отдельного `UnitOfWork` интерфейса в usecase нет.

#### Transport HTTP (`internal/auth/transport/http/`, пакет `httpauth`)
- Один файл — один хендлер: `XxxHandler{ uc *usecase.Xxx }`, `NewXxxHandler(uc)`, `ServeHTTP(w, r)`.
- DTO с camelCase json-тегами лежат в `dto.go`, **приватные** типы (`registerRequest`, `tokensResponse` …).
- Никакой валидации в транспорте — все правила формата проверяет домен через VO-конструкторы; транспорт реагирует только на ошибку `json.Decode` через `writeBadBody` (AUTH-012).
- Маппинг доменных ошибок → HTTP в одном месте — `error_mapper.go:17-46` через `errors.Is`. Формат envelope `{"error":{"code":"AUTH-NNN","message":"..."}}` пишется через `pkg/httpx.WriteJSONError`.
- Регистрация маршрутов — `routes.go:10-31`: структура `Deps` агрегирует все usecase + `TokenIssuer` + `Clock`, функция `RegisterRoutes(r chi.Router, deps Deps)` создаёт `r.Route("/auth", ...)` с двумя группами: публичной и защищённой через `authmw.RequireAuth`.

#### Repository postgres + sqlc (`internal/auth/repository/postgres/`)
- `queries/<table>.sql` — сырые SQL-запросы для sqlc; `db/` — сгенерированный код (`db.go`, `models.go`, `<table>.sql.go`).
- Адаптер репозитория — `<entity>_repository.go`. Принимает `*db.Queries` (если транзакции не нужны) или `*pgxpool.Pool` напрямую (если нужны — `refresh_token_repository.go:24-26`, `Rotate` — `BeginTx`/`WithTx`/`Commit`).
- Маппинг через `mapper.go`: `rowToDomain` использует доменные `NewXxx`, оборачивая ошибки `fmt.Errorf("user row: <field>: %w", err)`. Финал — `domain.ReconstructXxx`.
- Ошибки Postgres: `pgx.ErrNoRows` → доменная `ErrXxxNotFound`; unique-violation детектируется через `pgerr.go:9-17` (`*pgconn.PgError`, `Code == "23505"` + `ConstraintName`) и мапится в доменную `ErrEmailAlreadyTaken`/`ErrUsernameAlreadyTaken`.
- UUID мапится через `github.com/google/uuid.UUID` (sqlc override). Не-nullable `timestamptz` → `time.Time`. Nullable `timestamptz` → `pgtype.Timestamptz` (запись через `pgtype.Timestamptz{Time:t, Valid:true}`).

#### Композиция (DI)
- Composition root — `cmd/server/main.go::run` (`main.go:64-168`). Builder-пакета нет — всё собирается прямо в `run`.
- Порядок: `pgxpool.New` → `db.New(pool)` → репозитории → адаптеры (bcrypt/jwt) → runtime-зависимости (`realClock`, `realUUID`, `cryptoRand` из `cmd/server/runtime.go:10-26`) → usecases → роутер → middleware → mount роутов в `mux.Route("/api/v1", ...)` → `http.Server` + graceful shutdown.

#### Архитектурные правила (`prompts/`)
- `Architecture Layers.txt:67-73`: `domain/` ← stdlib; `usecase/` ← domain; `transport/`/`repository/` ← usecase + domain; кросс-доменный импорт `transport/`/`repository/` запрещён.
- `Domain Model.txt`: rich domain, приватные поля, два конструктора (`NewXxx` для usecase, `ReconstructXxx` для репозитория), VO с валидацией в конструкторе, sentinel-ошибки в `domain/errors.go`, связи между сущностями — по id (не вложенными объектами).
- `RepoModel.txt:152-159`: `pgx.ErrNoRows` → доменная `NotFound`; unique violation по constraint name → доменная ошибка; sqlc-структуры наружу `usecase/`/`domain/` не выносить; транзакции через `pgx.Tx + Queries.WithTx`.
- `Builder.txt`: Builder только в тестах, живёт в тест-пакете, принимает `*testing.T`, `Build()` падает через `t.Fatalf`.
- `Go style.txt`: gofmt + golangci-lint обязательны; интерфейсы в потребителе, маленькие; SQL только sqlc, без ORM; `panic` в продовом коде запрещён, кроме composition root.

### 2. Миграции и sqlc

#### Существующие миграции (`migrations/`)
- `0001_init.up/down.sql` — `CREATE EXTENSION pgcrypto` (для `gen_random_uuid()`).
- `0002_users.up/down.sql` — `CREATE EXTENSION citext`; таблица `users` (`id uuid PK default gen_random_uuid()`, `email citext UNIQUE`, `password_hash text`, `username text UNIQUE`, `created_at timestamptz default now()`, named CHECK на длину username).
- `0003_refresh_tokens.up/down.sql` — таблица `refresh_tokens` (`user_id uuid REFERENCES users(id) ON DELETE CASCADE`, `token_hash bytea UNIQUE` с CHECK на `octet_length=32`, `expires_at`, `created_at default now()`, nullable `revoked_at`); индекс `refresh_tokens_user_id_idx`.

#### Соглашения о миграциях
- 4-значный префикс, snake_case имя, парность up/down: `<NNNN>_<name>.<up|down>.sql`.
- Заголовок: первая строка — имя файла-комментарий, вторая — короткое описание.
- `IF NOT EXISTS` только для расширений (на up). На down: `DROP TABLE IF EXISTS`.
- UUID для всех PK/FK; `timestamptz` для всех времён; `text` для строк (или `citext` для регистронезависимых); `bytea` для бинарных хешей.
- CHECK именованные: `<table>_<field>_<rule>_check`. Индексы именованные: `<table>_<col>_idx`.
- FK inline через `REFERENCES … ON DELETE CASCADE`. ENUM-типов в проекте пока нет — для `channels.type ('text'|'voice')` и `room_members.role ('owner'|'admin'|'member')` придётся выбрать стиль (CHECK-constraint на text vs Postgres ENUM-тип) — прецедента в кодовой базе нет.

#### sqlc workflow (`sqlc.yaml`)
- `version: "2"`, единственная запись в `sql:` под auth-домен.
- `engine: postgresql`, `schema: "migrations"`, `queries: "internal/auth/repository/postgres/queries"`, `sql_package: pgx/v5`, `package: db`, `out: "internal/auth/repository/postgres/db"`.
- Все `emit_*` отключены (нет интерфейсов, json/db-тегов, prepared queries, pointer-for-null).
- Overrides: `uuid` → `github.com/google/uuid.UUID`; `timestamptz` (и `pg_catalog.timestamptz`) → `time.Time`. Nullable `timestamptz` остаётся дефолтным `pgtype.Timestamptz`.
- **Под room/channel конфиг придётся расширить** — добавить блоки `sql:` для `internal/room/repository/postgres/queries` и `internal/channel/repository/postgres/queries` (sqlc v2 поддерживает массив).

#### Команды Makefile
- Миграции: `make migrate-up` (`migrate -path migrations -database $DATABASE_URL up`) и `make migrate-down` (`down 1`). Команды `migrate create` в Makefile **нет** — файлы новых миграций создаются вручную или прямым вызовом `migrate create -ext sql -dir migrations -seq <name>`.
- sqlc: `make sqlc` → `sqlc generate`.
- Прочее: `run`, `build`, `test` (`-race -count=1`), `lint` (`golangci-lint run`), `fmt` (gofmt+goimports `-local github.com/dovgalb/project-rupor`), `vet`, `dc-up/down/logs`, `init-project`.
- `DATABASE_URL` подставляется из `.env` с дефолтом `postgres://rupor:rupor@localhost:5432/rupor?sslmode=disable`.

#### Стиль .sql-запросов sqlc
- Файл = таблица: `internal/auth/repository/postgres/queries/<table>.sql`.
- `-- name: <PascalName> :<kind>`. Используются `:exec`, `:one`, `:execrows`. `:many` пока нет (потребуется для списков комнат/каналов/участников).
- Позиционные плейсхолдеры `$1`, `$2`. Без `sqlc.arg()`.
- `SELECT` всегда явно перечисляет столбцы — никогда `SELECT *`.
- `RETURNING` **не используется**: id генерится в Go и передаётся в `INSERT`.
- `:execrows` применяется когда важно число обновлённых строк (идемпотентный `UPDATE ... WHERE ... AND <cond> IS NULL`).

### 3. HTTP-роутинг, middleware, композиция

#### Composition root (`cmd/server/main.go`)
1. `slog` JSON-логгер.
2. `config.Load(config.OsLookuper{})` (readme errors → exit 2).
3. `pgxpool.New(ctx, cfg.DatabaseURL())` + defer Close.
4. `db.New(pool)` → репозитории → bcrypt/jwt-адаптеры → runtime (`realClock/realUUID/cryptoRand`) → usecases.
5. `chi.NewRouter()` → глобальные middleware → `mux.Route("/api/v1", ...)` → `httpauth.RegisterRoutes(r, httpauth.Deps{...})`.
6. `http.Server` с `ReadHeaderTimeout: 10s`, graceful shutdown по `SIGINT/SIGTERM` с timeout 5s.

#### Глобальный middleware-стек (порядок применения)
1. `httpx/middleware.RequestID(uuidGen)` — принимает входящий `X-Request-ID` или генерирует UUID; кладёт в контекст и в response header.
2. `httpx/middleware.Recover(logger)` — ловит panic, логирует, отвечает 500 `{"error":{"code":"INTERNAL"}}`.
3. `httpx/middleware.Logger(logger, userIDHook)` — access-log; `userIDHook` извлекает `user_id` через `authmw.UserIDFromContext` (инъецируется в `main.go:106-112`, чтобы `pkg/httpx` не зависел от `internal/auth`).
4. `httpx/middleware.CORS(cfg.CORSAllowedOrigins(), false)` — whitelist origins, без wildcard. Methods: `GET, POST, PUT, DELETE, OPTIONS`. Headers: `Authorization, Content-Type, X-Request-ID`. Preflight → 204.

#### Auth middleware
- `internal/auth/transport/http/middleware/auth.go:17` — `RequireAuth(issuer usecase.TokenIssuer, clock usecase.Clock) func(http.Handler) http.Handler`. Парсит `Authorization: Bearer <jwt>`, вызывает `issuer.VerifyAccess`, кладёт `domain.UserID` в контекст. Ошибки → 401 (`AUTH-010` invalid / `AUTH-011` expired) через `httpx.WriteJSONError`.
- Применяется на уровне chi-группы внутри Mount-функции домена: `r.Group(func(r chi.Router) { r.Use(authmw.RequireAuth(deps.TokenIssuer, deps.Clock)); ... })`.

#### Извлечение userID
- Ключи контекста — приватные неэкспортируемые типы:
  - `pkg/httpx/contextkeys.go` — `httpx.WithRequestID`/`httpx.RequestIDFromContext`.
  - `internal/auth/transport/http/middleware/contextkeys.go:9-24` — `authmw.WithUserID`/`authmw.UserIDFromContext`.
- В хендлере: `uid, ok := authmw.UserIDFromContext(r.Context())`. Защитная ветка `!ok` → `mapError(domain.ErrAccessTokenInvalid)` (`me_handler.go:20-25`).

#### Mount-паттерн для новых доменов
Эталон — `internal/auth/transport/http/routes.go:10-31`:
```go
type Deps struct {
    Register    *usecase.RegisterUser
    // … usecases
    TokenIssuer usecase.TokenIssuer
    Clock       usecase.Clock
}
func RegisterRoutes(r chi.Router, deps Deps) { r.Route("/auth", ...) }
```
Для PR-2:
1. `internal/room/transport/http/routes.go` — пакет `httproom`, `Deps{...}`, `RegisterRoutes(r chi.Router, deps Deps)` → `r.Route("/rooms", func(r chi.Router) { r.Use(authmw.RequireAuth(...)); ... })`.
2. `internal/channel/transport/http/routes.go` — аналогично, либо `r.Route("/rooms/{roomID}/channels", ...)`, либо вложенным mount внутри room-роутера. Path-параметры — `chi.URLParam(r, "roomID")`.
3. В `cmd/server/main.go` после `httpauth.RegisterRoutes(r, ...)` добавить `httproom.RegisterRoutes(r, httproom.Deps{TokenIssuer: issuer, Clock: clock, ...})` и аналогично для channel — `issuer` и `clock` уже инициализированы.
4. Импорт `internal/auth/transport/http/middleware` из транспорта `room`/`channel` для `RequireAuth` и `UserIDFromContext` — кросс-доменный, но `arch_test.go` его сейчас не запрещает (см. ниже).

#### arch_test.go — enforced правила
`arch_test.go` парсит import-блоки non-test `.go`-файлов и ругается на нарушения:
- `TestArchitecture_DomainImports` (61): `internal/auth/domain` ← stdlib + `github.com/google/uuid`.
- `TestArchitecture_UseCaseImports` (82): `internal/auth/usecase` ← stdlib + `uuid` + `internal/auth/domain`.
- `TestArchitecture_RepositoriesIsolated` (104): `postgres`/`jwt`/`bcrypt` адаптеры не импортируют друг друга и не импортируют `internal/auth/transport/...`.
- `TestArchitecture_UsecaseUsedAsContract` (133): `internal/auth/transport/http` (только корень) не импортирует `internal/auth/repository/...`.
- `TestArchitecture_PkgHttpxImports` (149): `pkg/httpx` и `pkg/httpx/middleware` ← stdlib + сам `pkg/httpx`. Никаких `internal/...`.
- `TestArchitecture_AuthMiddlewareImports` (176): `internal/auth/transport/http/middleware` ← stdlib + `uuid` + `internal/auth/usecase` + `internal/auth/domain` + `pkg/httpx`.
**Для room и channel аналогичные правила сейчас не enforced — их можно дописать по образцу.**

#### Обработка ошибок
- Единый writer: `pkg/httpx.WriteJSONError(w, status, code, message)` — формат `{"error":{"code":"...","message":"..."}}`, Content-Type `application/json; charset=utf-8`.
- В каждом домене — свой `error_mapper.go` со своим префиксом кодов (auth → `AUTH-NNN`). Для room/channel логично завести `ROOM-NNN`/`CHANNEL-NNN` (соглашения нет — выбирается на старте).

### 4. Текущее состояние room/channel и стиль тестов

#### Скелеты доменов
- `internal/room/{domain,usecase,transport/http,repository/postgres}/.gitkeep` — все 4 файла по 0 байт, Go-кода нет.
- `internal/channel/{domain,usecase,transport/http,repository/postgres}/.gitkeep` — то же.
- `internal/user/`, `internal/chat/`, `internal/voice/` — те же 4 пустых `.gitkeep`. Реализации нет. Функциональность «текущий пользователь» уже есть в `internal/auth/` (`User`, `GetCurrentUser`, `/auth/me`).

#### Стиль тестов (по образцу `internal/auth/`)
- Только стандартный `testing`. Без testify/ginkgo/gomega.
- Пакет тестов внешний: `package usecase_test`, `package httpauth_test`, `package postgres_test`.
- `t.Parallel()` в каждом тесте и подтесте.
- `t.Helper()` обязателен в хелперах.
- Хелперы-конструкторы фикстур: `mustEmail(t, raw)`, `mustUser(t, ...)` (`internal/auth/usecase/fakes_test.go:201-268`).
- Моки — **только ручные фейки** в том же тест-пакете (`fakeUserRepo`, `fixedClock`, `fixedUUID`, `fixedRand`). `internal/auth/usecase/fakes_test.go:14-198`. Никаких gomock/mockery.
- Детерминизм: время — `fixedClock` с фиксированной `time.Date(2026,5,10,12,0,0,0,time.UTC)`; UUID/rand — через интерфейсы.
- SUT-pattern: `newRegisterSUT(t)` собирает все зависимости и use case (`register_user_test.go:15-42`).
- Сравнения: ошибки — `errors.Is`; значения — явное `!=` + `t.Fatalf("X = %q, want %q", got, want)`.
- Интеграционные тесты: build-tag `//go:build integration`, env `TEST_DATABASE_URL` (если не задано — `t.Skip`); очистка `TRUNCATE ... CASCADE`. Без testcontainers/dockertest — внешняя БД. Сейчас все интеграционные стоят на `t.Skip("integration: requires Postgres, see issue 1.5")`.
- HTTP-тесты: `httptest.NewServer` поверх `chi.Router`, реальные usecase + bcrypt/jwt + фейковые репозитории, `t.Cleanup(srv.Close)`, `bcrypt.MinCost` для скорости.
- Compile-check тесты в `repository/{bcrypt,jwt,postgres}/compile_check_test.go` — статически проверяют, что реализация удовлетворяет интерфейсу.

#### Требования из `prompts/Tests Style.txt` и `Domain model test.txt`
- Уровни тестов: 1) domain (без I/O) → 2) usecase с фейками → 3) repository против реального Postgres → 4) HTTP через `httptest` → 5) e2e опционально. Не инвертировать пирамиду.
- Domain-тест: пакет `domain_test` (чёрный ящик), без моков и фреймворков, `< 10ms`, table-driven по умолчанию (с захватом `tc := tc`). Проверяем конструкторы и инварианты (отказ невалидным входам, конкретные доменные ошибки), бизнес-методы, value objects (валидация + нормализация), методы-запросы, неизменность состояния при ошибке. Не тестируем БД/HTTP/usecase.
- Запрещено: `time.Sleep`, зависимость от сети/даты/timezone/глобальных env (только `t.Setenv`), `reflect.DeepEqual` для сущностей с временем.
- Перед PR: `go test ./...`, `go test -race ./...`, тесты на новые ветки, нет `t.Skip` без обоснования.

### 5. Документация фичи

#### `docs/<N_имя>/` — обязательная пачка файлов
- `README.md` — front-matter `date/feature/status/research`, бизнес-контекст, scope/out-of-scope, критерии приёмки, таблица документов.
- `01-architecture.md` — Logical (C4 L1→L2→L3, граф зависимостей).
- `02-behavior.md` — Process (DFD, sequence diagrams, error/edge cases).
- `03-decisions.md` — Decision (ADR, риски, open questions).
- `04-testing.md` — Quality (coverage mapping, тест-кейсы).
- `05-events.md` — опционально, доменные события.
- `06-repo-model.md` — Repository model (есть в `1_3_auth_domen`, опущен в `1_4_auth_middleware` намеренно).
- `07-standards.md` — Compliance-матрица по стандартам в `prompts/`.
- `08-api-contract.md` — API surface.
- Шаблон зафиксирован в `docs/1_4_auth_middleware/README.md:58-72`.
- Существующие фичи: `docs/1_1_project-structure`, `1_2_db_schema`, `1_3_auth_domen`, `1_4_auth_middleware`. Для PR-2 ожидается папка вида `docs/2_1_rooms_and_channels/` (точное имя пока не зафиксировано).

#### `manual_qa/<N_имя>/` — ручные QA-сценарии
- Пер-фича пронумерованные `.http`-файлы для VS Code REST Client / JetBrains HTTP Client.
- Пример: `manual_qa/1_3_auth/{00_flow,01_register,02_login,03_refresh,04_me,99_health}.http` + `README.md`.
- В шапке — `@host`, `@origin`, `@access_token`. README описывает подготовку стенда (`make dc-up && make migrate-up && make run`), требуемые env, переиспользование токенов из соседних флоу.

### 6. Общий план — Фаза 2 (`.claude/plans/general_plan.md:127-138`)

Готовность 0%, все задачи `[ ]`. Содержание:
- Миграции: `rooms`, `room_members` (роли owner/admin/member), `invites`, `channels` (тип text/voice).
- Домен `room`: сущности, роли, инварианты прав.
- Usecase room: создать/получить/удалить комнату, список комнат пользователя; генерация инвайта, вступление по коду; список участников.
- Домен `channel`: CRUD каналов внутри комнаты, проверка прав.
- HTTP: `POST/GET/DELETE /rooms`, `/rooms/:id/invite`, `/rooms/join/:code`, `/rooms/:id/members`; `POST/GET/DELETE /rooms/:id/channels`.
- Репозитории Postgres (sqlc) для room/channel.
- Тесты.

Сущности и связи (Room owner+members+channels+invite, Channel `text|voice`) описаны в `general_plan.md:5-25`. Сообщения вынесены в Фазу 3 — не входят в PR-2.

Невыполнен также подпункт фазы 1 — **1.5 Тесты** (`general_plan.md:121-124`): unit usecase auth с моками, интеграционные HTTP auth, e2e. По факту unit/usecase и HTTP-тесты уже написаны, а интеграционные `postgres_test` стоят на `t.Skip(... see issue 1.5)`.

## Ссылки на код

### Эталон auth
- `internal/auth/domain/user.go:5-37` — Entity User
- `internal/auth/domain/refresh_token.go:5-65` — Entity RefreshToken c бизнес-методом `Revoke`
- `internal/auth/domain/email.go:8-26`, `username.go:8-30`, `password.go:8-21` — VO с валидацией/нормализацией
- `internal/auth/domain/errors.go:6-30` — sentinel ошибки
- `internal/auth/usecase/ports.go:12-44` — все интерфейсы зависимостей
- `internal/auth/usecase/register_user.go:11-75` — каноничный usecase
- `internal/auth/transport/http/routes.go:10-31` — Mount-функция и Deps
- `internal/auth/transport/http/dto.go:9-51` — DTO + json helpers
- `internal/auth/transport/http/error_mapper.go:17-54` — маппинг доменных ошибок
- `internal/auth/transport/http/middleware/auth.go:17-45` — RequireAuth
- `internal/auth/transport/http/middleware/contextkeys.go:9-24` — UserID context
- `internal/auth/transport/http/me_handler.go:11-41` — пример защищённого хендлера
- `internal/auth/repository/postgres/user_repository.go:19-60` — адаптер на `*db.Queries`
- `internal/auth/repository/postgres/refresh_token_repository.go:19-86` — адаптер с транзакцией (`Rotate`)
- `internal/auth/repository/postgres/mapper.go:11-77` — row↔domain
- `internal/auth/repository/postgres/pgerr.go:9-17` — детектор unique-violation
- `internal/auth/repository/postgres/queries/users.sql:1-13`, `queries/refresh_tokens.sql:1-15` — стиль sqlc-запросов

### Инфраструктура
- `cmd/server/main.go:37-168` — composition root
- `cmd/server/runtime.go:10-26` — Clock/UUID/Rand адаптеры
- `cmd/server/health.go:7` — публичный health-handler
- `pkg/httpx/jsonerror.go:18` — `WriteJSONError`
- `pkg/httpx/contextkeys.go:5-16` — RequestID context
- `pkg/httpx/middleware/{requestid,recover,logger,cors,responsewriter}.go` — глобальные middleware
- `arch_test.go:61-201` — арх-правила (только под auth)
- `migrations/0001_init.*`, `0002_users.*`, `0003_refresh_tokens.*` — стиль миграций
- `sqlc.yaml:1-27` — конфиг sqlc
- `Makefile:48-55` — migrate-up/down, sqlc generate

### Тесты и документация
- `internal/auth/usecase/register_user_test.go:15-42` — SUT-pattern
- `internal/auth/usecase/fakes_test.go:14-268` — каталог фейков и must-хелперов
- `internal/auth/repository/postgres/integration_helpers_test.go:1-35` — pattern интеграционных тестов
- `internal/auth/repository/postgres/user_repository_integration_test.go:19-67` — пример со `t.Skip`
- `internal/auth/transport/http/setup_test.go:163-216` — pattern HTTP-тестов
- `prompts/Tests Style.txt`, `prompts/Domain model test.txt` — требования к тестам
- `docs/1_4_auth_middleware/README.md:58-72` — шаблон документации фичи
- `manual_qa/1_3_auth/README.md`, `manual_qa/1_4_auth_middleware/README.md` — шаблон ручных QA
- `.claude/plans/general_plan.md:5-25, 50-62, 121-138` — описание сущностей, API-контракт, задачи Фазы 2

## Архитектурные наблюдения

- **Слои**: `domain ← usecase ← {transport,repository}`; composition в `cmd/server`. Интерфейсы зависимостей — в потребителе (usecase). Доменные ошибки — sentinel в `domain/errors.go`.
- **Поток данных (запрос)**: HTTP → middleware (RequestID/Recover/Logger/CORS) → chi router → group `RequireAuth` (положил `UserID` в контекст) → handler (`json.Decode` → `usecase.Execute`) → `mapper.go` → `db.Queries` (sqlc, pgx).
- **Поток данных (ошибка)**: domain error (`errors.New`) → usecase оборачивает или пробрасывает → `error_mapper.mapError(err)` → `httpx.WriteJSONError` → JSON envelope.
- **Ключевые зависимости**: `chi/v5` (роутер), `pgx/v5 + pgxpool` (БД), `sqlc` (codegen), `golang-migrate` (миграции), `google/uuid`, `bcrypt`, `golang-jwt/jwt/v5`, `slog`.
- **Изоляция `pkg/httpx`**: не импортирует `internal/...` (enforced). Любая интеграция (например, логирование `user_id`) — через инъецируемые hooks из main.
- **Кросс-доменный импорт**: `internal/<domain>/transport/http` может импортировать `internal/auth/transport/http/middleware` для `RequireAuth`/`UserIDFromContext` — единственный санкционированный кросс-доменный мост; arch_test это не запрещает.
- **Единый источник конфигурации**: `config/config.go` через env, fail-fast.
- **Готовых соглашений нет** для: (1) ENUM vs CHECK для ролей/типов каналов в Postgres; (2) расширения `sqlc.yaml` под несколько доменов; (3) кодов ошибок `ROOM-*`/`CHANNEL-*`; (4) арх-тестов под `room`/`channel`.
