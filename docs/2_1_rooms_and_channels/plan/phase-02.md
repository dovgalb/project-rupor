---
phase: 2
name: Room domain
layer: domain
depends_on: [phase-01]
plan: ./README.md
---

# Phase 2: Доменный слой room (entities, VO, ошибки, тесты)

## Цель

Реализовать чистый доменный слой `internal/room/domain/`: сущности `Room`, `Membership`, `Invite`, value objects, sentinel-ошибки, табличные тесты. Никакого I/O, никаких импортов кроме stdlib и `github.com/google/uuid`.

## Контекст

После Phase 01 у нас есть таблицы и пустые `db/` пакеты. Domain пишется автономно — он ничего не знает про БД и HTTP. Эталон стиля — `internal/auth/domain/` (см. `../research.md` §«Эталонный паттерн домена auth»).

Доменная модель полностью описана в `../01-architecture.md` §3.1; маппинг для будущих репозиториев — в `../06-repo-model.md`. Все доменные ошибки и инварианты — в `../01-architecture.md` §3.1.5.

## Файлы для создания

### Value Objects

#### `internal/room/domain/room_id.go`
- Тип `RoomID struct { value uuid.UUID }`.
- `NewRoomID(id uuid.UUID) (RoomID, error)` — отвергает `uuid.Nil` → `ErrInvalidRoomID`.
- Методы: `UUID() uuid.UUID`, `String() string`, `IsZero() bool`.
- По образцу `internal/auth/domain/user_id.go:5-16`.

#### `internal/room/domain/user_id.go`
- Свой `UserID` (не из `auth/domain`). Зачем: `domain` не может импортировать `auth/domain` (`../03-decisions.md` §D-08). Структура и методы — те же.

#### `internal/room/domain/invite_id.go`
- Аналогично `RoomID`.

#### `internal/room/domain/room_name.go`
- Тип `RoomName struct { value string }`.
- `NewRoomName(raw string) (RoomName, error)`:
  - `strings.TrimSpace`.
  - Длина после trim: 1..64 (см. `../01-architecture.md` §3.1.2).
  - Без управляющих символов (использовать `unicode.IsControl` в цикле по рунам).
  - Ошибка → `ErrInvalidRoomName`.
- `String() string`.

#### `internal/room/domain/role.go`
- `type Role uint8` с константами:
  - `RoleMember Role = iota`
  - `RoleAdmin`
  - `RoleOwner`
- `String() string` — `"member"`/`"admin"`/`"owner"`; для невалидного значения возвращает `"invalid"` (для безопасной отладки).
- `ParseRole(raw string) (Role, error)` — switch по строкам, `ErrInvalidRole` для прочего.
- Метод `func (r Role) IsValid() bool`.

#### `internal/room/domain/invite_code.go`
- Константы (приватные): `inviteCodeLength = 8`, `crockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"` (без I/L/O/U).
- Тип `InviteCode struct { value string }`.
- `NewInviteCode(raw string) (InviteCode, error)`:
  - `strings.TrimSpace` + `strings.ToUpper`.
  - Длина строго 8.
  - Каждый символ должен быть в `crockfordAlphabet`.
  - Ошибка → `ErrInvalidInviteCode`.
- `String() string`.
- Хелпер для тестов и адаптеров: экспортированный `InviteCodeAlphabet() string` (возвращает `crockfordAlphabet`).

### Entities

#### `internal/room/domain/room.go`
- Структура `Room` с приватными полями `id RoomID`, `ownerID UserID`, `name RoomName`, `createdAt time.Time`.
- `NewRoom(id RoomID, ownerID UserID, name RoomName, createdAt time.Time) (*Room, error)`:
  - `id.IsZero()` → `ErrInvalidRoomID`.
  - `ownerID.IsZero()` → `ErrInvalidUserID`.
  - `createdAt.IsZero()` → `ErrInvalidCreatedAt`.
  - `RoomName` уже валидирован конструктором VO.
- `ReconstructRoom(...)` — алиас, вызывает `NewRoom` (`../01-architecture.md` §3.1.1).
- Геттеры: `ID()`, `OwnerID()`, `Name()`, `CreatedAt()`.
- Метод `TransferOwnership(newOwner UserID) error`:
  - `newOwner.IsZero()` → `ErrInvalidUserID`, состояние не меняется.
  - Иначе `r.ownerID = newOwner`. Возвращает nil.
  - **Зачем нужен сейчас**: без HTTP-эндпоинта, но обязателен по rich-domain (`../03-decisions.md` §D-17). Покрывается доменными тестами.

#### `internal/room/domain/membership.go`
- Структура `Membership` с приватными полями `roomID RoomID`, `userID UserID`, `role Role`, `joinedAt time.Time`.
- `NewMembership(roomID RoomID, userID UserID, role Role, joinedAt time.Time) (*Membership, error)`:
  - Стандартные zero-проверки.
  - `!role.IsValid()` → `ErrInvalidRole`.
