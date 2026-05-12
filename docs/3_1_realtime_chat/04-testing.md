---
parent: ./README.md
view: quality
---

# 04 — Testing (Quality View)

Стратегия тестов соответствует `prompts/Tests Style.txt`, `prompts/Domain model test.txt`. Используется только stdlib `testing` (никаких testify/gomock). Фейки пишутся вручную, рядом с тестом. `t.Parallel()` всегда, где безопасно. AAA-структура с пустыми строками-разделителями.

## Coverage Mapping

Каждый код ошибки из `02-behavior.md` и `08-api-contract.md` имеет хотя бы один тест.

### Chat use case

| Use Case / Error | Код ошибки | Слой | Тест |
|---|---|---|---|
| SendMessage / happy path | — | `usecase` | `TestSendMessage_AsMember_PersistsAndPublishes` |
| SendMessage / пустой текст | — (`ErrInvalidMessageText`) | `domain` | `TestNewMessageText_Empty_ReturnsErrInvalid` |
| SendMessage / только пробелы | — (`ErrInvalidMessageText`) | `domain` | `TestNewMessageText_OnlyWhitespace_ReturnsErrInvalid` |
| SendMessage / длинный текст > 4000 рун | CHAT-001 | `usecase`/`http` | `TestSendMessage_TooLong_ReturnsErrInvalidText` |
| SendMessage / control character | CHAT-001 | `domain` | `TestNewMessageText_ControlChar_ReturnsErrInvalid` |
| SendMessage / канал не найден | CHAT-002 | `usecase` | `TestSendMessage_UnknownChannel_ReturnsErrChannelNotFound` |
| SendMessage / канал voice | CHAT-003 | `usecase` | `TestSendMessage_VoiceChannel_ReturnsErrChannelNotText` |
| SendMessage / не член комнаты | CHAT-004 | `usecase` | `TestSendMessage_NotMember_ReturnsErrAccessDenied` |
| SendMessage / Save error → не publish'им | INTERNAL | `usecase` | `TestSendMessage_SaveFails_DoesNotPublish` |
| SendMessage / Publish error → не возвращаем клиенту | — | `usecase` | `TestSendMessage_PublishFails_ReturnsSuccess` |
| ListMessages / happy path с before/limit | — | `usecase` | `TestListMessages_AsMember_ReturnsBeforeCursor` |
| ListMessages / пустая история | — | `usecase` | `TestListMessages_EmptyChannel_ReturnsEmpty` |
| ListMessages / before указывает на несуществующий id | — | `usecase` | `TestListMessages_BeforeUnknown_ReturnsEmpty` |
| ListMessages / limit < 1 или > 100 | CHAT-006 | `http` | `TestListMessages_LimitOutOfRange_Returns400` |
| ListMessages / не член комнаты | CHAT-004 | `usecase` | `TestListMessages_NotMember_ReturnsErrAccessDenied` |
| ListMessages / канал не найден | CHAT-002 | `usecase` | `TestListMessages_UnknownChannel_ReturnsErrChannelNotFound` |
| HTTP / невалидный channelID | CHAT-005 | `http` | `TestListMessages_InvalidChannelUUID_Returns400` |
| HTTP / невалидный before | CHAT-005 | `http` | `TestListMessages_InvalidBeforeUUID_Returns400` |
| HTTP / нет токена | AUTH-010 | `http` | `TestListMessages_NoToken_Returns401` |
| HTTP / истёкший токен | AUTH-011 | `http` | `TestListMessages_ExpiredToken_Returns401` |
| WS / невалидный токен | AUTH-010 | `transport/ws` | `TestWS_Connect_InvalidToken_Returns401BeforeUpgrade` |
| WS / истёкший токен | AUTH-011 | `transport/ws` | `TestWS_Connect_ExpiredToken_Returns401BeforeUpgrade` |
| WS / нет токена | AUTH-010 | `transport/ws` | `TestWS_Connect_NoToken_Returns401BeforeUpgrade` |
| WS / subscribe на чужой канал | CHAT-004 | `transport/ws` | `TestWS_Subscribe_NotMember_SendsErrorFrame` |
| WS / message.send в чужой канал | CHAT-004 | `transport/ws` | `TestWS_MessageSend_NotMember_SendsErrorFrame` |
| WS / message.send в voice | CHAT-003 | `transport/ws` | `TestWS_MessageSend_VoiceChannel_SendsErrorFrame` |
| WS / невалидный JSON-фрейм | — | `transport/ws` | `TestWS_InvalidJSON_ClosesWith1003` |
| WS / фрейм больше 64 KB | — | `transport/ws` | `TestWS_OversizedFrame_ClosesWith1009` |
| WS / graceful shutdown | — | `transport/ws` | `TestWS_ServerShutdown_ClosesWith1001` |

