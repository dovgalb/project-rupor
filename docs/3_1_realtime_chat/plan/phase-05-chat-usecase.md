---
phase: 5
name: chat-usecase
layer: usecase
depends_on: [phase-02]
plan: ./README.md
---

# Phase 5: Use case `internal/chat/usecase/`

## Цель

Объявить порты и реализовать use case'ы `SendMessage` и `ListMessages`. После этой фазы chat готов как функциональный сервис на уровне use case (тесты с фейками покрывают весь бизнес-flow). Транспорт и адаптеры — отдельно в phase-04 (repository) / phase-06 (HTTP) / phase-07 (WS).

## Контекст

Phase-02 создала домен. Порты — `consumer-owned interfaces` (см. `prompts/Clean architecture.txt`). Реализации (phase-04) подключаются через композицию в `cmd/server/main.go` (phase-09).

Шаблон — `internal/channel/usecase/`. Стиль тестов — ручные фейки, SUT-паттерн, `t.Parallel()`, AAA. См. `internal/channel/usecase/create_channel_test.go` и `fakes_test.go`.

## Файлы для создания

### `internal/chat/usecase/ports.go`

**Назначение:** Объявление всех портов use case-слоя.

```go
package usecase

import (
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/chat/domain"
)

type RoleRequirement int

const (
    RoleAnyMember    RoleRequirement = iota
    RoleAdminOrOwner
    RoleOwnerOnly
)

// MembershipQuery — проверка прав на действие в канале.
// Контракт ошибок:
//   nil                                — пользователь имеет требуемую роль.
//   domain.ErrChatAccessDenied         — не член комнаты канала.
//   domain.ErrChatInsufficientRole     — член, но роль ниже требуемой.
//   domain.ErrChannelNotFound          — канал не существует.
//   обёрнутая fmt.Errorf               — техническая ошибка.
type MembershipQuery interface {
    Require(ctx context.Context, channelID domain.ChannelID, userID domain.UserID, req RoleRequirement) error
}

// ChannelInfo возвращает MessageRepository.ChannelOf — нужен для проверки kind перед записью.
type ChannelInfo struct {
    ChannelID domain.ChannelID
    RoomID    domain.RoomID
    Kind      string // "text" | "voice"
}

const (
    ChannelKindText  = "text"
    ChannelKindVoice = "voice"
)

// MessageRepository — порт хранения сообщений.
type MessageRepository interface {
    Save(ctx context.Context, m *domain.Message) error
    ListByChannel(ctx context.Context, channelID domain.ChannelID, before domain.MessageID, limit int) ([]*domain.Message, error)
    ChannelOf(ctx context.Context, channelID domain.ChannelID) (ChannelInfo, error)
}

// Broadcaster — порт публикации событий в real-time канал.
// Контракт: реализация не блокирует, ошибки не пробрасывает (best-effort).
type Broadcaster interface {
    PublishToChannel(channelID domain.ChannelID, eventType string, payload any)
    PublishToRoom(roomID domain.RoomID, eventType string, payload any)
}

type Clock         interface{ Now() time.Time }
type UUIDGenerator interface{ New() uuid.UUID }
```

### `internal/chat/usecase/send_message.go`

**Назначение:** Use case отправки сообщения. Оркестрирует проверку прав → проверку типа канала → создание сущности → сохранение → broadcast.

