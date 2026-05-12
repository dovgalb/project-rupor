---
phase: 4
name: Room usecase
layer: usecase
depends_on: [phase-02]
plan: ./README.md
---

# Phase 4: Use case слой room (7 use case + ports + fakes + unit-тесты)

## Цель

Реализовать `internal/room/usecase/`: интерфейсы зависимостей (ports), 7 use case (`CreateRoom`, `GetRoom`, `ListUserRooms`, `DeleteRoom`, `ListMembers`, `RegenerateInvite`, `JoinByCode`), фейковые реализации в тест-пакете, unit-тесты по coverage mapping из `../04-testing.md`.

## Контекст

После Phase 02 у нас есть `internal/room/domain/` со всеми сущностями и VO. Use case слой оркеструет вызовы домена и репозиториев. **Не импортирует transport, repository, sqlc — только `internal/room/domain` + stdlib + `github.com/google/uuid`** (по `prompts/Architecture Layers.txt:70`).

Сценарии и sequence-диаграммы — в `../02-behavior.md` UC-R1..UC-R7. Архитектура — `../01-architecture.md` §3.1. Эталон стиля — `internal/auth/usecase/` (см. `../research.md`).

## Файлы для создания

### Ports

#### `internal/room/usecase/ports.go`

Все интерфейсы зависимостей. Маленькие, ролевые (по `prompts/Go style.txt`):

```go
package usecase

import (
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/room/domain"
)

type RoomRepository interface {
    SaveWithOwner(ctx context.Context, room *domain.Room, ownerMembership *domain.Membership) error
    FindByID(ctx context.Context, id domain.RoomID) (*domain.Room, error)
    ListByMember(ctx context.Context, userID domain.UserID) ([]RoomWithRole, error)
    Delete(ctx context.Context, id domain.RoomID) error
}

type RoomWithRole struct {
    Room *domain.Room
    Role domain.Role
}

type MembershipRepository interface {
    FindByPair(ctx context.Context, roomID domain.RoomID, userID domain.UserID) (*domain.Membership, error)
    ListByRoom(ctx context.Context, roomID domain.RoomID) ([]*domain.Membership, error)
    Add(ctx context.Context, m *domain.Membership) error
}

type InviteRepository interface {
    // RegenerateActive в одной транзакции отзывает все активные коды комнаты
    // (UPDATE invites SET revoked_at=now WHERE room_id=$1 AND revoked_at IS NULL),
    // затем INSERT нового invite. Если новый код коллизирует по invites_active_code
    // partial unique index — возвращает ErrInviteCodeCollision (для ретрая в usecase).
    RegenerateActive(ctx context.Context, invite *domain.Invite) error
    FindActiveByCode(ctx context.Context, code domain.InviteCode) (*domain.Invite, error)
}

type InviteCodeGenerator interface {
    New() (domain.InviteCode, error)
}

type Clock interface {
    Now() time.Time
}

type UUIDGenerator interface {
    New() uuid.UUID
}

// ErrInviteCodeCollision — внутренний sentinel для управляемого ретрая в RegenerateInvite.
// Возвращается InviteRepository из RegenerateActive при unique violation на invites_active_code.
// НЕ доменная ошибка, не должна всплыть наружу transport-слоя.
var ErrInviteCodeCollision = errors.New("usecase: invite code collision (retry)")
```

`ErrInviteCodeCollision` живёт в `usecase/`, а не в `domain/`, потому что это контракт между repository-адаптером и usecase, а не доменная ошибка.

### Use cases

Каждый use case — отдельный файл с `XxxInput`, `XxxOutput`, `Xxx` структурой, `NewXxx(...)` конструктором, единственным методом `Execute(ctx, in) (out, error)`.

#### `internal/room/usecase/create_room.go` (UC-R1, см. `../02-behavior.md`)

```
type CreateRoomInput struct {
    ActorID uuid.UUID
    Name    string
}

type CreateRoomOutput struct {
    Room *domain.Room
}

type CreateRoom struct {
    rooms RoomRepository
    clock Clock
    uuids UUIDGenerator
}

func NewCreateRoom(rooms RoomRepository, clock Clock, uuids UUIDGenerator) *CreateRoom
func (uc *CreateRoom) Execute(ctx context.Context, in CreateRoomInput) (CreateRoomOutput, error)
```