### Room (`member.joined`)

| Сценарий | Тест |
|---|---|
| JoinByCode успешно публикует member.joined | `TestJoinByCode_Success_PublishesMemberJoined` (usecase + fake publisher) |
| JoinByCode publish ошибся — membership всё равно сохранён | `TestJoinByCode_PublishError_DoesNotRollbackMembership` |
| JoinByCode по неверному коду — publish не вызывается | `TestJoinByCode_InvalidCode_DoesNotPublish` |

### Hub (`pkg/websocket`)

| Сценарий | Тест |
|---|---|
| Register + Publish → подписчики получают фрейм | `TestHub_PublishToChannel_DeliversToSubscribers` |
| Несколько подписчиков на тот же топик | `TestHub_MultipleSubscribers_ReceiveSameFrame` |
| Подписка на чужой топик не доставляется | `TestHub_PublishToTopic_DoesNotLeakToOtherTopics` |
| Disconnect снимает все подписки | `TestHub_Disconnect_RemovesSubscriptions` |
| Двойная подписка — идемпотентна | `TestHub_DoubleSubscribe_NoDuplicateDelivery` |
| Shutdown закрывает все соединения с 1001 | `TestHub_Shutdown_ClosesAllConns` |
| Race: subscribe || publish (race detector) | `TestHub_ConcurrentSubscribePublish_NoRace` |
| Slow consumer таймаут | `TestHub_SlowClient_GetsDisconnected` |

---

## Chat / domain — Test Cases

### Message (1 файл, ~5 тестов)

| Тест | Что проверяет |
|---|---|
| `TestNewMessage_Valid_ReturnsMessage` | Все геттеры возвращают то, что было передано |
| `TestNewMessage_ZeroID_ReturnsErrInvalidMessageID` | nil-uuid не пропускается |
| `TestNewMessage_ZeroChannelID_ReturnsErrInvalidChannelID` | nil-uuid не пропускается |
| `TestNewMessage_ZeroAuthorID_ReturnsErrInvalidAuthorID` | nil-uuid не пропускается |
| `TestNewMessage_ZeroCreatedAt_ReturnsErrInvalidCreatedAt` | zero time не пропускается |

### MessageText VO (1 файл, ~9 кейсов как table-driven)

| Кейс | Ожидание |
|---|---|
| "hello" | ok, нормализовано в "hello" |
| "  hello  " | ok, нормализовано в "hello" (TrimSpace) |
| "" | `ErrInvalidMessageText` |
| "   " | `ErrInvalidMessageText` |
| 4000 рун (граница) | ok |
| 4001 руна | `ErrInvalidMessageText` |
| "hello\nworld" | ok (`\n` разрешён) |
| "hello\tworld" | ok (`\t` разрешён) |
| "\x00null" | `ErrInvalidMessageText` (control character) |

### MessageID / ChannelID / UserID / RoomID (~4 кейса каждый)

Стандартный набор: valid → ok; nil-uuid → ErrInvalid...; `IsZero()` корректно; `String()` корректно.

---

## Chat / usecase — Test Cases

### SendMessage (1 файл, ~10 тестов)

| Тест | Что проверяет |
|---|---|
| `TestSendMessage_AsMember_PersistsAndPublishes` | Save вызван, Publish вызван, output корректен |
| `TestSendMessage_NotMember_ReturnsErrAccessDenied` | Membership.Require вернул ErrChatAccessDenied; Save НЕ вызван; Publish НЕ вызван |
| `TestSendMessage_UnknownChannel_ReturnsErrChannelNotFound` | `MessageRepository.ChannelOf` вернул NotFound |
| `TestSendMessage_VoiceChannel_ReturnsErrChannelNotText` | ChannelOf вернул kind=voice |
| `TestSendMessage_EmptyText_ReturnsErrInvalidMessageText` | домен валидирует |
| `TestSendMessage_TooLong_ReturnsErrInvalidMessageText` | 4001 руна |
| `TestSendMessage_SaveFails_DoesNotPublish` | Save вернул ошибку, Publish НЕ вызван |
| `TestSendMessage_PublishFails_ReturnsSuccess` | Publish паника игнорируется, output корректен (best-effort) |
| `TestSendMessage_ChannelOfFails_ReturnsError` | Технический сбой ChannelOf |
| `TestSendMessage_MembershipFails_ReturnsError` | Технический сбой MembershipQuery |

### ListMessages (1 файл, ~7 тестов)

