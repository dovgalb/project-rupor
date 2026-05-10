---
parent: ./README.md
view: repository
---

# 06 — Repo Model — `User` и `RefreshToken`

Документ описывает маппинг между доменными сущностями `internal/auth/domain/` и схемой БД (`migrations/0002_users`, `migrations/0003_refresh_tokens`), а также sqlc-сигнатуры и обработку ошибок драйвера. Базовый стандарт — `prompts/RepoModel.txt`.

## Маппинг `User` ↔ `users`

| Поле сущности (VO / тип) | Столбец БД (тип) | Конверсия |
|--------------------------|------------------|-----------|
| `ID() UserID` (VO над `uuid.UUID`) | `id uuid` | `domain.NewUserID(row.ID)` ↔ `user.ID().UUID()` |
| `Email() Email` (VO над `string`) | `email citext` | `string(row.Email)` → `domain.NewEmail(...)`; `email.String()` → `pgtype.Text` для citext-параметра (`pgx/v5` принимает `string`) |
| `Username() Username` (VO над `string`) | `username text` | `domain.NewUsername(row.Username)` ↔ `username.String()` |
| `PasswordHash() PasswordHash` (VO над `string`) | `password_hash text` | `domain.NewPasswordHash(row.PasswordHash)` ↔ `hash.String()` |
| `CreatedAt() time.Time` | `created_at timestamptz` | прямое значение (`time.Time` в UTC) |

Маппер живёт в `internal/auth/repository/postgres/mapper.go`:

```go
func userRowToDomain(row db.User) (*domain.User, error) {
    email, err := domain.NewEmail(string(row.Email))
    if err != nil {
        return nil, fmt.Errorf("user row: email: %w", err)
    }
    username, err := domain.NewUsername(row.Username)
    if err != nil {
        return nil, fmt.Errorf("user row: username: %w", err)
    }
    hash, err := domain.NewPasswordHash(row.PasswordHash)
    if err != nil {
        return nil, fmt.Errorf("user row: password_hash: %w", err)
    }
    return domain.ReconstructUser(
        domain.NewUserID(row.ID),
        email,
        username,
        hash,
        row.CreatedAt,
    )
}

func domainToInsertUserParams(u *domain.User) db.InsertUserParams {
    return db.InsertUserParams{
        ID:           u.ID().UUID(),
        Email:        u.Email().String(),
        Username:     u.Username().String(),
        PasswordHash: u.PasswordHash().String(),
        CreatedAt:    u.CreatedAt(),
    }
}
```

Ошибки маппинга — баг данных (`prompts/RepoModel.txt:96`); репозиторий обернёт их и вернёт наверх. Use case логирует `ERROR` и возвращает 500.

## Маппинг `RefreshToken` ↔ `refresh_tokens`

| Поле сущности | Столбец БД | Конверсия |
|---------------|------------|-----------|
| `ID() RefreshTokenID` | `id uuid` | `domain.NewRefreshTokenID(row.ID)` ↔ `token.ID().UUID()` |
| `UserID() UserID` | `user_id uuid` | `domain.NewUserID(row.UserID)` ↔ `token.UserID().UUID()` |
| `TokenHash() TokenHash` (VO над `[32]byte`) | `token_hash bytea` | `domain.NewTokenHash(row.TokenHash)` ↔ `token.TokenHash().Bytes()[:]` (slice для pgx) |
| `ExpiresAt() time.Time` | `expires_at timestamptz` | прямое значение |
| `CreatedAt() time.Time` | `created_at timestamptz` | прямое значение |
| `RevokedAt() time.Time` (zero = активный) | `revoked_at timestamptz NULL` | `pgtype.Timestamptz` ↔ доменное опциональное поле; маппер: `if row.RevokedAt.Valid { token.revokedAt = row.RevokedAt.Time }` |

Маппер для `RefreshToken`:

```go
func refreshTokenRowToDomain(row db.RefreshToken) (*domain.RefreshToken, error) {
    hash, err := domain.NewTokenHash(row.TokenHash)
    if err != nil {
        return nil, fmt.Errorf("refresh_token row: token_hash: %w", err)
    }
    var revokedAt time.Time
    if row.RevokedAt.Valid {
        revokedAt = row.RevokedAt.Time
    }
    return domain.ReconstructRefreshToken(
        domain.NewRefreshTokenID(row.ID),
        domain.NewUserID(row.UserID),
        hash,
        row.ExpiresAt,
        row.CreatedAt,
        revokedAt,
    )
}

func domainToInsertRefreshTokenParams(t *domain.RefreshToken) db.InsertRefreshTokenParams {
    bytes := t.TokenHash().Bytes()
    return db.InsertRefreshTokenParams{
        ID:        t.ID().UUID(),
        UserID:    t.UserID().UUID(),
        TokenHash: bytes[:],
        ExpiresAt: t.ExpiresAt(),
        CreatedAt: t.CreatedAt(),
    }
}
```

