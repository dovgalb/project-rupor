---
parent: ./README.md
view: standards
---

# 07 — Standards Compliance

Соответствие дизайна каждому из восьми стандартов в `prompts/`. Статус ✅ — стандарт применим и соблюдается; ⚠️ — есть оговорка/расхождение, описанное в «Уточнения».

| Стандарт | Статус | Ключевые точки compliance |
|----------|--------|---------------------------|
| `Architecture Layers.txt` | ✅ | Раскладка `internal/auth/{domain,usecase,transport/http,repository/{postgres,jwt,bcrypt}}/` (`01-architecture.md`). Терминология: `usecase/` (`prompts/Architecture Layers.txt:120`), интерфейсы по роли — `UserRepository`, `RefreshTokenRepository`, `PasswordHasher`, `TokenIssuer`, `Clock`, `UUIDGenerator`, `RandomBytes` (`prompts/Architecture Layers.txt:121`), DTO use case'ов — `RegisterUserInput`/`Output` (`prompts/Architecture Layers.txt:122`), транспортные DTO — `RegisterUserRequest`/`Response` (`prompts/Architecture Layers.txt:123`). Граф зависимостей в `01-architecture.md` направлен внутрь по `prompts/Architecture Layers.txt:67-74`. Composition root в `cmd/server/main.go` по `prompts/Architecture Layers.txt:62-66`. См. ⚠️ в «Уточнения». |
| `Builder.txt` | ✅ N/A | В этой фиче builder-pattern **не применяется**. Сущности `User` (5 полей) и `RefreshToken` (6 полей) имеют простые конструкторы, обходимые `must*`-хелперами в тестах (`prompts/Builder.txt:14-19` явно говорит «у структуры 1–3 поля. Проще передать их прямо в конструктор» — у нас на грани). Если в задаче 1.5 / фазе 2 появятся сущности с 8+ полями (например, `Room` с rolemap'ом) — там будет builder. Решение зафиксировано в `04-testing.md`. Отклонений нет. |
| `Clean architecture.txt` | ✅ | `cmd/server/main.go` — единственная точка склейки (`prompts/Clean architecture.txt:30-38`). Правило зависимостей внутрь — выполнено: `domain` → stdlib + uuid (`prompts/Domain Model.txt:25` явно допускает); `usecase` → `domain` + stdlib + uuid; `transport` и `repository/*` → `usecase` + `domain`. Никаких обратных импортов. JWT, bcrypt, sqlc — детали в адаптерах (`prompts/Clean architecture.txt:51-54`). |
| `Domain Model.txt` | ✅ | Rich-модель: приватные поля, конструкторы `New*` / `Reconstruct*` (`prompts/Domain Model.txt:124-129`); инварианты в конструкторах (`prompts/Domain Model.txt:32-50`); геттеры, без сеттеров (`prompts/Domain Model.txt:80-99`). VO: `Email`, `Username`, `Password`, `PasswordHash`, `UserID`, `RefreshTokenID`, `TokenHash` — иммутабельные, валидируются в конструкторе (`prompts/Domain Model.txt:55-77`). Связь между `User` и `RefreshToken` — через `UserID`, не вложенный объект (`prompts/Domain Model.txt:152-156`). Доменные ошибки в `domain/errors.go` (`prompts/Domain Model.txt:131-150`). См. ⚠️ в «Уточнения» по `Activate/Ban`. |
| `Domain model test.txt` | ✅ | Чёрный ящик `package domain_test` (`prompts/Domain model test.txt:67`); `t.Parallel()` везде (`prompts/Domain model test.txt:117-122`); table-driven для VO (`prompts/Domain model test.txt:30-62`); `errors.Is` для sentinel (`prompts/Domain model test.txt:101`); `t.Helper()` в фикстурах (`prompts/Domain model test.txt:71-91`); time/UUID инжектируется через конструкторы (`prompts/Domain model test.txt:93-98`). |
| `Go style.txt` | ✅ | Go 1.25 (`go.mod:3`). `gofmt`/`goimports` через `make lint` (`Makefile:38-43`). Конфигурация — env (`prompts/Go style.txt:84-87`, ADR-004). Логирование — `log/slog` (`prompts/Go style.txt:91`); пароли/токены не логируются (`prompts/Go style.txt:94`). SQL — sqlc (`prompts/Go style.txt:97-101`). Контекст — первым параметром (`prompts/Go style.txt:47-52`). Ранние возвраты, без `else` после `return` (`prompts/Go style.txt:54-55`). См. ⚠️ в «Уточнения» по новым зависимостям. |
| `RepoModel.txt` | ✅ | См. `06-repo-model.md`, раздел «Соответствие `prompts/RepoModel.txt`» — все 12 правил отмечены. Особо: маппинг через `Reconstruct*` (`prompts/RepoModel.txt:79-81`), две функции на сущность (`prompts/RepoModel.txt:84-88`), VO разворачиваются в адаптере (`prompts/RepoModel.txt:90-93`), `pgx.ErrNoRows` → доменная ошибка (`prompts/RepoModel.txt:152-156`), unique violation по имени constraint → `ErrEmailAlreadyTaken`/`ErrUsernameAlreadyTaken` (`prompts/RepoModel.txt:138-144`). |
| `Tests Style.txt` | ✅ | Стандартная библиотека `testing` (`prompts/Tests Style.txt:24-26`) — без testify/gomock. `t.Parallel()` (`prompts/Tests Style.txt:28-30`). Table-driven (`prompts/Tests Style.txt:32-49`). Фейки руками (`prompts/Tests Style.txt:75-100`). Use case тесты — black-box `package usecase_test` (`prompts/Tests Style.txt:115-118`). HTTP-тесты — `httptest.NewServer` поверх собранного роутера (`prompts/Tests Style.txt:147-162`). Время и UUID — инжектируются (`prompts/Tests Style.txt:55-62`). AAA-структура (`prompts/Tests Style.txt:170-202`). |

## Уточнения

### Расхождение 1: новые зависимости в `go.mod`

`prompts/Go style.txt:111-117` запрещает добавлять новые зависимости без согласования. В этой фиче добавляются 4:
- `github.com/jackc/pgx/v5` — драйвер БД (ADR-001)
- `github.com/golang-jwt/jwt/v5` — JWT (ADR-002)
- `github.com/google/uuid` — UUID-обёртка (ADR-003)
- `golang.org/x/crypto/bcrypt` — хеш паролей (`general_plan.md:108` явно требует bcrypt)

Все четыре **согласованы с пользователем** (см. `03-decisions.md`, ADR-001..003 + явное упоминание в `general_plan.md`). Это не отклонение от стандарта, а корректная реализация процедуры согласования.

`golang.org/x/crypto` — официальное расширение Go, `prompts/Go style.txt` его явно не упоминает в исключениях, поэтому формально засчитывается как «новая зависимость» и проходит ту же процедуру.

### Расхождение 2: расширение раскладки `repository/`

`prompts/Architecture Layers.txt:46-58` приводит примером только `repository/postgres/`. В этой фиче появляются **три параллельных подпапки** в `repository/`: `postgres/`, `jwt/`, `bcrypt/`.

Обоснование (`03-decisions.md`, ADR-005):
- Все три — адаптеры для интерфейсов из `usecase/`, формально соответствуют слою «Interface Adapters» (`prompts/Architecture Layers.txt:46-58`).
- Альтернатива (новый слой `internal/auth/security/`) ломает структуру `domain/usecase/transport/repository/`, физически зафиксированную фазой 1.1 (`docs/1_1_project-structure/03-decisions.md:12`, ADR-001).
- Альтернатива (`pkg/jwt/`) преждевременна: нет потребителей вне auth-домена.

Это отклонение от **примера** в стандарте, но не от **правила**. Стандарт фиксирует «реализации репозиториев и адаптеров — в `internal/<домен>/repository/...`», и три подпапки именно туда и кладутся.

### Расхождение 3: `domain.User` без бизнес-методов

`prompts/Domain Model.txt:24-50` ожидает rich-модель с бизнес-методами. В нашей `User` бизнес-методов **нет**: только конструкторы и геттеры. Это связано со scope MVP:
- В схеме нет полей жизненного цикла (`status`, `deleted_at`, `email_verified_at`) — `docs/1_2_db_schema/users_table/03-decisions.md`, решения 8–9.
- Нет операций `Activate`, `Ban`, `Verify` — они появятся, когда соответствующие поля будут добавлены отдельной миграцией.

Сейчас `User` — структура с инвариантами на создание, но без операций жизненного цикла. Это **корректно** по стандарту: «Если нет бизнес-операций — методов нет; добавятся, когда появятся правила». Стандарт явно не требует «обязательно иметь бизнес-метод» (`prompts/Domain Model.txt:107-113`).

`RefreshToken`, наоборот, имеет бизнес-методы (`Revoke`, `IsActive`, `IsExpired`, `IsRevoked`) — это rich-модель в полном смысле.

### Расхождение 4: inline-проверка JWT в `MeHandler` без middleware

`prompts/Architecture Layers.txt:50-58` относит middleware к `transport/http/`. В фазе 1.4 он там и появится. В фазе 1.3 `MeHandler` сам извлекает `Authorization: Bearer` и вызывает `TokenIssuer.VerifyAccess` — это inline-логика, которая в 1.4 переедет в middleware (`03-decisions.md`, ADR-009).

Это временное отклонение от паттерна «middleware → handler → use case» — на 1 фазу. Контракт `TokenIssuer.VerifyAccess` стабилизирован в 1.3 и не изменится в 1.4.

### Расхождение 5: интеграционные тесты репозиториев — отложены или с `t.Skip`

`prompts/Tests Style.txt:120-145` требует интеграционных тестов репозиториев против реального PostgreSQL. В фазе 1.3 эти тесты могут быть:
- Полностью реализованы (если CI-среда позволяет — Docker postgres-сервис в `.github/workflows/ci.yml`).
- Помечены `t.Skip("integration: requires Postgres")` или build tag `//go:build integration` с активацией в задаче 1.5 (`general_plan.md:120-123`: «Интеграционные тесты HTTP-хендлеров auth», «e2e через curl/http-файл»).

**Решение:** интеграционные тесты пишутся как `//go:build integration`, чтобы `make test` оставался быстрым (≤2с) и не зависел от Postgres. Активация и подключение в CI — задача 1.5. Unit-тесты (с фейками) пишутся в фазе 1.3 без оговорок и обеспечивают coverage для всех `AUTH-*` кодов через usecase/handler/jwt/bcrypt-уровни.

Это **прагматичное отклонение** от стандарта на одну фазу, согласованное с декомпозицией плана.

### Расхождение 6: dummy-bcrypt при `ErrUserNotFound` в Login

`prompts/Domain Model.txt:171-173` и `Tests Style.txt:170-202` (AAA-структура) не описывают сценарий «выполнить бесполезную операцию ради таймингового выравнивания». В `LoginUser.Execute` мы вызываем `hasher.Verify` даже если пользователь не найден (`02-behavior.md`, edge case в Use Case 2; `03-decisions.md`, ADR-008).

Это техническая защита от user-enumeration (OWASP ASVS V2.1.1). Не противоречит стандартам — это просто специфическое use case-правило, документированное в дизайн-документах.

## Pre-flight чек перед коммитом (по `prompts/Go style.txt:119-126`)

Реализация фазы 1.3 должна проходить эти команды без правок:

1. `gofmt -l .` — пустой вывод
2. `go vet ./...` — exit 0
3. `golangci-lint run` — exit 0, 0 issues
4. `go test ./... -race -count=1` — все unit-тесты зелёные (~80 тестов, см. `04-testing.md`)
5. `go build ./...` — exit 0
6. `make sqlc` — генерация без ошибок, сгенерированный `internal/auth/repository/postgres/db/` без диффа после `gofmt`
7. `git diff --check` — нет whitespace-ошибок

CI workflow (`.github/workflows/ci.yml`) выполняет шаги 1–5. Шаг 6 (`make sqlc`) — добавится в CI или станет частью pre-commit hook (отдельная фича). Шаг 7 — локально перед коммитом.

## Список новых правил, появляющихся с 1.3 (для ревью на этапе 2 и далее)

- **Все авторизованные хендлеры используют `TokenIssuer.VerifyAccess`** — фаза 1.4 завернёт это в middleware, но контракт зафиксирован сейчас.
- **Refresh-токен — это `crypto/rand.Read(32)` + `base64url`** (не JWT) — `03-decisions.md`, ADR-011. Любая будущая фича, нуждающаяся в server-side revokable токене, использует этот же паттерн.
- **Адаптеры лежат в `repository/`** — не в новых слоях `security/`/`infrastructure/`. Если в фазе 2/3 появится email-sender, file-storage и т.п. — они тоже идут в `internal/<domain>/repository/`.
- **Sql-запросы в `internal/<domain>/repository/postgres/queries/`** — настроено в `sqlc.yaml`. Если в фазе 2 появится `internal/room/`, у sqlc.yaml будет второй блок `sql:` с тем же шаблоном.