| Тест | Что проверяет |
|---|---|
| `TestListMessages_AsMember_ReturnsItems` | ok с limit=10, before=nil |
| `TestListMessages_WithBeforeCursor_ReturnsBefore` | before передан корректно |
| `TestListMessages_NotMember_ReturnsErrAccessDenied` | Membership.Require вернул ErrChatAccessDenied |
| `TestListMessages_UnknownChannel_ReturnsErrChannelNotFound` | ChannelOf вернул NotFound |
| `TestListMessages_EmptyChannel_ReturnsEmpty` | ListByChannel вернул [] |
| `TestListMessages_LimitNormalized` | limit=0 → 50, limit=200 → 100 (на уровне use case или handler — см. `08-api-contract.md`) |
| `TestListMessages_NextBeforeCursor` | при `len(items) == limit` возвращается next cursor |

### Stubs / Mocks

`internal/chat/usecase/fakes_test.go` содержит:

```go
type fakeMessageRepo struct {
    mu          sync.Mutex
    byID        map[uuid.UUID]*domain.Message
    listByChan  map[uuid.UUID][]*domain.Message
    channelKind map[uuid.UUID]channelInfo
    saveErr     error
    listErr     error
    channelOfErr error
}

type fakeMembershipQuery struct {
    role map[membershipKey]usecase.RoleRequirement  // expected role
    err  map[membershipKey]error                     // injected error
}

type fakeBroadcaster struct {
    mu       sync.Mutex
    channelEvents []publishedEvent
    roomEvents    []publishedEvent
    panicNext bool
}

type fixedClock struct{ now time.Time }
type fixedUUID  struct{ next []uuid.UUID; idx int }
```

