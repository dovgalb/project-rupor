---
date: 2026-05-11
feature: 2_1_rooms_and_channels
design: ../README.md
status: draft
---

# План кода: 2.1 Комнаты и каналы

## Overview

Реализация фазы 2 MVP по утверждённому дизайну в `../`. План делит работу на 10 фаз снизу вверх (миграции → domain → usecase → repository → transport → DI). Подробности конкретных решений — в `../03-decisions.md`; контракт API — в `../08-api-contract.md`; маппинг и SQL — в `../06-repo-model.md`.

## Phase Strategy

**Bottom-up**, room и channel в отдельных фазах одного слоя.

Почему bottom-up:
- Каждая фаза тестируется изолированно: domain — без I/O, usecase — с фейками, repository — против реального Postgres, transport — через `httptest`. Это естественный порядок для нашего стиля тестов (`prompts/Tests Style.txt`).
- Composition root (`cmd/server/main.go`) собирается в самом конце, когда все детали готовы.

Почему room и channel в отдельных фазах:
- Domain'ы независимы друг от друга в разрезе entities/VO; их объединение в одну фазу не уменьшает работу, а размывает фокус ревью.
- Единственная точка их встречи — `MembershipQuery` port: объявляется в `internal/channel/usecase` (фаза 05), реализуется адаптером в `internal/room/repository/postgres/membership_query.go` (фаза 06). Зависимость 05 → 06 явная.
- Транспортные слои разные (`/rooms*` против `/rooms/{id}/channels*`), удобнее ревьюить раздельно.

Phase 02+03 (domain) можно реализовать параллельно. Phase 04+05 — параллельно после 02+03. Phase 07 — параллельно с 06 (зависит только от 05). Phase 08+09 — параллельно после 06+07. Distance между параллельными фазами implementer выбирает сам.

## Phases

| # | Фаза | Слой | Зависимости | Status |
|---|------|------|-------------|--------|
| 01 | Migrations + sqlc.yaml + sqlc generate | migrations | none | ☐ |
| 02 | Room domain (entities, VO, errors) | domain | 01 | ☐ |
| 03 | Channel domain (entities, VO, errors) | domain | 01 | ☐ |
| 04 | Room usecase (7 use case + ports + fakes) | usecase | 02 | ☐ |
| 05 | Channel usecase (3 use case + MembershipQuery port + fakes) | usecase | 03 | ☐ |
| 06 | Room repository postgres (3 repo + MembershipQueryAdapter) | repository | 01, 04, 05 | ☐ |
| 07 | Channel repository postgres (ChannelRepository) | repository | 01, 05 | ☐ |
| 08 | Room transport HTTP (handlers + dto + routes) | transport | 04 (06 для smoke) | ☐ |
| 09 | Channel transport HTTP (handlers + dto + routes) | transport | 05 (07 для smoke) | ☐ |
| 10 | Composition + arch_test + manual_qa | infra | 01..09 | ☐ |

## File Map

### New Files

**Миграции** (Phase 01):
- `migrations/0004_rooms.up.sql` / `.down.sql` — таблица rooms
- `migrations/0005_room_members.up.sql` / `.down.sql` — таблица room_members + partial unique для one-owner
- `migrations/0006_invites.up.sql` / `.down.sql` — таблица invites + 2 partial unique
- `migrations/0007_channels.up.sql` / `.down.sql` — таблица channels + UNIQUE(room_id, name)