- `ReconstructMembership(...)` — алиас.
- Геттеры: `RoomID()`, `UserID()`, `Role()`, `JoinedAt()`.
- Бизнес-методы:
  - `Promote() error` — `member` → `admin`; `admin`/`owner` — no-op (возвращают nil без изменений).
  - `Demote() error` — `admin` → `member`; `member` — no-op; `owner` → `ErrCannotDemoteOwner`, состояние не меняется.
- Методы-запросы (см. матрицу в `../04-testing.md` §«Membership permissions matrix»):
  - `CanReadRoom() bool` — true для всех валидных ролей.
  - `CanReadMembers() bool` — true для всех.
  - `CanReadChannels() bool` — true для всех.
  - `CanCreateChannel() bool` — `role == RoleOwner || role == RoleAdmin`.
  - `CanDeleteChannel() bool` — то же.
  - `CanGenerateInvite() bool` — то же.
  - `CanDeleteRoom() bool` — `role == RoleOwner`.
  - `CanKick(target Membership) bool` — таблица из `../04-testing.md`:
    - actor owner: target admin → true, target member → true, target owner (себя) → false.
    - actor admin: target member → true, иначе false.
    - actor member: всегда false.

#### `internal/room/domain/invite.go`
- Структура `Invite` с приватными полями `id InviteID`, `roomID RoomID`, `code InviteCode`, `createdBy UserID`, `createdAt time.Time`, `revokedAt time.Time` (zero == активен).
- `NewInvite(id InviteID, roomID RoomID, code InviteCode, createdBy UserID, createdAt time.Time) (*Invite, error)`:
  - Zero-проверки на id/roomID/createdBy/createdAt.
  - `code` уже валиден.
  - `revokedAt` устанавливается в `time.Time{}`.
- `ReconstructInvite(... revokedAt time.Time) (*Invite, error)` — дополнительно принимает `revokedAt` (для восстановления revoked записей из БД).
- Геттеры: `ID()`, `RoomID()`, `Code()`, `CreatedBy()`, `CreatedAt()`, `RevokedAt()`.
- Метод `Revoke(now time.Time) error`:
  - Если уже revoked (`!i.revokedAt.IsZero()`) → `ErrInviteAlreadyRevoked`, состояние не меняется.
  - Если `now.IsZero()` → `ErrInvalidCreatedAt` (защита от багов вызова).
  - Иначе `i.revokedAt = now`, возвращает nil.
- Запросы: `IsActive() bool` (`i.revokedAt.IsZero()`), `IsRevoked() bool`.

### Sentinel ошибки

#### `internal/room/domain/errors.go`

Все `var ErrXxx = errors.New("room: ...")`. Список — в `../01-architecture.md` §3.1.5:

```
// валидация VO/Entity
ErrInvalidRoomID
ErrInvalidUserID
ErrInvalidInviteID
ErrInvalidCreatedAt
ErrInvalidRoomName
ErrInvalidRole
ErrInvalidInviteCode

// бизнес-инварианты
ErrRoomNotFound
ErrNotMember
ErrAlreadyMember
ErrInsufficientRole
ErrInviteNotFound
ErrInviteAlreadyRevoked
ErrCannotDemoteOwner
```

По образцу `internal/auth/domain/errors.go:6-30`. Группировать комментариями.

### Тесты (package `domain_test`, чёрный ящик)

Все тесты — отдельные файлы рядом с production кодом. `t.Parallel()`, `t.Helper()`, табличные где это уменьшает дублирование. Стиль — `prompts/Domain model test.txt`.

#### `internal/room/domain/room_test.go`
- `TestNewRoom_Valid_Constructs`
- `TestNewRoom_ZeroID_ReturnsErrInvalidRoomID`
- `TestNewRoom_ZeroOwnerID_ReturnsErrInvalidUserID`
- `TestNewRoom_ZeroCreatedAt_ReturnsErrInvalidCreatedAt`
- `TestRoom_TransferOwnership_ChangesOwner`
- `TestRoom_TransferOwnership_ZeroNewOwner_ReturnsErr` (плюс state-preservation check)

#### `internal/room/domain/membership_test.go`
- `TestNewMembership_Valid_Constructs`
- `TestNewMembership_ZeroRoomID_ReturnsErr`
- `TestNewMembership_ZeroUserID_ReturnsErr`
- `TestNewMembership_InvalidRole_ReturnsErrInvalidRole`
- `TestMembership_Permissions_TableDriven` — матрица из `../04-testing.md`. Структура case'а:
  ```go
  cases := []struct{
      role     domain.Role
      method   string         // имя метода для подписи в t.Run
      expected bool
      callFn   func(*domain.Membership) bool
  }{...}
  ```