Заметим: при INSERT мы НЕ передаём `revoked_at` — он остаётся `NULL` по умолчанию. Активный токен — это всегда строка с `revoked_at IS NULL`.

## sqlc Queries

Файлы запросов:
- `internal/auth/repository/postgres/queries/users.sql`
- `internal/auth/repository/postgres/queries/refresh_tokens.sql`

### `users.sql`

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

`GetUserByEmail` опирается на UNIQUE-индекс `users_email_key`, созданный автоматически под `email citext UNIQUE` (`migrations/0002_users.up.sql:8`). Регистр-нечувствительность поиска обеспечивается типом `citext`.

`InsertUser` принимает `id` и `created_at` явно, чтобы use case (а не БД) контролировал генерацию идентификаторов и временных меток — это требование детерминизма для тестов (`prompts/Tests Style.txt:55-62`).

### `refresh_tokens.sql`

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

`RevokeRefreshTokenByHash` помечен `:execrows` — sqlc генерирует метод, возвращающий `int64` (число затронутых строк). Используется в `Rotate` для определения race-условия (`affected == 0` → токен уже отозван).

`GetRefreshTokenByHash` опирается на UNIQUE-индекс `refresh_tokens_token_hash_key` (`migrations/0003_refresh_tokens.up.sql:7`).

В фазе 1.3 **не вводим** `RevokeAllRefreshTokensByUser` — он понадобится для logout-all и reuse-detection (`docs/1_2_db_schema/token_and_index/06-repo-model.md:97-101`), но эти фичи вне scope (см. `03-decisions.md`, OQ-3, OQ-4).

## Реализация `Rotate` через `pgx.Tx`

`RefreshTokenRepository.Rotate(ctx, oldHash, newToken, revokedAt)` живёт в `internal/auth/repository/postgres/refresh_token_repository.go` и использует прямой `pgxpool.Pool` (а не только `*db.Queries`):

```go
type RefreshTokenRepository struct {
    pool *pgxpool.Pool
    q    *db.Queries
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
    return &RefreshTokenRepository{pool: pool, q: db.New(pool)}
}

func (r *RefreshTokenRepository) Rotate(
    ctx context.Context,
    oldHash domain.TokenHash,
    newToken *domain.RefreshToken,
    revokedAt time.Time,
) error {
    tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx) // no-op if commit succeeded

    qtx := r.q.WithTx(tx)
    oldBytes := oldHash.Bytes()
    affected, err := qtx.RevokeRefreshTokenByHash(ctx, db.RevokeRefreshTokenByHashParams{
        TokenHash: oldBytes[:],
        RevokedAt: pgtype.Timestamptz{Time: revokedAt, Valid: true},
    })
    if err != nil {
        return fmt.Errorf("revoke old: %w", err)
    }
    if affected == 0 {
        return domain.ErrRefreshTokenRevoked
    }

    if err := qtx.InsertRefreshToken(ctx, domainToInsertRefreshTokenParams(newToken)); err != nil {
        if isUniqueViolation(err, "refresh_tokens_token_hash_key") {
            return fmt.Errorf("insert new: hash collision: %w", err)
        }
        return fmt.Errorf("insert new: %w", err)
    }

    if err := tx.Commit(ctx); err != nil {
        return fmt.Errorf("commit: %w", err)
    }
    return nil
}
```

`pgx.TxOptions{}` использует уровень изоляции `read committed` — достаточно, потому что вся транзакция сериализуется через UNIQUE constraint на `token_hash` (любой race на INSERT упадёт в `unique_violation`).

## Маппинг ошибок драйвера → доменные ошибки

`internal/auth/repository/postgres/pgerr.go`:

```go
func isUniqueViolation(err error, constraintName string) bool {
    var pgErr *pgconn.PgError
    if !errors.As(err, &pgErr) {
        return false
    }
    return pgErr.Code == "23505" && pgErr.ConstraintName == constraintName
}
```

Маппинг в `UserRepository.Save`:

```go
func (r *UserRepository) Save(ctx context.Context, u *domain.User) error {
    if err := r.q.InsertUser(ctx, domainToInsertUserParams(u)); err != nil {
        if isUniqueViolation(err, "users_email_key") {
            return domain.ErrEmailAlreadyTaken
        }
        if isUniqueViolation(err, "users_username_key") {
            return domain.ErrUsernameAlreadyTaken
        }
        return fmt.Errorf("insert user: %w", err)
    }
    return nil
}
```

