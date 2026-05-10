---
date: 2026-05-07
researcher: Claude
commit: 6ffa4f9
branch: feature/PR-1_3-auth_domen
research_question: "Текущее состояние кодовой базы под задачу 1.3 «Домен auth» (general_plan.md:105-112): что уже есть как фундамент, какие стандарты и решения зафиксированы, где находятся точки входа для добавления слоёв domain/usecase/transport/repository."
---

# Исследование: Baseline под задачу 1.3 «Домен auth»

## Резюме

Задача 1.3 «Домен auth» (`.claude/plans/general_plan.md:105-112`) на момент коммита `6ffa4f9` ветки `feature/PR-1_3-auth_domen` **не начата в коде**: все шесть подпапок `internal/auth/{domain,usecase,transport/http,repository/postgres}/` содержат только `.gitkeep` (`internal/auth/domain/.gitkeep`, `internal/auth/usecase/.gitkeep`, `internal/auth/transport/http/.gitkeep`, `internal/auth/repository/postgres/.gitkeep`). Go-кода в `internal/` нет ни в одном домене.

Что уже сделано как фундамент под 1.3:

1. **Инфраструктура (фаза 1.1, `general_plan.md:89-97`)** — `cmd/server/main.go` поднимает chi-роутер с одним эндпоинтом `GET /api/v1/health`, `config/` грузит `JWT_SECRET`, `DATABASE_URL`, `SERVER_PORT` через интерфейс `Lookuper`, `Makefile` даёт команды `run/test/lint/migrate-up/sqlc`, CI поднят, `golangci-lint` v2 настроен на минимальный набор линтеров.
2. **Схема БД (фаза 1.2, `general_plan.md:99-103`)** — миграции `0001_init` (расширение `pgcrypto`), `0002_users` (таблица `users` + расширение `citext`), `0003_refresh_tokens` (таблица `refresh_tokens` с FK на `users` и индексом по `user_id`). Имена ограничений зафиксированы как контракт для будущего маппинга unique-violation в доменные ошибки.
3. **Архитектурные стандарты** — восемь промптов в `prompts/` (Architecture Layers, Clean architecture, Domain Model, Builder, RepoModel, Go style, Tests Style, Domain model test) задают жёсткие правила слоистой архитектуры, направления зависимостей, стиля и тестирования. Терминология фиксирована (`usecase/`, не `service/`; `UserRepository`, `PasswordHasher`, `TokenIssuer`, `Clock`, `UUIDGenerator`).
4. **Дизайн-документы по схеме БД** — `docs/1_2_db_schema/users_table/` и `docs/1_2_db_schema/token_and_index/` фиксируют решения, которые **уже частично определяют контракт домена auth 1.3**: имена ограничений, плановый маппинг полей, плановые сигнатуры sqlc-запросов.

`go.mod` содержит ровно одну прикладную зависимость — `github.com/go-chi/chi/v5 v5.2.1`. `sqlc.yaml` пуст (`sql: []`). `sqlc generate` ничего не сгенерирует, пока в 1.3 не добавятся блоки и SQL-файлы. Любое расширение `go.mod` (bcrypt, JWT-библиотека, pgx) требует согласования по `prompts/Go style.txt:111-117`.

## Детальные результаты

### 1. Текущее состояние `internal/`

