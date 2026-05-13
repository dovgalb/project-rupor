---
phase: 4
name: chat-repository
layer: repository
depends_on: [phase-01, phase-02]
plan: ./README.md
---

# Phase 4: Repository `chat/repository/postgres` + `MembershipQueryChatAdapter`

## Цель

Реализовать постгрес-репозиторий сообщений (через sqlc) и адаптер `MembershipQueryChatAdapter` для проверки прав chat-домена. После этой фазы есть SQL-уровень, готовый принимать вызовы use case'ов (которые ещё нет — будут в phase-05).

## Контекст

Phase-01 создала таблицу `messages`. Phase-02 определила домен `chat`. Sqlc-конфиг для chat ещё не подключён — это часть phase-09 (composition root); в этой фазе мы пишем `queries/messages.sql` и **локально** запускаем `make sqlc` для генерации, но изменения в `sqlc.yaml` фиксируем в phase-09.

Шаблон репозитория — `internal/channel/repository/postgres/` (4 файла: repository, mapper, pgerr, queries; плюс compile_check_test, integration_helpers_test, integration_test). См. [`../06-repo-model.md §Структура репозитория`](../06-repo-model.md).

## Файлы для создания

### `internal/chat/repository/postgres/queries/messages.sql`

**Назначение:** SQL-запросы chat-репозитория. Точно из [`../06-repo-model.md §sqlc Queries`](../06-repo-model.md):

```sql
-- name: InsertMessage :exec
INSERT INTO messages (id, channel_id, author_id, text, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetMessageByID :one
SELECT id, channel_id, author_id, text, created_at
FROM messages
WHERE id = $1;

-- name: ListMessagesBeforeCursor :many
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
SELECT id, room_id, kind
FROM channels
WHERE id = $1;
```

**Замечание:** `sqlc.narg('before_id')` — nullable parameter. После генерации sqlc создаст структуру `ListMessagesBeforeCursorParams` с полем `BeforeID pgtype.UUID`. На уровне репозитория конструируется `pgtype.UUID{Bytes: before.UUID(), Valid: !before.IsZero()}`.

### `internal/chat/repository/postgres/db/` — сгенерируется sqlc

После запуска `make sqlc` появятся:
- `internal/chat/repository/postgres/db/db.go` — `Queries`, `New(*pgxpool.Pool)`
- `internal/chat/repository/postgres/db/models.go` — `Message`, `Channel` (т.к. `GetChannelKind` ссылается на channels)
- `internal/chat/repository/postgres/db/messages.sql.go` — методы запросов

⚠️ **Внимание:** sqlc сгенерирует структуру `db.Channel` в этом пакете (потому что `GetChannelKind` использует `channels`). Это допустимо — sqlc-структуры разных пакетов независимы; дублирование структуры `Channel` между `internal/channel/repository/postgres/db/` и `internal/chat/repository/postgres/db/` — не проблема (это типизированные row-структуры, не доменные сущности). См. [`../06-repo-model.md §Альтернатива (отбрасываем)`](../06-repo-model.md).

### `internal/chat/repository/postgres/mapper.go`

**Назначение:** Свободные функции маппинга `messageRowToDomain` и `domainToInsertMessageParams`. Шаблон — `internal/channel/repository/postgres/mapper.go`.

```go
package postgres

import (
    "fmt"

    "github.com/jackc/pgx/v5/pgtype"
    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/chat/domain"
    "github.com/dovgalb/project-rupor/internal/chat/repository/postgres/db"
)

func messageRowToDomain(row db.Message) (*domain.Message, error) {
    id, err := domain.NewMessageID(row.ID)
    if err != nil {
        return nil, fmt.Errorf("message row: id: %w", err)
    }
    channelID, err := domain.NewChannelID(row.ChannelID)
    if err != nil {
        return nil, fmt.Errorf("message row: channel_id: %w", err)
    }
    authorID, err := domain.NewUserID(row.AuthorID)
    if err != nil {
        return nil, fmt.Errorf("message row: author_id: %w", err)
    }
    text, err := domain.NewMessageText(row.Text)
    if err != nil {
        return nil, fmt.Errorf("message row: text: %w", err)
    }
    return domain.ReconstructMessage(id, channelID, authorID, text, row.CreatedAt.UTC())
}

func domainToInsertMessageParams(m *domain.Message) db.InsertMessageParams {
    return db.InsertMessageParams{
        ID:        m.ID().UUID(),
        ChannelID: m.ChannelID().UUID(),
        AuthorID:  m.AuthorID().UUID(),
        Text:      m.Text().String(),
        CreatedAt: m.CreatedAt().UTC(),
    }
}

func optionalUUID(id domain.MessageID) pgtype.UUID {
    if id.IsZero() {
        return pgtype.UUID{Valid: false}
    }
    return pgtype.UUID{Bytes: id.UUID(), Valid: true}
}
```

