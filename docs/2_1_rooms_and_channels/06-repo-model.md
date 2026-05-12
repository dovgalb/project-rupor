---
parent: ./README.md
view: repository
---

# 06 — Repo Model

Маппинг доменных сущностей в строки БД, sqlc-сигнатуры, миграции. Следует `prompts/RepoModel.txt`.

## Маппинг сущность ↔ модель БД

### `room.domain.Room` ↔ `rooms`

| Поле сущности (VO) | Поле модели (raw) | Конверсия запись | Конверсия чтение |
|--------------------|-------------------|------------------|------------------|
| `ID() RoomID` | `id uuid` | `room.ID().UUID()` | `domain.NewRoomID(row.ID)` |
| `OwnerID() UserID` | `owner_id uuid` | `room.OwnerID().UUID()` | `domain.NewUserID(row.OwnerID)` |
| `Name() RoomName` | `name text` | `room.Name().String()` | `domain.NewRoomName(row.Name)` |
| `CreatedAt() time.Time` | `created_at timestamptz NOT NULL` | как есть | как есть |

Финальный шаг чтения: `domain.ReconstructRoom(id, ownerID, name, createdAt)`.

### `room.domain.Membership` ↔ `room_members`

| Поле сущности | Поле модели | Конверсия |
|---------------|-------------|-----------|
| `RoomID() RoomID` | `room_id uuid` | `.UUID()` ↔ `NewRoomID` |
| `UserID() UserID` | `user_id uuid` | `.UUID()` ↔ `NewUserID` |
| `Role() Role` | `role text` (enum-string) | `role.String()` ↔ `domain.ParseRole(row.Role)` |
| `JoinedAt() time.Time` | `joined_at timestamptz NOT NULL` | как есть |

Identity = `(room_id, user_id)`. PK составной.

### `room.domain.Invite` ↔ `invites`

| Поле сущности | Поле модели | Конверсия |
|---------------|-------------|-----------|
| `ID() InviteID` | `id uuid` | `.UUID()` ↔ `NewInviteID` |
| `RoomID() RoomID` | `room_id uuid` | `.UUID()` ↔ `NewRoomID` |
| `Code() InviteCode` | `code text` (8 chars) | `.String()` ↔ `NewInviteCode` |
| `CreatedBy() UserID` | `created_by uuid` | `.UUID()` ↔ `NewUserID` |
| `CreatedAt() time.Time` | `created_at timestamptz NOT NULL` | как есть |
| `RevokedAt() time.Time` (zero == active) | `revoked_at timestamptz NULL` | `if row.RevokedAt.Valid { revokedAt = row.RevokedAt.Time } else { revokedAt = time.Time{} }`. Запись: `pgtype.Timestamptz{Time: revokedAt, Valid: !revokedAt.IsZero()}` |

### `channel.domain.Channel` ↔ `channels`

| Поле сущности | Поле модели | Конверсия |
|---------------|-------------|-----------|
| `ID() ChannelID` | `id uuid` | `.UUID()` ↔ `NewChannelID` |
| `RoomID() RoomID` | `room_id uuid` | `.UUID()` ↔ `NewRoomID` (channel-домен) |
| `Name() ChannelName` | `name text` | `.String()` ↔ `NewChannelName` |
| `Kind() ChannelKind` | `kind text` (enum-string) | `kind.String()` ↔ `domain.ParseChannelKind` |
| `CreatedAt() time.Time` | `created_at timestamptz NOT NULL` | как есть |

## sqlc Queries (сигнатуры)

Все файлы — в директориях `internal/<domain>/repository/postgres/queries/`. Стиль повторяет `internal/auth/repository/postgres/queries/users.sql` (PascalCase имена, позиционные `$N`, без `RETURNING`).

### `internal/room/repository/postgres/queries/rooms.sql`

```sql
-- name: InsertRoom :exec
INSERT INTO rooms (id, owner_id, name, created_at)
VALUES ($1, $2, $3, $4);

-- name: GetRoomByID :one
SELECT id, owner_id, name, created_at
FROM rooms
WHERE id = $1;

-- name: ListRoomsByMember :many
SELECT r.id, r.owner_id, r.name, r.created_at, m.role
FROM rooms r
JOIN room_members m ON m.room_id = r.id
WHERE m.user_id = $1
ORDER BY r.created_at DESC;

-- name: DeleteRoomByID :execrows
DELETE FROM rooms
WHERE id = $1;
```

### `internal/room/repository/postgres/queries/room_members.sql`