Поток (по UC-R1 sequence в `../02-behavior.md`):
1. `domain.NewUserID(in.ActorID)` — если zero, доменная ошибка.
2. `domain.NewRoomName(in.Name)` — может вернуть `ErrInvalidRoomName`.
3. `roomID := domain.NewRoomID(uuids.New())`.
4. `now := clock.Now()`.
5. `room := domain.NewRoom(roomID, actorID, name, now)`.
6. `owner := domain.NewMembership(roomID, actorID, domain.RoleOwner, now)`.
7. `rooms.SaveWithOwner(ctx, room, owner)` — обернуть техническую ошибку через `fmt.Errorf("usecase: create_room: %w", err)`.
8. Вернуть `CreateRoomOutput{Room: room}`.

#### `internal/room/usecase/get_room.go` (UC-R2)

```
type GetRoomInput struct {
    ActorID uuid.UUID
    RoomID  uuid.UUID
}
type GetRoomOutput struct {
    Room *domain.Room
}
type GetRoom struct {
    rooms       RoomRepository
    memberships MembershipRepository
}
```

Поток:
1. Парсить ID'ы через `domain.NewRoomID`/`domain.NewUserID`.
2. `m, err := memberships.FindByPair(ctx, roomID, actorID)` — `ErrNotMember` пробрасываем как есть.
3. `room, err := rooms.FindByID(ctx, roomID)` — `ErrRoomNotFound` пробрасываем.
4. Вернуть.

Не используем `m` для проверки прав — `CanReadRoom() == true` для всех ролей. Просто факт существования membership подтверждает доступ.

#### `internal/room/usecase/list_user_rooms.go` (UC-R3)

```
type ListUserRoomsInput struct {
    ActorID uuid.UUID
}
type ListUserRoomsOutput struct {
    Items []RoomWithRole  // тип из ports.go
}
type ListUserRooms struct {
    rooms RoomRepository
}
```

Поток: парсинг → `rooms.ListByMember(ctx, userID)`. Пустой список — нормально.

#### `internal/room/usecase/delete_room.go` (UC-R4)

```
type DeleteRoomInput struct {
    ActorID uuid.UUID
    RoomID  uuid.UUID
}
type DeleteRoom struct {
    rooms       RoomRepository
    memberships MembershipRepository
}
```

Поток:
1. Парсинг ID'ов.
2. `m := memberships.FindByPair(ctx, roomID, actorID)` — `ErrNotMember`.
3. `if !m.CanDeleteRoom() → return domain.ErrInsufficientRole` (специально не ErrNotMember — у пользователя есть membership).
4. `rooms.Delete(ctx, roomID)` — `ErrRoomNotFound`.

#### `internal/room/usecase/list_members.go` (UC-R5)

```
type ListMembersInput struct {
    ActorID uuid.UUID
    RoomID  uuid.UUID
}
type ListMembersOutput struct {
    Items []*domain.Membership
}
type ListMembers struct {
    memberships MembershipRepository
}
```

Поток:
1. Парсинг.
2. `actorMembership := memberships.FindByPair(ctx, roomID, actorID)` — `ErrNotMember`.
3. (`CanReadMembers()` true для всех ролей — проверка не нужна.)
4. `memberships.ListByRoom(ctx, roomID)`.

#### `internal/room/usecase/regenerate_invite.go` (UC-R6)

```
type RegenerateInviteInput struct {
    ActorID uuid.UUID
    RoomID  uuid.UUID
}
type RegenerateInviteOutput struct {
    Invite *domain.Invite
}
type RegenerateInvite struct {
    invites     InviteRepository
    memberships MembershipRepository
    codes       InviteCodeGenerator
    clock       Clock
    uuids       UUIDGenerator
}

const maxInviteCodeRetries = 3
```

Поток:
1. Парсинг.
2. `m := memberships.FindByPair(ctx, roomID, actorID)` — `ErrNotMember`.
3. `if !m.CanGenerateInvite() → return domain.ErrInsufficientRole`.
4. Цикл до `maxInviteCodeRetries`:
   - `code := codes.New()`.
   - `inviteID := domain.NewInviteID(uuids.New())`.
   - `invite := domain.NewInvite(inviteID, roomID, code, actorID, clock.Now())`.
   - `err := invites.RegenerateActive(ctx, invite)`.
   - Если `err == ErrInviteCodeCollision` — продолжить цикл; иначе — вернуть.
5. После 3 ретраев — `fmt.Errorf("usecase: regenerate_invite: max retries exceeded: %w", err)` (превратится в INTERNAL 500).

