---
parent: ./README.md
view: repository
---

# Repo Model — таблица `refresh_tokens`

Документ описывает схему таблицы `refresh_tokens`, которую создаёт миграция `0003_refresh_tokens`, и фиксирует, как она будет маппиться на будущий доменный слой `auth` (задача 1.3 — вне scope этой фичи).

## DDL (целевая схема)

```sql
-- migrations/0003_refresh_tokens.up.sql

CREATE TABLE refresh_tokens (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz NULL,
    CONSTRAINT refresh_tokens_token_hash_length_check CHECK (octet_length(token_hash) = 32)
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens(user_id);
```

```sql
-- migrations/0003_refresh_tokens.down.sql

DROP TABLE IF EXISTS refresh_tokens;
```

## Сводка по столбцам

| Столбец | Тип | NULL | Default | Ограничения | Назначение |
|---------|-----|------|---------|-------------|------------|
| `id` | `uuid` | NOT NULL | `gen_random_uuid()` | PRIMARY KEY (`refresh_tokens_pkey`) | Стабильный идентификатор записи; используется как ключ при отзыве |
| `user_id` | `uuid` | NOT NULL | — | FK `refresh_tokens_user_id_fkey` → `users(id)` ON DELETE CASCADE; индекс `refresh_tokens_user_id_idx` (b-tree) | Владелец токена |
| `token_hash` | `bytea` | NOT NULL | — | UNIQUE (`refresh_tokens_token_hash_key`); CHECK `octet_length(token_hash) = 32` (`refresh_tokens_token_hash_length_check`) | `sha256(refresh_token)` — 32 сырых байта; raw токен в БД не хранится |
| `expires_at` | `timestamptz` | NOT NULL | — | — | Момент истечения срока действия (UTC); момент задаёт приложение в задаче 1.3 |
| `created_at` | `timestamptz` | NOT NULL | `now()` | — | Момент выпуска токена (UTC) |
| `revoked_at` | `timestamptz` | NULL | — | — | Момент отзыва токена. NULL = активный; не-NULL = отозван (logout / ротация / админ-отзыв) |

## Имена объектов БД (фиксируется этой фичей)

| Объект | Имя | Тип |
|--------|-----|-----|
| Таблица | `refresh_tokens` | table |
| Первичный ключ | `refresh_tokens_pkey` | constraint (PK) |
| FK на `users(id)` | `refresh_tokens_user_id_fkey` | constraint (FK) |
| Уникальность `token_hash` | `refresh_tokens_token_hash_key` | constraint (UNIQUE) |
| Длина `token_hash` | `refresh_tokens_token_hash_length_check` | constraint (CHECK) |
| Индекс под `refresh_tokens_pkey` | `refresh_tokens_pkey` | b-tree (создаётся автоматически под PK) |
| Индекс под `refresh_tokens_token_hash_key` | `refresh_tokens_token_hash_key` | b-tree (создаётся автоматически под UNIQUE) |
| Индекс по `user_id` | `refresh_tokens_user_id_idx` | b-tree (явный) |

`refresh_tokens_token_hash_key` — это имя, которое явно ожидается в задаче 1.3 для маппинга unique-violation (например, на доменную ошибку «токен уже выпущен» или для повтора генерации) по аналогии с `users_email_key` (`prompts/RepoModel.txt:138-140`).

## Соответствие `prompts/RepoModel.txt` (задача 1.3 — для справки)

Маппинг ниже носит информационный характер и фиксирует контракт между этой миграцией и будущим репозиторием `internal/auth/repository/postgres/`. В рамках задачи 1.2 Go-код не пишется.

Предполагаемая доменная сущность `auth.RefreshToken` (задача 1.3) — обычная сущность с инвариантами «токен либо активен, либо отозван», «истечение задаётся при выпуске», «hash длиной 32 байта»:

