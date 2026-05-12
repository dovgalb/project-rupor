---
date: 2026-05-11
researcher: Claude
commit: 93d8a0e
branch: feature/PR-1_5_tests
research_question: "Текущее состояние тестов auth: unit usecase (с моками репо), интеграционные HTTP-хендлеров, e2e через .http"
---

# Исследование: Базовое состояние тестов auth (тикет 1.5)

## Резюме

Тесты auth уже массово существуют, написаны на стандартном `testing` без внешних фреймворков (нет `testify`, нет `mockgen`, нет `testcontainers`). Все unit-тесты используют `t.Parallel()` и собственные фейки.

- **Usecase**: все 4 сценария (Register, Login, RefreshAccess, GetCurrentUser) покрыты — 22 теста с фейками в `fakes_test.go`.
- **HTTP-транспорт**: 6 файлов тестов (~774 строки) поднимают реальный chi-роутер через `httptest.NewServer` с настоящими usecase + fake-репозиториями (usecase НЕ мокается). Покрытие — happy path, валидация DTO, маппинг доменных ошибок в коды AUTH-001…AUTH-012, INTERNAL.
- **Domain / JWT / Bcrypt**: чистые unit-тесты value objects, лайфцикла refresh-токена, security-edge-cases JWT (alg=none, tampering, expired).
- **Postgres-репозитории**: 9 интеграционных тестов **все помечены `t.Skip`** под build-tag `//go:build integration`. Ожидают внешнюю Postgres через env `TEST_DATABASE_URL`. Нет testcontainers, нет отдельного docker-compose.test.yml.
- **E2E (.http)**: 6 файлов для `1_3_auth/` (full flow + per-endpoint) и 4 файла для `1_4_auth_middleware/`. Используется VS Code REST Client с цепочками переменных `{{login.response.body.accessToken}}`.
- **CI**: `.github/workflows/ci.yml` запускает `go test ./... -race -count=1` + golangci-lint v2.12.1. Интеграционных шагов с БД в CI нет (build-tag `integration` не передаётся).

## Детальные результаты

### 1. Unit-тесты слоя usecase

**Расположение**: `internal/auth/usecase/`

**Фейки** (`fakes_test.go:1-269`):
- `fakeUserRepo` (15-53): in-memory `byID` / `byEmail` карты; `saveErr` для инжекции ошибок
- `fakeRefreshRepo` (56-98): карта `byHash`; параметры `saveErr`, `findErr`, `rotateErr`
- `fakeHasher` (101-145): счётчик `verifyCalls`, флаги `hashErr` / `verifyErr` / `verifyReject`
- `fakeIssuer` (148-164): лог `issued`, `accessTTL`, `issueErr`
- `fixedClock` (167-169), `fixedUUID` (172-181) — детерминированные генераторы
- `fixedRand` (184-197) — преднастроенные буферы случайных байт
- Helpers `mustEmail`, `mustUsername`, `mustUser` и т.п. с `t.Helper()`

**Тестовые файлы**:
- `register_user_test.go:1-152` — 6 кейсов: Success, InvalidEmail, InvalidUsername, InvalidPassword, EmailTaken, UsernameTaken
- `login_user_test.go:1-188` — 6 кейсов: Success, BadEmailFormat, UserNotFound, WrongPassword, **DummyHashCalledOnUserNotFound** (timing-attack защита), RandReadFails
- `refresh_access_test.go:1-191` — 7 кейсов: Success, NotFound (3 sub: bad base64, wrong length, missing), Revoked, Expired, RaceReturnsRevoked, TokenIssuerFails
- `get_current_user_test.go:1-53` — 3 кейса: Success, BadUUID, NotFound

**Зависимости**: используются настоящие usecase + все фейки портов из `ports.go:12-44`.

### 2. Тесты HTTP-транспорта

**Расположение**: `internal/auth/transport/http/`

**Тестовый стенд** (`setup_test.go:173-216`):
- `httptest.NewServer` + реальный chi-роутер из `routes.go`
- Настоящие usecase, bcrypt hasher, JWT issuer
- Фейк-репозитории `fakeUserRepo` / `fakeRefreshRepo` (общие с usecase-тестами по концепции, реализованы локально)
- Часы зафиксированы: `2026-05-10 12:00:00 UTC`
- `seqUUID`, `seqRand` — последовательные генераторы

**Тестовые файлы**:
- `register_handler_test.go:37-148` — 5 кейсов: Success (201), MalformedBody (AUTH-012), InvalidEmail (AUTH-001), EmailTaken (AUTH-004), UnsupportedMethod (405)
- `login_handler_test.go:20-118` — 4 кейса: Success (токены + timestamps), MalformedBody (AUTH-012), InvalidCredentials (AUTH-006 в 3 sub: формат email / not found / wrong password), InternalError (AUTH→INTERNAL через `seqRand` ошибку)
- `refresh_handler_test.go:36-167` — 5 кейсов: Success, MalformedBody, NotFound (AUTH-007), Revoked (AUTH-008 — инжекция revoke в fake-репо), Expired (AUTH-009 — подмена сохранённого токена)
- `me_handler_test.go:30-144` — 5 кейсов: Success, NoAuth (AUTH-010), BadScheme, InvalidSignature, TokenExpired (AUTH-011)
- `middleware/auth_test.go:45-274` — 8 кейсов RequireAuth: Success, NoHeader, BadScheme, EmptyToken, InvalidSignature, TokenExpired, BadSubClaim, **NextNotCalledOnError** (параметризованный)
- `middleware/contextkeys_test.go:26-47` — 2 кейса: RoundTrip, AbsentReturnsFalse