#### `internal/room/usecase/join_by_code.go` (UC-R7)

```
type JoinByCodeInput struct {
    ActorID uuid.UUID
    Code    string
}
type JoinByCodeOutput struct {
    Room *domain.Room
}
type JoinByCode struct {
    invites     InviteRepository
    memberships MembershipRepository
    rooms       RoomRepository
    clock       Clock
}
```

Поток:
1. `actorID := domain.NewUserID(in.ActorID)`.
2. `code := domain.NewInviteCode(in.Code)` — `ErrInvalidInviteCode`.
3. `invite := invites.FindActiveByCode(ctx, code)` — `ErrInviteNotFound`.
4. `existing, err := memberships.FindByPair(ctx, invite.RoomID(), actorID)`:
   - Если `err == nil` (membership уже есть) → `ErrAlreadyMember`.
   - Если `err == ErrNotMember` → продолжаем.
   - Иначе — обернуть и вернуть.
5. `m := domain.NewMembership(invite.RoomID(), actorID, domain.RoleMember, clock.Now())`.
6. `memberships.Add(ctx, m)` — может вернуть `ErrAlreadyMember` (race), пробросить.
7. `room := rooms.FindByID(ctx, invite.RoomID())` — `ErrRoomNotFound` пробросить.
8. Вернуть room.

### Fakes (package `usecase_test`)

#### `internal/room/usecase/fakes_test.go`

По образцу `internal/auth/usecase/fakes_test.go:14-268`. Все ручные структуры с явным состоянием:

- `fakeRoomRepo`:
  - Поля: `rooms map[uuid.UUID]*domain.Room`, `memberships map[mkey]*domain.Membership`, `saveWithOwnerErr error` (для теста ошибок), `mu sync.Mutex` (тесты `t.Parallel()` не разделяют один экземпляр).
  - Методы: `SaveWithOwner`, `FindByID` (`ErrRoomNotFound` если нет), `ListByMember`, `Delete` (`ErrRoomNotFound` если нет).
- `fakeMembershipRepo`:
  - Поля: `data map[mkey]*domain.Membership`.
  - `FindByPair`, `ListByRoom`, `Add` — `Add` проверяет дубликат → `ErrAlreadyMember`.
- `fakeInviteRepo`:
  - Поля: `invites map[uuid.UUID]*domain.Invite`, `nextRegenCollisions int` (счётчик принудительных коллизий для теста ретраев).
  - `RegenerateActive` — отзывает старые активные для room_id, INSERT нового; если `nextRegenCollisions > 0` — декрементирует и возвращает `ErrInviteCodeCollision`.
  - `FindActiveByCode` — линейный поиск по `invites` с фильтром `IsActive() && code == ...`.
- `fakeInviteCodeGen`:
  - Поле `queue []string`. `New()` отдаёт следующий код из очереди (через `domain.NewInviteCode`); если очередь пуста — детерминированно выдаёт `"AAAAAAAA"`.
- `fixedClock` — `time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)`.
- `fixedUUID` — последовательность UUID, выдаёт по очереди.
- Хелперы: `mustRoomID(t, uuid.UUID)`, `mustUserID(t, uuid.UUID)`, `mustRoomName(t, string)`, `mustInviteCode(t, string)`.

### Тесты (по одному файлу на use case)

Каждый тест-файл в `package usecase_test`. SUT-конструктор внутри файла:

```go
type createRoomSUT struct {
    rooms *fakeRoomRepo
    clock *fixedClock
    uuids *fixedUUID
    uc    *usecase.CreateRoom
}

func newCreateRoomSUT(t *testing.T) *createRoomSUT {
    t.Helper()
    rooms := newFakeRoomRepo()
    clock := newFixedClock()
    uuids := newFixedUUID(uuid.MustParse("11111111-..."))
    return &createRoomSUT{
        rooms: rooms,
        clock: clock,
        uuids: uuids,
        uc:    usecase.NewCreateRoom(rooms, clock, uuids),
    }
}
```

#### `internal/room/usecase/create_room_test.go`
- `TestCreateRoom_Valid_PersistsRoomAndOwnerMembership`
- `TestCreateRoom_InvalidName_ReturnsErrInvalidRoomName`
- `TestCreateRoom_RepoFails_ReturnsWrappedError`
- `TestCreateRoom_UsesInjectedClockAndUUID`

