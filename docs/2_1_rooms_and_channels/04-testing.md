---
parent: ./README.md
view: quality
---

# 04 — Testing (Quality View)

Стратегия следует `prompts/Tests Style.txt` и `prompts/Domain model test.txt`. Уровни: 1) domain (без I/O) → 2) usecase с фейками → 3) repository против реального Postgres (build-tag `integration`) → 4) HTTP через `httptest`. Никакого testify — только stdlib.

## Coverage mapping (use case → error code → тест)

| Use Case | Error Code | HTTP | Тест (имя файла, имя функции) |
|----------|------------|------|--------------------------------|
| CreateRoom | `ROOM-001` | 400 | `internal/room/usecase/create_room_test.go::TestCreateRoom_InvalidName_ReturnsErrInvalidRoomName` |
| CreateRoom | `ROOM-009` | 400 | `internal/room/transport/http/create_room_handler_test.go::TestCreateRoom_BadBody_Returns400` |
| CreateRoom | (success 201) | 201 | `internal/room/usecase/create_room_test.go::TestCreateRoom_Valid_PersistsRoomAndOwnerMembership` |
| GetRoom | `ROOM-002` | 404 | `internal/room/transport/http/get_room_handler_test.go::TestGetRoom_InvalidUUID_Returns404` |
| GetRoom | `ROOM-003` | 403 | `internal/room/usecase/get_room_test.go::TestGetRoom_NotMember_ReturnsErrNotMember` |
| GetRoom | (success 200) | 200 | `internal/room/usecase/get_room_test.go::TestGetRoom_AsMember_ReturnsRoom` |
| ListUserRooms | (success) | 200 | `internal/room/usecase/list_user_rooms_test.go::TestListUserRooms_ReturnsRoomsWithRoles` |
| ListUserRooms | (empty) | 200 | `...::TestListUserRooms_NoRooms_ReturnsEmptyItems` |
| DeleteRoom | `ROOM-002` | 404 | `internal/room/transport/http/delete_room_handler_test.go::TestDeleteRoom_NotFound_Returns404` |
| DeleteRoom | `ROOM-003` | 403 | `internal/room/usecase/delete_room_test.go::TestDeleteRoom_NotMember_ReturnsErrNotMember` |
| DeleteRoom | `ROOM-005` | 403 | `internal/room/usecase/delete_room_test.go::TestDeleteRoom_AsAdmin_ReturnsErrInsufficientRole` |
| DeleteRoom | (success 204) | 204 | `internal/room/usecase/delete_room_test.go::TestDeleteRoom_AsOwner_DeletesRoom` |
| ListMembers | `ROOM-002` | 404 | `internal/room/transport/http/list_members_handler_test.go::TestListMembers_InvalidUUID_Returns404` |
| ListMembers | `ROOM-003` | 403 | `internal/room/usecase/list_members_test.go::TestListMembers_NotMember_ReturnsErrNotMember` |
| ListMembers | (success) | 200 | `internal/room/usecase/list_members_test.go::TestListMembers_AsMember_ReturnsAll` |
| RegenerateInvite | `ROOM-002` | 404 | `internal/room/transport/http/regenerate_invite_handler_test.go::TestRegenerateInvite_InvalidUUID_Returns404` |
| RegenerateInvite | `ROOM-003` | 403 | `internal/room/usecase/regenerate_invite_test.go::TestRegenerateInvite_NotMember_ReturnsErrNotMember` |
| RegenerateInvite | `ROOM-004` | 403 | `internal/room/usecase/regenerate_invite_test.go::TestRegenerateInvite_AsMember_ReturnsErrInsufficientRole` |
| RegenerateInvite | (success) | 200 | `internal/room/usecase/regenerate_invite_test.go::TestRegenerateInvite_AsAdmin_RevokesOldAndCreatesNew` |
| RegenerateInvite | (collision retry) | 200 | `internal/room/usecase/regenerate_invite_test.go::TestRegenerateInvite_FirstCodeCollides_RetriesAndSucceeds` |
| JoinByCode | `ROOM-008` | 400 | `internal/room/usecase/join_by_code_test.go::TestJoinByCode_InvalidFormat_ReturnsErrInvalidInviteCode` |
| JoinByCode | `ROOM-007` | 404 | `internal/room/usecase/join_by_code_test.go::TestJoinByCode_NoActiveInvite_ReturnsErrInviteNotFound` |
| JoinByCode | `ROOM-006` | 409 | `internal/room/usecase/join_by_code_test.go::TestJoinByCode_AlreadyMember_ReturnsErrAlreadyMember` |
| JoinByCode | (success) | 200 | `internal/room/usecase/join_by_code_test.go::TestJoinByCode_NewMember_AddsMembershipAndReturnsRoom` |
| CreateChannel | `CHANNEL-001` | 400 | `internal/channel/usecase/create_channel_test.go::TestCreateChannel_InvalidName_ReturnsErrInvalidChannelName` |
| CreateChannel | `CHANNEL-002` | 400 | `internal/channel/usecase/create_channel_test.go::TestCreateChannel_InvalidKind_ReturnsErrInvalidChannelKind` |
| CreateChannel | `CHANNEL-003` | 404 | `internal/channel/transport/http/create_channel_handler_test.go::TestCreateChannel_InvalidRoomUUID_Returns404` |
| CreateChannel | `CHANNEL-004` | 409 | `internal/channel/usecase/create_channel_test.go::TestCreateChannel_DuplicateName_ReturnsErrChannelNameAlreadyTaken` |
| CreateChannel | `CHANNEL-005` | 400 | `internal/channel/transport/http/create_channel_handler_test.go::TestCreateChannel_BadBody_Returns400` |
| CreateChannel | `CHANNEL-006` | 403 | `internal/channel/usecase/create_channel_test.go::TestCreateChannel_NotMember_ReturnsErrAccessDenied` |
| CreateChannel | `CHANNEL-007` | 403 | `internal/channel/usecase/create_channel_test.go::TestCreateChannel_AsMember_ReturnsErrInsufficientRole` |
| CreateChannel | (success) | 201 | `internal/channel/usecase/create_channel_test.go::TestCreateChannel_AsAdmin_PersistsChannel` |
| ListChannels | `CHANNEL-006` | 403 | `internal/channel/usecase/list_channels_test.go::TestListChannels_NotMember_ReturnsErrAccessDenied` |
| ListChannels | (success) | 200 | `internal/channel/usecase/list_channels_test.go::TestListChannels_AsMember_ReturnsAll` |
| DeleteChannel | `CHANNEL-003` | 404 | `internal/channel/usecase/delete_channel_test.go::TestDeleteChannel_NotFound_ReturnsErrChannelNotFound` |
| DeleteChannel | `CHANNEL-006` | 403 | `internal/channel/usecase/delete_channel_test.go::TestDeleteChannel_NotMember_ReturnsErrAccessDenied` |
| DeleteChannel | `CHANNEL-007` | 403 | `internal/channel/usecase/delete_channel_test.go::TestDeleteChannel_AsMember_ReturnsErrInsufficientRole` |
| DeleteChannel | (success 204) | 204 | `internal/channel/usecase/delete_channel_test.go::TestDeleteChannel_AsAdmin_DeletesChannel` |

