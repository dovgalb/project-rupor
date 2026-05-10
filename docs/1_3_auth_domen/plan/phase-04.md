---
phase: 4
name: internal/auth/repository/postgres
layer: adapter
depends_on: phase-01, phase-03
plan: ./README.md
---

# Phase 4: `internal/auth/repository/postgres/`

## Цель

Реализовать sqlc-запросы и Postgres-адаптеры для портов `UserRepository` и `RefreshTokenRepository`. После фазы — `make sqlc` генерирует пакет `db/`, реализации проходят интеграционные тесты против реального Postgres (`//go:build integration`); `go build ./...` зелёный, юнит-тесты на маппер — зелёные.

## Контекст

- Все sql-запросы и контракты — `../06-repo-model.md:111-156`.
- Реализация `Rotate` через `pgx.Tx` — `../06-repo-model.md:158-211`. Атомарность критична (`../03-decisions.md` ADR-006, ADR-010).
- Маппер ошибок — `../06-repo-model.md:213-261`. `pgx.ErrNoRows` → `domain.ErrUserNotFound` / `domain.ErrRefreshTokenNotFound`; `*pgconn.PgError` `code 23505` + `ConstraintName` → доменные ошибки уникальности.
- Имена ограничений в БД фиксированы фазой 1.2: `users_email_key`, `users_username_key`, `refresh_tokens_token_hash_key` (`migrations/0002_users.up.sql:8,12-13`, `migrations/0003_refresh_tokens.up.sql`).

## Файлы для создания

### `internal/auth/repository/postgres/queries/users.sql`

Точное содержимое — `../06-repo-model.md:113-127`:

```sql
-- name: InsertUser :exec
INSERT INTO users (id, email, password_hash, username, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetUserByID :one
SELECT id, email, password_hash, username, created_at
FROM   users
WHERE  id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, username, created_at
FROM   users
WHERE  email = $1;
```

### `internal/auth/repository/postgres/queries/refresh_tokens.sql`

Точное содержимое — `../06-repo-model.md:135-150`:

```sql
-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
FROM   refresh_tokens
WHERE  token_hash = $1;

-- name: RevokeRefreshTokenByHash :execrows
UPDATE refresh_tokens
SET    revoked_at = $2
WHERE  token_hash = $1
  AND  revoked_at IS NULL;
```

### `internal/auth/repository/postgres/db/`

Создаётся `make sqlc` после написания queries. Содержимое — сгенерированное (3+ файлов в зависимости от sqlc-версии: `db.go`, `models.go`, `users.sql.go`, `refresh_tokens.sql.go`). Файлы коммитим в git (`prompts/RepoModel.txt:163-166`, sqlc генерация — часть исходного кода).

После генерации проверить, что типы соответствуют ожиданиям маппера (`../06-repo-model.md:23-100`):
- `db.User.ID uuid.UUID`, `db.User.Email string` (citext в pgx-типах = string), `db.User.PasswordHash string`, `db.User.Username string`, `db.User.CreatedAt time.Time`.
- `db.RefreshToken.TokenHash []byte`, `db.RefreshToken.RevokedAt pgtype.Timestamptz`.

Если sqlc сгенерировал что-то иначе (например, `pgtype.UUID` вместо `uuid.UUID`) — диагностика: проверить `sqlc.yaml`, флаг `sql_package: "pgx/v5"` критичен. Не подгонять маппер под отклонения; чинить генератор.

### `internal/auth/repository/postgres/mapper.go`

Точное содержимое — `../06-repo-model.md:22-101`. Ключевые функции:
- `userRowToDomain(row db.User) (*domain.User, error)`
- `domainToInsertUserParams(u *domain.User) db.InsertUserParams`
- `refreshTokenRowToDomain(row db.RefreshToken) (*domain.RefreshToken, error)`
- `domainToInsertRefreshTokenParams(t *domain.RefreshToken) db.InsertRefreshTokenParams`

Все четыре — чистые функции без I/O (`prompts/RepoModel.txt:98-101`). Тесты — round-trip без БД (см. ниже).

