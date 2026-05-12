---
phase: 7
name: Channel repository (postgres)
layer: repository
depends_on: [phase-01, phase-05]
plan: ./README.md
---

# Phase 7: Channel repository postgres (ChannelRepository + queries + integration tests)

## Цель

Реализовать `internal/channel/repository/postgres/`: SQL-запрос, единственный адаптер `ChannelRepository`, маппинг row↔domain, обработку ошибок Postgres, integration-тесты.

## Контекст

После Phase 01 у нас есть таблица `channels` и пустой пакет `internal/channel/repository/postgres/db/`. После Phase 05 есть порт `ChannelRepository`. Реализация — простая, без транзакций; все операции через `*db.Queries`.

Channel-repo **не зависит от room** — все права уже проверены в usecase через `MembershipQuery`. Здесь — только CRUD каналов.

Маппинг и SQL — `../06-repo-model.md` §«internal/channel/repository/postgres/queries/channels.sql».

## Файлы для создания

### SQL queries

#### `internal/channel/repository/postgres/queries/channels.sql`
Содержимое — точно как в `../06-repo-model.md` §«channels.sql». Три запроса: `InsertChannel`, `ListChannelsByRoom`, `DeleteChannelInRoom` (`:execrows`).

После сохранения — `make sqlc`.

### Утилиты

#### `internal/channel/repository/postgres/pgerr.go`
По образцу `internal/auth/repository/postgres/pgerr.go:9-17`. Дублируем `isUniqueViolation` — **не** импортируем из room/repo (это нарушение слоёв; маленький приватный helper лучше повторить).

```go
package postgres  // alias channelpg в main

import (
    "errors"

    "github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error, constraint string) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
    }
    return false
}
```

### Mapper

#### `internal/channel/repository/postgres/mapper.go`
Одна пара функций (Channel ↔ row):

```go
func channelRowToDomain(row db.Channel) (*domain.Channel, error) {
    id, err := domain.NewChannelID(row.ID)
    if err != nil { return nil, fmt.Errorf("channel row: id: %w", err) }
    roomID, err := domain.NewRoomID(row.RoomID)
    if err != nil { return nil, fmt.Errorf("channel row: room_id: %w", err) }
    name, err := domain.NewChannelName(row.Name)
    if err != nil { return nil, fmt.Errorf("channel row: name: %w", err) }
    kind, err := domain.ParseChannelKind(row.Kind)
    if err != nil { return nil, fmt.Errorf("channel row: kind: %w", err) }
    return domain.ReconstructChannel(id, roomID, name, kind, row.CreatedAt)
}

func domainToInsertChannelParams(c *domain.Channel) db.InsertChannelParams {
    return db.InsertChannelParams{
        ID:        c.ID().UUID(),
        RoomID:    c.RoomID().UUID(),
        Name:      c.Name().String(),
        Kind:      c.Kind().String(),
        CreatedAt: c.CreatedAt(),
    }
}
```

#### `internal/channel/repository/postgres/mapper_test.go`
- `TestChannelMapper_RoundTrip_BothKinds` — табличный по text/voice.

### Repository

#### `internal/channel/repository/postgres/channel_repository.go`

```go
type ChannelRepository struct {
    q *db.Queries
}

func NewChannelRepository(q *db.Queries) *ChannelRepository
```

Методы:

- **`Save(ctx, ch)`** — `q.InsertChannel(ctx, domainToInsertChannelParams(ch))`. На `isUniqueViolation(err, "channels_room_id_name_key")` → `domain.ErrChannelNameAlreadyTaken`. Прочее — обернуть `fmt.Errorf("channel repo: insert: %w", err)`.
- **`ListByRoom(ctx, roomID)`** — `q.ListChannelsByRoom(ctx, roomID.UUID())`. Маппинг каждой row через `channelRowToDomain`.
- **`DeleteInRoom(ctx, channelID, roomID)`** — `q.DeleteChannelInRoom(ctx, db.DeleteChannelInRoomParams{ID: channelID.UUID(), RoomID: roomID.UUID()})`. Возвращает `int64` (rows affected). Если 0 → `domain.ErrChannelNotFound`.