- `TestMembership_Promote_MemberToAdmin_Succeeds`
- `TestMembership_Promote_AdminNoOp`
- `TestMembership_Promote_OwnerNoOp`
- `TestMembership_Demote_AdminToMember_Succeeds`
- `TestMembership_Demote_MemberNoOp`
- `TestMembership_Demote_Owner_ReturnsErrCannotDemoteOwner` (+ state-preservation)
- `TestMembership_CanKick_TableDriven` — пары (actor.role, target.role).

#### `internal/room/domain/invite_test.go`
- `TestNewInvite_Valid_ConstructsActive` — IsActive == true сразу.
- `TestNewInvite_ZeroFields_ReturnErrors` — табличный по полям.
- `TestInvite_Revoke_SetsRevokedAt` — IsActive == false после Revoke.
- `TestInvite_Revoke_AlreadyRevoked_ReturnsErr` (+ state-preservation).
- `TestInvite_Revoke_ZeroNow_ReturnsErr`.
- `TestReconstructInvite_RevokedRow_RestoresState`.

#### `internal/room/domain/room_name_test.go`
- `TestNewRoomName_Valid_Normalizes` — табличный (typical, with leading/trailing spaces, max length).
- `TestNewRoomName_TooShort_ReturnsErr` — пустая, только пробелы.
- `TestNewRoomName_TooLong_ReturnsErr` — 65, 100 символов.
- `TestNewRoomName_ControlCharacter_ReturnsErr` — табличный (`\n`, `\t`, `\r`, `\x00`).

#### `internal/room/domain/role_test.go`
- `TestParseRole_TableDriven` — `"owner"`, `"admin"`, `"member"`, `""`, `"OWNER"`, `"foo"`.
- `TestRole_String_TableDriven` — round-trip Role → String → ParseRole.
- `TestRole_IsValid` — табличный.

#### `internal/room/domain/invite_code_test.go`
- `TestNewInviteCode_Valid_Normalizes` — табличный (lower → upper, with spaces).
- `TestNewInviteCode_InvalidLength_ReturnsErr` — 7, 9, 0.
- `TestNewInviteCode_InvalidChar_ReturnsErr` — содержит `I`, `L`, `O`, `U`, `*`, `-`.
- `TestInviteCodeAlphabet_HasNoConfusables` — assertion: alphabet length 32, не содержит I/L/O/U.

### Тестовые билдеры

#### `internal/room/domain/testing_builders_test.go` (package `domain_test`)
- `RoomBuilder` — chainable, принимает `*testing.T`. Дефолты:
  - `id = uuid.New()` (детерминирован в рамках одного билдера? Берём `uuid.New()` — это OK для domain-тестов; внутри теста все билды независимы).
  - `ownerID = uuid.New()`.
  - `name = "test-room"`.
  - `createdAt = time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)`.
- Методы: `WithID(uuid.UUID)`, `WithOwner(uuid.UUID)`, `WithName(string)`, `CreatedAt(time.Time)`, `Build() *domain.Room`.
- Аналогично `MembershipBuilder` (методы `Owner()`, `Admin()`, `Member()`), `InviteBuilder` (`Revoked(now)`).
- По образцу `prompts/Builder.txt` §«Пример полного builder-а».

## Файлы для модификации

- `internal/room/domain/.gitkeep` — удалить.

## Ключевые решения

- **Свой `UserID` в room/domain** — не импортируем из `auth/domain`. Цена — три строки кода и одно value object на домен; выгода — изоляция (`../03-decisions.md` §D-08).
- **`Role` как `uint8` enum**, не string — экономнее, проще сравнения; маппинг ↔ string выполняет `ParseRole`/`String()` на границе репозитория. Никаких string-сравнений в бизнес-логике.
- **`InviteCode` принимает регистр любой**, нормализует в upper-case — UX-удобство, чтобы пользователь мог ввести код с пробелами и в нижнем регистре.
- **`TransferOwnership` оставлен в Room** без HTTP-эндпоинта — rich-domain (`../03-decisions.md` §D-17). Тестируется domain-тестами.
- **`Revoke(zeroTime)` отвергается** — защита от багов; в реальном пути `Clock.Now()` zero не возвращает.

## Verification

- [ ] Все 9 production-файлов и 7 тест-файлов созданы.
- [ ] `go build ./internal/room/domain/...` чистый.
- [ ] `go test ./internal/room/domain/... -race -count=1` зелёный.
- [ ] `go test ./internal/room/domain/... -v` показывает все ожидаемые тесты из `../04-testing.md` §«internal/room/domain — Test Cases».
- [ ] `go vet ./internal/room/domain/...` чистый.
- [ ] `golangci-lint run ./internal/room/domain/...` чистый.
- [ ] Нет импортов вне stdlib + `github.com/google/uuid` (можно проверить вручную: `grep -r "import" internal/room/domain/*.go` — non-test файлы).
- [ ] `internal/room/domain/.gitkeep` удалён.
- [ ] Покрытие конструкторов и доменных ошибок: `go test -cover ./internal/room/domain/` > 90%.