## `internal/room/domain` — Test Cases (черный ящик, package `domain_test`)

### `Room` entity (4 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestNewRoom_Valid_Constructs` | NewRoom возвращает entity с корректными getter'ами |
| `TestNewRoom_ZeroID_ReturnsErrInvalidRoomID` | id == zero UUID отклоняется |
| `TestNewRoom_ZeroOwnerID_ReturnsErrInvalidUserID` | ownerID == zero UUID отклоняется |
| `TestNewRoom_ZeroCreatedAt_ReturnsErrInvalidCreatedAt` | createdAt == zero time отклоняется |
| `TestRoom_TransferOwnership_ChangesOwner` | После TransferOwnership(newOwner) `OwnerID()` возвращает newOwner |
| `TestRoom_TransferOwnership_ZeroNewOwner_ReturnsErr` | TransferOwnership с zero UserID отклоняется, состояние не меняется |

### `Membership` entity и матрица прав (15 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestNewMembership_Valid_Constructs` | конструктор happy path |
| `TestNewMembership_InvalidRole_ReturnsErrInvalidRole` | Role вне {owner, admin, member} отклоняется |
| `TestNewMembership_ZeroRoomID_ReturnsErr` | roomID == zero отклоняется |
| `TestNewMembership_ZeroUserID_ReturnsErr` | userID == zero отклоняется |
| `TestMembership_Permissions_TableDriven` | **Табличный** тест по матрице: для каждой роли (owner/admin/member) каждый `CanXxx()` метод возвращает ожидаемое булево. См. матрицу ниже. |
| `TestMembership_Promote_MemberToAdmin_Succeeds` | member → admin |
| `TestMembership_Promote_AdminNoOp` | Promote admin не падает, role остаётся admin |
| `TestMembership_Promote_OwnerNoOp` | Promote owner не падает, role остаётся owner |
| `TestMembership_Demote_AdminToMember_Succeeds` | admin → member |
| `TestMembership_Demote_MemberNoOp` | Demote member не падает |
| `TestMembership_Demote_Owner_ReturnsErrCannotDemoteOwner` | Demote owner возвращает ошибку, role не меняется |
| `TestMembership_CanKick_TableDriven` | **Табличный** по парам (actor.role, target.role) |

