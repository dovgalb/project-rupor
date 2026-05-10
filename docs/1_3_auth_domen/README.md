---
date: 2026-05-07
feature: 1_3_auth_domen
status: draft
research: ../../.thoughts/research/2026-05-07-1_3-auth-domen-baseline.md
---

# 1.3 Домен auth — Документы дизайна

## Бизнес-контекст

Задача 1.3 «Домен auth» (`.claude/plans/general_plan.md:105-112`) даёт платформе первый рабочий бизнес-домен и закрывает Фазу 1: после неё пользователь может зарегистрироваться, авторизоваться и получить пару токенов (access + refresh), необходимых для всех последующих защищённых эндпоинтов (`rooms`, `channels`, `messages`, WebSocket).

Без 1.3 всё остальное в плане заблокировано: фазы 2–3 опираются на наличие `userID` в контексте запроса (`general_plan.md:115`, JWT-middleware фазы 1.4), фаза 5 (фронтенд) — на login-флоу. Миграции `0002_users` и `0003_refresh_tokens` (фаза 1.2) уже создали БД-уровень контракта; фаза 1.3 наполняет четыре слоя `internal/auth/` Go-кодом.

Scope строго ограничен подпунктами тикета (`general_plan.md:106-112`):
- Доменные сущности `User`, `RefreshToken` + value objects (`Email`, `Username`, `Password`, `PasswordHash`, `UserID`, `RefreshTokenID`, `TokenHash`).
- Сценарии `Register` / `Login` / `Refresh` / `GetCurrentUser`.
- Хеширование паролей через `golang.org/x/crypto/bcrypt`.
- Генерация и валидация JWT (HS256) для access-токена; refresh — 32 случайных байта (sha256 хранится в БД).
- sqlc-запросы и реализация репозиториев в `internal/auth/repository/postgres/`.
- HTTP-хендлеры `POST /api/v1/auth/register`, `/login`, `/refresh`, `GET /api/v1/auth/me` + DTO.
- Подключение роутов в `cmd/server/main.go`.

Вне scope:
- JWT-middleware (фаза 1.4 — `general_plan.md:114-118`). В 1.3 хендлер `/auth/me` валидирует токен напрямую через `TokenIssuer.VerifyAccess`; это временное прямое использование (см. `03-decisions.md`, ADR-009).
- CORS, recover, request-logging middleware (фаза 1.4).
- Интеграционные тесты против реального PostgreSQL — задача 1.5 (`general_plan.md:120-123`); в 1.3 пишутся unit/handler-тесты с фейками.
- Cleanup истёкших/отозванных refresh-токенов (откладывается, см. `docs/1_2_db_schema/token_and_index/03-decisions.md`, решение 14).
- E-mail верификация, восстановление пароля, multi-device-сессии — не в MVP (`docs/1_2_db_schema/users_table/03-decisions.md`, решения 8–9).

## Критерии приёмки