### Compile-check

#### `internal/channel/repository/postgres/compile_check_test.go`

```go
package postgres_test

import (
    pg "github.com/dovgalb/project-rupor/internal/channel/repository/postgres"
    chuc "github.com/dovgalb/project-rupor/internal/channel/usecase"
)

var _ chuc.ChannelRepository = (*pg.ChannelRepository)(nil)
```

### Integration tests (build tag `integration`)

#### `internal/channel/repository/postgres/integration_helpers_test.go`
Аналогично room: `newTestPool`, `truncate`, `t.Skip(... see issue 1.5)`. Truncate включает `channels` и `rooms` (для FK).

Поскольку channel зависит от room (FK), helper `seedRoom(t, pool, roomID, ownerID)` пишет напрямую SQL-ом (или через `roompg.NewRoomRepository(pool).SaveWithOwner`, но это создаёт зависимость от room/repo в тестах channel-repo — лучше прямой INSERT для тестовой изоляции). Решение: прямой SQL `INSERT INTO rooms (id, owner_id, name, created_at) VALUES (...)` + `INSERT INTO room_members (...)` в helper'е.

#### `internal/channel/repository/postgres/channel_repository_integration_test.go`
- `TestChannelRepositoryIntegration_Save_Persists` — INSERT и проверка `ListByRoom` находит канал.
- `TestChannelRepositoryIntegration_UniqueNamePerRoom` — два канала с одинаковым `(room_id, name)` → второй падает с `ErrChannelNameAlreadyTaken`.
- `TestChannelRepositoryIntegration_DeleteInRoom_NotFound_ReturnsErr` — DELETE несуществующего → `ErrChannelNotFound`.
- `TestChannelRepositoryIntegration_DeleteInRoom_WrongRoom_ReturnsErr` — channel есть, но roomID не совпадает → `ErrChannelNotFound`.
- `TestChannelRepositoryIntegration_RoomDelete_CascadesChannels` — DELETE FROM rooms → channels исчезают.

## Файлы для модификации

- `internal/channel/repository/postgres/.gitkeep` — удалён в Phase 01.

## Ключевые решения

- **`pgerr.go` дублирован, не импортирован из room** — приватный helper из 9 строк не стоит кросс-доменной связи. Альтернатива (вынести в `pkg/dbutil/`) обсудима в будущем; пока локальная копия (`prompts/Go style.txt` — «явное лучше неявного»).
- **`ChannelRepository` не имеет транзакций** — единственная мутация (`DeleteInRoom`) атомарна одним SQL.
- **`ListByRoom` без пагинации** — каналов в комнате типично < 100. При росте — добавим LIMIT/OFFSET без breaking change на уровне порта.
- **Тестовый seed для room — прямой SQL** — избегаем зависимости от `room/repository` в тестах channel-repo (изоляция теста).

## Verification

- [ ] `queries/channels.sql` создан, `make sqlc` отрабатывает; в `db/` появился `channels.sql.go`.
- [ ] `pgerr.go`, `mapper.go`, `channel_repository.go` созданы.
- [ ] `compile_check_test.go` компилируется.
- [ ] `mapper_test.go` зелёный.
- [ ] `go build ./internal/channel/repository/postgres/...` чистый.
- [ ] `go vet`, `golangci-lint` чисто.
- [ ] При поднятом Postgres и `TEST_DATABASE_URL`: `go test -tags=integration ./internal/channel/repository/postgres/...` зелёный (или `t.Skip`).
- [ ] Импорты non-test файлов: stdlib + `pgx`, `pgconn` + свой usecase/domain. **Нет** импорта `internal/room/...` или `internal/auth/...`.
