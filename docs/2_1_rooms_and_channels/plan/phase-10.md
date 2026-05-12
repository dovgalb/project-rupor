---
phase: 10
name: Composition root + arch tests + manual QA
layer: infra
depends_on: [phase-01, phase-02, phase-03, phase-04, phase-05, phase-06, phase-07, phase-08, phase-09]
plan: ./README.md
---

# Phase 10: DI композиция + arch_test расширение + manual_qa + финализация

## Цель

Собрать всё в `cmd/server/main.go`: создать room/channel-репозитории, usecase'ы, MembershipQueryAdapter, зарегистрировать роуты. Расширить `arch_test.go` восемью новыми архитектурными тестами. Создать `manual_qa/2_1_rooms_and_channels/` с .http-сценариями. Прогнать smoke и финально проверить критерии приёмки.

## Контекст

После фаз 01-09:
- 4 миграции, расширенный sqlc.yaml.
- 2 полностью реализованных и протестированных домена + usecase + repository + transport.
- Все unit/integration/HTTP-тесты зелёные изолированно.

Что осталось — собрать в working binary и зафиксировать инварианты архитектуры.

## Файлы для модификации

### `cmd/server/main.go`

**Добавить импорты** (между `main.go:19` и `main.go:29` — внутри существующего блока импортов проекта):

```go
// добавить:
roompg "github.com/dovgalb/project-rupor/internal/room/repository/postgres"
roomdb "github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
httproom "github.com/dovgalb/project-rupor/internal/room/transport/http"
roomusecase "github.com/dovgalb/project-rupor/internal/room/usecase"

channelpg "github.com/dovgalb/project-rupor/internal/channel/repository/postgres"
channeldb "github.com/dovgalb/project-rupor/internal/channel/repository/postgres/db"
httpchannel "github.com/dovgalb/project-rupor/internal/channel/transport/http"
channelusecase "github.com/dovgalb/project-rupor/internal/channel/usecase"
```

Алиасы выбраны так, чтобы не конфликтовать с уже используемыми (`postgres` уже занят auth — поэтому `roompg`/`channelpg`; `db` тоже занят — поэтому `roomdb`/`channeldb`; `usecase` — `roomusecase`/`channelusecase`).

**Расширить блок composition** (вставить после `main.go:101` — после `meUC := usecase.NewGetCurrentUser(userRepo)`):

```go
// === Room composition ===
roomQueries := roomdb.New(pool)
roomRepo := roompg.NewRoomRepository(pool)
membershipRepo := roompg.NewMembershipRepository(roomQueries)
inviteRepo := roompg.NewInviteRepository(pool)
inviteCodeGen := roompg.NewBase32CodeGen(randSrc)

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
membershipQuery := roompg.NewMembershipQueryAdapter(roomQueries)

createChannelUC := channelusecase.NewCreateChannel(channelRepo, membershipQuery, clock, uuids)
listChannelsUC := channelusecase.NewListChannels(channelRepo, membershipQuery)
deleteChannelUC := channelusecase.NewDeleteChannel(channelRepo, membershipQuery)
```

**Расширить Mount-блок** (внутри `main.go:119-129` — после `httpauth.RegisterRoutes(...)` блока):

```go
httproom.RegisterRoutes(r, httproom.Deps{
    CreateRoom:       createRoomUC,
    GetRoom:          getRoomUC,
    ListUserRooms:    listRoomsUC,
    DeleteRoom:       deleteRoomUC,
    ListMembers:      listMembersUC,
    RegenerateInvite: regenInviteUC,
    JoinByCode:       joinByCodeUC,
    TokenIssuer:      issuer,
    Clock:            clock,
})

httpchannel.RegisterRoutes(r, httpchannel.Deps{
    CreateChannel: createChannelUC,
    ListChannels:  listChannelsUC,
    DeleteChannel: deleteChannelUC,
    TokenIssuer:   issuer,
    Clock:         clock,
})
```

### `arch_test.go`

**Добавить 8 новых тестовых функций** (после `arch_test.go:201` — в конец файла). По образцу существующих `TestArchitecture_*`. Полный список — в `../04-testing.md` §«Architectural Tests`.

```go
const (
    roomDomainPath  = modulePath + "/internal/room/domain"
    roomUsecasePath = modulePath + "/internal/room/usecase"
    roomRepoBase    = modulePath + "/internal/room/repository"
    roomTransBase   = modulePath + "/internal/room/transport"

    channelDomainPath  = modulePath + "/internal/channel/domain"
    channelUsecasePath = modulePath + "/internal/channel/usecase"
    channelRepoBase    = modulePath + "/internal/channel/repository"
    channelTransBase   = modulePath + "/internal/channel/transport"

    authMiddlewarePath = modulePath + "/internal/auth/transport/http/middleware"
    pkgHttpxPath       = modulePath + "/pkg/httpx"
    pkgHttpxMwPath     = modulePath + "/pkg/httpx/middleware"
)

