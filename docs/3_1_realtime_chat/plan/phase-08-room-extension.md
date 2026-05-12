---
phase: 8
name: room-extension
layer: usecase
depends_on: [phase-03]
plan: ./README.md
---

# Phase 8: Расширение `room/usecase.JoinByCode` — публикация `member.joined`

## Цель

Добавить порт `room/usecase.RoomEventsPublisher`, расширить `JoinByCode` для публикации события `member.joined` после успешного вступления пользователя в комнату. Реализация publisher'а — `pkg/websocket.Hub` (через тонкий wrapper в `cmd/server/main.go`).

## Контекст

Существующий код `internal/room/usecase/join_by_code.go` уже работает: парсит invite-code, проверяет инвайт, создаёт membership через `MembershipRepository.Add`. Эта фаза добавляет ОДИН вызов publisher'а после успешного `Add`.

Существующие порты `room/usecase` — в `internal/room/usecase/ports.go`. Существующие use case'ы — `CreateRoom`, `GetRoom`, `ListUserRooms`, `DeleteRoom`, `ListMembers`, `RegenerateInvite`, `JoinByCode`.

Решение и контракт публикатора — [§ D-07](../03-decisions.md), [`../05-events.md §member.joined`](../05-events.md).

## Файлы для модификации

### `internal/room/usecase/ports.go`

**Добавить интерфейс в конец файла:**

```go
// RoomEventsPublisher — публикация доменных событий из room.JoinByCode и других use case'ов.
// Контракт: best-effort, ошибки не пробрасываются, реализация не блокирует.
type RoomEventsPublisher interface {
    PublishMemberJoined(roomID domain.RoomID, userID domain.UserID, joinedAt time.Time)
}
```

Если в файле уже есть импорт `"time"` — он остаётся. Иначе — добавить.

### `internal/room/usecase/join_by_code.go`

**Изменения:**

1. **Добавить поле `events RoomEventsPublisher` в struct `JoinByCode`:**

```go
type JoinByCode struct {
    invites    InviteRepository
    members    MembershipRepository
    rooms      RoomRepository
    clock      Clock
    events     RoomEventsPublisher   // НОВОЕ
}
```

2. **Изменить сигнатуру `NewJoinByCode`:**

```go
func NewJoinByCode(
    invites InviteRepository,
    members MembershipRepository,
    rooms RoomRepository,
    clock Clock,
    events RoomEventsPublisher,    // НОВЫЙ параметр в конце
) *JoinByCode {
    return &JoinByCode{
        invites: invites,
        members: members,
        rooms:   rooms,
        clock:   clock,
        events:  events,
    }
}
```

3. **В `Execute(...)` после успешного `members.Add(...)` добавить публикацию:**

```go
// ... существующий код ...

if err := uc.members.Add(ctx, m); err != nil {
    return JoinByCodeOutput{}, err
}

// НОВОЕ: best-effort publish.
func() {
    defer func() { _ = recover() }()
    uc.events.PublishMemberJoined(roomID, userID, m.JoinedAt())
}()

return JoinByCodeOutput{ /* ... */ }, nil
```

⚠️ Перед редактированием прочитать существующий `join_by_code.go` целиком, чтобы корректно вписаться в код. Использовать переменные точно как они называются (например, `roomID`, `userID`, `m` — проверить).

### `internal/room/usecase/join_by_code_test.go`

**Добавить 3 теста** из [`../04-testing.md §Room (member.joined)`](../04-testing.md):

- `TestJoinByCode_Success_PublishesMemberJoined` — после успешного join'а publisher вызван ровно один раз с правильными параметрами.
- `TestJoinByCode_PublishError_DoesNotRollbackMembership` — публикатор паникует/возвращает ошибку → `Execute` возвращает успех; membership сохранён в fake-репо.
- `TestJoinByCode_InvalidCode_DoesNotPublish` — invite не найден → `Execute` возвращает ошибку, publisher НЕ вызывается.

### `internal/room/usecase/fakes_test.go`

**Добавить fake-publisher в существующий `fakes_test.go`:**

```go
type fakeRoomEventsPublisher struct {
    mu           sync.Mutex
    memberJoined []memberJoinedCall
    panicNext    bool
}

type memberJoinedCall struct {
    RoomID   domain.RoomID
    UserID   domain.UserID
    JoinedAt time.Time
}

func newFakeRoomEventsPublisher() *fakeRoomEventsPublisher {
    return &fakeRoomEventsPublisher{}
}

func (p *fakeRoomEventsPublisher) PublishMemberJoined(roomID domain.RoomID, userID domain.UserID, joinedAt time.Time) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.panicNext {
        p.panicNext = false
        panic("simulated publisher panic")
    }
    p.memberJoined = append(p.memberJoined, memberJoinedCall{roomID, userID, joinedAt})
}
```

## Файлы для создания

Нет новых файлов в этой фазе. Изменения — only в room/usecase.

## Ключевые решения

- **`events RoomEventsPublisher` — НОВЫЙ параметр конструктора, последний по позиции** — минимизирует визуальный шум при diff'е существующего кода. Совместимость со старым кодом ломается (компиляция упадёт там, где `NewJoinByCode` вызывается со старой сигнатурой), но это ровно одно место — `cmd/server/main.go` (изменяется в phase-09).
- **`defer recover()`** в `Execute` — гарантирует, что panic в publisher'е не сломает business-flow. См. [§ D-07](../03-decisions.md) и аналогично `chat/usecase.SendMessage` ([phase-05](./phase-05-chat-usecase.md)).
- **Порт объявлен в `room/usecase/ports.go`** (consumer-owned), реализация — `pkg/websocket.Hub` через wrapper в `main.go` (phase-09).
- **Параметры publisher'а — VO room/domain типов** (`domain.RoomID`, `domain.UserID`), не uuid — потому что publisher живёт в usecase, а usecase оперирует доменными типами. Wrapper в `main.go` конвертирует VO в `uuid.UUID` для hub'а.

## Verification

- [ ] `go build ./internal/room/usecase/...` без ошибок.
- [ ] `go test ./internal/room/usecase/...` зелёный, включая существующие тесты + 3 новых.
- [ ] `go test -race ./internal/room/usecase/...` зелёный.
- [ ] Существующие тесты `JoinByCode` обновлены под новую сигнатуру `NewJoinByCode` (последний параметр — fake publisher).
- [ ] `TestJoinByCode_Success_PublishesMemberJoined` — проверка ровно одного вызова publisher с правильными VO.
- [ ] `TestJoinByCode_PublishError_DoesNotRollbackMembership` — panic игнорируется, membership сохранён.
- [ ] `TestJoinByCode_InvalidCode_DoesNotPublish` — publisher НЕ вызван при ошибке `Find`.
- [ ] Импорты `internal/room/usecase` — только stdlib + uuid + `internal/room/domain`. Никаких новых импортов. (Arch-тест без изменений — он уже это проверяет.)