```sql
-- name: InsertRoomMember :exec
INSERT INTO room_members (room_id, user_id, role, joined_at)
VALUES ($1, $2, $3, $4);

-- name: GetRoomMember :one
SELECT room_id, user_id, role, joined_at
FROM room_members
WHERE room_id = $1 AND user_id = $2;

-- name: ListRoomMembers :many
SELECT room_id, user_id, role, joined_at
FROM room_members
WHERE room_id = $1
ORDER BY joined_at ASC;

-- name: DeleteRoomMember :execrows
DELETE FROM room_members
WHERE room_id = $1 AND user_id = $2;
```

### `internal/room/repository/postgres/queries/invites.sql`

```sql
-- name: InsertInvite :exec
INSERT INTO invites (id, room_id, code, created_by, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: RevokeActiveInvitesByRoom :execrows
UPDATE invites
SET revoked_at = $2
WHERE room_id = $1 AND revoked_at IS NULL;

-- name: GetActiveInviteByCode :one
SELECT id, room_id, code, created_by, created_at, revoked_at
FROM invites
WHERE code = $1 AND revoked_at IS NULL;

-- name: GetActiveInviteByRoom :one
SELECT id, room_id, code, created_by, created_at, revoked_at
FROM invites
WHERE room_id = $1 AND revoked_at IS NULL;
```

### `internal/channel/repository/postgres/queries/channels.sql`

```sql
-- name: InsertChannel :exec
INSERT INTO channels (id, room_id, name, kind, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ListChannelsByRoom :many
SELECT id, room_id, name, kind, created_at
FROM channels
WHERE room_id = $1
ORDER BY created_at ASC;

-- name: DeleteChannelInRoom :execrows
DELETE FROM channels
WHERE id = $1 AND room_id = $2;
```

## Транзакции

| Операция | Где живёт | Подход |
|----------|-----------|--------|
| `RoomRepository.SaveWithOwner(room, owner)` | `room/repository/postgres/room_repository.go` | `pgxpool.Pool.BeginTx` → `db.Queries.WithTx(tx)` → `InsertRoom` → `InsertRoomMember(role='owner')` → `Commit`. По образцу `internal/auth/repository/postgres/refresh_token_repository.go:50-86` (`Rotate`). Конструктор `NewRoomRepository` принимает `*pgxpool.Pool` |
| `InviteRepository.RegenerateActive(roomID, code, createdBy, now)` | `room/repository/postgres/invite_repository.go` | `BeginTx` → `RevokeActiveInvitesByRoom(roomID, now)` → `InsertInvite(...)` → `Commit`. Если `INSERT` падает по unique violation на `invites_active_code` (коллизия) → возвращает `errInviteCodeCollision` (внутренняя ошибка пакета, не доменная) для ретрая в usecase. Конструктор принимает `*pgxpool.Pool` |
| Остальные операции | `MembershipRepository`, `ChannelRepository`, `RoomRepository.{FindByID,ListByMember,Delete}` | Используют `*db.Queries` напрямую (через `pool` или uniform handle). По аналогии с `internal/auth/repository/postgres/user_repository.go:23-25` |

## Маппинг ошибок Postgres → доменные ошибки

`internal/room/repository/postgres/pgerr.go` (новый, по образцу auth-овского):

```go
func isUniqueViolation(err error, constraint string) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
    }
    return false
}

func isForeignKeyViolation(err error, constraint string) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23503" && pgErr.ConstraintName == constraint
    }
    return false
}
```

| Источник ошибки | Constraint | Доменная ошибка |
|-----------------|------------|-----------------|
| `pgx.ErrNoRows` от `GetRoomByID` | — | `domain.ErrRoomNotFound` |
| `pgx.ErrNoRows` от `GetRoomMember` | — | `domain.ErrNotMember` |
| `pgx.ErrNoRows` от `GetActiveInviteByCode` / `GetActiveInviteByRoom` | — | `domain.ErrInviteNotFound` |
| Unique violation на `InsertRoomMember` | `room_members_pkey` | `domain.ErrAlreadyMember` |
| Unique violation на `InsertRoomMember` (попытка второго owner) | `room_members_one_owner_per_room` (partial unique idx) | `domain.ErrOwnerAlreadyExists` (внутренняя — не должна всплыть наружу при корректной логике; маппим в `INTERNAL`) |
| Unique violation на `InsertInvite` | `invites_active_code` (partial unique idx) | `errInviteCodeCollision` (для ретрая в usecase) |
| `DeleteRoomByID` или `DeleteChannelInRoom` вернул 0 rows | — | `domain.ErrRoomNotFound` / `domain.ErrChannelNotFound` |

`internal/channel/repository/postgres/pgerr.go` — аналогичный, плюс:

| Источник | Constraint | Доменная ошибка |
|----------|------------|-----------------|
| Unique violation на `InsertChannel` | `channels_room_id_name_key` | `channel.domain.ErrChannelNameAlreadyTaken` |
| `pgx.ErrNoRows` (для будущего `GetChannelByID`) | — | `channel.domain.ErrChannelNotFound` |
| `DeleteChannelInRoom` вернул 0 rows | — | `channel.domain.ErrChannelNotFound` |

## MembershipQueryAdapter (мост channel → room)

**Расположение:** `internal/room/repository/postgres/membership_query.go`. Это единственное место, где room-репозиторий импортирует `internal/channel/usecase` и `internal/channel/domain`.

**Сигнатура:**

```go
package roompg

import (
    "context"
    "errors"

    "github.com/dovgalb/project-rupor/internal/channel/domain"
    "github.com/dovgalb/project-rupor/internal/channel/usecase"
    roomdom "github.com/dovgalb/project-rupor/internal/room/domain"
    "github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
)

type MembershipQueryAdapter struct {
    q *db.Queries
}

func NewMembershipQueryAdapter(q *db.Queries) *MembershipQueryAdapter { ... }

func (a *MembershipQueryAdapter) Require(
    ctx context.Context,
    roomID domain.RoomID,         // channel.domain.RoomID
    userID domain.UserID,         // channel.domain.UserID
    req usecase.RoleRequirement,
) error {
    row, err := a.q.GetRoomMember(ctx, db.GetRoomMemberParams{
        RoomID: roomID.UUID(),    // bridge через uuid.UUID
        UserID: userID.UUID(),
    })
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return domain.ErrChannelAccessDenied
        }
        return fmt.Errorf("membership query: %w", err)
    }
    role, perr := roomdom.ParseRole(row.Role)
    if perr != nil {
        return fmt.Errorf("membership query: parse role: %w", perr)
    }
    if !satisfies(role, req) {
        return domain.ErrChannelInsufficientRole
    }
    return nil
}

func satisfies(role roomdom.Role, req usecase.RoleRequirement) bool {
    switch req {
    case usecase.RoleAnyMember:
        return true
    case usecase.RoleAdminOrOwner:
        return role == roomdom.RoleAdmin || role == roomdom.RoleOwner
    case usecase.RoleOwnerOnly:
        return role == roomdom.RoleOwner
    default:
        return false
    }
}
```

**Замечание о bridge через `uuid.UUID`:** `channel.domain.RoomID` и `room.domain.RoomID` — два независимых типа, оба обёртки над `uuid.UUID`. В адаптере мы переходим через `uuid.UUID` (метод `.UUID()` на любом из VO даёт сырой `uuid.UUID`, который мы передаём в sqlc). Никаких неявных приведений типов между двумя VO нет.

## Миграции

Файлы создаются в `migrations/`. Стиль повторяет `0002_users.up.sql` / `0003_refresh_tokens.up.sql`: первая строка-комментарий с именем файла, вторая — описание; CHECK-constraints именованные; индексы именованные; `IF NOT EXISTS` только для extensions (которых тут нет — `pgcrypto` уже подключён в 0001).

### `migrations/0004_rooms.up.sql`

```sql
-- 0004_rooms.up.sql
-- Создание таблицы rooms (комнаты — аналог серверов в Discord)

CREATE TABLE rooms (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT rooms_name_length_check CHECK (char_length(name) BETWEEN 1 AND 64)
);

CREATE INDEX rooms_owner_id_idx ON rooms(owner_id);
```

### `migrations/0004_rooms.down.sql`

```sql
-- 0004_rooms.down.sql
-- Откат таблицы rooms

DROP TABLE IF EXISTS rooms;
```

### `migrations/0005_room_members.up.sql`

```sql
-- 0005_room_members.up.sql
-- Создание таблицы room_members с ролями owner/admin/member

CREATE TABLE room_members (
    room_id     uuid        NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        text        NOT NULL,
    joined_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id),
    CONSTRAINT room_members_role_check CHECK (role IN ('owner', 'admin', 'member'))
);

-- Гарантия "ровно один owner на комнату"
CREATE UNIQUE INDEX room_members_one_owner_per_room
    ON room_members(room_id)
    WHERE role = 'owner';

-- Для ListUserRooms
CREATE INDEX room_members_user_id_idx ON room_members(user_id);
```

### `migrations/0005_room_members.down.sql`

```sql
-- 0005_room_members.down.sql
-- Откат таблицы room_members

DROP TABLE IF EXISTS room_members;
```

### `migrations/0006_invites.up.sql`