В `FindByID` / `FindByEmail`:

```go
func (r *UserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
    row, err := r.q.GetUserByID(ctx, id.UUID())
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, domain.ErrUserNotFound
        }
        return nil, fmt.Errorf("get user by id: %w", err)
    }
    return userRowToDomain(row)
}
```

Аналогично для `RefreshTokenRepository.FindByHash` → `domain.ErrRefreshTokenNotFound`.

`pgx.ErrNoRows` — стандартная sentinel-ошибка `pgx/v5` при `:one` запросе, возвращающем 0 строк.

## sqlc.yaml

Конфигурация генератора:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations"
    queries: "internal/auth/repository/postgres/queries"
    gen:
      go:
        sql_package: "pgx/v5"
        package: "db"
        out: "internal/auth/repository/postgres/db"
        emit_interface: false
        emit_json_tags: false
        emit_db_tags: false
        emit_pointers_for_null_types: false
        emit_prepared_queries: false
```

`emit_pointers_for_null_types: false` → nullable-поля будут как `pgtype.Timestamptz` с полем `Valid` (а не `*time.Time`). Это явный паттерн `prompts/RepoModel.txt:104-113` (раздел «Nullable-поля»).

`emit_interface: false` — мы не используем интерфейс `Querier`; для тестов фейкуем не sqlc-слой, а бизнес-интерфейс репо (`UserRepository` из `usecase/`).

## Зависимости миграций

Новых миграций **нет**. Используются:
- `0001_init` — расширение `pgcrypto` (`gen_random_uuid()`) — нужно для `id uuid DEFAULT gen_random_uuid()` в обеих таблицах. **В sqlc-параметрах мы передаём id явно**, поэтому DEFAULT не активен; но миграция остаётся как поддержка.
- `0002_users` — таблица `users`.
- `0003_refresh_tokens` — таблица `refresh_tokens` + индекс `refresh_tokens_user_id_idx`.

## Соответствие `prompts/RepoModel.txt`

| Правило | Применение |
|---------|-----------|
| `prompts/RepoModel.txt:79-81` (маппинг через `Reconstruct*`) | ✅ `userRowToDomain` и `refreshTokenRowToDomain` вызывают `domain.ReconstructUser` / `domain.ReconstructRefreshToken`. |
| `prompts/RepoModel.txt:84-88` (две функции на сущность) | ✅ `userRowToDomain` / `domainToInsertUserParams`; `refreshTokenRowToDomain` / `domainToInsertRefreshTokenParams`. |
| `prompts/RepoModel.txt:90-93` (VO разворачиваются на границе) | ✅ `Email.String()` ↔ `string(row.Email)`; `TokenHash.Bytes()[:]` ↔ `row.TokenHash`. |
| `prompts/RepoModel.txt:96` (ошибка маппинга — баг данных) | ✅ `userRowToDomain` оборачивает доменные ошибки в `fmt.Errorf("user row: ...: %w", err)`. |
| `prompts/RepoModel.txt:98-101` (маппер не делает I/O) | ✅ `userRowToDomain` — чистая функция от `db.User`. |
| `prompts/RepoModel.txt:104-113` (nullable) | ✅ `revoked_at` через `pgtype.Timestamptz` + `Valid` flag. |
| `prompts/RepoModel.txt:118-122` (id как VO) | ✅ `domain.NewUserID(row.ID)` ↔ `user.ID().UUID()`. |
| `prompts/RepoModel.txt:138-144` (маппинг unique violation) | ✅ `isUniqueViolation(err, "users_email_key")` → `domain.ErrEmailAlreadyTaken`. Имена ограничений — фикс из `docs/1_2_db_schema/users_table/03-decisions.md` решение 6. |
| `prompts/RepoModel.txt:147-150` (транзакции через `BeginTx`) | ✅ `Rotate` использует `pool.BeginTx` + `q.WithTx(tx)`. |
| `prompts/RepoModel.txt:152-159` (БД-ошибки → доменные) | ✅ `pgx.ErrNoRows` → `ErrUserNotFound`/`ErrRefreshTokenNotFound`; `pgconn.PgError` code `23505` → `ErrEmailAlreadyTaken` / `ErrUsernameAlreadyTaken`. |
| `prompts/RepoModel.txt:163-166` (не использовать ORM) | ✅ только sqlc + pgx. |
| `prompts/RepoModel.txt:165` (SQL в `repository/postgres/queries/`) | ✅ `users.sql`, `refresh_tokens.sql` в подпапке `queries/`. |