### `internal/chat/repository/postgres/pgerr.go`

**Назначение:** Утилиты обработки pg-ошибок (unique-violation, foreign-key-violation). Шаблон — `internal/channel/repository/postgres/pgerr.go`.

```go
package postgres

import (
    "errors"

    "github.com/jackc/pgx/v5/pgconn"
)

const (
    uniqueViolationCode     = "23505"
    foreignKeyViolationCode = "23503"
)

func isUniqueViolation(err error, constraintName string) bool {
    var pgErr *pgconn.PgError
    if !errors.As(err, &pgErr) {
        return false
    }
    return pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == constraintName
}

func isForeignKeyViolation(err error, constraintName string) bool {
    var pgErr *pgconn.PgError
    if !errors.As(err, &pgErr) {
        return false
    }
    return pgErr.Code == foreignKeyViolationCode && pgErr.ConstraintName == constraintName
}
```

### `internal/chat/repository/postgres/message_repository.go`

**Назначение:** Реализация `chat/usecase.MessageRepository`. Согласован с [`../06-repo-model.md §Структура репозитория`](../06-repo-model.md).

```go
package postgres

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

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
    if err := r.q.InsertMessage(ctx, domainToInsertMessageParams(m)); err != nil {
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
        return nil, fmt.Errorf("postgres: list messages: %w", err)
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

### `internal/chat/repository/postgres/compile_check_test.go`

```go
package postgres_test

import (
    pg "github.com/dovgalb/project-rupor/internal/chat/repository/postgres"
    chatuc "github.com/dovgalb/project-rupor/internal/chat/usecase"
)

var _ chatuc.MessageRepository = (*pg.MessageRepository)(nil)
```

### `internal/chat/repository/postgres/integration_helpers_test.go`

**Назначение:** Хелперы интеграционных тестов. Расширяет шаблон из `internal/channel/repository/postgres/integration_helpers_test.go`.

```go
//go:build integration

package postgres_test

func dbConn(t *testing.T) *pgxpool.Pool {
    t.Helper()
    url := os.Getenv("TEST_DATABASE_URL")
    if url == "" { t.Skip("TEST_DATABASE_URL not set") }
    pool, err := pgxpool.New(context.Background(), url)
    if err != nil { t.Fatalf("pgxpool.New: %v", err) }
    t.Cleanup(pool.Close)
    return pool
}

func truncate(t *testing.T, pool *pgxpool.Pool) {
    t.Helper()
    _, err := pool.Exec(context.Background(),
        "TRUNCATE messages, channels, room_members, invites, rooms, refresh_tokens, users CASCADE")
    if err != nil { t.Fatalf("truncate: %v", err) }
}

// seedChannel создаёт user → room → owner-membership → channel
// (text по умолчанию; kind можно переопределить параметром).
func seedChannel(t *testing.T, pool *pgxpool.Pool, channelID, roomID, ownerID uuid.UUID, kind string) {
    t.Helper()
    // ... аналогично seedRoom в channel-репо, но добавляется INSERT INTO channels
}
```

### `internal/chat/repository/postgres/message_repository_integration_test.go`

Build-tag `integration`. 10 тестов из [`../04-testing.md §Chat / repository/postgres`](../04-testing.md).

### `internal/room/repository/postgres/membership_query_chat.go`

**Назначение:** Новый адаптер `MembershipQueryChatAdapter` (отдельная структура, не модификация существующей). Реализует `chat/usecase.MembershipQuery`. Точно из [`../06-repo-model.md §MembershipQueryChatAdapter`](../06-repo-model.md).

```go
package postgres

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    chatdom "github.com/dovgalb/project-rupor/internal/chat/domain"
    chatuc "github.com/dovgalb/project-rupor/internal/chat/usecase"
    roomdom "github.com/dovgalb/project-rupor/internal/room/domain"
    "github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
)

type MembershipQueryChatAdapter struct {
    q *db.Queries
}

func NewMembershipQueryChatAdapter(pool *pgxpool.Pool) *MembershipQueryChatAdapter {
    return &MembershipQueryChatAdapter{q: db.New(pool)}
}