1. В `internal/auth/domain/` появляются сущности `User`, `RefreshToken`, value objects (`UserID`, `Email`, `Username`, `Password`, `PasswordHash`, `RefreshTokenID`, `TokenHash`) и доменные ошибки. Сущности — rich-модель по `prompts/Domain Model.txt:24-50`: приватные поля, конструкторы `New*` / `Reconstruct*`, инварианты в конструкторах.
2. В `internal/auth/usecase/` объявлены интерфейсы зависимостей (`UserRepository`, `RefreshTokenRepository`, `PasswordHasher`, `TokenIssuer`, `Clock`, `UUIDGenerator`, `RandomBytes`) и реализованы сценарии `RegisterUser`, `LoginUser`, `RefreshAccess`, `GetCurrentUser`.
3. В `internal/auth/repository/postgres/` реализованы `UserRepository` и `RefreshTokenRepository` через sqlc-запросы поверх `pgx/v5`. Имена ограничений `users_email_key`, `users_username_key`, `refresh_tokens_token_hash_key` явно мапятся на доменные ошибки (`prompts/RepoModel.txt:138-144`).
4. В `internal/auth/repository/jwt/` реализован `TokenIssuer` через `github.com/golang-jwt/jwt/v5` (HS256, секрет — `cfg.JWTSecret()`). В `internal/auth/repository/bcrypt/` реализован `PasswordHasher` через `golang.org/x/crypto/bcrypt`.
5. В `internal/auth/transport/http/` реализованы хендлеры `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `GET /auth/me` и DTO. JSON-формы запроса/ответа точно соответствуют `08-api-contract.md`.
6. `cmd/server/main.go` собирает граф зависимостей: `pgxpool.Pool` → sqlc `*db.Queries` → repositories → hasher → token-issuer → use cases → handlers → регистрация роутов под `r.Route("/api/v1/auth", ...)`. Используется единый shutdown-таймаут 5с.
7. `config/` дополнен полями `JWTAccessTTL` (env `JWT_ACCESS_TTL`, default `15m`) и `JWTRefreshTTL` (env `JWT_REFRESH_TTL`, default `720h`). Парсинг через `time.ParseDuration`. Ошибки конфига — `CONFIG-004`, `CONFIG-005`.
8. `sqlc.yaml` настроен: `sql_package: pgx/v5`, `engine: postgresql`, `schema: migrations`, `queries: internal/auth/repository/postgres/queries`, `out: internal/auth/repository/postgres/db`. `make sqlc` генерирует пакет `db` без ошибок.
9. Все коды ошибок диапазона `AUTH-001..AUTH-099` зафиксированы (`08-api-contract.md`) и покрыты тестами (`04-testing.md`).
10. `go build ./...`, `make lint`, `make test` — все три зелёные. Race-detector чистый.
11. `go.mod` пополняется ровно четырьмя новыми зависимостями: `github.com/jackc/pgx/v5`, `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid`, `golang.org/x/crypto`. Все согласованы заранее.
12. `internal/auth/domain/` и `internal/auth/usecase/` не содержат импортов вне stdlib (для domain — модулёт `github.com/google/uuid` явно разрешён `prompts/Domain Model.txt:25`; для usecase — только `domain` + stdlib + `uuid`).
13. Раскладка `internal/auth/{domain,usecase,transport/http,repository/{postgres,jwt,bcrypt}}/` — всё под `repository/` (см. ADR-005). Никаких новых верхних слоёв в `internal/auth/`.
14. Refresh-токен ротируется атомарно: `RefreshTokenRepository.Rotate(ctx, oldHash, *newToken)` помечает старый `revoked_at = now()` и вставляет новый в одной `pgx.Tx`. Тесты репозитория проверяют атомарность.

## Документы

| Файл | Разрез | Описание |
|------|--------|----------|
| [01-architecture.md](./01-architecture.md) | Logical | C4 L1 → L2 → L3, граф зависимостей слоёв `internal/auth/`, state-machine `RefreshToken` |
| [02-behavior.md](./02-behavior.md) | Process | DFD по 4 сценариям + sequence diagrams, error/edge cases |
| [03-decisions.md](./03-decisions.md) | Decision | 12 ADR (драйвер, JWT, UUID, TTL, refresh-стратегия, раскладка адаптеров, …), риски, open questions |
| [04-testing.md](./04-testing.md) | Quality | Coverage mapping `AUTH-*` ↔ тесты, тест-кейсы по слоям, builder-стратегия |
| [06-repo-model.md](./06-repo-model.md) | Repository | Маппинг `User`/`RefreshToken` на таблицы, sqlc-сигнатуры, обработка `*pgconn.PgError` |
| [07-standards.md](./07-standards.md) | Standards | Compliance-матрица по 8 стандартам `prompts/`, уточнения и расхождения |
| [08-api-contract.md](./08-api-contract.md) | Contract | REST-контракт 4 эндпоинтов: точные JSON request/response, error responses, тест-план |
| [research.md](../../.thoughts/research/2026-05-07-1_3-auth-domen-baseline.md) | Research | Baseline: что уже есть в коде, какие решения зафиксированы по `users` и `refresh_tokens` |

### Намеренно опущенные документы шаблона

- **`05-events.md`** — фича не публикует доменных событий с подписчиками. Hipotetический `UserRegistered` не имеет получателей до фазы 3 (WebSocket-hub в `pkg/websocket/` пока пуст). Если в будущем появятся подписчики (welcome-email, аудит-лог) — событие добавится отдельной миграцией кода в соответствующей фазе. В этой фиче use case `RegisterUser` возвращает результат напрямую, без шины.
