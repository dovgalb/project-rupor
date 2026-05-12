---
parent: ./README.md
---

# 06 — Repository Model (Postgres + sqlc)

Описание физической модели хранения сообщений и sqlc-запросов. Соответствует `prompts/RepoModel.txt` и шаблонам `internal/channel/repository/postgres/`.

## Маппинг сущность ↔ модель БД

### Таблица `messages`

| Поле сущности (VO) | Поле модели (raw) | Тип Postgres | Конверсия domain → db | Конверсия db → domain |
|---|---|---|---|---|
| `Message.ID()` (`MessageID`) | `id` | `uuid NOT NULL PRIMARY KEY DEFAULT gen_random_uuid()` | `m.ID().UUID()` | `domain.NewMessageID(row.ID)` |
| `Message.ChannelID()` (`ChannelID`) | `channel_id` | `uuid NOT NULL REFERENCES channels(id) ON DELETE CASCADE` | `m.ChannelID().UUID()` | `domain.NewChannelID(row.ChannelID)` |
| `Message.AuthorID()` (`UserID`) | `author_id` | `uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE` | `m.AuthorID().UUID()` | `domain.NewUserID(row.AuthorID)` |
| `Message.Text()` (`MessageText`) | `text` | `text NOT NULL` | `m.Text().String()` | `domain.NewMessageText(row.Text)` |
| `Message.CreatedAt()` (`time.Time`) | `created_at` | `timestamptz NOT NULL DEFAULT now()` | `m.CreatedAt()` | `row.CreatedAt` (уже `time.Time`) |

Все uuid-конверсии через тип `uuid.UUID` из `github.com/google/uuid`, как в existing-доменах. CreatedAt всегда UTC (`row.CreatedAt.UTC()` в маппере).

Repo model отдельной структуры (`messageRow`) не нужна — sqlc-сгенерированная `db.Message` подходит one-to-one. Маппинг через свободные функции `messageRowToDomain` / `domainToInsertMessageParams` (паттерн `internal/channel/repository/postgres/mapper.go`).

---

## Миграция

### `migrations/0008_messages.up.sql`

```sql
-- 0008_messages.up.sql
-- Создание таблицы messages (сообщения в text-каналах).

CREATE TABLE messages (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id  uuid        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id   uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT messages_text_length_check CHECK (char_length(text) BETWEEN 1 AND 16000)
);

-- Cursor pagination + чтение последних N сообщений канала.
CREATE INDEX messages_channel_created_id_idx
    ON messages (channel_id, created_at DESC, id DESC);

-- Поиск всех сообщений автора (на будущее: «мои сообщения», антиспам).
CREATE INDEX messages_author_id_idx
    ON messages (author_id);
```

**Замечания:**

- `CHECK char_length(text) BETWEEN 1 AND 16000` — защита на уровне БД (16000 байт ≈ верхняя оценка для 4000 рун UTF-8). Domain-валидация (`MessageText`) более строгая (1..4000 рун). Defense-in-depth.
- Композитный индекс `(channel_id, created_at DESC, id DESC)` обслуживает запрос `ListMessagesBeforeCursor` без отдельной сортировки.
- `ON DELETE CASCADE` по `channel_id` и `author_id` — при удалении канала или пользователя сообщения каскадно удаляются. Это согласовано с тем, как ведут себя `channels` и `room_members` в существующих миграциях (`migrations/0005_room_members.up.sql`, `migrations/0007_channels.up.sql`).

### `migrations/0008_messages.down.sql`

```sql
-- 0008_messages.down.sql
DROP TABLE IF EXISTS messages;
```

---

## sqlc Queries

Файл: `internal/chat/repository/postgres/queries/messages.sql`.

```sql
-- name: InsertMessage :exec
INSERT INTO messages (id, channel_id, author_id, text, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetMessageByID :one
SELECT id, channel_id, author_id, text, created_at
FROM messages
WHERE id = $1;

-- name: ListMessagesBeforeCursor :many
-- Возвращает до $3 сообщений канала, строго старше курсора $2.
-- Если $2 = nil/zero, возвращает самые последние $3 сообщений.
SELECT m.id, m.channel_id, m.author_id, m.text, m.created_at
FROM messages AS m
WHERE m.channel_id = $1
  AND (
    sqlc.narg('before_id')::uuid IS NULL
    OR (m.created_at, m.id) < (
        SELECT b.created_at, b.id FROM messages AS b WHERE b.id = sqlc.narg('before_id')
    )
  )
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg('limit')::int;

-- name: GetChannelKind :one
-- Резолвит room_id и kind канала за один round-trip.
-- Используется для проверки типа канала перед записью сообщения.
SELECT id, room_id, kind
FROM channels
WHERE id = $1;
```

**Замечания:**