**Room domain** (Phase 02):
- `internal/room/domain/room.go` — entity Room (NewRoom, ReconstructRoom, TransferOwnership, getters)
- `internal/room/domain/membership.go` — entity Membership (Promote/Demote/CanXxx/CanKick)
- `internal/room/domain/invite.go` — entity Invite (Revoke, IsActive, IsRevoked)
- `internal/room/domain/room_id.go` — VO RoomID
- `internal/room/domain/user_id.go` — VO UserID (свой, не из auth)
- `internal/room/domain/invite_id.go` — VO InviteID
- `internal/room/domain/room_name.go` — VO RoomName
- `internal/room/domain/role.go` — enum-VO Role + ParseRole
- `internal/room/domain/invite_code.go` — VO InviteCode (Crockford base32, 8 chars)
- `internal/room/domain/errors.go` — sentinel ошибки (ErrRoomNotFound, ErrNotMember, …)
- `internal/room/domain/room_test.go`, `membership_test.go`, `invite_test.go`, `room_name_test.go`, `role_test.go`, `invite_code_test.go` — domain tests (package `domain_test`)
- `internal/room/domain/testing_builders_test.go` — Builder'ы для тестов (RoomBuilder, MembershipBuilder, InviteBuilder)

**Channel domain** (Phase 03):
- `internal/channel/domain/channel.go` — entity Channel
- `internal/channel/domain/channel_id.go`, `room_id.go`, `user_id.go` — VO
- `internal/channel/domain/channel_name.go` — VO ChannelName
- `internal/channel/domain/channel_kind.go` — enum-VO ChannelKind + ParseChannelKind
- `internal/channel/domain/errors.go` — sentinel (ErrChannelNotFound, ErrChannelNameAlreadyTaken, ErrChannelAccessDenied, ErrChannelInsufficientRole, …)
- `internal/channel/domain/channel_test.go`, `channel_name_test.go`, `channel_kind_test.go`
- `internal/channel/domain/testing_builders_test.go`

**Room usecase** (Phase 04):
- `internal/room/usecase/ports.go` — все интерфейсы зависимостей (RoomRepository, MembershipRepository, InviteRepository, InviteCodeGenerator, Clock, UUIDGenerator)
- `internal/room/usecase/create_room.go` (+ `_test.go`)
- `internal/room/usecase/get_room.go` (+ `_test.go`)
- `internal/room/usecase/list_user_rooms.go` (+ `_test.go`)
- `internal/room/usecase/delete_room.go` (+ `_test.go`)
- `internal/room/usecase/list_members.go` (+ `_test.go`)
- `internal/room/usecase/regenerate_invite.go` (+ `_test.go`)
- `internal/room/usecase/join_by_code.go` (+ `_test.go`)
- `internal/room/usecase/fakes_test.go` — fakeRoomRepo, fakeMembershipRepo, fakeInviteRepo, fakeInviteCodeGen, fixedClock, fixedUUID, must-хелперы

**Channel usecase** (Phase 05):
- `internal/channel/usecase/ports.go` — ChannelRepository, MembershipQuery, RoleRequirement, Clock, UUIDGenerator
- `internal/channel/usecase/create_channel.go` (+ `_test.go`)
- `internal/channel/usecase/list_channels.go` (+ `_test.go`)
- `internal/channel/usecase/delete_channel.go` (+ `_test.go`)
- `internal/channel/usecase/fakes_test.go` — fakeChannelRepo, fakeMembershipQuery (with builder helpers), fixedClock, fixedUUID

**Room repository postgres** (Phase 06):
- `internal/room/repository/postgres/queries/rooms.sql` — sqlc запросы
- `internal/room/repository/postgres/queries/room_members.sql`
- `internal/room/repository/postgres/queries/invites.sql`
- `internal/room/repository/postgres/db/*` — сгенерированный sqlc-код (после `make sqlc`)
- `internal/room/repository/postgres/room_repository.go` — Save, SaveWithOwner, FindByID, ListByMember, Delete
- `internal/room/repository/postgres/membership_repository.go` — FindByPair, ListByRoom, Add (с маппингом unique violation в ErrAlreadyMember)
- `internal/room/repository/postgres/invite_repository.go` — RegenerateActive, FindActiveByCode (с маппингом unique violation в errInviteCodeCollision)
- `internal/room/repository/postgres/membership_query.go` — MembershipQueryAdapter (реализация channel.usecase.MembershipQuery)
- `internal/room/repository/postgres/mapper.go` — row ↔ domain маппинг (4 пары функций)
- `internal/room/repository/postgres/pgerr.go` — isUniqueViolation, isForeignKeyViolation
- `internal/room/repository/postgres/code_gen.go` — base32CodeGen (реализация InviteCodeGenerator) + конструктор от io.Reader
- `internal/room/repository/postgres/compile_check_test.go` — статические assertions
- `internal/room/repository/postgres/room_repository_integration_test.go` (build tag integration)
- `internal/room/repository/postgres/membership_repository_integration_test.go`
- `internal/room/repository/postgres/invite_repository_integration_test.go`
- `internal/room/repository/postgres/membership_query_integration_test.go`
- `internal/room/repository/postgres/integration_helpers_test.go` (build tag integration)
- `internal/room/repository/postgres/mapper_test.go` — round-trip тесты

