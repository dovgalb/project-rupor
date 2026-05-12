---
phase: 3
name: Channel domain
layer: domain
depends_on: [phase-01]
plan: ./README.md
---

# Phase 3: Доменный слой channel (entities, VO, ошибки, тесты)

## Цель

Реализовать `internal/channel/domain/`: сущность `Channel`, value objects (включая собственные `RoomID` и `UserID`), enum `ChannelKind`, sentinel-ошибки. Никакого I/O, импорты — только stdlib и `github.com/google/uuid`.

## Контекст

После Phase 01 пакет `internal/channel/repository/postgres/db/` существует с пустыми моделями. Channel-домен пишется автономно. Стиль и правила — те же, что в Phase 02 (см. `prompts/Domain Model.txt`, `prompts/Domain model test.txt`).

Channel-домен **не импортирует room-домен** — связь между ними строится через порт `MembershipQuery` в Phase 05. См. `../03-decisions.md` §D-07. Поэтому здесь объявляются СВОИ `RoomID` и `UserID` — параллельно одноимённым в `room/domain` и `auth/domain` (`../03-decisions.md` §D-08).

Доменная модель — `../01-architecture.md` §3.2; ошибки — `../01-architecture.md` §3.2.3.

## Файлы для создания

### Value Objects

#### `internal/channel/domain/channel_id.go`
- `ChannelID struct { value uuid.UUID }` с конструктором `NewChannelID`, методами `UUID()`, `String()`, `IsZero()`. По образцу `internal/room/domain/room_id.go` (Phase 02).

#### `internal/channel/domain/room_id.go`
- Собственный `RoomID` (не из room/domain). Та же структура, что в Phase 02.

#### `internal/channel/domain/user_id.go`
- Собственный `UserID`.

#### `internal/channel/domain/channel_name.go`
- `ChannelName struct { value string }`.
- `NewChannelName(raw string) (ChannelName, error)`:
  - `strings.TrimSpace`.
  - Длина 1..64.
  - Без `unicode.IsControl` символов.
  - Ошибка → `ErrInvalidChannelName`.
- `String() string`.

#### `internal/channel/domain/channel_kind.go`
- `type ChannelKind uint8`:
  - `ChannelKindText ChannelKind = iota`
  - `ChannelKindVoice`
- `String() string` — `"text"` / `"voice"`; для невалидного — `"invalid"`.
- `ParseChannelKind(raw string) (ChannelKind, error)` — switch, иначе `ErrInvalidChannelKind`.
- `IsValid() bool`.

### Entity

#### `internal/channel/domain/channel.go`
- Структура `Channel` с приватными полями `id ChannelID`, `roomID RoomID`, `name ChannelName`, `kind ChannelKind`, `createdAt time.Time`.
- `NewChannel(id ChannelID, roomID RoomID, name ChannelName, kind ChannelKind, createdAt time.Time) (*Channel, error)`:
  - `id.IsZero()` → `ErrInvalidChannelID`.
  - `roomID.IsZero()` → `ErrInvalidRoomID`.
  - `!kind.IsValid()` → `ErrInvalidChannelKind`.
  - `createdAt.IsZero()` → `ErrInvalidCreatedAt`.
  - `name` — уже валиден через VO.
- `ReconstructChannel(...)` — алиас, для репозитория.
- Геттеры: `ID()`, `RoomID()`, `Name()`, `Kind()`, `CreatedAt()`.

В PR-2 у Channel **нет** мутирующих бизнес-методов. Rename/move отложены.

### Sentinel ошибки

#### `internal/channel/domain/errors.go`

```
// валидация VO/Entity
ErrInvalidChannelID
ErrInvalidRoomID
ErrInvalidUserID
ErrInvalidChannelName
ErrInvalidChannelKind
ErrInvalidCreatedAt

// бизнес-инварианты
ErrChannelNotFound
ErrChannelNameAlreadyTaken
ErrChannelAccessDenied      // нет membership в комнате
ErrChannelInsufficientRole  // есть membership, но недостаточно прав
```

Все sentinel `var ErrXxx = errors.New("channel: ...")`. Группировать комментариями (по образцу `internal/auth/domain/errors.go:6-30`).

`ErrChannelAccessDenied` и `ErrChannelInsufficientRole` объявляются здесь, **не в порту** — это доменные понятия, и адаптер `MembershipQueryAdapter` (Phase 06) импортирует их из `channel/domain`. См. `../03-decisions.md` §D-19.

### Тесты (package `domain_test`)

#### `internal/channel/domain/channel_test.go`
- `TestNewChannel_Valid_Constructs` — happy path.
- `TestNewChannel_ZeroID_ReturnsErr`
- `TestNewChannel_ZeroRoomID_ReturnsErr`
- `TestNewChannel_ZeroCreatedAt_ReturnsErr`
- (валидность name и kind проверяется конструкторами VO в их собственных файлах — здесь не дублируем)

#### `internal/channel/domain/channel_name_test.go`
- `TestNewChannelName_Valid_Normalizes` — табличный.
- `TestNewChannelName_TooShort_ReturnsErr` — пустая, только пробелы.
- `TestNewChannelName_TooLong_ReturnsErr` — 65, 100.
- `TestNewChannelName_ControlCharacter_ReturnsErr` — табличный.

#### `internal/channel/domain/channel_kind_test.go`
- `TestParseChannelKind_Valid_TableDriven` — `"text"`, `"voice"`, регистр strict (`"TEXT"` отклоняется — нет нормализации в kind, в отличие от code; обоснование: kind приходит из своего домена/контракта, а не от пользователя).
- `TestParseChannelKind_Invalid_ReturnsErr` — `"video"`, `""`, `"Text"`.
- `TestChannelKind_String_TableDriven` — round-trip.
- `TestChannelKind_IsValid` — табличный.

### Тестовые билдеры

#### `internal/channel/domain/testing_builders_test.go` (package `domain_test`)
- `ChannelBuilder` с дефолтами (uuid, "general", text, fixed time 2026-05-11). Методы: `WithName`, `WithRoom`, `WithKind`, `Voice()`, `Text()`, `Build()`.

## Файлы для модификации

- `internal/channel/domain/.gitkeep` — удалить.

## Ключевые решения

- **Свои `RoomID`, `UserID`** — не импортируем `room/domain` или `auth/domain` (`../03-decisions.md` §D-08).
- **`ParseChannelKind` без нормализации регистра** — kind не вводится пользователем, приходит из API-контракта. `"text"` и `"TEXT"` — разные значения; принимаем только lowercase.
- **`ErrChannelAccessDenied` и `ErrChannelInsufficientRole` в `channel/domain`** — нейтральные понятия, не зависят от понятия `Membership` (которое живёт в room/domain). Это позволяет адаптеру в Phase 06 транслировать `room.domain.ErrNotMember` → `channel.domain.ErrChannelAccessDenied` без обратной зависимости.
- **Channel — entity без мутирующих методов в PR-2** — будущие фазы добавят rename/move/visibility-управление.

## Verification

- [ ] 6 production-файлов и 4 тест-файла созданы (плюс testing_builders_test.go).
- [ ] `go build ./internal/channel/domain/...` чистый.
- [ ] `go test ./internal/channel/domain/... -race -count=1` зелёный.
- [ ] Нет импортов вне stdlib + `github.com/google/uuid`. Особенно — нет импорта `internal/room/domain` или `internal/auth/domain`.
- [ ] `golangci-lint run ./internal/channel/domain/...` чистый.
- [ ] `internal/channel/domain/.gitkeep` удалён.
- [ ] `go test -cover ./internal/channel/domain/` > 90%.
