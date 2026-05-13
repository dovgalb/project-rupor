---
phase: 2
name: chat-domain
layer: domain
depends_on: none
plan: ./README.md
---

# Phase 2: Домен `internal/chat/domain/`

## Цель

Создать чистый доменный слой `chat`: сущность `Message`, value objects (`MessageID`, `MessageText`, `ChannelID`, `UserID`, `RoomID`), доменные ошибки. Полное покрытие юнит-тестами без БД и моков.

## Контекст

Шаблон — `internal/channel/domain/` (5 файлов VO + entity + errors + tests). Стиль — [`prompts/Domain Model.txt`](../../../prompts/Domain%20Model.txt) (rich entity, приватные поля, конструкторы с инвариантами, без сеттеров). Тесты — [`prompts/Domain model test.txt`](../../../prompts/Domain%20model%20test.txt) (чёрный ящик, table-driven, `t.Parallel()`, без моков).

Импорты разрешены: stdlib + `github.com/google/uuid`. Никаких других зависимостей (см. [01-architecture.md § Граф зависимостей](../01-architecture.md)).

## Файлы для создания

### `internal/chat/domain/message_id.go`

**Назначение:** VO `MessageID` — обёртка над `uuid.UUID` с инвариантом «не nil».

**Детали реализации:**
- Структура `type MessageID struct { value uuid.UUID }` (приватное поле).
- `func NewMessageID(raw uuid.UUID) (MessageID, error)` — отвергает `uuid.Nil`, возвращает `ErrInvalidMessageID`.
- Методы: `(m MessageID) UUID() uuid.UUID`, `(m MessageID) String() string`, `(m MessageID) IsZero() bool`.
- Шаблон 1-в-1 повторяет `internal/channel/domain/channel_id.go`.

### `internal/chat/domain/channel_id.go`

**Назначение:** VO `ChannelID`. По шаблону `MessageID`. Доменная ошибка — `ErrInvalidChannelID`.

### `internal/chat/domain/user_id.go`

**Назначение:** VO `UserID`. По шаблону `MessageID`. Доменная ошибка — `ErrInvalidAuthorID` (имя — `AuthorID`, потому что chat-домен видит пользователей как «авторов сообщений»; внутри VO — `UserID`).

### `internal/chat/domain/room_id.go`

**Назначение:** VO `RoomID`. По шаблону `MessageID`. Доменная ошибка — `ErrInvalidRoomID`.

### `internal/chat/domain/message_text.go`

**Назначение:** VO `MessageText` — обёртка над string с валидацией длины и набора символов. Соответствует [§ D-14](../03-decisions.md).

**Детали реализации:**
- Структура `type MessageText struct { value string }`.
- `func NewMessageText(raw string) (MessageText, error)`:
  1. `raw = strings.TrimSpace(raw)` (как в `internal/channel/domain/channel_name.go`).
  2. Если длина в рунах < 1 → `ErrInvalidMessageText`.
  3. Если длина в рунах > 4000 → `ErrInvalidMessageText`.
  4. Итерация по рунам: `unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t'` → `ErrInvalidMessageText`.
  5. Итерация по рунам: `unicode.Is(unicode.Cf, r)` → `ErrInvalidMessageText` (категория Cf — invisible-formatting, anti-spoofing; см. `channel_name.go:24-28`).
- Метод `(t MessageText) String() string` возвращает `t.value`.
- **Константы:** `const (MessageTextMinLen = 1; MessageTextMaxLen = 4000)` — в файле, не магические числа.

### `internal/chat/domain/message.go`

**Назначение:** Rich entity `Message`. Иммутабельна (нет бизнес-методов в MVP, см. [§ D-04](../03-decisions.md)).

**Детали реализации:**
```go
type Message struct {
    id        MessageID
    channelID ChannelID
    authorID  UserID
    text      MessageText
    createdAt time.Time
}

func NewMessage(
    id MessageID,
    channelID ChannelID,
    authorID UserID,
    text MessageText,
    createdAt time.Time,
) (*Message, error) {
    if id.IsZero()        { return nil, ErrInvalidMessageID }
    if channelID.IsZero() { return nil, ErrInvalidChannelID }
    if authorID.IsZero()  { return nil, ErrInvalidAuthorID }
    if createdAt.IsZero() { return nil, ErrInvalidCreatedAt }
    return &Message{id, channelID, authorID, text, createdAt}, nil
}

func ReconstructMessage(...) (*Message, error) { return NewMessage(...) }

// Геттеры — без сеттеров:
func (m *Message) ID() MessageID         { return m.id }
func (m *Message) ChannelID() ChannelID  { return m.channelID }
func (m *Message) AuthorID() UserID      { return m.authorID }
func (m *Message) Text() MessageText     { return m.text }
func (m *Message) CreatedAt() time.Time  { return m.createdAt }
```