func TestArchitecture_RoomDomainImports(t *testing.T) {
    t.Parallel()
    allowed := map[string]struct{}{
        "github.com/google/uuid": {},
    }
    imports := collectImports(t, "internal/room/domain")
    for file, ims := range imports {
        for _, im := range ims {
            if isStdlib(im) { continue }
            if _, ok := allowed[im]; ok { continue }
            t.Fatalf("%s imports forbidden %q", file, im)
        }
    }
}

func TestArchitecture_RoomUseCaseImports(t *testing.T) {
    t.Parallel()
    allowed := map[string]struct{}{
        "github.com/google/uuid": {},
        roomDomainPath:           {},
    }
    imports := collectImports(t, "internal/room/usecase")
    for file, ims := range imports {
        for _, im := range ims {
            if isStdlib(im) { continue }
            if _, ok := allowed[im]; ok { continue }
            t.Fatalf("%s imports forbidden %q", file, im)
        }
    }
}

func TestArchitecture_RoomRepoMayImplementChannelPort(t *testing.T) {
    t.Parallel()
    // room/repository/postgres допустимо импортирует channel/usecase и channel/domain
    // (для MembershipQueryAdapter), но НЕ channel/transport и НЕ другие домены.
    forbiddenPrefixes := []string{
        modulePath + "/internal/channel/transport/",
        modulePath + "/internal/auth/",
    }
    imports := collectImports(t, "internal/room/repository/postgres")
    for file, ims := range imports {
        for _, im := range ims {
            for _, prefix := range forbiddenPrefixes {
                if strings.HasPrefix(im, prefix) {
                    t.Fatalf("%s imports forbidden %q", file, im)
                }
            }
        }
    }
}

func TestArchitecture_RoomTransportImports(t *testing.T) {
    t.Parallel()
    allowed := map[string]struct{}{
        "github.com/go-chi/chi/v5":         {},
        "github.com/google/uuid":           {},
        roomDomainPath:                     {},
        roomUsecasePath:                    {},
        authMiddlewarePath:                 {},
        modulePath + "/internal/auth/usecase": {},
        modulePath + "/internal/auth/domain":  {},
        pkgHttpxPath:                       {},
    }
    imports := collectImports(t, "internal/room/transport/http")
    for file, ims := range imports {
        for _, im := range ims {
            if isStdlib(im) { continue }
            if _, ok := allowed[im]; ok { continue }
            if strings.HasPrefix(im, roomRepoBase+"/") {
                t.Fatalf("transport %s imports adapter %q", file, im)
            }
            if strings.HasPrefix(im, modulePath+"/internal/channel/") {
                t.Fatalf("transport %s imports cross-domain %q", file, im)
            }
            t.Fatalf("transport %s imports forbidden %q", file, im)
        }
    }
}

func TestArchitecture_ChannelDomainImports(t *testing.T) {
    t.Parallel()
    allowed := map[string]struct{}{
        "github.com/google/uuid": {},
    }
    imports := collectImports(t, "internal/channel/domain")
    for file, ims := range imports {
        for _, im := range ims {
            if isStdlib(im) { continue }
            if _, ok := allowed[im]; ok { continue }
            t.Fatalf("%s imports forbidden %q", file, im)
        }
    }
}

func TestArchitecture_ChannelUseCaseImports(t *testing.T) {
    t.Parallel()
    allowed := map[string]struct{}{
        "github.com/google/uuid": {},
        channelDomainPath:        {},
    }
    imports := collectImports(t, "internal/channel/usecase")
    for file, ims := range imports {
        for _, im := range ims {
            if isStdlib(im) { continue }
            if _, ok := allowed[im]; ok { continue }
            // Особенно — никаких room.*
            if strings.HasPrefix(im, modulePath+"/internal/room/") {
                t.Fatalf("channel/usecase %s imports room %q (must use port)", file, im)
            }
            t.Fatalf("%s imports forbidden %q", file, im)
        }
    }
}

