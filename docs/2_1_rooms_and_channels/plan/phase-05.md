---
phase: 5
name: Channel usecase
layer: usecase
depends_on: [phase-03]
plan: ./README.md
---

# Phase 5: Use case слой channel (3 use case + порт MembershipQuery + fakes + тесты)

## Цель

Реализовать `internal/channel/usecase/`: интерфейсы зависимостей (включая ключевой кросс-доменный порт `MembershipQuery`), 3 use case (`CreateChannel`, `ListChannels`, `DeleteChannel`), фейки, тесты.

## Контекст

Channel-usecase **не импортирует room** — связь только через порт `MembershipQuery`. Сам порт объявляется здесь, в `channel/usecase/ports.go`. Реализация порта (`MembershipQueryAdapter`) живёт в Phase 06 — `internal/room/repository/postgres/membership_query.go` — и зависит от **этого** пакета (импортирует `channel/usecase` и `channel/domain`). Это санкционированное «обратное» кросс-доменное направление, описанное в `../03-decisions.md` §D-07/D-19.

Сценарии — `../02-behavior.md` UC-C1..UC-C3. Архитектура — `../01-architecture.md` §3.2.4 (порт MembershipQuery).

## Файлы для создания

### Ports

#### `internal/channel/usecase/ports.go`

```go
package usecase

import (
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/channel/domain"
)

type ChannelRepository interface {
    Save(ctx context.Context, ch *domain.Channel) error
    ListByRoom(ctx context.Context, roomID domain.RoomID) ([]*domain.Channel, error)
    DeleteInRoom(ctx context.Context, channelID domain.ChannelID, roomID domain.RoomID) error
}

type RoleRequirement int

const (
    RoleAnyMember    RoleRequirement = iota // owner | admin | member
    RoleAdminOrOwner                        // admin | owner
    RoleOwnerOnly                           // owner
)

// MembershipQuery — кросс-доменный порт. Реализуется адаптером в room/repository/postgres.
//
// Контракт ошибок:
//   nil                                — пользователь имеет требуемую роль (или выше).
//   domain.ErrChannelAccessDenied      — пользователь не является членом комнаты.
//   domain.ErrChannelInsufficientRole  — является членом, но роль ниже требуемой.
//   обёрнутая через fmt.Errorf         — техническая ошибка (БД и т.п.).
type MembershipQuery interface {
    Require(ctx context.Context, roomID domain.RoomID, userID domain.UserID, req RoleRequirement) error
}

type Clock interface {
    Now() time.Time
}

type UUIDGenerator interface {
    New() uuid.UUID
}
```

`RoleRequirement` — нейтральный enum, не использует `room.domain.Role`. Адаптер транслирует.

### Use cases

#### `internal/channel/usecase/create_channel.go` (UC-C1)

```
type CreateChannelInput struct {
    ActorID uuid.UUID
    RoomID  uuid.UUID
    Name    string
    Kind    string
}
type CreateChannelOutput struct {
    Channel *domain.Channel
}
type CreateChannel struct {
    channels    ChannelRepository
    membership  MembershipQuery
    clock       Clock
    uuids       UUIDGenerator
}
```

Поток (по UC-C1 sequence):
1. Парсинг ID'ов.
2. Парсинг `name` через `domain.NewChannelName`.
3. Парсинг `kind` через `domain.ParseChannelKind`.
4. `membership.Require(ctx, roomID, actorID, RoleAdminOrOwner)` — может вернуть `ErrChannelAccessDenied` или `ErrChannelInsufficientRole`.
5. `chID := domain.NewChannelID(uuids.New())`.
6. `now := clock.Now()`.
7. `ch := domain.NewChannel(chID, roomID, name, kind, now)`.
8. `channels.Save(ctx, ch)` — может вернуть `ErrChannelNameAlreadyTaken`.

#### `internal/channel/usecase/list_channels.go` (UC-C2)

```
type ListChannelsInput struct {
    ActorID uuid.UUID
    RoomID  uuid.UUID
}
type ListChannelsOutput struct {
    Items []*domain.Channel
}
type ListChannels struct {
    channels   ChannelRepository
    membership MembershipQuery
}
```

Поток:
1. Парсинг.
2. `membership.Require(ctx, roomID, actorID, RoleAnyMember)`.
3. `channels.ListByRoom(ctx, roomID)`.

#### `internal/channel/usecase/delete_channel.go` (UC-C3)

```
type DeleteChannelInput struct {
    ActorID   uuid.UUID
    RoomID    uuid.UUID
    ChannelID uuid.UUID
}
type DeleteChannel struct {
    channels   ChannelRepository
    membership MembershipQuery
}
```

Поток:
1. Парсинг.
2. `membership.Require(ctx, roomID, actorID, RoleAdminOrOwner)`.
3. `channels.DeleteInRoom(ctx, channelID, roomID)` — может вернуть `ErrChannelNotFound`.