```go
package usecase

import (
    "context"
    "errors"
    "fmt"

    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/chat/domain"
)

type SendMessage struct {
    messages    MessageRepository
    membership  MembershipQuery
    broadcaster Broadcaster
    clock       Clock
    uuids       UUIDGenerator
}

func NewSendMessage(
    messages MessageRepository,
    membership MembershipQuery,
    broadcaster Broadcaster,
    clock Clock,
    uuids UUIDGenerator,
) *SendMessage {
    return &SendMessage{messages, membership, broadcaster, clock, uuids}
}

type SendMessageInput struct {
    ActorID   uuid.UUID
    ChannelID uuid.UUID
    Text      string
}

type SendMessageOutput struct {
    MessageID uuid.UUID
    ChannelID uuid.UUID
    AuthorID  uuid.UUID
    Text      string
    CreatedAt time.Time
}

func (uc *SendMessage) Execute(ctx context.Context, in SendMessageInput) (SendMessageOutput, error) {
    actorID, err := domain.NewUserID(in.ActorID)
    if err != nil {
        return SendMessageOutput{}, err
    }
    channelID, err := domain.NewChannelID(in.ChannelID)
    if err != nil {
        return SendMessageOutput{}, err
    }
    text, err := domain.NewMessageText(in.Text)
    if err != nil {
        return SendMessageOutput{}, err
    }

    // 1. Проверка прав на канал (членство в комнате канала).
    if err := uc.membership.Require(ctx, channelID, actorID, RoleAnyMember); err != nil {
        return SendMessageOutput{}, err
    }

    // 2. Проверка типа канала (только text).
    info, err := uc.messages.ChannelOf(ctx, channelID)
    if err != nil {
        return SendMessageOutput{}, err
    }
    if info.Kind != ChannelKindText {
        return SendMessageOutput{}, domain.ErrChannelNotText
    }

    // 3. Создание сущности.
    msgID, err := domain.NewMessageID(uc.uuids.New())
    if err != nil {
        return SendMessageOutput{}, fmt.Errorf("send message: generate id: %w", err)
    }
    now := uc.clock.Now().UTC()
    msg, err := domain.NewMessage(msgID, channelID, actorID, text, now)
    if err != nil {
        return SendMessageOutput{}, fmt.Errorf("send message: construct: %w", err)
    }

    // 4. Сохранение в БД.
    if err := uc.messages.Save(ctx, msg); err != nil {
        return SendMessageOutput{}, err
    }

    // 5. Broadcast — best-effort. Паника/ошибка не пробрасывается.
    func() {
        defer func() { _ = recover() }()
        uc.broadcaster.PublishToChannel(channelID, "message.new", messagePayload(msg))
    }()

    return SendMessageOutput{
        MessageID: msg.ID().UUID(),
        ChannelID: msg.ChannelID().UUID(),
        AuthorID:  msg.AuthorID().UUID(),
        Text:      msg.Text().String(),
        CreatedAt: msg.CreatedAt(),
    }, nil
}

// messagePayload собирает payload события `message.new`.
// Snake_case согласован с 05-events.md.
func messagePayload(m *domain.Message) map[string]any {
    return map[string]any{
        "id":         m.ID().String(),
        "channel_id": m.ChannelID().String(),
        "author_id":  m.AuthorID().String(),
        "text":       m.Text().String(),
        "created_at": m.CreatedAt().Format(time.RFC3339Nano),
    }
}
```

**Детали реализации:**
- Order проверок: **сначала Require, потом ChannelOf** — не утекаем информацию о существовании канала пользователю не-члену (если канал есть, но user не член — он получит CHAT-004, а не CHAT-002).
- Broadcast в `defer recover` — гарантирует, что panic в hub'е не сломает Save. См. [§ D-08](../03-decisions.md).
- Использование `_ = recover()` — игнорирование panic. Логирование оставляется на сторону hub'а ([phase-03](./phase-03-pkg-websocket.md)).

⚠️ **Open:** `errors.Is` для domain-errors не нужен здесь — мы пробрасываем как есть. Маппинг в HTTP/WS-коды — на стороне transport (phase-06, phase-07).

### `internal/chat/usecase/list_messages.go`

```go
package usecase

import (
    "context"
    "fmt"

    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/chat/domain"
)

type ListMessages struct {
    messages   MessageRepository
    membership MembershipQuery
}

func NewListMessages(messages MessageRepository, membership MembershipQuery) *ListMessages {
    return &ListMessages{messages, membership}
}

type ListMessagesInput struct {
    ActorID   uuid.UUID
    ChannelID uuid.UUID
    Before    uuid.UUID // uuid.Nil если не задан
    Limit     int
}

type ListMessagesOutput struct {
    Items      []*domain.Message
    NextBefore uuid.UUID // uuid.Nil если страница последняя
}

const (
    DefaultLimit = 50
    MinLimit     = 1
    MaxLimit     = 100
)

func (uc *ListMessages) Execute(ctx context.Context, in ListMessagesInput) (ListMessagesOutput, error) {
    actorID, err := domain.NewUserID(in.ActorID)
    if err != nil {
        return ListMessagesOutput{}, err
    }
    channelID, err := domain.NewChannelID(in.ChannelID)
    if err != nil {
        return ListMessagesOutput{}, err
    }
    var beforeID domain.MessageID
    if in.Before != uuid.Nil {
        beforeID, err = domain.NewMessageID(in.Before)
        if err != nil {
            return ListMessagesOutput{}, err
        }
    }

    limit := in.Limit
    if limit == 0 {
        limit = DefaultLimit
    }
    if limit < MinLimit || limit > MaxLimit {
        return ListMessagesOutput{}, fmt.Errorf("limit out of range [%d..%d]: got %d",
            MinLimit, MaxLimit, in.Limit)
    }

    // Проверка прав.
    if err := uc.membership.Require(ctx, channelID, actorID, RoleAnyMember); err != nil {
        return ListMessagesOutput{}, err
    }

    items, err := uc.messages.ListByChannel(ctx, channelID, beforeID, limit)
    if err != nil {
        return ListMessagesOutput{}, err
    }

    var nextBefore uuid.UUID
    if len(items) == limit {
        nextBefore = items[len(items)-1].ID().UUID()
    }
    return ListMessagesOutput{Items: items, NextBefore: nextBefore}, nil
}
```