**Матрица прав (для `TestMembership_Permissions_TableDriven`):**

| Метод | owner | admin | member |
|-------|-------|-------|--------|
| `CanReadRoom` | ✓ | ✓ | ✓ |
| `CanReadMembers` | ✓ | ✓ | ✓ |
| `CanReadChannels` | ✓ | ✓ | ✓ |
| `CanCreateChannel` | ✓ | ✓ | ✗ |
| `CanDeleteChannel` | ✓ | ✓ | ✗ |
| `CanGenerateInvite` | ✓ | ✓ | ✗ |
| `CanDeleteRoom` | ✓ | ✗ | ✗ |

**Матрица CanKick (actor.role × target.role):**

| actor \ target | owner | admin | member |
|----------------|-------|-------|--------|
| owner | ✗ | ✓ | ✓ |
| admin | ✗ | ✗ | ✓ |
| member | ✗ | ✗ | ✗ |

### `Invite` entity (5 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestNewInvite_Valid_ConstructsActive` | конструктор + IsActive == true |
| `TestNewInvite_ZeroFields_ReturnErrors` | табличный по полям id/roomID/code/createdBy/createdAt |
| `TestInvite_Revoke_SetsRevokedAt` | Revoke(now) переводит в IsRevoked, IsActive == false |
| `TestInvite_Revoke_AlreadyRevoked_ReturnsErr` | Двойной Revoke отклоняется, состояние не меняется |
| `TestReconstructInvite_RevokedRow_RestoresState` | rebuild из БД с заданным revoked_at — IsActive == false |

### Value Objects (8 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestNewRoomName_Valid_Normalizes` | trim, типичные имена |
| `TestNewRoomName_TooShort_ReturnsErr` | пустая строка после trim |
| `TestNewRoomName_TooLong_ReturnsErr` | 65+ символов |
| `TestNewRoomName_ControlCharacter_ReturnsErr` | `\n`, `\t`, ` ` отклоняются |
| `TestParseRole_TableDriven` | "owner"/"admin"/"member" → Role; всё остальное → ErrInvalidRole |
| `TestRole_String_TableDriven` | round-trip Role → String → ParseRole |
| `TestNewInviteCode_Valid_Normalizes` | upper-case, trim, ровно 8 символов из Crockford |
| `TestNewInviteCode_InvalidLength_ReturnsErr` | 7 или 9 символов |
| `TestNewInviteCode_InvalidChar_ReturnsErr` | `I`, `L`, `O`, `U`, нижний регистр после нормализации остаются — табличный |