### Fakes (package `usecase_test`)

#### `internal/channel/usecase/fakes_test.go`

- `fakeChannelRepo`:
  - `data map[uuid.UUID]*domain.Channel`.
  - На `Save` проверяет дубликат `(roomID, name)` → `ErrChannelNameAlreadyTaken`.
  - `ListByRoom` фильтрует по roomID.
  - `DeleteInRoom` проверяет принадлежность; если канала нет или roomID не совпал → `ErrChannelNotFound`.
- `fakeMembershipQuery`:
  - Карта `map[mkey]RoleRequirement`-эквивалентного состояния. Но удобнее — карта `map[mkey]role` где `role` — внутренний тестовый enum (Owner/Admin/Member/None).
  - Методы-builders для тестов:
    - `withRole(roomID uuid.UUID, userID uuid.UUID, role string)` — `"owner"`/`"admin"`/`"member"`.
    - `notMember(roomID uuid.UUID, userID uuid.UUID)`.
  - `Require(...)` — switch по сохранённой роли + req. Если нет записи — `ErrChannelAccessDenied`. Если роль ниже req — `ErrChannelInsufficientRole`.
- `fixedClock`, `fixedUUID` — те же, что в Phase 04 (дублируем код — так делается в auth: `internal/auth/transport/http/setup_test.go` и `internal/auth/usecase/fakes_test.go` имеют свои копии). Упрощение через общий `internal/testfixtures` отложим до фазы рефакторинга.

### Тесты

#### `internal/channel/usecase/create_channel_test.go`
- `TestCreateChannel_AsAdmin_PersistsChannel`
- `TestCreateChannel_AsOwner_PersistsChannel`
- `TestCreateChannel_InvalidName_ReturnsErrInvalidChannelName`
- `TestCreateChannel_InvalidKind_ReturnsErrInvalidChannelKind`
- `TestCreateChannel_NotMember_ReturnsErrAccessDenied`
- `TestCreateChannel_AsMember_ReturnsErrInsufficientRole`
- `TestCreateChannel_DuplicateName_ReturnsErrChannelNameAlreadyTaken`

#### `internal/channel/usecase/list_channels_test.go`
- `TestListChannels_AsMember_ReturnsAll`
- `TestListChannels_AsOwner_ReturnsAll`
- `TestListChannels_NotMember_ReturnsErrAccessDenied`
- `TestListChannels_EmptyRoom_ReturnsEmptyItems`

#### `internal/channel/usecase/delete_channel_test.go`
- `TestDeleteChannel_AsAdmin_DeletesChannel`
- `TestDeleteChannel_AsMember_ReturnsErrInsufficientRole`
- `TestDeleteChannel_NotMember_ReturnsErrAccessDenied`
- `TestDeleteChannel_NotFound_ReturnsErrChannelNotFound`
- `TestDeleteChannel_WrongRoomID_ReturnsErrChannelNotFound` — channel существует, но `DeleteInRoom(channelID, otherRoomID)` возвращает NotFound.

## Файлы для модификации

- `internal/channel/usecase/.gitkeep` — удалить.

## Ключевые решения

- **`MembershipQuery` объявлен в channel/usecase**, не в room — порт принадлежит потребителю (Hexagonal/Ports & Adapters, `prompts/Architecture Layers.txt:11`, `prompts/Clean architecture.txt`).
- **`RoleRequirement` — нейтральный enum в channel/usecase** — не зависит от `room.domain.Role`. Адаптер делает трансляцию (Phase 06).
- **`ErrChannelAccessDenied` vs `ErrChannelInsufficientRole`** — два разных кода ошибок (CHANNEL-006, CHANNEL-007), потому что фронту нужна разница UX (`../03-decisions.md` §D-19).
- **`DeleteInRoom(channelID, roomID)` — два аргумента** — защита от подмены roomID в URL: SQL будет `WHERE id=$1 AND room_id=$2`. Это инвариант, выраженный в сигнатуре порта.

## Verification

- [ ] `internal/channel/usecase/ports.go` создан.
- [ ] 3 use-case файла созданы.
- [ ] `internal/channel/usecase/fakes_test.go` создан.
- [ ] 3 `*_test.go` файла созданы; все тесты из `../04-testing.md` §«internal/channel/usecase» присутствуют.
- [ ] `go build ./internal/channel/usecase/...` чистый.
- [ ] `go test ./internal/channel/usecase/... -race -count=1` зелёный.
- [ ] Импорты non-test файлов: stdlib + `github.com/google/uuid` + `internal/channel/domain`. **Нет** импорта `internal/room/...`.
- [ ] `golangci-lint run ./internal/channel/usecase/...` чистый.
- [ ] `internal/channel/usecase/.gitkeep` удалён.
- [ ] `go test -cover ./internal/channel/usecase/` ≥ 85%.