**Channel repository postgres** (Phase 07):
- `internal/channel/repository/postgres/queries/channels.sql`
- `internal/channel/repository/postgres/db/*` — sqlc gen
- `internal/channel/repository/postgres/channel_repository.go` — Save, ListByRoom, DeleteInRoom
- `internal/channel/repository/postgres/mapper.go`
- `internal/channel/repository/postgres/pgerr.go`
- `internal/channel/repository/postgres/compile_check_test.go`
- `internal/channel/repository/postgres/channel_repository_integration_test.go`
- `internal/channel/repository/postgres/integration_helpers_test.go`
- `internal/channel/repository/postgres/mapper_test.go`

**Room transport HTTP** (Phase 08):
- `internal/room/transport/http/dto.go` — приватные request/response типы + jsonDecode/jsonEncode
- `internal/room/transport/http/error_mapper.go` — mapError, writeError, writeBadBody, ROOM-NNN коды
- `internal/room/transport/http/routes.go` — Deps + RegisterRoutes (Mount /rooms внутри chi.Router)
- `internal/room/transport/http/create_room_handler.go`, `get_room_handler.go`, `list_rooms_handler.go`, `delete_room_handler.go`, `list_members_handler.go`, `regenerate_invite_handler.go`, `join_by_code_handler.go`
- `internal/room/transport/http/setup_test.go` — общий setup httptest-сервера (по образцу auth/setup_test.go)
- Соответствующие `<handler>_test.go` файлы

**Channel transport HTTP** (Phase 09):
- `internal/channel/transport/http/dto.go`
- `internal/channel/transport/http/error_mapper.go` — CHANNEL-NNN коды
- `internal/channel/transport/http/routes.go`
- `internal/channel/transport/http/create_channel_handler.go`, `list_channels_handler.go`, `delete_channel_handler.go`
- `internal/channel/transport/http/setup_test.go`
- Соответствующие `<handler>_test.go`

**Composition + manual QA** (Phase 10):
- `manual_qa/2_1_rooms_and_channels/00_flow.http`, `01_create_room.http`, `02_get_list_rooms.http`, `03_invite.http`, `04_join.http`, `05_members.http`, `06_channels.http`, `07_delete_room.http`, `99_smoke.http`, `README.md`

### Modified Files

- `sqlc.yaml` — заменить целиком: `sql:` массив с тремя записями (auth, room, channel). См. Phase 01 + `../06-repo-model.md` §«sqlc.yaml — расширение».
- `cmd/server/main.go:19-29` — добавить импорты пакетов room/channel.
- `cmd/server/main.go:67-101` — после блока auth-инициализации добавить блок room+channel (репозитории, usecase, code generator, MembershipQueryAdapter).
- `cmd/server/main.go:119-129` — внутри `mux.Route("/api/v1", ...)` добавить `httproom.RegisterRoutes(...)` и `httpchannel.RegisterRoutes(...)`.
- `arch_test.go` (после строки 201) — добавить 8 новых `TestArchitecture_*` функций (см. Phase 10).
- `.gitignore` — если потребуется (новые `db/` каталоги для sqlc уже не игнорируются — auth такой же).
- `internal/room/{domain,usecase,transport/http,repository/postgres}/.gitkeep` — удалить после первого реального файла в каталоге.
- `internal/channel/{domain,usecase,transport/http,repository/postgres}/.gitkeep` — удалить.
- Удалить пустые `internal/{user,chat,voice}/...` НЕ нужно — это других фаз скелет.