## `internal/channel/domain` — Test Cases (4 + 5 = 9 тестов)

### `Channel` entity (4 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestNewChannel_Valid_Constructs` | конструктор + getters |
| `TestNewChannel_ZeroFields_ReturnErrors` | id/roomID/createdAt zero |
| `TestNewChannel_InvalidName_ReturnsErr` | через NewChannelName |
| `TestNewChannel_InvalidKind_ReturnsErr` | через ParseChannelKind |

### Value Objects (5 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestNewChannelName_Valid_Normalizes` | trim |
| `TestNewChannelName_TooLong_ReturnsErr` | 65+ |
| `TestNewChannelName_Empty_ReturnsErr` | пусто после trim |
| `TestParseChannelKind_Valid_TableDriven` | "text"/"voice" → ChannelKind |
| `TestParseChannelKind_Invalid_ReturnsErr` | "video", "" |

## `internal/room/usecase` — Test Cases с фейками

Файл фейков — `internal/room/usecase/fakes_test.go`. Все фейки — ручные структуры в `package usecase_test` с явным состоянием (по образцу `internal/auth/usecase/fakes_test.go`).

### Stubs / Fakes

- `fakeRoomRepo` — in-memory map[RoomID]Room. Поведение: `SaveWithOwner`, `FindByID`, `ListByMember`, `Delete`. Гонка не моделируется.
- `fakeMembershipRepo` — in-memory map[(RoomID, UserID)]Membership. `FindByPair`, `ListByRoom`, `Add`. На `Add` мапит существующую пару → `ErrAlreadyMember`.
- `fakeInviteRepo` — in-memory + фоновое поле `nextCollisions int` для `RegenerateActive`, чтобы тестировать ретраи.
- `fakeInviteCodeGen` — список заранее заготовленных кодов. `New()` возвращает следующий из списка по очереди.
- `fixedClock` — фиксированный `time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)`.
- `fixedUUID` — фиксированный набор UUID, выдаётся по очереди.

### CreateRoom (4 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestCreateRoom_Valid_PersistsRoomAndOwnerMembership` | sут пишет и Room, и Membership(owner) в одной транзакции — фейк подтверждает оба вызова |
| `TestCreateRoom_InvalidName_ReturnsErrInvalidRoomName` | имя "" → ошибка |
| `TestCreateRoom_RepoFails_ReturnsWrappedError` | fakeRoomRepo.SaveWithOwner возвращает ошибку → usecase оборачивает |
| `TestCreateRoom_UsesInjectedClockAndUUID` | createdAt и id берутся из инжектированных портов |

### GetRoom, DeleteRoom, ListUserRooms, ListMembers, RegenerateInvite, JoinByCode

См. coverage mapping выше — файлы `get_room_test.go`, `delete_room_test.go`, `list_user_rooms_test.go`, `list_members_test.go`, `regenerate_invite_test.go`, `join_by_code_test.go`.

Каждый usecase-тест:
- Покрывает happy path + каждую ошибку из `02-behavior.md`.
- Использует SUT-конструктор `newXxxSUT(t)` (паттерн из `internal/auth/usecase/register_user_test.go:15-42`).
- Применяет `t.Parallel()`.
- Сравнивает ошибки через `errors.Is`.

## `internal/channel/usecase` — Test Cases с фейками

### Stubs / Fakes

- `fakeChannelRepo` — in-memory map[ChannelID]Channel. `Save`, `ListByRoom`, `DeleteInRoom`. На `Save` мапит дубликат `(roomID, name)` → `ErrChannelNameAlreadyTaken`.
- `fakeMembershipQuery` — fluent builder вида `withRole(roomID, userID, role)` или `notMember(roomID, userID)`. `Require(...)` возвращает `nil`/`ErrChannelAccessDenied`/`ErrChannelInsufficientRole`.
- `fixedClock`, `fixedUUID` — те же типы (можно вынести в `internal/testfixtures/timegen` позже; на старте дублируем).