#### `internal/room/usecase/get_room_test.go`
- `TestGetRoom_AsMember_ReturnsRoom`
- `TestGetRoom_NotMember_ReturnsErrNotMember`
- `TestGetRoom_RoomDeletedAfterMembershipCheck_ReturnsErrRoomNotFound` (race-edge)

#### `internal/room/usecase/list_user_rooms_test.go`
- `TestListUserRooms_ReturnsRoomsWithRoles`
- `TestListUserRooms_NoRooms_ReturnsEmptyItems`

#### `internal/room/usecase/delete_room_test.go`
- `TestDeleteRoom_AsOwner_DeletesRoom`
- `TestDeleteRoom_AsAdmin_ReturnsErrInsufficientRole`
- `TestDeleteRoom_AsMember_ReturnsErrInsufficientRole`
- `TestDeleteRoom_NotMember_ReturnsErrNotMember`

#### `internal/room/usecase/list_members_test.go`
- `TestListMembers_AsMember_ReturnsAll`
- `TestListMembers_AsAdmin_ReturnsAll`
- `TestListMembers_NotMember_ReturnsErrNotMember`

#### `internal/room/usecase/regenerate_invite_test.go`
- `TestRegenerateInvite_AsAdmin_RevokesOldAndCreatesNew`
- `TestRegenerateInvite_AsOwner_Succeeds`
- `TestRegenerateInvite_AsMember_ReturnsErrInsufficientRole`
- `TestRegenerateInvite_NotMember_ReturnsErrNotMember`
- `TestRegenerateInvite_FirstCodeCollides_RetriesAndSucceeds` — `fakeInviteRepo.nextRegenCollisions = 1`, проверить что в результате 2 вызова `codes.New()` и итоговый код — второй из очереди.
- `TestRegenerateInvite_AllRetriesCollide_ReturnsWrappedError` — `nextRegenCollisions = 5`, проверить что после 3 попыток возвращается обёрнутая ошибка.

#### `internal/room/usecase/join_by_code_test.go`
- `TestJoinByCode_NewMember_AddsMembershipAndReturnsRoom`
- `TestJoinByCode_AlreadyMember_ReturnsErrAlreadyMember`
- `TestJoinByCode_NoActiveInvite_ReturnsErrInviteNotFound`
- `TestJoinByCode_InvalidFormat_ReturnsErrInvalidInviteCode`
- `TestJoinByCode_RaceWithDuplicateInsert_ReturnsErrAlreadyMember` — Add возвращает ErrAlreadyMember, пробросить.

## Файлы для модификации

- `internal/room/usecase/.gitkeep` — удалить.

## Ключевые решения

- **`ErrInviteCodeCollision` живёт в `usecase/ports.go`**, не в `domain/`. Это контракт между repo-адаптером и usecase для управляемого ретрая (`../03-decisions.md` §D-04, §D-12).
- **`SaveWithOwner` принимает оба объекта** — единая транзакционная операция, не два метода. Это инкапсулирует инвариант «room + owner membership atomic» в repository (`../03-decisions.md` §D-11).
- **`RoomWithRole` — отдельный value-type в ports.go** — usecase возвращает доменную сущность плюс роль текущего пользователя (которая семантически принадлежит контексту запроса, не самой комнате). Альтернатива (`map[RoomID]Role`) хуже читается.
- **Никаких `panic`** — все ошибки возвращаются.
- **Все методы принимают `context.Context` первым параметром** — `prompts/Go style.txt`.
- **Фейки моделируют только то, что нужно для тестов** — никакого `ListByCode`, никаких лишних методов.

## Verification

- [ ] `internal/room/usecase/ports.go` создан.
- [ ] 7 use-case файлов созданы.
- [ ] `internal/room/usecase/fakes_test.go` создан.
- [ ] 7 `*_test.go` файлов созданы; все тесты из `../04-testing.md` §«internal/room/usecase» присутствуют.
- [ ] `go build ./internal/room/usecase/...` чистый.
- [ ] `go test ./internal/room/usecase/... -race -count=1` зелёный.
- [ ] `go vet ./internal/room/usecase/...` и `golangci-lint run ./internal/room/usecase/...` чисто.
- [ ] Импорты non-test файлов: только stdlib + `github.com/google/uuid` + `internal/room/domain`. Никаких repo, transport, channel.
- [ ] `internal/room/usecase/.gitkeep` удалён.
- [ ] `go test -cover ./internal/room/usecase/` ≥ 85%.