**Замечание:** "limit out of range" — техническая ошибка, не доменная. Transport-слой (phase-06) парсит/нормализует limit на стороне handler'а и отдаёт CHAT-006 ДО вызова use case. Use case остаётся защищён на случай прямого вызова.

### `internal/chat/usecase/fakes_test.go`

**Назначение:** Ручные фейки портов для тестов use case'ов. Структура из [`../04-testing.md §Stubs`](../04-testing.md).

```go
package usecase_test

import (
    "context"
    "sync"
    "time"

    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/chat/domain"
    "github.com/dovgalb/project-rupor/internal/chat/usecase"
)

type fakeMessageRepo struct {
    mu           sync.Mutex
    saved        []*domain.Message
    listByChan   map[uuid.UUID][]*domain.Message
    channelInfo  map[uuid.UUID]usecase.ChannelInfo
    saveErr      error
    listErr      error
    channelOfErr error
}

func newFakeMessageRepo() *fakeMessageRepo { /* ... */ }
func (r *fakeMessageRepo) Save(...) error { ... }
func (r *fakeMessageRepo) ListByChannel(...) ([]*domain.Message, error) { ... }
func (r *fakeMessageRepo) ChannelOf(...) (usecase.ChannelInfo, error) { ... }

type fakeMembershipQuery struct {
    err  error // глобальная инжекция ошибки
}

type fakeBroadcaster struct {
    mu        sync.Mutex
    channels  []publishedEvent
    rooms     []publishedEvent
    panicNext bool
}

type publishedEvent struct {
    TopicID   uuid.UUID
    EventType string
    Payload   any
}

type fixedClock struct{ now time.Time }
type fixedUUID  struct{ next []uuid.UUID; idx int }
```

### `internal/chat/usecase/send_message_test.go`

10 тестов из [`../04-testing.md §SendMessage`](../04-testing.md).

SUT-паттерн:
```go
type sendMessageSUT struct {
    uc          *usecase.SendMessage
    messages    *fakeMessageRepo
    membership  *fakeMembershipQuery
    broadcaster *fakeBroadcaster
    clock       *fixedClock
    uuids       *fixedUUID
    actor       uuid.UUID
    channel     uuid.UUID
    msgID       uuid.UUID
}

func newSendMessageSUT(t *testing.T) *sendMessageSUT { /* default-валидные ID, channelInfo с kind=text */ }
```

### `internal/chat/usecase/list_messages_test.go`

7 тестов из [`../04-testing.md §ListMessages`](../04-testing.md).

## Файлы для модификации

Нет в этой фазе.

## Ключевые решения

- **Order проверок: Require → ChannelOf** — anti-info-leak. См. внутреннее замечание в `send_message.go`.
- **Broadcast в defer recover** — best-effort delivery, [§ D-08](../03-decisions.md).
- **`limit out of range` — ошибка use case, не handler-only** — defense in depth. Handler нормализует limit ДО вызова, use case бережёт от прямого вызова.
- **DTO use case'ов — простые uuid.UUID, не VO** — `SendMessageInput` принимает raw uuid, использует VO внутри (см. шаблон `internal/channel/usecase/create_channel.go:30-37`). Это позволяет handler'у не парсить VO самому.
- **Payload `message.new` — `map[string]any` со snake_case** — отслеживается явно в `messagePayload()`. Альтернатива (типизированная struct с json-тегами) — оверкилл для одного payload'а.

## Verification

- [ ] `go build ./internal/chat/usecase/...` без ошибок.
- [ ] `go test ./internal/chat/usecase/...` зелёный.
- [ ] `go test -race ./internal/chat/usecase/...` зелёный.
- [ ] Все тесты из [04-testing.md §Chat / usecase](../04-testing.md) написаны и проходят (~17 тестов: 10 SendMessage + 7 ListMessages).
- [ ] Импорты `internal/chat/usecase` — только stdlib + uuid + `internal/chat/domain` (визуально; arch-тест в phase-09).
- [ ] `errors.Is` корректно работает для всех доменных ошибок (тесты).
- [ ] `TestSendMessage_PublishFails_ReturnsSuccess` — panic в broadcaster'е НЕ пробрасывается клиенту.
- [ ] `TestSendMessage_SaveFails_DoesNotPublish` — broadcaster НЕ вызывается, если Save упал.