| Поле будущего домена `auth.RefreshToken` | Столбец БД | Конверсия (планируемая в 1.3) |
|------------------------------------------|------------|-------------------------------|
| `id domain.RefreshTokenID` (VO над `uuid.UUID`) | `id uuid` | `domain.NewRefreshTokenID(row.ID)` ↔ `token.ID().UUID()` |
| `userID domain.UserID` | `user_id uuid` | `domain.NewUserID(row.UserID)` ↔ `token.UserID().UUID()` |
| `tokenHash domain.TokenHash` (VO над `[32]byte`) | `token_hash bytea` | `[32]byte(row.TokenHash)` ↔ `token.Hash()[:]` |
| `expiresAt time.Time` | `expires_at timestamptz` | прямое значение (`time.Time` в UTC) |
| `createdAt time.Time` | `created_at timestamptz` | прямое значение |
| `revokedAt time.Time` (zero = активный) или `optional[time.Time]` | `revoked_at timestamptz NULL` | `sql.NullTime` / `pgtype.Timestamptz` ↔ доменное опциональное поле; маппер разворачивает по правилу 6 в `prompts/RepoModel.txt:104-113` |

Решение «как именно представлять опциональное `revoked_at` в домене» (zero-value vs `Optional[T]` vs отдельное состояние enum) — открытый вопрос задачи 1.3. Миграция от этого решения не зависит: с точки зрения SQL поле всегда `timestamptz NULL`.

## sqlc Queries (плановые сигнатуры — задача 1.3)

В рамках задачи 1.2 запросы sqlc **не добавляем**. `sqlc.yaml` (`sqlc.yaml:1-2`) остаётся пустым. Ниже — ожидаемый набор запросов в задаче 1.3 для контекста (НЕ часть scope этой миграции):

```sql
-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
FROM   refresh_tokens
WHERE  token_hash = $1;

-- name: RevokeRefreshTokenByHash :exec
UPDATE refresh_tokens
SET    revoked_at = now()
WHERE  token_hash = $1
  AND  revoked_at IS NULL;

-- name: RevokeAllRefreshTokensByUser :exec
UPDATE refresh_tokens
SET    revoked_at = now()
WHERE  user_id = $1
  AND  revoked_at IS NULL;
```

Запрос `GetRefreshTokenByHash` опирается на индекс `refresh_tokens_token_hash_key`. Запрос `RevokeAllRefreshTokensByUser` опирается на индекс `refresh_tokens_user_id_idx` (решение 8 в `03-decisions.md`).

## Соответствие стандартам

| Стандарт | Применимость к миграции | Статус |
|----------|-------------------------|--------|
| `prompts/Architecture Layers.txt:60-65` | Миграция = внешний слой; не импортирует ничего из `domain/`/`usecase/`/`transport/`. | ✅ |
| `prompts/RepoModel.txt:138-140` | Имя `refresh_tokens_token_hash_key` детерминировано и предсказуемо для будущего маппинга unique-violation. | ✅ |
| `prompts/RepoModel.txt:118-122` | Конвенция id (uuid в БД ↔ VO в домене) сохраняется. | ✅ N/A на DDL |
| `prompts/Clean architecture.txt` | DDL не нарушает направление зависимостей. | ✅ N/A на DDL |
| `prompts/Domain Model.txt`, `Builder.txt`, `Go style.txt`, `Tests Style.txt`, `Domain model test.txt` | Применимы к Go-коду; миграция — чистый SQL. | N/A |

## Зависимости миграции

- **Расширение `pgcrypto`** — подключено `0001_init` (`migrations/0001_init.up.sql:4`), даёт `gen_random_uuid()` для PK.
- **Таблица `users`** — создаётся `0002_users` (`migrations/0002_users.up.sql:6-13`); её существование требуется для FK `refresh_tokens.user_id → users(id)`.
- **Расширение `citext`** — НЕ требуется. Подключение, выполненное `0002_users`, не используется в этой миграции и не трогается.
- **Порядок миграций** `0001 → 0002 → 0003` обеспечивается префиксами; `golang-migrate` накатывает по возрастанию (`Makefile:54-58`).