См. coverage mapping выше — файлы `create_channel_test.go`, `list_channels_test.go`, `delete_channel_test.go`.

## Repo Model Round-Trip Tests

Round-trip тесты (Entity → row → Entity без потери данных) живут в `_test.go` файлах рядом с маппером.

| Тест | Что проверяет |
|------|---------------|
| `TestRoomMapper_RoundTrip_AllFieldsPreserved` | `domainToRoomRow(r) → roomRowToDomain(row)` равно исходному Room по getter'ам |
| `TestMembershipMapper_RoundTrip_AllRoles` | Табличный по owner/admin/member |
| `TestInviteMapper_RoundTrip_Active` | Активный invite с zero `revoked_at` |
| `TestInviteMapper_RoundTrip_Revoked` | Revoked invite с заданным `revoked_at` |
| `TestChannelMapper_RoundTrip_BothKinds` | Табличный по text/voice |

## Integration Tests (репозитории)

Build-tag: `//go:build integration`. Подключение через env `TEST_DATABASE_URL`. Очистка — `TRUNCATE rooms, room_members, invites, channels CASCADE` в `t.Cleanup`. По образцу `internal/auth/repository/postgres/integration_helpers_test.go:1-35`.

| Тест | Файл | Что проверяет |
|------|------|---------------|
| `TestRoomRepositoryIntegration_SaveWithOwner_Atomic` | `internal/room/repository/postgres/room_repository_integration_test.go` | После SaveWithOwner в обеих таблицах есть строки; rollback при искусственном fail второй INSERT возвращает обе таблицы в исходное состояние |
| `TestRoomRepositoryIntegration_Delete_Cascades` | … | DELETE room сносит room_members, invites, channels |
| `TestRoomRepositoryIntegration_ListByMember_OrderedDesc` | … | ORDER BY created_at DESC |
| `TestMembershipRepositoryIntegration_OneOwnerPartialUnique` | `…/membership_repository_integration_test.go` | Попытка вставить второго owner в одну комнату падает по unique-violation |
| `TestInviteRepositoryIntegration_RegenerateActive_RevokesAndInserts` | `…/invite_repository_integration_test.go` | После RegenerateActive старый invite revoked, новый active |
| `TestInviteRepositoryIntegration_OnlyOneActivePerRoom` | … | Партиальный unique index на `(room_id) WHERE revoked_at IS NULL` |
| `TestInviteRepositoryIntegration_FindActiveByCode_IgnoresRevoked` | … | Revoked не возвращается |
| `TestChannelRepositoryIntegration_UniqueNamePerRoom` | `internal/channel/repository/postgres/channel_repository_integration_test.go` | Попытка двух каналов с одинаковым name в одной комнате падает по unique-violation |
| `TestMembershipQueryAdapterIntegration_RequireAdminOrOwner` | `internal/room/repository/postgres/membership_query_integration_test.go` | Адаптер корректно возвращает nil/ErrChannelAccessDenied/ErrChannelInsufficientRole в зависимости от роли |

Выше тесты покрывают именно те инварианты, которые БД защищает (D-04, D-05, D-16). Помеченные `t.Skip("integration: requires Postgres, see issue 1.5")` — пока 1.5 не активирован.

## HTTP Tests (per-handler через httptest)

Файл `internal/room/transport/http/setup_test.go` — общий setup сервера с реальными usecase + фейковыми репо (по образцу `internal/auth/transport/http/setup_test.go:163-216`). Аналогично `internal/channel/transport/http/setup_test.go`.

Тесты идут в файлах `<handler_name>_test.go`. Каждый покрывает: 200/2xx, основные 4xx из coverage mapping, формат JSON-ответа.

## Architectural Tests (`arch_test.go`)

Расширение существующего `arch_test.go` (новые `Test*` функции):