### Files NOT modified

- `Makefile` — текущие цели (`make sqlc`, `make migrate-up`, `make test`) уже покрывают всё нужное.
- `docker-compose.yml`, `config/`, `.env.example`, `pkg/httpx*` — не требуют изменений.
- Существующий `internal/auth/...` — никаких правок.

## DI Integration

### Init chain position

Текущая цепочка в `cmd/server/main.go::run`:

1. `pgxpool.New` (`main.go:67`) — общий пул для auth/room/channel.
2. Auth-блок (`main.go:73-101`): `db.New(pool)` → repos → bcrypt/jwt → runtime → dummyHash → 4 usecase'а.
3. `chi.NewRouter()` (`main.go:103`).
4. Глобальные middleware (`main.go:114-117`).
5. Mount /api/v1 (`main.go:119-129`).
6. `http.Server` + graceful shutdown (`main.go:131-167`).

**Новый блок room+channel вставляется между шагом 2 и шагом 3** (т.е. между текущей строкой `main.go:101` и `main.go:103`).

### Composition root changes (Phase 10, детальный шаг)

```go
// после main.go:101 — конец auth-блока
//
// === Room composition ===
roomQueries := roomdb.New(pool)
roomRepo := roompg.NewRoomRepository(pool)            // принимает pool, т.к. SaveWithOwner транзакционен
membershipRepo := roompg.NewMembershipRepository(roomQueries)
inviteRepo := roompg.NewInviteRepository(pool)        // принимает pool, т.к. RegenerateActive транзакционен
inviteCodeGen := roompg.NewBase32CodeGen(randSrc)     // randSrc уже есть (cryptoRand)

createRoomUC := roomusecase.NewCreateRoom(roomRepo, clock, uuids)
getRoomUC := roomusecase.NewGetRoom(roomRepo, membershipRepo)
listRoomsUC := roomusecase.NewListUserRooms(roomRepo)
deleteRoomUC := roomusecase.NewDeleteRoom(roomRepo, membershipRepo)
listMembersUC := roomusecase.NewListMembers(membershipRepo)
regenInviteUC := roomusecase.NewRegenerateInvite(inviteRepo, membershipRepo, inviteCodeGen, clock, uuids)
joinByCodeUC := roomusecase.NewJoinByCode(inviteRepo, membershipRepo, roomRepo, clock)

// === Channel composition ===
channelQueries := channeldb.New(pool)
channelRepo := channelpg.NewChannelRepository(channelQueries)
membershipQuery := roompg.NewMembershipQueryAdapter(roomQueries)  // реализует channel.usecase.MembershipQuery

createChannelUC := channelusecase.NewCreateChannel(channelRepo, membershipQuery, clock, uuids)
listChannelsUC := channelusecase.NewListChannels(channelRepo, membershipQuery)
deleteChannelUC := channelusecase.NewDeleteChannel(channelRepo, membershipQuery)
```

И в Mount-блоке (внутри `mux.Route("/api/v1", func(r chi.Router) { ... })` после `httpauth.RegisterRoutes(...)`):

```go
httproom.RegisterRoutes(r, httproom.Deps{
    CreateRoom:        createRoomUC,
    GetRoom:           getRoomUC,
    ListUserRooms:     listRoomsUC,
    DeleteRoom:        deleteRoomUC,
    ListMembers:       listMembersUC,
    RegenerateInvite:  regenInviteUC,
    JoinByCode:        joinByCodeUC,
    TokenIssuer:       issuer,
    Clock:             clock,
})

httpchannel.RegisterRoutes(r, httpchannel.Deps{
    CreateChannel: createChannelUC,
    ListChannels:  listChannelsUC,
    DeleteChannel: deleteChannelUC,
    TokenIssuer:   issuer,
    Clock:         clock,
})
```