func (a *MembershipQueryChatAdapter) Require(
    ctx context.Context,
    channelID chatdom.ChannelID,
    userID chatdom.UserID,
    req chatuc.RoleRequirement,
) error {
    rawRole, err := a.q.GetMemberForChannel(ctx, db.GetMemberForChannelParams{
        ID:     channelID.UUID(),
        UserID: userID.UUID(),
    })
    if err == nil {
        role, parseErr := roomdom.ParseRole(rawRole)
        if parseErr != nil {
            return fmt.Errorf("postgres: parse role: %w", parseErr)
        }
        if !chatSatisfies(req, role) {
            return chatdom.ErrChatInsufficientRole
        }
        return nil
    }
    if !errors.Is(err, pgx.ErrNoRows) {
        return fmt.Errorf("postgres: get member for channel: %w", err)
    }

    exists, err := a.q.ChannelExists(ctx, channelID.UUID())
    if err != nil {
        return fmt.Errorf("postgres: channel exists: %w", err)
    }
    if !exists {
        return chatdom.ErrChannelNotFound
    }
    return chatdom.ErrChatAccessDenied
}

func chatSatisfies(req chatuc.RoleRequirement, role roomdom.Role) bool {
    switch req {
    case chatuc.RoleAnyMember:    return role == roomdom.RoleMember || role == roomdom.RoleAdmin || role == roomdom.RoleOwner
    case chatuc.RoleAdminOrOwner: return role == roomdom.RoleAdmin || role == roomdom.RoleOwner
    case chatuc.RoleOwnerOnly:    return role == roomdom.RoleOwner
    default: return false
    }
}
```

### `internal/room/repository/postgres/membership_query_chat_integration_test.go`

Build-tag `integration`. 3 теста из [`../04-testing.md §MembershipQueryAdapter`](../04-testing.md).

## Файлы для модификации

### `internal/room/repository/postgres/queries/room_members.sql`

Добавляются два новых запроса в конец файла:

```sql
-- name: GetMemberForChannel :one
SELECT rm.role
FROM channels AS c
JOIN room_members AS rm ON rm.room_id = c.room_id
WHERE c.id = $1 AND rm.user_id = $2;

-- name: ChannelExists :one
SELECT EXISTS(SELECT 1 FROM channels WHERE id = $1) AS exists;
```

Существующие запросы (`InsertRoomMember`, `GetRoomMember`, `ListRoomMembers`, `DeleteRoomMember`) — без изменений.

### `internal/room/repository/postgres/db/room_members.sql.go`

Автоматически после `make sqlc` — добавятся методы `GetMemberForChannel`, `ChannelExists`. Существующие методы и структуры — без изменений.

## Ключевые решения

- **Дублирование sqlc-структуры `Channel` в `chat/repository/postgres/db/`** — допустимо, см. [`../06-repo-model.md §GetChannelKind alternative`](../06-repo-model.md).
- **`MembershipQueryChatAdapter` как отдельная структура** — [§ D-01](../03-decisions.md). НЕ модификация существующего `MembershipQueryAdapter` (он остаётся для channel-домена).
- **Два запроса в `Require`** в негативной ветке — приемлемо для MVP, см. [`../06-repo-model.md §SQL для chat`](../06-repo-model.md).
- **`row.CreatedAt.UTC()` в маппере** — гарантирует, что время всегда в UTC независимо от настроек pool'а.

## Verification

- [ ] `make sqlc` без ошибок (после добавления запросов и пути в `sqlc.yaml` — последнее в phase-09).
  - Локально для phase-04: можно временно добавить chat-блок в `sqlc.yaml` для тестирования; финальный commit `sqlc.yaml` в phase-09.
- [ ] `go build ./internal/chat/repository/postgres/...` без ошибок.
- [ ] `go build ./internal/room/repository/postgres/...` без ошибок.
- [ ] Compile-check `internal/chat/repository/postgres/compile_check_test.go` компилируется → `MessageRepository` реализует `MessageRepository` (использованы алиасы; phase-05 создаст этот интерфейс — если phase-05 ещё не сделана, переставить compile_check_test в phase-05).
- [ ] Интеграционные тесты `go test -tags=integration ./internal/chat/repository/postgres/...` зелёные (с `TEST_DATABASE_URL`).
- [ ] Интеграционные тесты `MembershipQueryChatAdapter` зелёные.
- [ ] Round-trip тест: создать `Message` → `Save` → `ListByChannel` → сравнить с исходным.
- [ ] `ChannelOf` для несуществующего канала возвращает `domain.ErrChannelNotFound`, не INTERNAL.

**Ordering note:** интерфейс `chat/usecase.MessageRepository` объявлен в phase-05. Compile-check теста имеет неразрешённую зависимость до phase-05. Решение: либо разбить phase-04 на 4a (SQL + queries + sqlc generation) и 4b (impl + compile_check); либо запустить phase-04 после phase-05 (но domain-фаза без use case не критична). **Рекомендация:** делать phase-04 и phase-05 как одну логическую под-итерацию: сначала ports.go из phase-05, затем repository из phase-04. План выше написан так, что implementer'у это очевидно.