```sql
-- 0006_invites.up.sql
-- Создание таблицы invites (короткие base32-коды для вступления в комнаты)

CREATE TABLE invites (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id     uuid        NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    code        text        NOT NULL,
    created_by  uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz NULL,
    CONSTRAINT invites_code_length_check CHECK (char_length(code) = 8)
);

-- Один активный код на комнату
CREATE UNIQUE INDEX invites_active_room
    ON invites(room_id)
    WHERE revoked_at IS NULL;

-- Глобальная уникальность активного кода
CREATE UNIQUE INDEX invites_active_code
    ON invites(code)
    WHERE revoked_at IS NULL;

-- Для GetActiveInviteByCode (партиальный индекс выше уже его покрывает,
-- этот — для будущей выборки истории)
CREATE INDEX invites_room_id_idx ON invites(room_id);
```

### `migrations/0006_invites.down.sql`

```sql
-- 0006_invites.down.sql
-- Откат таблицы invites

DROP TABLE IF EXISTS invites;
```

### `migrations/0007_channels.up.sql`

```sql
-- 0007_channels.up.sql
-- Создание таблицы channels (text/voice внутри комнаты)

CREATE TABLE channels (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id     uuid        NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    name        text        NOT NULL,
    kind        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT channels_name_length_check CHECK (char_length(name) BETWEEN 1 AND 64),
    CONSTRAINT channels_kind_check CHECK (kind IN ('text', 'voice')),
    CONSTRAINT channels_room_id_name_key UNIQUE (room_id, name)
);

CREATE INDEX channels_room_id_idx ON channels(room_id);
```

### `migrations/0007_channels.down.sql`

```sql
-- 0007_channels.down.sql
-- Откат таблицы channels

DROP TABLE IF EXISTS channels;
```

## sqlc.yaml — расширение

Текущий `sqlc.yaml` содержит одну запись для auth. Расширяем массив `sql:`:

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
        overrides:
          - db_type: "uuid"
            nullable: false
            go_type: { import: "github.com/google/uuid", type: "UUID" }
          - db_type: "timestamptz"
            nullable: false
            go_type: "time.Time"
          - db_type: "pg_catalog.timestamptz"
            nullable: false
            go_type: "time.Time"

  - engine: "postgresql"
    schema: "migrations"
    queries: "internal/room/repository/postgres/queries"
    gen:
      go:
        sql_package: "pgx/v5"
        package: "db"
        out: "internal/room/repository/postgres/db"
        emit_interface: false
        emit_json_tags: false
        emit_db_tags: false
        emit_pointers_for_null_types: false
        emit_prepared_queries: false
        overrides:
          - db_type: "uuid"
            nullable: false
            go_type: { import: "github.com/google/uuid", type: "UUID" }
          - db_type: "timestamptz"
            nullable: false
            go_type: "time.Time"
          - db_type: "pg_catalog.timestamptz"
            nullable: false
            go_type: "time.Time"

  - engine: "postgresql"
    schema: "migrations"
    queries: "internal/channel/repository/postgres/queries"
    gen:
      go:
        sql_package: "pgx/v5"
        package: "db"
        out: "internal/channel/repository/postgres/db"
        emit_interface: false
        emit_json_tags: false
        emit_db_tags: false
        emit_pointers_for_null_types: false
        emit_prepared_queries: false
        overrides:
          - db_type: "uuid"
            nullable: false
            go_type: { import: "github.com/google/uuid", type: "UUID" }
          - db_type: "timestamptz"
            nullable: false
            go_type: "time.Time"
          - db_type: "pg_catalog.timestamptz"
            nullable: false
            go_type: "time.Time"
```

Все три блока генерируют пакет с одним именем `db`, но в разных каталогах — Go-пути не пересекаются (`internal/<domain>/repository/postgres/db`).

## Соответствие `prompts/RepoModel.txt`

- ✅ Маппинг через доменные конструкторы (`ReconstructRoom`, `ReconstructMembership`, `ReconstructInvite`, `ReconstructChannel`) — без рефлексии и приватных сеттеров.
- ✅ Два направления — две функции (`domainToXxxRow` / `xxxRowToDomain`), живут в `mapper.go` рядом с репозиторием.
- ✅ Value objects разворачиваются на границе репозитория.
- ✅ Ошибки маппинга (`ParseRole`, `NewInviteCode`) оборачиваются `fmt.Errorf("xxx row: <field>: %w", err)`.
- ✅ Маппер не делает I/O.
- ✅ Nullable `revoked_at` обработан через `pgtype.Timestamptz` (см. таблицу выше).
- ✅ `role` и `kind` — Postgres-string + домен ParseXxx.
- ✅ UUID-VO разворачиваются в `uuid.UUID` через `.UUID()`.
- ✅ `db.*` структуры наружу `usecase/` не выходят.
- ✅ Транзакции через `pgx.Tx` + `Queries.WithTx`.