- **Расположение**: `internal/<6 доменов>/{domain,usecase,transport/http,repository/postgres}/.gitkeep`
- **Описание**: 24 пустые папки, созданные фазой 1.1 (`docs/1_1_project-structure/03-decisions.md:12`, ADR-001). Раскладка «домен → слой» зафиксирована физически — отклоняться от неё нельзя.
- **Наполнение в `internal/auth/`**: ноль `.go`-файлов. Все четыре слоя ждут реализации в задаче 1.3:
  - `internal/auth/domain/` — будет содержать сущности (`User`, опционально `RefreshToken`), value objects (`UserID`, `Email`, `Username`, `Password`/`PasswordHash`, `RefreshTokenID`, `TokenHash`), доменные ошибки (`ErrInvalidEmail`, `ErrInvalidUsername`, `ErrEmailAlreadyTaken`, …).
  - `internal/auth/usecase/` — будет содержать сценарии `Register` / `Login` / `Refresh` (`general_plan.md:107`) + интерфейсы зависимостей (`UserRepository`, `RefreshTokenRepository`, `PasswordHasher`, `TokenIssuer`, `Clock`, `UUIDGenerator`).
  - `internal/auth/repository/postgres/` — будет содержать sqlc-запросы и реализацию интерфейсов из `usecase/`.
  - `internal/auth/transport/http/` — будет содержать хендлеры `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `GET /auth/me` + DTO (`general_plan.md:111`).

### 2. Composition root: `cmd/server/main.go`

- **Расположение**: `cmd/server/main.go:1-91`, `cmd/server/health.go:1-12`
- **Описание**: единственная точка входа (`docs/1_1_project-structure/03-decisions.md:13`, ADR-002). Загружает конфиг, поднимает `chi.NewRouter`, регистрирует один маршрут `r.Get("/api/v1/health", healthHandler)` (`cmd/server/main.go:50-52`), запускает `http.Server` на порту из конфига, ждёт `SIGINT/SIGTERM`, делает graceful shutdown с таймаутом 5 секунд (`cmd/server/main.go:19, 82-87`).
- **Логирование**: `slog.New(slog.NewJSONHandler(os.Stdout, ...))` с уровнем `INFO`, установлен как default (`cmd/server/main.go:22-25`). По `prompts/Go style.txt:91` — единая библиотека логирования.
- **Тест**: `cmd/server/health_test.go` — `httptest.NewRecorder` + прямой вызов `healthHandler` (`docs/1_1_project-structure/04-testing.md:48-50`).
- **Зависимости (текущий импорт)**: `context`, `errors`, `log/slog`, `net/http`, `os`, `os/signal`, `strconv`, `syscall`, `time`, `github.com/go-chi/chi/v5`, `github.com/dovgalb/project-rupor/config` (`cmd/server/main.go:3-17`).
- **Точка расширения для 1.3**: задача 1.3 явно требует «Подключение роутов в `cmd/server/main.go`» (`general_plan.md:112`). Это — место, где будут собираться репозитории (через `pgxpool` или другой пул), хешер, токен-эмиттер и регистрироваться роуты `/api/v1/auth/*` под уже существующим префиксом `r.Route("/api/v1", ...)` (`cmd/server/main.go:50`). Префикс `/api/v1/auth/*` зарезервирован в `docs/1_1_project-structure/08-api-contract.md:77`.

### 3. Конфигурация: `config/`

- **Расположение**: `config/config.go:1-103`, `config/config_test.go:1-147`
- **Описание**: `Config` со структурой `serverPort int`, `databaseURL string`, `jwtSecret string` (приватные поля + геттеры `ServerPort()`, `DatabaseURL()`, `JWTSecret()` — `config/config.go:28-36`). `Load(Lookuper) (*Config, error)` валидирует обязательность `JWT_SECRET` (`CONFIG-001`), `DATABASE_URL` (`CONFIG-002`), парсит `SERVER_PORT` (default 8080, диапазон 1..65535, `CONFIG-003` — `config/config.go:48-103`).
- **Что уже есть для 1.3**: `cfg.JWTSecret()` готов к использованию в JWT-эмиттере. `cfg.DatabaseURL()` — для подключения пула БД (драйвер БД пока не выбран в `go.mod`, см. п. 9).
- **Чего пока нет**: TTL access/refresh токенов как параметры конфига. По `docs/1_2_db_schema/token_and_index/03-decisions.md:22` (решение 11) и open question там же — это решение откладывается на 1.3, при проектировании `TokenIssuer`. Если выбираем env-параметры — добавляются в `config/config.go` рядом с `JWT_SECRET` по тому же паттерну.
- **`Lookuper` интерфейс**: тестируемость без `os.Setenv` (`config/config.go:38-46`, `prompts/Tests Style.txt:204-211`). `OsLookuper{}` — продовая реализация.

### 4. Схема БД: миграции 0001, 0002, 0003

- **Расположение**: `migrations/0001_init.up.sql:1-4`, `migrations/0002_users.up.sql:1-13`, `migrations/0003_refresh_tokens.up.sql:1-14`, парные `*.down.sql`.
- **`0001_init`**: `CREATE EXTENSION IF NOT EXISTS pgcrypto;` (для `gen_random_uuid()`) — `migrations/0001_init.up.sql:4`.
- **`0002_users`**: расширение `citext` + таблица `users`:
  - `id uuid PRIMARY KEY DEFAULT gen_random_uuid()`
  - `email citext NOT NULL UNIQUE` (имя ограничения — `users_email_key`, по дефолту Postgres)
  - `password_hash text NOT NULL`
  - `username text NOT NULL UNIQUE` (имя — `users_username_key`)
  - `created_at timestamptz NOT NULL DEFAULT now()`
  - `CONSTRAINT users_username_length_check CHECK (char_length(username) BETWEEN 3 AND 32)`
  - Источник: `migrations/0002_users.up.sql:6-13`.
- **`0003_refresh_tokens`**: таблица `refresh_tokens` + индекс по `user_id`:
  - `id uuid PRIMARY KEY DEFAULT gen_random_uuid()`
  - `user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE`
  - `token_hash bytea NOT NULL UNIQUE` (имя — `refresh_tokens_token_hash_key`)
  - `expires_at timestamptz NOT NULL`
  - `created_at timestamptz NOT NULL DEFAULT now()`
  - `revoked_at timestamptz NULL`
  - `CONSTRAINT refresh_tokens_token_hash_length_check CHECK (octet_length(token_hash) = 32)` — sha256 = 32 байта
  - `CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens(user_id);`
  - Источник: `migrations/0003_refresh_tokens.up.sql:4-14`.

### 5. ADR-фиксации схемы БД (контракты для домена auth 1.3)

- **Расположение**: `docs/1_2_db_schema/users_table/03-decisions.md:8-23`, `docs/1_2_db_schema/users_table/06-repo-model.md:62-68`, `docs/1_2_db_schema/token_and_index/03-decisions.md:8-27`, `docs/1_2_db_schema/token_and_index/06-repo-model.md:66-75`.
- **Контракт по `users`**:
  - PK — `uuid` со стороны БД, в домене — VO `domain.UserID` через `domain.NewUserID(row.ID)` ↔ `user.ID().UUID()` (`docs/1_2_db_schema/users_table/06-repo-model.md:64`).
  - `email citext` ↔ `domain.Email` через `string(row.Email)` → `domain.NewEmail(...)` (`docs/1_2_db_schema/users_table/06-repo-model.md:65`).
  - `username text` ↔ `domain.Username` через `domain.NewUsername(row.Username)` (`docs/1_2_db_schema/users_table/06-repo-model.md:67`).
  - `password_hash text` — домен принимает уже захешированное значение от `PasswordHasher` (`docs/1_2_db_schema/users_table/06-repo-model.md:66`). Сам алгоритм (bcrypt) — техническая деталь, не закрепляется в БД (`docs/1_2_db_schema/users_table/03-decisions.md:15`, решение 4).
  - `created_at` — UTC, прямое значение `time.Time` (`docs/1_2_db_schema/users_table/06-repo-model.md:68`).
  - **Имя `users_email_key`** — критично: маппинг unique-violation на `domain.ErrEmailAlreadyTaken` (`prompts/RepoModel.txt:138-140`, `docs/1_2_db_schema/users_table/03-decisions.md:17`, решение 6).
  - **Полей `status`, `deleted_at`, `email_verified_at` НЕТ** — сознательно отказано (`docs/1_2_db_schema/users_table/03-decisions.md:19-20`, решения 8–9 и Open Questions).
  - Регистрозависимость `username` — БД-уровневая (регистрозависимая); если домен потребует case-insensitive — задача VO `Username` (нормализация перед сохранением, `docs/1_2_db_schema/users_table/03-decisions.md:38`).
- **Контракт по `refresh_tokens`**:
  - В БД хранится **только sha256-хеш** raw-токена, не сам токен (`docs/1_2_db_schema/token_and_index/03-decisions.md:13`, решение 2). Тип — `bytea`, длина — 32 байта (`docs/1_2_db_schema/token_and_index/03-decisions.md:14`, решение 3).
  - Плановое поле домена `auth.RefreshToken` — `tokenHash domain.TokenHash` (VO над `[32]byte`), маппинг `[32]byte(row.TokenHash)` ↔ `token.Hash()[:]` (`docs/1_2_db_schema/token_and_index/06-repo-model.md:70`).
  - Состояние «активен/отозван» — через nullable `revoked_at`. Запрос активных: `WHERE revoked_at IS NULL AND expires_at > now()` (`docs/1_2_db_schema/token_and_index/03-decisions.md:20`, решение 9).
  - **Ротация при refresh** — выбранная стратегия (`docs/1_2_db_schema/token_and_index/03-decisions.md:13`, решение 2). При каждом refresh старый токен помечается `revoked_at = now()`, выпускается новый.
  - Каскадное удаление токенов при удалении пользователя — `ON DELETE CASCADE` (`docs/1_2_db_schema/token_and_index/03-decisions.md:16`, решение 5).
  - **Имя `refresh_tokens_token_hash_key`** — контракт для маппинга unique-violation в задаче 1.3 (`docs/1_2_db_schema/token_and_index/03-decisions.md:18`, решение 7).
  - Поля аудита (`device_id`, `user_agent`, `ip`), TTL и лимит активных сессий — **не** в MVP (`docs/1_2_db_schema/token_and_index/03-decisions.md:24`, решение 13; Open Questions внизу того же файла).

### 6. Плановые sqlc-запросы (заявлены в ADR, не реализованы)

- **Расположение**: `docs/1_2_db_schema/token_and_index/06-repo-model.md:81-101` — для `refresh_tokens`. Аналог для `users` в `docs/1_2_db_schema/users_table/06-repo-model.md:81-83` явно отложен на 1.3 без сигнатур.
- **Заявленные запросы по `refresh_tokens`** (намечены, не написаны):
  - `InsertRefreshToken(:exec)` — `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3);`
  - `GetRefreshTokenByHash(:one)` — `SELECT id, user_id, token_hash, expires_at, created_at, revoked_at FROM refresh_tokens WHERE token_hash = $1;`
  - `RevokeRefreshTokenByHash(:exec)` — `UPDATE … SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL;`
  - `RevokeAllRefreshTokensByUser(:exec)` — `UPDATE … SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL;`
- **Заявленные запросы по `users`** — упомянуты в декларациях (`InsertUser`, `GetUserByID`, `GetUserByEmail`) в `docs/1_2_db_schema/users_table/06-repo-model.md:83`, но без точных сигнатур.
- **Состояние `sqlc.yaml`**: `version: "2"; sql: []` (`sqlc.yaml:1-2`). Конфиг генератора пуст; задача 1.3 явно включает его настройку (`docs/1_2_db_schema/token_and_index/03-decisions.md:27`, решение 16; `docs/1_2_db_schema/users_table/03-decisions.md:23`, решение 12).

### 7. Архитектурные правила слоёв

- **Источник**: `prompts/Architecture Layers.txt:14-66`, `prompts/Clean architecture.txt:18-38`.
- **Жёсткие правила импорта** (`prompts/Architecture Layers.txt:67-74`):
  - `domain/` — только stdlib. Никаких сторонних библиотек, JSON-тегов, тегов БД.
  - `usecase/` — только `domain/` + stdlib. Объявляет интерфейсы зависимостей, не знает про HTTP/SQL/JSON.
  - `transport/...` и `repository/...` импортируют `usecase/` и `domain/`, но не наоборот.
  - `cmd/server/` собирает граф зависимостей.
  - Запрещено: одна доменная папка импортирует `transport/` или `repository/` другой доменной папки.
- **Терминология (фиксирована)** (`prompts/Architecture Layers.txt:118-123`):
  - Слой сценариев — `usecase/` (не `service/`).
  - Интерфейсы по роли: `UserRepository`, `PasswordHasher`, `TokenIssuer`, `Clock`, `UUIDGenerator`.
  - DTO use case'ов — `RegisterUserInput` / `RegisterUserOutput`. Транспортные DTO — `RegisterUserRequest` / `RegisterUserResponse`.
- **Поток регистрации (заранее описан)** (`prompts/Architecture Layers.txt:75-83`):
  1. HTTP-запрос → `internal/auth/transport/http/register_handler.go`
  2. Хендлер парсит JSON → DTO use case'а
  3. Use case вызывает `domain.NewUser(...)`, который проверяет инварианты
  4. Use case вызывает `UserRepository.Save(ctx, user)` (интерфейс в `usecase/`)
  5. Реализация в `internal/auth/repository/postgres/` транслирует в SQL через sqlc
  6. Use case возвращает output → хендлер сериализует JSON.
- **Что считается бизнес-правилом, а что — нет** (`prompts/Clean architecture.txt:42-54`): bcrypt и HS256 явно отнесены к **деталям**, живут в адаптерах. Доменное правило — «пользователь не может зарегистрироваться с пустым именем».

### 8. Стандарты доменной модели и репозитория

- **Domain Model** (`prompts/Domain Model.txt`):
  - Rich-модель: приватные поля, конструкторы с инвариантами, методы бизнес-операций, не публичные сеттеры (`prompts/Domain Model.txt:32-50, 80-99`).
  - Value Objects: иммутабельные, валидируются в конструкторе (`prompts/Domain Model.txt:55-77`). Пример `Email` — `NewEmail(raw string)` с trim+lowercase+regex.
  - Два конструктора: `NewUser(...)` для новой сущности, `ReconstructUser(...)` для восстановления из БД (`prompts/Domain Model.txt:124-129`).
  - Связи между сущностями — через id, не вложенные объекты (`prompts/Domain Model.txt:152-156`). `RefreshToken` будет ссылаться на `UserID`, а не на `*User`.
  - Доменные ошибки в `domain/errors.go`: sentinel через `errors.New` или типизированные (`prompts/Domain Model.txt:131-150`).
- **Repo Model** (`prompts/RepoModel.txt`):
  - Маппинг через доменные конструкторы (`Reconstruct*`), не через прямой доступ к приватным полям (`prompts/RepoModel.txt:79-81`).
  - Две функции на сущность: `domainToRow` / `rowToDomain` (`prompts/RepoModel.txt:84-88`).
  - Value objects разворачиваются на границе репозитория (`prompts/RepoModel.txt:90-93`).
  - Маппинг ошибок БД → доменные ошибки в репозитории: `sql.ErrNoRows` → `domain.ErrXxxNotFound`; unique violation по известному constraint → доменная ошибка (`prompts/RepoModel.txt:152-159`). Проверка через `*pgconn.PgError` — если используем `pgx`.
  - **Никакого ORM**: только sqlc (`prompts/RepoModel.txt:163`).
- **Builder** (`prompts/Builder.txt`): паттерн только для тестов — `NewUserBuilder(t).WithEmail(...).Banned().Build()`. Build вызывает реальный `domain.ReconstructUser` — не обходит инварианты (`prompts/Builder.txt:53-63`). В продовом коде Builder запрещён без отдельного согласования (`prompts/Builder.txt:144-150`).

### 9. Стандарты Go и тестов

- **Go style** (`prompts/Go style.txt`):
  - Go 1.25 (см. `go.mod:3`), `gofmt`/`goimports` обязательны (`prompts/Go style.txt:7-9`), `golangci-lint` через `make lint` (`Makefile:38-39`).
  - Конфиг — только env (`prompts/Go style.txt:84-87`).
  - Логирование — `log/slog` (`prompts/Go style.txt:91`), не PII/секреты (`prompts/Go style.txt:94`).
  - SQL — только sqlc (`prompts/Go style.txt:97-101`).
  - **Запрещено без согласования**: новые зависимости в `go.mod`, своя DI, свой роутер, `reflect` в hot path, `unsafe`, `_` для игнора ошибок (`prompts/Go style.txt:111-117`).
  - Контекст первым параметром в I/O-функциях, не класть в структуры (`prompts/Go style.txt:47-52`).
- **Tests Style** (`prompts/Tests Style.txt`):
  - Стандартная библиотека `testing`, без testify/ginkgo (`prompts/Tests Style.txt:24-26`).
  - Фейки руками, не gomock (`prompts/Tests Style.txt:75-100`). Пример `fakeUserRepository` уже описан в стандарте.
  - Use case тесты — black-box (`package usecase_test`) (`prompts/Tests Style.txt:115-118`).
  - Интеграционные тесты репозиториев — против реального PostgreSQL (`prompts/Tests Style.txt:120-145`).
  - Интеграционные тесты HTTP — `httptest.NewServer` поверх собранного роутера (`prompts/Tests Style.txt:147-162`).
  - Время и UUID — инжектируются (`prompts/Tests Style.txt:55-62`).
- **Domain model tests** (`prompts/Domain model test.txt`): без моков, без БД, без HTTP. Table-driven там, где варианты входов. `t.Parallel()` везде.

### 10. Текущий `go.mod` и зависимости

- **Расположение**: `go.mod:1-5`, `go.sum:1-2`
- **Модуль**: `github.com/dovgalb/project-rupor`
- **Go**: `1.25`
- **Прикладные зависимости**: ровно одна — `github.com/go-chi/chi/v5 v5.2.1`
- **Чего нет в `go.mod` и потребуется решать в 1.3**:
  - **Драйвер БД** — `pgx` или `database/sql` + `lib/pq`. Не выбран. Нужно согласование с пользователем.
  - **bcrypt** — `golang.org/x/crypto/bcrypt`. Указан в задаче явно (`general_plan.md:108`), но добавление — новая зависимость.
  - **JWT-библиотека** — `general_plan.md:109` требует «Генерация и валидация JWT (access + refresh)», конкретная библиотека не выбрана. Кандидаты: `github.com/golang-jwt/jwt/v5`, `github.com/lestrrat-go/jwx`. **Любая — новая зависимость, требует согласования** (`prompts/Go style.txt:111-113`).
  - **UUID** — `prompts/Domain Model.txt:25` упоминает `github.com/google/uuid` «если согласовано». В текущем `go.mod` его нет. На уровне БД UUID генерируется `gen_random_uuid()`, так что в принципе можно жить без библиотеки в домене (Reconstruct получает уже готовый `[16]byte` через драйвер).

### 11. Makefile и CI

- **Расположение**: `Makefile:1-92`, `.github/workflows/ci.yml` (не прочитан в этом исследовании, упоминается в `docs/1_1_project-structure/03-decisions.md:22`).
- **Команды для 1.3**:
  - `make migrate-up` — `migrate -path migrations -database "$DATABASE_URL" up` (`Makefile:48-49`). Требует локально установленного `golang-migrate` CLI (`docs/1_1_project-structure/03-decisions.md:21`, ADR-010).
  - `make sqlc` — `sqlc generate` (`Makefile:54-55`). Сейчас генерирует ноль файлов из-за пустого `sql: []`.
  - `make test` — `go test ./... -race -count=1` (`Makefile:35-36`).
  - `make lint` — `golangci-lint run` (`Makefile:38-39`).
  - `make run` — `go run ./cmd/server` (`Makefile:29-30`).
- **Линтеры включены**: `govet` (с `enable-all`, выключен только `fieldalignment`), `staticcheck`, `errcheck`, `ineffassign`, `unused`, форматтеры `gofmt`, `goimports` с `local-prefixes: github.com/dovgalb/project-rupor` (`.golangci.yml:7-29`).
- **`init_project`**: `Makefile:71-92` — полная инициализация окружения (env, deps, hooks, db, migrations).

### 12. Контракт `/api/v1/*` и резерв префикса

- **Расположение**: `cmd/server/main.go:50-52`, `docs/1_1_project-structure/08-api-contract.md:74-82`.
- **Сейчас**: один маршрут `r.Get("/api/v1/health", healthHandler)` под группой `r.Route("/api/v1", ...)`.
- **Резерв для 1.3**: префикс `/api/v1/auth/*` явно зафиксирован в `docs/1_1_project-structure/08-api-contract.md:77`. Маршруты задачи 1.3 (`general_plan.md:111`):
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `GET  /api/v1/auth/me` — требует JWT-middleware (`general_plan.md:115`), но middleware — задача **1.4**, отдельная фаза.
- **Авторизация**: формат — `Authorization: Bearer <jwt>` (`CLAUDE.md` проекта, раздел «API»).

### 13. Существующие исследовательские документы

- **Расположение**: `.thoughts/research/`
  - `2026-05-04-project-structure.md` — research под фазу 1.1
  - `2026-05-05-users-table-migration.md` — research под `0002_users`
  - `2026-05-07-tokens-and-index.md` — research под `0003_refresh_tokens`
- В этих документах детальный анализ предыдущих фаз; для 1.3 они дают исходную точку, но не пересекаются по scope.

## Ссылки на код

- `cmd/server/main.go:21-46` — точка входа: загрузка конфига, обработка ошибок, переход в `run`.
- `cmd/server/main.go:48-91` — `run`: chi-роутер, http.Server, graceful shutdown с `signal.NotifyContext` и `srv.Shutdown` с таймаутом 5с.
- `cmd/server/main.go:50-52` — место регистрации будущих `/api/v1/auth/*` маршрутов.
- `cmd/server/health.go:7-11` — текущий health-handler (прообраз стиля для будущих хендлеров: явный `Content-Type`, явный `WriteHeader`, ручная сериализация без `encoding/json`).
- `config/config.go:28-36` — структура `Config` и геттеры (используем `cfg.JWTSecret()` и `cfg.DatabaseURL()` в 1.3).
- `config/config.go:48-77` — `Load(Lookuper)` и валидация обязательных переменных.
- `config/config_test.go:10-15` — `mapLookuper` как пример фейка для тестов.
- `internal/auth/domain/.gitkeep` — пустая папка для сущностей и VO.
- `internal/auth/usecase/.gitkeep` — пустая папка для сценариев и интерфейсов.
- `internal/auth/repository/postgres/.gitkeep` — пустая папка для sqlc-обёртки.
- `internal/auth/transport/http/.gitkeep` — пустая папка для хендлеров.
- `migrations/0002_users.up.sql:6-13` — DDL `users`.
- `migrations/0003_refresh_tokens.up.sql:4-14` — DDL `refresh_tokens` + индекс по `user_id`.
- `sqlc.yaml:1-2` — пустая конфигурация генератора (требует наполнения в 1.3).
- `go.mod:1-5` — модуль и единственная зависимость `chi/v5`.
- `Makefile:48-55` — таргеты `migrate-up`, `migrate-down`, `sqlc`.
- `.golangci.yml:7-19` — состав линтеров.
- `prompts/Architecture Layers.txt:14-66` — описание четырёх слоёв.
- `prompts/Architecture Layers.txt:118-123` — фиксация терминологии (`usecase/`, `UserRepository`, `PasswordHasher`, `TokenIssuer`, …).
- `prompts/Domain Model.txt:124-129` — паттерн «два конструктора» (`New*` / `Reconstruct*`).
- `prompts/RepoModel.txt:138-144` — пример маппинга unique-violation в `domain.ErrEmailAlreadyTaken` и опора на имя `users_email_key`.
- `prompts/Go style.txt:111-117` — список запрещённого без согласования (включая новые зависимости).
- `docs/1_2_db_schema/users_table/03-decisions.md:14-22` — фиксированные решения по схеме `users` (типы, имена, отказ от `status`/`deleted_at`).
- `docs/1_2_db_schema/users_table/06-repo-model.md:62-68` — плановый маппинг `users` → `domain.User`.
- `docs/1_2_db_schema/token_and_index/03-decisions.md:13-25` — решения по `refresh_tokens` (хеш+ротация, FK с CASCADE, отказ от полей аудита).
- `docs/1_2_db_schema/token_and_index/06-repo-model.md:66-75` — плановый маппинг `refresh_tokens` → `domain.RefreshToken`.
- `docs/1_2_db_schema/token_and_index/06-repo-model.md:81-101` — плановые сигнатуры sqlc-запросов для refresh-токенов.
- `docs/1_1_project-structure/08-api-contract.md:74-82` — резерв префиксов `/api/v1/auth/*`, `/api/v1/rooms/*` и т.д.
- `.claude/plans/general_plan.md:105-112` — определение задачи 1.3.

## Архитектурные наблюдения

- **Используемый паттерн**: Clean Architecture в Go-адаптации с раскладкой «домен → слой» (`internal/<домен>/{domain,usecase,transport/http,repository/postgres}/`). Ровно эта структура повторена для всех шести доменов через `.gitkeep`-плейсхолдеры — отступать от неё нельзя.
- **Граф зависимостей слоёв** (`prompts/Architecture Layers.txt:67-74`):
  - `cmd/server` → импортирует `transport/http`, `repository/postgres`, `usecase/`, `pkg/`, `config/`.
  - `transport/http` → `usecase/`, `domain/`.
  - `repository/postgres` → `usecase/` (реализует интерфейсы), `domain/` (доменные ошибки/типы).
  - `usecase/` → `domain/`.
  - `domain/` → только stdlib.
- **Поток данных регистрации (плановый, по 1.3)**:
  1. `POST /api/v1/auth/register` → `internal/auth/transport/http/register_handler.go`
  2. парсинг JSON → `RegisterUserRequest` → маппинг в `RegisterUserInput`
  3. `RegisterUser.Execute(ctx, input)` в `internal/auth/usecase/`
  4. внутри: `domain.NewEmail`, `domain.NewUsername`, `PasswordHasher.Hash(password)`, `domain.NewUser(...)`
  5. `UserRepository.Save(ctx, user)` → реализация в `internal/auth/repository/postgres/` через sqlc
  6. ответ `201 Created` с `RegisterUserResponse` (id/email/username/createdAt).
- **Поток данных login**: тот же путь, плюс `RefreshTokenRepository.Save(ctx, *RefreshToken)` (хеш в БД, raw token — клиенту), плюс `TokenIssuer.IssueAccess(userID, now)` и `TokenIssuer.IssueRefresh(userID, now)`.
- **Поток данных refresh** (с ротацией): найти запись по `sha256(rawRefresh)` → проверить `revoked_at IS NULL AND expires_at > now()` → выпустить новые токены → пометить старый `revoked_at = now()` (атомарно в одной транзакции, либо `RevokeRefreshTokenByHash` + `InsertRefreshToken`).
- **Ключевые контракты с уже принятыми решениями**:
  - Имя `users_email_key` — для маппинга unique-violation в `domain.ErrEmailAlreadyTaken`.
  - Имя `users_username_key` — для маппинга в `domain.ErrUsernameAlreadyTaken`.
  - Имя `refresh_tokens_token_hash_key` — для маппинга unique-violation refresh-токенов.
  - Имя `refresh_tokens_user_id_idx` — индекс под запросы logout-all/list-active.
  - Длина `token_hash` = 32 байта (sha256). VO `domain.TokenHash` должен быть над `[32]byte`.
  - `password_hash` — `text` без длины: алгоритм может смениться. На уровне 1.3 — bcrypt (`general_plan.md:108`).
  - **`status`/`deleted_at`/`email_verified_at` в `users` отсутствуют** — `domain.User` 1.3 строится без полей жизненного цикла (только `id`, `email`, `username`, `passwordHash`, `createdAt`).
  - **TTL access/refresh** — открытый вопрос задачи 1.3 (`docs/1_2_db_schema/token_and_index/03-decisions.md`, нижний open question). Решается при проектировании `TokenIssuer`.
- **Открытые точки выбора в 1.3 (требуют согласования с пользователем)**:
  - Драйвер БД (`pgx` vs `database/sql`+`lib/pq`).
  - JWT-библиотека (`golang-jwt` vs `lestrrat-go/jwx` vs ручная реализация на stdlib `crypto/hmac`).
  - Использовать `github.com/google/uuid` или нет (либо domain работает с `[16]byte`/string).
  - Стратегия refresh: ротация (выбрана в ADR схемы) — закрепляется в `usecase.Refresh` атомарным reissue.
  - TTL access (стандартно 15 мин) и TTL refresh (стандартно 7–30 дней) как параметры конфига или константы.
  - Один транзакционный шаблон (UnitOfWork в use case или `WithTx` на репозитории — `prompts/RepoModel.txt:147-150` оставляет это решение на этап проектирования).
  - Формат password validation (минимальная длина, политика символов) — это VO `Password`, и решение должно быть зафиксировано в `domain/`.
  - Нормализация `Email` (trim+lowercase в VO) и `Username` (regex/нижний регистр).
  - Тип `domain.User.passwordHash` — string, или VO `PasswordHash` с проверкой формата bcrypt.
- **Зависимости миграций**: `0003 → требует 0002 → требует 0001`. Расширение `citext` подключено в `0002`, не используется в `0003`. Расширение `pgcrypto` подключено в `0001`, используется обоими (`gen_random_uuid()`).