**Error mapper** (`error_mapper.go:17-55`): полный маппинг domain-ошибок:
- AUTH-001 (email) / AUTH-002 (username) / AUTH-003 (password) — валидация
- AUTH-004 (email taken) / AUTH-005 (username taken)
- AUTH-006 (invalid creds) / AUTH-007 (refresh not found) / AUTH-008 (revoked) / AUTH-009 (refresh expired)
- AUTH-010 (access invalid) / AUTH-011 (access expired)
- AUTH-012 (malformed JSON)
- INTERNAL (fallback 500)

### 3. Domain / JWT / Bcrypt

**Domain** (`internal/auth/domain/`): 13 `*_test.go`, все pure unit + `t.Parallel()`, без зависимостей.
- Value objects: `email_test.go:11-76`, `username_test.go:11-57`, `password_test.go:11-45`, `password_hash_test.go:10-17`, `token_hash_test.go:11-61`, `user_id_test.go:12-19`, `refresh_token_id_test.go:12-19`
- Aggregates: `user_test.go:13-90` (NewUser/Reconstruct), `refresh_token_test.go:14-147` (Create/Revoke/IsActive — 4 состояния)
- Фикстуры: `testing_fixtures_test.go:11-64` — helpers `mustEmail/...` для использования в других тестах

**JWT** (`internal/auth/repository/jwt/token_issuer_test.go:40-169`): 5 кейсов
- RoundTrip (issue + verify)
- VerifyAccess_RejectsAlgNone (запрет `alg=none`)
- VerifyAccess_RejectsWrongSecret
- VerifyAccess_RejectsExpired
- VerifyAccess_RejectsBadSub (sub не UUID)
- VerifyAccess_RejectsTampered (подмена payload)
- `compile_check_test.go` — `_ usecase.TokenIssuer = (*TokenIssuer)(nil)`

**Bcrypt** (`internal/auth/repository/bcrypt/password_hasher_test.go:22-72`): 3 кейса — RoundTrip, VerifyMismatch, VerifyInvalidHash. Плюс compile-check.

### 4. Интеграционные тесты Postgres

**Расположение**: `internal/auth/repository/postgres/`

- `integration_helpers_test.go:1-35` — build-tag `//go:build integration`. Функция `dbConn(t)` читает `TEST_DATABASE_URL` из env, открывает `pgxpool.Pool`. `truncate(t, pool)` чистит таблицы CASCADE. **Нет testcontainers**, **нет docker-compose.test.yml** — ожидается уже поднятая Postgres.
- `user_repository_integration_test.go:18-66` — 5 кейсов: Save_RoundTrip, Save_DuplicateEmail, Save_DuplicateUsername, FindByEmail_CaseInsensitive, FindByID_NotFound. **Все `t.Skip("...")`**.
- `refresh_token_repository_integration_test.go:12-30` — 4 кейса: Save_RoundTrip, FindByHash_NotFound, Rotate_Atomic, Rotate_AlreadyRevoked. **Все `t.Skip`**.
- `mapper_test.go:69-188` — 2 unit-теста маппинга domain ↔ db (round-trip всех полей, включая `pgtype.Timestamptz` для revoked-кейса). Эти тесты НЕ под build-tag.
- `compile_check_test.go` — проверка соответствия `UserRepository` / `RefreshTokenRepository` интерфейсам usecase.

### 5. E2E через .http

**`manual_qa/1_3_auth/`** (6 файлов):
- `00_flow.http` — happy-path цепочка: register → login → me → refresh → me (новый) → refresh (старым → revoked). Цепочка через `{{login.response.body.accessToken}}`.
- `01_register.http` — 201 / 409 (email|username taken) / 400 (invalid email|username|password|malformed JSON) / 405 (GET)
- `02_login.http` — 200 / 401 (wrong pass / missing user / invalid email — все AUTH-006 для timing-safety) / 400
- `03_refresh.http` — 200 / 401 (revoked / not found / bad base64 / wrong length)
- `04_me.http` — 200 / 401 (no header / empty / bad scheme / garbage / tampered signature / expired)
- `99_health.http` — liveness probe

Динамические переменные: `{{$timestamp}}` для уникальных email; `@accessToken` / `@refreshToken` подставляются вручную или из предыдущего request'а.