**Замечание:** `text` НЕ проверяется в конструкторе — он уже валиден по конструкции `MessageText` (нельзя получить невалидный `MessageText` в обход `NewMessageText`).

### `internal/chat/domain/errors.go`

**Назначение:** Все доменные ошибки chat.

```go
package domain

import "errors"

// Инварианты конструкторов:
var (
    ErrInvalidMessageID  = errors.New("chat: invalid message id")
    ErrInvalidChannelID  = errors.New("chat: invalid channel id")
    ErrInvalidAuthorID   = errors.New("chat: invalid author id")
    ErrInvalidRoomID     = errors.New("chat: invalid room id")
    ErrInvalidMessageText = errors.New("chat: invalid message text")
    ErrInvalidCreatedAt  = errors.New("chat: invalid created_at")
)

// Бизнес-ошибки:
var (
    ErrMessageNotFound       = errors.New("chat: message not found")
    ErrChannelNotFound       = errors.New("chat: channel not found")
    ErrChannelNotText        = errors.New("chat: channel is not text")
    ErrChatAccessDenied      = errors.New("chat: access denied")
    ErrChatInsufficientRole  = errors.New("chat: insufficient role")
)
```

### `internal/chat/domain/testing_builders_test.go`

**Назначение:** Test builder для `Message`. Используется в тестах domain и usecase. Шаблон — `internal/channel/domain/testing_builders_test.go`.

**Детали реализации:**
```go
package domain_test

type MessageBuilder struct {
    t         *testing.T
    id        uuid.UUID
    channelID uuid.UUID
    authorID  uuid.UUID
    text      string
    createdAt time.Time
}

func NewMessageBuilder(t *testing.T) *MessageBuilder { /* default-валидные значения */ }
func (b *MessageBuilder) WithText(s string) *MessageBuilder { ... }
func (b *MessageBuilder) WithChannelID(id uuid.UUID) *MessageBuilder { ... }
func (b *MessageBuilder) WithAuthorID(id uuid.UUID) *MessageBuilder { ... }
func (b *MessageBuilder) WithCreatedAt(t time.Time) *MessageBuilder { ... }
func (b *MessageBuilder) Build() *domain.Message { /* вызывает NewMessage, падает через t.Fatalf */ }
```

## Тесты

Состав и наименование — точно из [../04-testing.md §Chat / domain](../04-testing.md). Все тесты — package `domain_test` (чёрный ящик).

### `message_test.go`
- `TestNewMessage_Valid_ReturnsMessage`
- `TestNewMessage_ZeroID_ReturnsErrInvalidMessageID`
- `TestNewMessage_ZeroChannelID_ReturnsErrInvalidChannelID`
- `TestNewMessage_ZeroAuthorID_ReturnsErrInvalidAuthorID`
- `TestNewMessage_ZeroCreatedAt_ReturnsErrInvalidCreatedAt`

### `message_text_test.go`
Table-driven, 9 кейсов из [04-testing.md §MessageText VO](../04-testing.md).

### `message_id_test.go`, `channel_id_test.go`, `user_id_test.go`, `room_id_test.go`
Каждый — 4 кейса: valid → ok; nil-uuid → ErrInvalid...; `IsZero()` корректно; `String()` корректно.

## Файлы для модификации

Нет. Phase-02 — чистое добавление новых файлов.

## Ключевые решения

- **Свои VO в `chat/domain`** для `ChannelID`, `UserID`, `RoomID` — не импортирует чужие домены (см. [§ D-13](../03-decisions.md)). Это повторяет паттерн `internal/channel/domain/`, который имеет свои `UserID` и `RoomID`.
- **`ReconstructMessage` — алиас `NewMessage`** — см. [`../07-standards.md`](../07-standards.md) § Уточнение 2. Шаблон `internal/channel/domain/channel.go:41-49`.
- **Нет бизнес-методов** — фаза 3.1 не редактирует и не удаляет сообщения (см. [§ D-04](../03-decisions.md)).

## Verification

- [ ] `go build ./internal/chat/domain/...` без ошибок.
- [ ] `go test ./internal/chat/domain/...` зелёный.
- [ ] `go test -race ./internal/chat/domain/...` зелёный.
- [ ] `go vet ./internal/chat/domain/...` чистый.
- [ ] Все поля сущности — Value Objects (импортные `uuid.UUID` напрямую — только в конструкторах VO, не в `Message`).
- [ ] Геттеры без сеттеров. Поля приватные.
- [ ] Все доменные ошибки совпадают с перечнем в [`../06-repo-model.md`](../06-repo-model.md) и [`../08-api-contract.md`](../08-api-contract.md).
- [ ] Импорты `chat/domain` — только stdlib + `github.com/google/uuid`. (Архитектурный тест добавится в phase-09.)