- `sqlc.narg('before_id')` (nullable arg) поддерживается sqlc через `omit_unchanged_update_columns` / `nullable params`. Параметр становится `pgtype.UUID` или `*uuid.UUID` в Go (зависит от настройки `emit_pointers_for_null_types`, которая в проекте `false`, — поэтому будет `pgtype.UUID`). Маппер репозитория собирает аргумент: `pgtype.UUID{Bytes: ..., Valid: !before.IsZero()}`.
- `GetChannelKind` живёт в chat-queries, а не в channel-queries, потому что используется только chat'ом. Это допустимая локальная денормализация SQL-уровня — структура `channels` сгенерируется в `internal/chat/repository/postgres/db/models.go` дополнительно к существующей в channel-репо.

  **Альтернатива (отбрасываем):** добавить `GetChannelKind` в `internal/channel/repository/postgres/queries/channels.sql` и сделать новый порт `channel/usecase.ChannelKindReader`, реализованный в channel-репо, и потребляемый chat'ом. Минусы: chat начинает зависеть от channel/usecase, что нарушает D-01. Поэтому SQL-запрос дублируется (повторение SQL, не повторение бизнес-логики — приемлемо).

---

## `MembershipQueryChatAdapter` — отдельная структура

Файл: `internal/room/repository/postgres/membership_query.go`. Существующий тип `MembershipQueryAdapter` (для channel-домена) НЕ меняется. Добавляется новая структура `MembershipQueryChatAdapter`, реализующая `chat/usecase.MembershipQuery`:

```go
// Существующий тип сохраняется без изменений:
type MembershipQueryAdapter struct{ pool *pgxpool.Pool }

func NewMembershipQueryAdapter(pool *pgxpool.Pool) *MembershipQueryAdapter { ... }

func (a *MembershipQueryAdapter) Require(
    ctx context.Context,
    roomID chdom.RoomID,
    userID chdom.UserID,
    req chuc.RoleRequirement,
) error { /* существующая логика */ }

// Новая структура для chat-домена (рядом, в том же пакете):
type MembershipQueryChatAdapter struct{ pool *pgxpool.Pool }

func NewMembershipQueryChatAdapter(pool *pgxpool.Pool) *MembershipQueryChatAdapter {
    return &MembershipQueryChatAdapter{pool: pool}
}

func (a *MembershipQueryChatAdapter) Require(
    ctx context.Context,
    channelID chatdom.ChannelID,
    userID chatdom.UserID,
    req chatuc.RoleRequirement,
) error { /* see SQL below */ }
```

**Подход:** две отдельные структуры в одном пакете `internal/room/repository/postgres/`. Каждая реализует «свой» порт (`channel/usecase.MembershipQuery` либо `chat/usecase.MembershipQuery`). Биндинг в `cmd/server/main.go`:

```go
membershipQueryForChannel := roompg.NewMembershipQueryAdapter(pool)
membershipQueryForChat    := roompg.NewMembershipQueryChatAdapter(pool)
```

Преимущества vs объединённой структуры:
- Нет коллизии имён метода `Require` (разные сигнатуры → один тип их не сможет иметь без переименования).
- `arch_test.go` проверяет каждый тип независимо.
- Один SQL-запрос в каждом адаптере, без shared state.

arch_test.go разрешает `room/repository/postgres` импортировать `channel/usecase`, `channel/domain`, `chat/usecase`, `chat/domain` (расширяется на chat по аналогии с уже существующим разрешением для channel).

### SQL для chat (новый запрос)

Файл: `internal/room/repository/postgres/queries/room_members.sql` пополняется:

```sql
-- name: GetMemberForChannel :one
-- Резолвит role пользователя в комнате указанного канала за один запрос.
-- Возвращает 0 строк, если канал не существует ИЛИ пользователь не член.
SELECT rm.role
FROM channels AS c
JOIN room_members AS rm ON rm.room_id = c.room_id
WHERE c.id = $1 AND rm.user_id = $2;

-- name: ChannelExists :one
-- Дисамбигуация: канал не существует vs пользователь не член.
SELECT EXISTS(SELECT 1 FROM channels WHERE id = $1) AS exists;
```

**Логика `MembershipQueryChatAdapter.Require`:**

1. Вызвать `GetMemberForChannel(channelID, userID)`.
2. Если возвращена строка → парсить role, проверить `satisfies(req, role)` → `nil` либо `chatdom.ErrChatInsufficientRole`.
3. Если `pgx.ErrNoRows` → вызвать `ChannelExists(channelID)`.
4. Если `exists == false` → `chatdom.ErrChannelNotFound`.
5. Если `exists == true` → `chatdom.ErrChatAccessDenied` (канал есть, пользователь не член).
6. Любая другая ошибка → `fmt.Errorf("postgres: chat membership require: %w", err)`.

Возвращаемые состояния:

| Условие | Возврат |
|---|---|
| Член комнаты канала, роль удовлетворяет `req` | `nil` |
| Член комнаты, роль ниже `req` | `chatdom.ErrChatInsufficientRole` (зарезервировано; в фазе 3 chat использует только `RoleAnyMember`) |
| Канал существует, не член | `chatdom.ErrChatAccessDenied` |
| Канал не существует | `chatdom.ErrChannelNotFound` |
| Технический сбой БД | обёрнутая `fmt.Errorf` |