| Тест | Что enforce'ит |
|------|----------------|
| `TestArchitecture_RoomDomainImports` | `internal/room/domain` ← stdlib + `github.com/google/uuid` |
| `TestArchitecture_RoomUseCaseImports` | `internal/room/usecase` ← stdlib + `uuid` + `internal/room/domain` |
| `TestArchitecture_RoomRepoMayImplementChannelPort` | `internal/room/repository/postgres` может импортировать `internal/channel/usecase` (только пакет; и `internal/channel/domain` для error-маппинга), но НЕ `internal/channel/transport/...` |
| `TestArchitecture_RoomTransportImports` | `internal/room/transport/http` ← stdlib + `chi` + свой usecase/domain + `pkg/httpx*` + `internal/auth/transport/http/middleware`. Никаких repo-адаптеров и никакого channel.* |
| `TestArchitecture_ChannelDomainImports` | `internal/channel/domain` ← stdlib + `uuid` |
| `TestArchitecture_ChannelUseCaseImports` | `internal/channel/usecase` ← stdlib + `uuid` + `internal/channel/domain` (НЕТ импорта `internal/room/...` — связь только через порт MembershipQuery) |
| `TestArchitecture_ChannelTransportImports` | `internal/channel/transport/http` ← stdlib + `chi` + свой usecase/domain + `pkg/httpx*` + `internal/auth/transport/http/middleware` |
| `TestArchitecture_ChannelRepoIsolated` | `internal/channel/repository/postgres` ← stdlib + `pgx`/`sqlc` + свой usecase/domain. Никаких импортов room.* |

## Compile-check tests

Файлы `compile_check_test.go` в каждом репозитории-адаптере, по образцу `internal/auth/repository/postgres/`:

| Файл | Проверяет |
|------|-----------|
| `internal/room/repository/postgres/compile_check_test.go` | `*RoomRepository`, `*MembershipRepository`, `*InviteRepository` удовлетворяют интерфейсам из `internal/room/usecase`; `*MembershipQueryAdapter` удовлетворяет `internal/channel/usecase.MembershipQuery` |
| `internal/channel/repository/postgres/compile_check_test.go` | `*ChannelRepository` удовлетворяет `internal/channel/usecase.ChannelRepository` |

## Test Count Summary

| Модуль | Domain | Usecase | Mapper round-trip | Repo integration | HTTP | Arch | Compile-check | Total |
|--------|--------|---------|-------------------|------------------|------|------|---------------|-------|
| room | 32 | 22 | 4 | 7 | 12 | 4 | 1 | 82 |
| channel | 9 | 12 | 1 | 1 | 8 | 4 | 1 | 36 |
| **Итого** | **41** | **34** | **5** | **8** | **20** | **8** | **2** | **118** |

Числа — оценка по плану из coverage mapping и matrices. Финальные числа подтвердятся в фазе реализации.

## Manual QA

Папка `manual_qa/2_1_rooms_and_channels/` (по образцу `manual_qa/1_3_auth/` и `manual_qa/1_4_auth_middleware/`):

| Файл | Сценарий |
|------|----------|
| `00_flow.http` | end-to-end сценарий: register two users → user A creates room → A invites → user B joins → A creates channel → B lists channels → A deletes channel |
| `01_create_room.http` | POST /rooms (happy + 401 без токена + 400 на пустое name) |
| `02_get_list_rooms.http` | GET /rooms и GET /rooms/{id} |
| `03_invite.http` | POST /rooms/{id}/invite + повторный вызов (старый код revoked) |
| `04_join.http` | POST /rooms/join/{code} (happy + 404 на revoked + 409 если уже member) |
| `05_members.http` | GET /rooms/{id}/members |
| `06_channels.http` | POST/GET/DELETE /rooms/{id}/channels (happy + 403 как member + 409 на дубликат имени) |
| `07_delete_room.http` | DELETE /rooms/{id} as owner / as admin (403) |
| `99_smoke.http` | health check |
| `README.md` | подготовка стенда (`make dc-up && make migrate-up && make run`), env переменные, ссылки на сценарии |