### `internal/auth/repository/postgres/pgerr.go`

```go
package postgres

import (
    "errors"

    "github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error, constraintName string) bool {
    var pgErr *pgconn.PgError
    if !errors.As(err, &pgErr) {
        return false
    }
    return pgErr.Code == "23505" && pgErr.ConstraintName == constraintName
}
```

### `internal/auth/repository/postgres/user_repository.go`

```go
package postgres

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"

    "github.com/dovgalb/project-rupor/internal/auth/domain"
    "github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
)

type UserRepository struct {
    q *db.Queries
}

func NewUserRepository(q *db.Queries) *UserRepository {
    return &UserRepository{q: q}
}

func (r *UserRepository) Save(ctx context.Context, u *domain.User) error {
    if err := r.q.InsertUser(ctx, domainToInsertUserParams(u)); err != nil {
        if isUniqueViolation(err, "users_email_key") {
            return domain.ErrEmailAlreadyTaken
        }
        if isUniqueViolation(err, "users_username_key") {
            return domain.ErrUsernameAlreadyTaken
        }
        return fmt.Errorf("postgres: insert user: %w", err)
    }
    return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
    row, err := r.q.GetUserByID(ctx, id.UUID())
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrUserNotFound
        }
        return nil, fmt.Errorf("postgres: get user by id: %w", err)
    }
    return userRowToDomain(row)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
    row, err := r.q.GetUserByEmail(ctx, email.String())
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrUserNotFound
        }
        return nil, fmt.Errorf("postgres: get user by email: %w", err)
    }
    return userRowToDomain(row)
}
```

Compile-time проверка `var _ usecase.UserRepository = (*UserRepository)(nil)` — НЕ добавляем здесь, потому что это создаст обратную зависимость `repository → usecase` для типа интерфейса. Альтернатива: не добавлять компайл-проверку, или добавить её через build-tag в `compile_check_test.go`. **Решение:** добавляем в `compile_check_test.go` (см. ниже).

### `internal/auth/repository/postgres/refresh_token_repository.go`

```go
package postgres

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgtype"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/dovgalb/project-rupor/internal/auth/domain"
    "github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db"
)

type RefreshTokenRepository struct {
    pool *pgxpool.Pool
    q    *db.Queries
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
    return &RefreshTokenRepository{pool: pool, q: db.New(pool)}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, t *domain.RefreshToken) error {
    if err := r.q.InsertRefreshToken(ctx, domainToInsertRefreshTokenParams(t)); err != nil {
        if isUniqueViolation(err, "refresh_tokens_token_hash_key") {
            return fmt.Errorf("postgres: insert refresh: hash collision: %w", err)
        }
        return fmt.Errorf("postgres: insert refresh: %w", err)
    }
    return nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, h domain.TokenHash) (*domain.RefreshToken, error) {
    bytes := h.Bytes()
    row, err := r.q.GetRefreshTokenByHash(ctx, bytes[:])
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrRefreshTokenNotFound
        }
        return nil, fmt.Errorf("postgres: get refresh by hash: %w", err)
    }
    return refreshTokenRowToDomain(row)
}

func (r *RefreshTokenRepository) Rotate(
    ctx context.Context,
    oldHash domain.TokenHash,
    newToken *domain.RefreshToken,
    revokedAt time.Time,
) error {
    tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return fmt.Errorf("postgres: begin tx: %w", err)
    }
    defer func() { _ = tx.Rollback(ctx) }() // no-op if already committed

    qtx := r.q.WithTx(tx)
    oldBytes := oldHash.Bytes()
    affected, err := qtx.RevokeRefreshTokenByHash(ctx, db.RevokeRefreshTokenByHashParams{
        TokenHash: oldBytes[:],
        RevokedAt: pgtype.Timestamptz{Time: revokedAt, Valid: true},
    })
    if err != nil {
        return fmt.Errorf("postgres: revoke old refresh: %w", err)
    }
    if affected == 0 {
        return domain.ErrRefreshTokenRevoked
    }

    if err := qtx.InsertRefreshToken(ctx, domainToInsertRefreshTokenParams(newToken)); err != nil {
        if isUniqueViolation(err, "refresh_tokens_token_hash_key") {
            return fmt.Errorf("postgres: insert new refresh: hash collision: %w", err)
        }
        return fmt.Errorf("postgres: insert new refresh: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("postgres: commit rotate: %w", err)
    }
    return nil
}
```