SUT-паттерн как в `internal/channel/usecase/create_channel_test.go:15-48`:

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
```

---

## Chat / transport/http — Test Cases

`setup_test.go` (по образцу `internal/channel/transport/http/setup_test.go:233-276`) поднимает `httptest.NewServer` с реальным chi-роутером, реальным `usecase`, fake-репо (под `sync.Mutex`), реальным `jwtadapter.NewTokenIssuer`.

### ListMessagesHandler (~10 тестов)

| Тест | Что проверяет |
|---|---|
| `TestListMessages_AsMember_Returns200` | Happy path, формат JSON совпадает с `08-api-contract.md` |
| `TestListMessages_NoToken_Returns401_AUTH010` | Authorization header отсутствует |
| `TestListMessages_ExpiredToken_Returns401_AUTH011` | Token с прошедшим exp |
| `TestListMessages_InvalidChannelUUID_Returns400_CHAT005` | path-param не uuid |
| `TestListMessages_InvalidBeforeUUID_Returns400_CHAT005` | query before не uuid |
| `TestListMessages_LimitZero_Returns400_CHAT006` | limit=0 |
| `TestListMessages_LimitNegative_Returns400_CHAT006` | limit=-1 |
| `TestListMessages_LimitTooBig_Returns400_CHAT006` | limit=101 |
| `TestListMessages_NotMember_Returns403_CHAT004` | пользователь не член комнаты |
| `TestListMessages_UnknownChannel_Returns404_CHAT002` | channel_id не существует |

---

## Chat / transport/ws — Test Cases

`setup_test.go` поднимает `httptest.NewServer` с реальным WSHandler, реальным hub, реальным usecase, fake-репо. Клиент — тоже `nhooyr.io/websocket`. Тесты записывают/читают фреймы напрямую.

### WSHandler (~12 тестов)

| Тест | Что проверяет |
|---|---|
| `TestWS_Connect_ValidToken_UpgradesAndAutosubscribesRooms` | Подключение, проверка что conn зарегистрирован, room-подписки созданы |
| `TestWS_Connect_NoToken_Returns401BeforeUpgrade` | 401 на HTTP-уровне до upgrade |
| `TestWS_Connect_InvalidToken_Returns401BeforeUpgrade` | то же для невалидного |
| `TestWS_Connect_ExpiredToken_Returns401BeforeUpgrade` | то же для expired |
| `TestWS_Subscribe_ToOwnChannel_Returns_subscribed_frame` | subscribe ok |
| `TestWS_Subscribe_NotMember_ReturnsErrorFrame_CHAT004` | subscribe на чужой channel |
| `TestWS_Subscribe_InvalidChannelID_ReturnsErrorFrame_CHAT005` | плохой uuid |
| `TestWS_MessageSend_AsMember_PersistsAndBroadcasts` | message.send → клиент A и B получают message.new |
| `TestWS_MessageSend_NotMember_ReturnsErrorFrame_CHAT004` | без подписки/без членства |
| `TestWS_MessageSend_VoiceChannel_ReturnsErrorFrame_CHAT003` | в voice |
| `TestWS_InvalidJSON_ClosesWith1003` | "{ not a json" → close 1003 |
| `TestWS_OversizedFrame_ClosesWith1009` | фрейм > 64 KB |
| `TestWS_ClientCloses_HubUnregistersAndCleansSubscriptions` | после client close — hub чист |
| `TestWS_ServerShutdown_ClosesWith1001` | shutdown closes all |

---

## Chat / repository/postgres — Test Cases (integration, build-tag)

Build tag `integration`. `TEST_DATABASE_URL` обязательная env. `truncate` через `TRUNCATE messages, channels, room_members, rooms, ... CASCADE`. Хелпер `seedChannel(t, pool, roomID, channelID, ownerID)` расширяет существующий `seedRoom` (`internal/channel/repository/postgres/integration_helpers_test.go:30-65`).

| Тест | Что проверяет |
|---|---|
| `TestMessageRepository_Save_Insertable` | Save новой сущности, потом GetByID возвращает её |
| `TestMessageRepository_Save_DuplicatePK_ReturnsError` | повторный Save с тем же id → unique violation → INTERNAL |
| `TestMessageRepository_Save_FKChannel_ReturnsForeignKeyError` | channel_id ссылается на несуществующий канал |
| `TestMessageRepository_ListByChannel_OrderDesc` | Сообщения в порядке `created_at DESC, id DESC` |
| `TestMessageRepository_ListByChannel_BeforeCursor` | cursor по id работает корректно |
| `TestMessageRepository_ListByChannel_LimitRespected` | возвращается ровно limit |
| `TestMessageRepository_ListByChannel_EmptyChannel_ReturnsEmpty` | канал без сообщений |
| `TestMessageRepository_GetChannelKind_Text` | text-канал |
| `TestMessageRepository_GetChannelKind_Voice` | voice-канал |
| `TestMessageRepository_GetChannelKind_NotFound` | ErrChannelNotFound |

### Compile-check (`internal/chat/repository/postgres/compile_check_test.go`)

```go
var _ chatuc.MessageRepository = (*pg.MessageRepository)(nil)
```

---

## Repo Model Round-Trip Tests

| Тест | Описание |
|---|---|
| `TestMessageRowRoundTrip_AllFieldsPreserved` | Domain → InsertParams → row → Domain. Все поля совпадают (`id`, `channelID`, `authorID`, `text`, `createdAt`). |
| `TestMessageRowRoundTrip_TextWithNewlines` | Текст с `\n` сохраняется как есть |
| `TestMessageRowRoundTrip_TextAt4000Runes` | Граничный размер |
| `TestMessageRowRoundTrip_CreatedAtTimezone` | createdAt всегда возвращается в UTC |

---

## MembershipQueryAdapter (расширение для chat)

| Тест | Что проверяет |
|---|---|
| `TestMembershipQueryAdapter_RequireChat_Member_ReturnsNil` | существующий member в комнате канала |
| `TestMembershipQueryAdapter_RequireChat_NotMember_ReturnsErrChatAccessDenied` | не член → правильная chat-доменная ошибка |
| `TestMembershipQueryAdapter_RequireChat_UnknownChannel_ReturnsErrChannelNotFound` | канал не существует |

---

## Integration Tests (HTTP + WS со всем стеком)

Build tag `integration`. Поднимают реальную БД и `httptest.NewServer` с полностью склееным графом зависимостей. Покрывают E2E happy-path.

| Тест | Сценарий |
|---|---|
| `TestE2E_SendThenList_RoundTrip` | message.send через WS → GET /messages вернул отправленное |
| `TestE2E_TwoClientsSameChannel_Broadcast` | client A шлёт, client B получает |
| `TestE2E_JoinByCode_PublishesMemberJoined` | A джоинится в комнату, существующий клиент B (подключенный к этой комнате) получает member.joined |

---

## Test Count Summary

| Модуль | Domain | UseCase | Transport/HTTP | Transport/WS | Repo Integration | Hub | Round-Trip | Adapter | Итого |
|---|---|---|---|---|---|---|---|---|---|
| chat | ~25 (5 Message + 9 MessageText + 4×3 ids) | ~17 (10 SendMessage + 7 ListMessages) | ~10 | ~14 | ~10 | — | ~4 | ~3 | **~83** |
| pkg/websocket | — | — | — | — | — | ~8 | — | — | **~8** |
| room (доп.) | — | ~3 (JoinByCode + Publisher) | — | — | — | — | — | — | **~3** |
| arch_test.go | — | — | — | — | — | — | — | — | **~7 (новых архтестов)** |
| E2E integration | — | — | — | — | — | — | — | — | **~3** |

**Всего:** ~104 теста по фазе 3.1.

Все тесты соответствуют `prompts/Tests Style.txt`: stdlib `testing`, ручные fakes, `t.Parallel`, AAA, явные сравнения через `errors.Is` и `t.Fatalf`, детерминированное время и UUID через инжекцию.