В типичном пути (член комнаты) делается ОДИН запрос. Второй запрос (`ChannelExists`) выполняется только в негативной ветке, что приемлемо.

**Альтернатива (отбрасываем):** один SQL с `LEFT JOIN room_members` и проверкой `channel.id IS NOT NULL AND rm.user_id IS NULL`. Это компактнее (1 запрос всегда), но даёт менее очевидный код и сложнее unit-тестируется. Для MVP — двухзапросный путь.

---

## sqlc.yaml — изменения

В `sqlc.yaml` добавляется 4-й блок:

```yaml
  - engine: "postgresql"
    schema: "migrations"
    queries: "internal/chat/repository/postgres/queries"
    gen:
      go:
        sql_package: "pgx/v5"
        package: "db"
        out: "internal/chat/repository/postgres/db"
        emit_interface: false
        emit_json_tags: false
        emit_db_tags: false
        emit_pointers_for_null_types: false
        emit_prepared_queries: false
        overrides:
          - db_type: "uuid"
            nullable: false
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
          - db_type: "timestamptz"
            nullable: false
            go_type: "time.Time"
          - db_type: "pg_catalog.timestamptz"
            nullable: false
            go_type: "time.Time"
```

Дополнительно `room` блок не меняется — `GetMemberForChannel` живёт там же, где `GetRoomMember`.

---

## Структура репозитория

```go
// internal/chat/repository/postgres/message_repository.go

package postgres

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v5/pgtype"

    "github.com/dovgalb/project-rupor/internal/chat/domain"
    "github.com/dovgalb/project-rupor/internal/chat/repository/postgres/db"
    "github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type MessageRepository struct {
    q *db.Queries
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
    return &MessageRepository{q: db.New(pool)}
}

func (r *MessageRepository) Save(ctx context.Context, m *domain.Message) error {
    err := r.q.InsertMessage(ctx, domainToInsertMessageParams(m))
    if err != nil {
        return fmt.Errorf("postgres: insert message: %w", err)
    }
    return nil
}

func (r *MessageRepository) ListByChannel(
    ctx context.Context,
    channelID domain.ChannelID,
    before domain.MessageID,
    limit int,
) ([]*domain.Message, error) {
    rows, err := r.q.ListMessagesBeforeCursor(ctx, db.ListMessagesBeforeCursorParams{
        ChannelID: channelID.UUID(),
        BeforeID:  optionalUUID(before),
        Limit:     int32(limit),
    })
    if err != nil {
        return nil, fmt.Errorf("postgres: list messages before cursor: %w", err)
    }
    out := make([]*domain.Message, 0, len(rows))
    for _, row := range rows {
        m, mErr := messageRowToDomain(row)
        if mErr != nil {
            return nil, mErr
        }
        out = append(out, m)
    }
    return out, nil
}

func (r *MessageRepository) ChannelOf(
    ctx context.Context,
    channelID domain.ChannelID,
) (usecase.ChannelInfo, error) {
    row, err := r.q.GetChannelKind(ctx, channelID.UUID())
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return usecase.ChannelInfo{}, domain.ErrChannelNotFound
        }
        return usecase.ChannelInfo{}, fmt.Errorf("postgres: get channel kind: %w", err)
    }
    roomID, _ := domain.NewRoomID(row.RoomID)
    return usecase.ChannelInfo{
        ChannelID: channelID,
        RoomID:    roomID,
        Kind:      row.Kind,
    }, nil
}
```

`usecase.ChannelInfo` — простой struct в `chat/usecase` (имеет VO-поля). `optionalUUID(before)` — helper, который возвращает `pgtype.UUID` с `Valid: !before.IsZero()`.

---

## Соответствие `prompts/RepoModel.txt`

| Правило | Где соблюдено |
|---|---|
| Repo model не нужна, если sqlc один-к-одному | `db.Message` подходит, своя структура не вводится |
| Маппинг через `Reconstruct` | `messageRowToDomain` вызывает `domain.NewMessage(...)` (т.к. `ReconstructMessage` = алиас, как в `internal/channel/domain/channel.go:41-49`) |
| Два направления — две функции | `messageRowToDomain` + `domainToInsertMessageParams` |
| VO разворачиваются на границе | `m.Text().String()` ↔ `domain.NewMessageText(row.Text)` |
| Ошибки маппинга — баг данных | `messageRowToDomain` возвращает обёрнутую ошибку с контекстом `"message row: text: %w"` |
| Маппер не делает I/O | Чистая функция от row к entity |
| Nullable-поля | `BeforeID` через `pgtype.UUID` (single nullable param) |
| Enum-ы | Kind возвращается как string из БД, парсится в usecase / repo при необходимости |
| Id-ы | `domain.NewMessageID(row.ID)` ↔ `m.ID().UUID()` |
| `sql.ErrNoRows` → доменная ошибка | `ChannelOf` мапит в `domain.ErrChannelNotFound` |
| Unique violation → доменная ошибка | На этой таблице нет уникальных constraint'ов, кроме PK; повторный insert с тем же UUID — INTERNAL (UUID-конфликты не должны случаться при правильной генерации) |