func TestArchitecture_ChannelTransportImports(t *testing.T) {
    t.Parallel()
    allowed := map[string]struct{}{
        "github.com/go-chi/chi/v5":           {},
        "github.com/google/uuid":             {},
        channelDomainPath:                    {},
        channelUsecasePath:                   {},
        authMiddlewarePath:                   {},
        modulePath + "/internal/auth/usecase": {},
        modulePath + "/internal/auth/domain":  {},
        pkgHttpxPath:                         {},
    }
    imports := collectImports(t, "internal/channel/transport/http")
    for file, ims := range imports {
        for _, im := range ims {
            if isStdlib(im) { continue }
            if _, ok := allowed[im]; ok { continue }
            if strings.HasPrefix(im, channelRepoBase+"/") {
                t.Fatalf("transport %s imports adapter %q", file, im)
            }
            if strings.HasPrefix(im, modulePath+"/internal/room/") {
                t.Fatalf("channel/transport %s imports room %q", file, im)
            }
            t.Fatalf("%s imports forbidden %q", file, im)
        }
    }
}

func TestArchitecture_ChannelRepoIsolated(t *testing.T) {
    t.Parallel()
    // channel/repository/postgres не должен импортировать ничего, кроме своих usecase/domain
    // и стандартных pgx/sqlc-зависимостей.
    forbiddenPrefixes := []string{
        modulePath + "/internal/room/",
        modulePath + "/internal/auth/",
        modulePath + "/internal/channel/transport/",
    }
    imports := collectImports(t, "internal/channel/repository/postgres")
    for file, ims := range imports {
        for _, im := range ims {
            for _, prefix := range forbiddenPrefixes {
                if strings.HasPrefix(im, prefix) {
                    t.Fatalf("%s imports forbidden %q", file, im)
                }
            }
        }
    }
}
```

### `.claude/plans/general_plan.md`

После прохождения всех verification-чекбоксов — пометить фазу 2 как завершённую. Изменения:
- Строки 127-138: каждый `[ ]` заменить на `[x]`.
- Сводная таблица (строки 174-183): `2. Комнаты и каналы | 100%` (или иной формат — выбрать в момент закрытия).

## Файлы для создания

### `manual_qa/2_1_rooms_and_channels/`

Структура повторяет `manual_qa/1_3_auth/` и `manual_qa/1_4_auth_middleware/`.

#### `manual_qa/2_1_rooms_and_channels/README.md`
- Подготовка стенда: `make dc-up && make migrate-up && make run`.
- Требуемые env: `JWT_SECRET`, `DATABASE_URL`.
- Описание сценариев и порядка их прохождения.
- Важно: токены и id'шки переиспользуются между файлами через `@` переменные в шапке.

#### `manual_qa/2_1_rooms_and_channels/00_flow.http`
End-to-end сценарий «два пользователя, одна комната»:
1. POST /auth/register (user A)
2. POST /auth/register (user B)
3. POST /auth/login (user A) → access_token_a
4. POST /auth/login (user B) → access_token_b
5. POST /rooms (user A) → room_id
6. POST /rooms/{room_id}/invite (user A) → invite_code
7. POST /rooms/join/{invite_code} (user B) → 200 (B стал member)
8. POST /rooms/{room_id}/channels (user A, kind=text) → channel_id
9. GET /rooms/{room_id}/channels (user B) → user B видит канал
10. DELETE /rooms/{room_id}/channels/{channel_id} (user A) → 204
11. DELETE /rooms/{room_id} (user A) → 204
12. GET /rooms/{room_id} (user B) → 403 ROOM-003 (CASCADE снёс membership)

#### `manual_qa/2_1_rooms_and_channels/01_create_room.http`
- POST /rooms — happy path (user A) → 201.
- POST /rooms без токена → 401 AUTH-010.
- POST /rooms с пустым name → 400 ROOM-001.
- POST /rooms с битым JSON → 400 ROOM-009.

#### `manual_qa/2_1_rooms_and_channels/02_get_list_rooms.http`
- GET /rooms (user A) → массив с одной комнатой, role="owner".
- GET /rooms/{id} (user A) → 200.
- GET /rooms/{id} (user B без membership) → 403 ROOM-003.
- GET /rooms/{плохой uuid} → 404 ROOM-002.

#### `manual_qa/2_1_rooms_and_channels/03_invite.http`
- POST /rooms/{id}/invite (owner) → 200, code1.
- POST /rooms/{id}/invite (owner) повторно → 200, code2 (≠ code1).
- POST /rooms/{id}/invite (user B как member, после join) → 403 ROOM-004.
- Попытка использовать code1 после перевыпуска → 404 ROOM-007 (см. 04_join.http).

#### `manual_qa/2_1_rooms_and_channels/04_join.http`
- POST /rooms/join/{code} (user B) → 200, room object.
- Повторный POST → 409 ROOM-006.
- POST /rooms/join/{плохой код} → 400 ROOM-008.
- POST /rooms/join/{отозванный код} → 404 ROOM-007.

#### `manual_qa/2_1_rooms_and_channels/05_members.http`
- GET /rooms/{id}/members (user A или B) → 200, массив с двумя записями.
- GET /rooms/{id}/members (user без membership) → 403 ROOM-003.

#### `manual_qa/2_1_rooms_and_channels/06_channels.http`
- POST /rooms/{id}/channels (owner, kind=text) → 201.
- POST /rooms/{id}/channels (owner, kind=voice) → 201.
- POST /rooms/{id}/channels (member) → 403 CHANNEL-007.
- POST /rooms/{id}/channels (kind="video") → 400 CHANNEL-002.
- POST /rooms/{id}/channels (имя дубликат) → 409 CHANNEL-004.
- GET /rooms/{id}/channels (любой member) → 200 с каналами.
- DELETE /rooms/{id}/channels/{cid} (owner) → 204.
- DELETE /rooms/{id}/channels/{cid} (member) → 403 CHANNEL-007.

#### `manual_qa/2_1_rooms_and_channels/07_delete_room.http`
- DELETE /rooms/{id} (owner) → 204.
- DELETE /rooms/{id} повторно → 404 ROOM-002.
- DELETE /rooms/{id} (admin) → 403 ROOM-005.
- (Сценарий проверяет CASCADE — после DELETE list channels на 404/403.)

#### `manual_qa/2_1_rooms_and_channels/99_smoke.http`
- GET /api/v1/health → 200.

## Ключевые решения

- **Алиасы импортов в main.go** (`roompg`, `channelpg`, `roomdb`, `channeldb`, `roomusecase`, `channelusecase`) — необходимо, потому что пакеты называются одинаково (`postgres`, `db`, `usecase`) для трёх доменов. Это уже принятая практика (`bcryptadapter`, `jwtadapter`, `httpauth`, `authmw` в текущем main.go).
- **`Clock` и `TokenIssuer` переиспользуются из auth-блока** — `clock`, `issuer` уже существуют в scope `run()`. Никакой дополнительной инициализации не требуется.
- **Порядок Mount: auth → room → channel** — функционально не важен (chi route-tree строится независимо), но для читаемости и совпадения с порядком фаз в плане.
- **Манифест manual_qa соответствует `../04-testing.md` §«Manual QA»** — те же 9 файлов и сценарии.
- **arch_test изменения «append-only»** — не трогаем существующие auth-тесты, только добавляем новые. Безопасно для регрессий.

## Verification

- [ ] `cmd/server/main.go` импорты расширены (8 новых строк в блоке импортов).
- [ ] Composition блок добавлен (15+ новых строк после :101).
- [ ] Mount блок расширен (`httproom.RegisterRoutes(...)` + `httpchannel.RegisterRoutes(...)` внутри `mux.Route("/api/v1", ...)`).
- [ ] `go build ./...` чистый (полная сборка проекта).
- [ ] `go vet ./...` чистый.
- [ ] `golangci-lint run` чистый.
- [ ] `go test ./... -race -count=1` — все тесты проходят (auth + room + channel + arch).
- [ ] `arch_test.go` — 8 новых `TestArchitecture_*` функций добавлены, все зелёные.
- [ ] При поднятом Postgres: `go test -tags=integration ./...` зелёный (с учётом `t.Skip` до закрытия 1.5).
- [ ] `make dc-up && make migrate-up && make run` — сервер стартует без паники.
- [ ] Все 9 файлов в `manual_qa/2_1_rooms_and_channels/` созданы.
- [ ] `00_flow.http` end-to-end проходит руками (от register до cascade-проверки).
- [ ] Каждый из других .http-файлов прогнан руками — все статусы и коды совпадают с `../08-api-contract.md`.
- [ ] 13 критериев приёмки из `../README.md` отмечены как выполненные.
- [ ] `.claude/plans/general_plan.md` фаза 2 помечена `[x]` (опционально — после слияния PR).