### `internal/auth/repository/postgres/compile_check_test.go`

```go
package postgres_test

import (
    "github.com/dovgalb/project-rupor/internal/auth/repository/postgres"
    "github.com/dovgalb/project-rupor/internal/auth/usecase"
)

var (
    _ usecase.UserRepository         = (*postgres.UserRepository)(nil)
    _ usecase.RefreshTokenRepository = (*postgres.RefreshTokenRepository)(nil)
)
```

Это **тестовый** файл, поэтому импорт `usecase` не создаёт production-обратной зависимости. Файл компилируется при `go test`, ловит расхождения сигнатур.

## Файлы для модификации

Нет.

## Тесты

### Юнит-тесты маппера (без БД)

`mapper_test.go` — round-trip тесты (`../04-testing.md:227-231`):
- `TestUserMapping_RoundTrip_AllFieldsPreserved` — `domain.User → domainToInsertUserParams → имитация db.User → userRowToDomain → исходные геттеры совпадают`.
- `TestRefreshTokenMapping_RoundTrip_AllFieldsPreserved` — то же для RefreshToken, отдельно для `revokedAt = zero` (Valid=false) и `revokedAt != zero` (Valid=true).

Эти 2 теста — без `//go:build integration`, запускаются в `make test`.

### Интеграционные тесты

Все остальные тесты — `//go:build integration` (`../04-testing.md:201-225`). Скелет `user_repository_integration_test.go` и `refresh_token_repository_integration_test.go`:

```go
//go:build integration

package postgres_test

// ... helpers: dbConn(t) *pgxpool.Pool, truncate(t, pool, "refresh_tokens", "users")
```

Содержание тестов — табличка из `../04-testing.md:207-225`. В фазе 1.3 эти тесты **могут оставаться непрогоняемыми в CI** — заработают в фазе 1.5 (`general_plan.md:120-123`). Но скелет должен компилироваться (`go build -tags=integration ./internal/auth/repository/postgres/...`) — это часть Definition of Done.

Альтернатива (минималка): пометить тела тестов `t.Skip("integration: requires Postgres, see issue 1.5")`. Но компилироваться обязаны (`prompts/Tests Style.txt:120-145`).

## Ключевые решения

- pgxpool.Pool инжектируется напрямую в `RefreshTokenRepository` (а не через `*db.Queries`), потому что нужен `BeginTx`. `UserRepository` получает только `*db.Queries` — у него нет транзакций (`../06-repo-model.md:160-170`).
- В `Rotate` используем `pgx.TxOptions{}` (default = `read committed`). Сериализуемость гарантируется UNIQUE-constraint'ом + UPDATE с `WHERE revoked_at IS NULL` (ADR-010).
- `compile_check_test.go` — единственный способ удостовериться, что адаптер удовлетворяет порту, без production-обратной зависимости.
- В `defer tx.Rollback(...)` глотаем ошибку через `_ = tx.Rollback(ctx)` — pgx возвращает `pgx.ErrTxClosed` после успешного `Commit`, это ожидаемо.

## Verification

- [ ] `make sqlc` успешен. Сгенерированные файлы попадают в `internal/auth/repository/postgres/db/`. Никаких файлов вне этого каталога не появляется.
- [ ] `go build ./internal/auth/repository/postgres/...` — зелёный.
- [ ] `go test ./internal/auth/repository/postgres/...` — зелёный (запустятся 2 round-trip + compile_check, integration пропускаются по build-tag).
- [ ] `go test -tags=integration ./internal/auth/repository/postgres/...` — собирается без ошибок (запуск может быть `t.Skip`).
- [ ] `make lint` зелёный.
- [ ] `git diff --name-only` ограничен `internal/auth/repository/postgres/**`.