### Initialization order

1. `pool` уже создан (`main.go:67`) — переиспользуется.
2. `roomQueries := roomdb.New(pool)` — `room/repository/postgres/db` пакет.
3. Room репозитории — три штуки. `RoomRepository` и `InviteRepository` принимают `*pgxpool.Pool` (для транзакций), `MembershipRepository` — `*db.Queries`.
4. `inviteCodeGen` — берёт `cryptoRand` (уже определён в `cmd/server/runtime.go`).
5. Семь room-usecase.
6. `channelQueries := channeldb.New(pool)`.
7. `ChannelRepository` принимает `*db.Queries` (нет транзакций в channel).
8. `MembershipQueryAdapter` — принимает `roomQueries` (живёт в `roompg`, импортирует `channel.usecase` для интерфейса и `channel.domain` для error sentinels).
9. Три channel-usecase.
10. Mount: `httproom.RegisterRoutes(r, ...)` и `httpchannel.RegisterRoutes(r, ...)` — внутри уже существующего `mux.Route("/api/v1", ...)`. Никаких новых глобальных middleware.

## Error Codes

**Range:** `ROOM-001..ROOM-009` + `CHANNEL-001..CHANNEL-007`

**Conflict check (на основе `../research.md`):** существующие коды занимают `AUTH-001..AUTH-012`. Префиксы `ROOM-*` и `CHANNEL-*` — свободные. Конфликтов нет.

| Code | Description | HTTP |
|------|-------------|------|
| `ROOM-001` | invalid room name | 400 |
| `ROOM-002` | room not found | 404 |
| `ROOM-003` | not a member | 403 |
| `ROOM-004` | insufficient role: admin or owner required | 403 |
| `ROOM-005` | only owner can delete room | 403 |
| `ROOM-006` | already a member | 409 |
| `ROOM-007` | invite not found or revoked | 404 |
| `ROOM-008` | invalid invite code | 400 |
| `ROOM-009` | invalid request body | 400 |
| `CHANNEL-001` | invalid channel name | 400 |
| `CHANNEL-002` | invalid channel kind | 400 |
| `CHANNEL-003` | channel or room not found | 404 |
| `CHANNEL-004` | channel name already taken | 409 |
| `CHANNEL-005` | invalid request body | 400 |
| `CHANNEL-006` | access denied: not a member | 403 |
| `CHANNEL-007` | insufficient role: admin or owner required | 403 |

## Success Criteria

- [ ] Все 10 фаз завершены и проверены (см. checkbox в каждом `phase-NN.md`).
- [ ] `go build ./...` чистый.
- [ ] `go test ./... -race -count=1` проходит (`make test`).
- [ ] `go test -tags=integration ./...` проходит при поднятом Postgres (`make dc-up && make migrate-up`); требует `TEST_DATABASE_URL`.
- [ ] `make lint` (`golangci-lint run`) чистый.
- [ ] `make sqlc` без ошибок и без диффа в сгенерированных файлах после повторного запуска.
- [ ] Все коды ошибок из `../08-api-contract.md` покрыты тестами (см. coverage mapping в `../04-testing.md`).
- [ ] Архитектурные тесты в `arch_test.go` для room/channel зелёные (8 новых `TestArchitecture_*`).
- [ ] API-контракт совпадает с реализацией (ручная сверка через `manual_qa/2_1_rooms_and_channels/*.http`).
- [ ] Все 13 критериев приёмки из `../README.md` выполнены и подтверждены `00_flow.http`.
- [ ] В `/.claude/plans/general_plan.md` фаза 2 отмечена как готовая (`[x]` по чек-листу `general_plan.md:127-138`).