**`manual_qa/1_4_auth_middleware/`** (4 файла):
- `00_smoke.http` — стек middleware + клиентский X-Request-ID + 404
- `01_cors_preflight.http` — allowed/denied origins, OPTIONS 204/405, Vary: Origin, Max-Age 600
- `02_request_id.http` — отбраковка инъекций (CRLF, non-ASCII, >128), генерация UUIDv4 при невалидном входе
- `03_require_auth.http` — отсутствие header / Basic / lowercase `bearer` / invalid JWT

### 6. CI и инфраструктура

- **`.github/workflows/ci.yml`**: триггеры push/PR на `main`, Go 1.25, шаги: `go test ./... -race -count=1` + `golangci-lint v2.12.1`. Timeout 10 мин. **Build-tag `integration` не передаётся** — все Postgres-тесты пропускаются.
- **`Makefile:35-36`**: `make test` = `go test ./... -race -count=1`.
- **docker-compose.yml**: единый PostgreSQL 16-alpine (без отдельного test-сервиса), креды `rupor/rupor`, healthcheck `pg_isready`.
- **Env**: `DATABASE_URL` для рантайма, `TEST_DATABASE_URL` ожидается для integration-тестов (нигде в Makefile/CI не задаётся).

## Ссылки на код

### Usecase tests
- `internal/auth/usecase/fakes_test.go:1-269` — все фейки
- `internal/auth/usecase/register_user_test.go:1-152`
- `internal/auth/usecase/login_user_test.go:1-188`
- `internal/auth/usecase/refresh_access_test.go:1-191`
- `internal/auth/usecase/get_current_user_test.go:1-53`
- `internal/auth/usecase/ports.go:12-44` — интерфейсы для моков

### HTTP transport tests
- `internal/auth/transport/http/setup_test.go:173-216` — общий стенд
- `internal/auth/transport/http/register_handler_test.go:37-148`
- `internal/auth/transport/http/login_handler_test.go:20-118`
- `internal/auth/transport/http/refresh_handler_test.go:36-167`
- `internal/auth/transport/http/me_handler_test.go:30-144`
- `internal/auth/transport/http/error_mapper.go:17-55`
- `internal/auth/transport/http/middleware/auth_test.go:45-274`
- `internal/auth/transport/http/middleware/contextkeys_test.go:26-47`

### Domain / repositories tests
- `internal/auth/domain/*_test.go` — 13 файлов
- `internal/auth/repository/jwt/token_issuer_test.go:40-169`
- `internal/auth/repository/bcrypt/password_hasher_test.go:22-72`
- `internal/auth/repository/postgres/integration_helpers_test.go:1-35` (build-tag `integration`)
- `internal/auth/repository/postgres/user_repository_integration_test.go:18-66` (all skipped)
- `internal/auth/repository/postgres/refresh_token_repository_integration_test.go:12-30` (all skipped)
- `internal/auth/repository/postgres/mapper_test.go:69-188`

### E2E / CI / infra
- `manual_qa/1_3_auth/00_flow.http` … `99_health.http`
- `manual_qa/1_4_auth_middleware/00_smoke.http` … `03_require_auth.http`
- `.github/workflows/ci.yml`
- `Makefile:35-36`
- `docker-compose.yml`
- `arch_test.go` — проверка изоляции слоёв (parse-based импорт-чек)

## Архитектурные наблюдения

- **Подход к мокированию**: ручные фейки в `*_test.go` файлах. `mockgen` / `testify/mock` не используются. Фейки внутрипакетные (`fakes_test.go`), не экспортируются.
- **Слойность тестов**: domain → unit pure; usecase → unit с фейками портов; transport → integration внутри пакета (реальный chi + настоящие usecase + fake-репо, без сети); repository/postgres → integration отдельным build-tag, **сейчас skipped**; pkg-уровень → unit для middleware.
- **Naming**: файлы `<feature>_test.go`, функции `Test<Component>_<Case>` или `Test<Component>_<Case>_<SubCase>`.
- **Assertion-стиль**: `errors.Is` / `errors.As` для доменных ошибок, `t.Fatalf("got %q, want %q", ...)` для значений. Без assertion-библиотек.
- **Детерминизм**: `fixedClock` / `seqUUID` / `seqRand` устраняют недетерминизм времени и случайностей. Время в HTTP-тестах: `2026-05-10 12:00:00 UTC`.
- **Tooling-зависимости**: только стандартный `testing`, `golang-jwt/v5`, `pgx/v5`, `bcrypt`, `uuid`. Внешнего тест-стека нет.
- **Незакрытые направления (факты, без оценки)**:
  - Все 9 Postgres integration-тестов помечены `t.Skip` — фактически не выполняются.
  - Build-tag `integration` не передаётся ни в CI, ни в Makefile (`make test` без `-tags=integration`).
  - Нет автоматизированного e2e-runnera для `.http` файлов — они исполняются вручную из IDE (VS Code REST Client / JetBrains HTTP Client).
  - В чек-листе тикета 1.5: "Unit-тесты usecase auth" — уже сделано (22 теста); "Интеграционные тесты HTTP-хендлеров" — уже сделано (6 файлов); "e2e через http-файл" — уже сделано (10 файлов).
