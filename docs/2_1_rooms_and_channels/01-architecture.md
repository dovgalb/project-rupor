---
parent: ./README.md
view: logical
---

# 01 — Architecture (Logical View)

C4-нарратив: L1 → L2 → L3. На L3 показаны два затронутых модуля (`internal/room`, `internal/channel`) и их связь с уже существующим `internal/auth` через middleware-пакет.

## C4 Level 1 — System Context

```mermaid
%% System Context — Rupor (после PR-2)
flowchart LR
    user(["«person»<br/>Member<br/>Авторизованный пользователь"]):::persona
    rupor["«system»<br/>Rupor<br/>Коммуникационная платформа"]:::system
    pg[("«system_db»<br/>PostgreSQL 16<br/>Хранит users, refresh_tokens,<br/>rooms, room_members, invites, channels")]:::db

    user -->|"HTTPS REST"| rupor
    rupor -->|"SQL / sqlc / pgx"| pg

    classDef persona fill:#08427b,color:#fff,stroke:#073b6f,stroke-width:1px
    classDef system  fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef db      fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef ext     fill:#999999,color:#fff,stroke:#6b6b6b,stroke-width:1px
```

Внешних систем PR-2 не привносит. WebSocket-сигнализация и WebRTC-инфраструктура появляются в фазах 3-4 и здесь не отображены.

## C4 Level 2 — Containers

```mermaid
%% Container Diagram — Rupor (PR-2)
flowchart LR
    user(["«person»<br/>Member"]):::persona
    httpc["«container»<br/>HTTP Client<br/>JetBrains HTTP Client / curl /<br/>будущий React-фронт"]:::ext

    subgraph rupor["Rupor"]
        api["«container»<br/>API Server<br/>Go 1.25 + chi v5<br/>HTTP REST"]:::system
        pg[("«container_db»<br/>PostgreSQL 16<br/>users, rooms, room_members,<br/>invites, channels, refresh_tokens")]:::db
    end

    user -->|"использует"| httpc
    httpc -->|"HTTPS<br/>JSON + Bearer JWT"| api
    api -->|"SQL via pgx pool"| pg

    classDef persona fill:#08427b,color:#fff,stroke:#073b6f,stroke-width:1px
    classDef system  fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef db      fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef ext     fill:#999999,color:#fff,stroke:#6b6b6b,stroke-width:1px
```

Затрагиваемые контейнеры:
- **API Server** (`cmd/server/main.go`) — добавляются новые модули `internal/room/`, `internal/channel/`, регистрация роутов `/rooms*` и `/rooms/{roomID}/channels*` внутри уже существующей группы `/api/v1`.
- **PostgreSQL** — добавляются 4 таблицы (`rooms`, `room_members`, `invites`, `channels`), миграции `0004…0007`.

Новых контейнеров нет (Rupor — монолит).

## C4 Level 3 — Components

### 3.1 Модуль `internal/room/`

```mermaid
flowchart TB
    subgraph room["internal/room"]
        subgraph d_room["domain"]
            E_Room["Room (entity)"]
            E_Membership["Membership (entity)"]
            E_Invite["Invite (entity)"]
            VO_RoomID["RoomID, MembershipID, InviteID, UserID"]
            VO_Other["RoomName, Role, InviteCode"]
            ERR["errors.go (sentinel)"]
        end
        subgraph u_room["usecase"]
            UC_Create["CreateRoom"]
            UC_Get["GetRoom"]
            UC_List["ListUserRooms"]
            UC_Delete["DeleteRoom"]
            UC_Members["ListMembers"]
            UC_Invite["RegenerateInvite"]
            UC_Join["JoinByCode"]
            P_Rooms["«port»<br/>RoomRepository"]
            P_Members["«port»<br/>MembershipRepository"]
            P_Invites["«port»<br/>InviteRepository"]
            P_Codes["«port»<br/>InviteCodeGenerator"]
            P_Clock["«port»<br/>Clock"]
            P_UUID["«port»<br/>UUIDGenerator"]
        end
        subgraph t_room["transport/http (httproom)"]
            H_Create["createRoomHandler"]
            H_Get["getRoomHandler"]
            H_List["listRoomsHandler"]
            H_Delete["deleteRoomHandler"]
            H_Members["listMembersHandler"]
            H_Invite["regenerateInviteHandler"]
            H_Join["joinByCodeHandler"]
            DTO["dto.go"]
            EM["error_mapper.go (ROOM-XXX)"]
            R["routes.go (Mount /rooms)"]
        end
        subgraph r_room["repository"]
            subgraph pg_room["postgres (roompg)"]
                Repo_Room["RoomRepository"]
                Repo_Mem["MembershipRepository"]
                Repo_Inv["InviteRepository"]
                Repo_MQ["MembershipQueryAdapter"]
                Mapper["mapper.go"]
                SQLC["db/* (sqlc gen)"]
            end
            subgraph rt_room["runtime"]
                CG["base32CodeGen"]
            end
        end
    end

    H_Create --> UC_Create
    H_Get --> UC_Get
    H_List --> UC_List
    H_Delete --> UC_Delete
    H_Members --> UC_Members
    H_Invite --> UC_Invite
    H_Join --> UC_Join

    UC_Create --> P_Rooms
    UC_Create --> P_Clock
    UC_Create --> P_UUID
    UC_Get --> P_Rooms
    UC_Get --> P_Members
    UC_List --> P_Rooms
    UC_Delete --> P_Rooms
    UC_Delete --> P_Members
    UC_Members --> P_Members
    UC_Invite --> P_Invites
    UC_Invite --> P_Members
    UC_Invite --> P_Codes
    UC_Join --> P_Invites
    UC_Join --> P_Members
    UC_Join --> P_Rooms

    Repo_Room -.implements.-> P_Rooms
    Repo_Mem  -.implements.-> P_Members
    Repo_Inv  -.implements.-> P_Invites
    CG        -.implements.-> P_Codes

    UC_Create --> E_Room
    UC_Create --> E_Membership
    UC_Invite --> E_Invite
    UC_Join --> E_Membership
```

#### 3.1.1 Доменные сущности

**`Room` (entity)**
- Поля (приватные): `id RoomID`, `name RoomName`, `ownerID UserID`, `createdAt time.Time`.
- `NewRoom(id, ownerID, name, createdAt) (*Room, error)` — для usecase. Инварианты: `id != Zero`, `ownerID != Zero`, `createdAt != Zero`.
- `ReconstructRoom(id, ownerID, name, createdAt) (*Room, error)` — для репозитория.
- Метод `TransferOwnership(newOwner UserID) error` — меняет ownerID на роли entity (мутация инкапсулирована, см. фазу 3+; для PR-2 присутствует ради честного rich-domain — без HTTP-эндпоинта).
- Геттеры: `ID()`, `OwnerID()`, `Name()`, `CreatedAt()`.

**`Membership` (entity)**
- Identity = пара `(roomID, userID)`. Поля: `roomID`, `userID`, `role Role`, `joinedAt`.
- `NewMembership(roomID, userID, role, joinedAt)` — конструктор с инвариантами (zero-проверка на id, валидная роль).
- `ReconstructMembership(...)` — для репозитория.
- Бизнес-методы:
  - `Promote() error` — `member` → `admin`. Owner промоут не требует.
  - `Demote() error` — `admin` → `member`. Owner понизить нельзя (`ErrCannotDemoteOwner`).
- Методы-запросы (доменная матрица прав):
  - `CanCreateRoom() bool` — всегда true для авторизованного, не привязано к Membership; технически живёт не здесь.
  - `CanReadRoom() bool` — true для всех ролей.
  - `CanReadMembers() bool` — true для всех ролей.
  - `CanReadChannels() bool` — true для всех ролей.
  - `CanCreateChannel() bool` — `owner | admin`.
  - `CanDeleteChannel() bool` — `owner | admin`.
  - `CanGenerateInvite() bool` — `owner | admin`.
  - `CanDeleteRoom() bool` — `owner` only.
  - `CanKick(target Membership) bool` — owner может kick admin/member; admin — только member; member — никого. (kick-эндпоинта в PR-2 нет, но метод доменно нужен и тестируется — для будущих фаз.)

**`Invite` (entity)**
- Поля: `id InviteID`, `roomID RoomID`, `code InviteCode`, `createdBy UserID`, `createdAt time.Time`, `revokedAt time.Time` (zero = активен).
- `NewInvite(id, roomID, code, createdBy, createdAt) (*Invite, error)` — для usecase.
- `ReconstructInvite(...)` — для репозитория.
- `Revoke(now time.Time) error` — отказывает уже отозванному (`ErrInviteAlreadyRevoked`).
- Запросы: `IsActive() bool`, `IsRevoked() bool`.

#### 3.1.2 Value objects

| VO | Тип-носитель | Правила |
|----|--------------|---------|
| `RoomID` | `uuid.UUID` | non-zero |
| `MembershipID` | — | составной ключ; не VO, а пара (roomID, userID) |
| `InviteID` | `uuid.UUID` | non-zero |
| `UserID` | `uuid.UUID` | non-zero. **Свой** в `room/domain/`, параллельный одноимённому из `auth/domain` (см. `03-decisions.md` §D-08) |
| `RoomName` | `string` | trim; длина 1..64; без управляющих символов |
| `Role` | enum | `RoleOwner | RoleAdmin | RoleMember`; парсер `ParseRole(string) (Role, error)`; `String()` |
| `InviteCode` | `string` | строго 8 символов; алфавит Crockford base32 без `I/L/O/U` (32 символа: `0123456789ABCDEFGHJKMNPQRSTVWXYZ`); upper-case при создании; парсер нормализует регистр |

#### 3.1.3 State machine: Invite

```
created (active) ──Revoke()──▶ revoked
                ◀──(больше не оживает; новый Invite — отдельная сущность с новым id и новым code)──
```

Логика «один активный код на комнату» обеспечивается на уровне **репозитория** через `RegenerateActive` (атомарно: `UPDATE invites SET revoked_at=now() WHERE room_id=$1 AND revoked_at IS NULL` + `INSERT new`) и подкреплена частичным уникальным индексом БД (см. `06-repo-model.md`).

#### 3.1.4 State machine: Membership.role

```
member ──Promote()──▶ admin ──Promote()──X (no-op / NoChange)
member ◀──Demote()── admin ──Demote()──X (no-op / NoChange)
owner ──Promote()──X (NoChange)
owner ──Demote()──X ErrCannotDemoteOwner

owner ↔ admin: только через TransferOwnership (Room.entity), который меняет (старый owner)→admin, (новый owner)→owner.
TransferOwnership в PR-2 не выставлен через HTTP — только доменный метод.
```

#### 3.1.5 Доменные ошибки (`internal/room/domain/errors.go`)

Все sentinel, префикс сообщения `room: …`.

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
ErrInsufficientRole       // member/admin тянет на owner-операцию или member на admin-операцию
ErrInviteNotFound         // активный код не найден или revoked
ErrInviteAlreadyRevoked
ErrCannotDemoteOwner
```

### 3.2 Модуль `internal/channel/`

```mermaid
flowchart TB
    subgraph channel["internal/channel"]
        subgraph d_ch["domain"]
            E_Ch["Channel (entity)"]
            VO_Ch["ChannelID, RoomID, UserID,<br/>ChannelName, ChannelKind"]
            ERR_Ch["errors.go (sentinel)"]
        end
        subgraph u_ch["usecase"]
            UC_CCreate["CreateChannel"]
            UC_CList["ListChannels"]
            UC_CDelete["DeleteChannel"]
            P_Channels["«port»<br/>ChannelRepository"]
            P_MQ["«port»<br/>MembershipQuery"]
            P_Clock_Ch["«port»<br/>Clock"]
            P_UUID_Ch["«port»<br/>UUIDGenerator"]
        end
        subgraph t_ch["transport/http (httpchannel)"]
            H_CCreate["createChannelHandler"]
            H_CList["listChannelsHandler"]
            H_CDelete["deleteChannelHandler"]
            DTO_Ch["dto.go"]
            EM_Ch["error_mapper.go (CHANNEL-XXX)"]
            R_Ch["routes.go (Mount /rooms/{roomID}/channels)"]
        end
        subgraph r_ch["repository/postgres (channelpg)"]
            Repo_Ch["ChannelRepository"]
            Mapper_Ch["mapper.go"]
            SQLC_Ch["db/* (sqlc gen)"]
        end
    end

    subgraph room_ext["internal/room (внешний для channel)"]
        Repo_MQ_Ext["MembershipQueryAdapter<br/>(реализация порта<br/>channel.usecase.MembershipQuery)"]
    end

    H_CCreate --> UC_CCreate
    H_CList --> UC_CList
    H_CDelete --> UC_CDelete

    UC_CCreate --> P_Channels
    UC_CCreate --> P_MQ
    UC_CCreate --> P_Clock_Ch
    UC_CCreate --> P_UUID_Ch
    UC_CList --> P_Channels
    UC_CList --> P_MQ
    UC_CDelete --> P_Channels
    UC_CDelete --> P_MQ

    Repo_Ch    -.implements.-> P_Channels
    Repo_MQ_Ext -.implements.-> P_MQ

    UC_CCreate --> E_Ch
```

#### 3.2.1 Доменные сущности

**`Channel` (entity)**
- Поля: `id ChannelID`, `roomID RoomID`, `name ChannelName`, `kind ChannelKind`, `createdAt time.Time`.
- `NewChannel(id, roomID, name, kind, createdAt) (*Channel, error)` — инварианты: id и roomID не zero, valid name/kind, createdAt не zero.
- `ReconstructChannel(...)` — для репозитория.
- В PR-2 у Channel нет мутирующих бизнес-методов (rename отложен).
- Геттеры: `ID()`, `RoomID()`, `Name()`, `Kind()`, `CreatedAt()`.

#### 3.2.2 Value objects

| VO | Тип | Правила |
|----|-----|---------|
| `ChannelID` | `uuid.UUID` | non-zero |
| `RoomID` | `uuid.UUID` | non-zero. **Свой** в `channel/domain/`, параллельно `room/domain/`. См. §D-08 |
| `UserID` | `uuid.UUID` | non-zero. **Свой** в `channel/domain/` |
| `ChannelName` | `string` | trim; длина 1..64; без управляющих символов |
| `ChannelKind` | enum | `ChannelKindText | ChannelKindVoice`; `ParseChannelKind(string) (ChannelKind, error)`; `String()` |

#### 3.2.3 Доменные ошибки

```
ErrInvalidChannelID
ErrInvalidRoomID
ErrInvalidChannelName
ErrInvalidChannelKind
ErrInvalidCreatedAt

ErrChannelNotFound
ErrChannelNameAlreadyTaken      // unique violation в (room_id, name)
ErrChannelAccessDenied          // от MembershipQuery: не member
ErrChannelInsufficientRole      // от MembershipQuery: member на admin-операцию
```

`ErrChannelAccessDenied` и `ErrChannelInsufficientRole` объявлены **внутри `channel/domain/`** — это нейтральные доменные понятия, чтобы channel/usecase не зависел от room/domain. Адаптер `MembershipQueryAdapter` (живёт в `internal/room/repository/postgres/` или в отдельном `internal/room/adapter/membershipquery/`) на чтении из `room_members` маппит «нет строки» в `channel.domain.ErrChannelAccessDenied`, а «есть, но роль не подходит» — в `ErrChannelInsufficientRole`.

#### 3.2.4 Порт `MembershipQuery`

Объявлен в `internal/channel/usecase/ports.go`. Внутри только нейтральные понятия `RoleAtLeast`, без знания о room.

```go
type RoleRequirement int

const (
    RoleAnyMember RoleRequirement = iota // owner | admin | member
    RoleAdminOrOwner                     // admin | owner
    RoleOwnerOnly                        // owner only
)

type MembershipQuery interface {
    // Require возвращает nil, если у пользователя в комнате есть требуемая роль;
    // ErrChannelAccessDenied — если он не member;
    // ErrChannelInsufficientRole — если member, но недостаточно прав.
    Require(ctx context.Context, roomID RoomID, userID UserID, req RoleRequirement) error
}
```

`RoomID`, `UserID`, `RoleRequirement` — все из `channel/domain/` или `channel/usecase/`. Реализация (`MembershipQueryAdapter` в `internal/room/...`) принимает на входе UUID-обёртки channel-домена и внутри переводит в room-домен (через `uuid.UUID`-bridge).

### 3.3 Композиция (cmd/server)

```mermaid
flowchart TB
    main["cmd/server/main.go"]
    cfg["config.Config"]
    pool["pgxpool.Pool"]
    qauth["auth db.Queries"]
    qroom["room db.Queries"]
    qch["channel db.Queries"]

    main --> cfg
    main --> pool
    pool --> qauth
    pool --> qroom
    pool --> qch

    qauth --> auth_repos["auth repositories"]
    qroom --> room_repos["room repositories +<br/>MembershipQueryAdapter"]
    qch --> ch_repos["channel repository"]

    auth_repos --> auth_uc["auth usecases"]
    room_repos --> room_uc["room usecases"]
    ch_repos --> ch_uc["channel usecases"]
    room_repos --> ch_uc

    auth_uc --> mount_auth["httpauth.RegisterRoutes(r, ...)"]
    room_uc --> mount_room["httproom.RegisterRoutes(r, ...)"]
    ch_uc --> mount_ch["httpchannel.RegisterRoutes(r, ...)"]

    mount_auth --> chi["chi.Router /api/v1"]
    mount_room --> chi
    mount_ch --> chi
```

Подробное описание шага DI и точек подключения — в `plan/README.md` после утверждения дизайна.

## Граф зависимостей модулей

```mermaid
flowchart BT
    httpx["pkg/httpx, pkg/httpx/middleware"]
    auth_dom["internal/auth/domain"]
    auth_uc["internal/auth/usecase"]
    auth_mw["internal/auth/transport/http/middleware"]
    auth_http["internal/auth/transport/http (httpauth)"]
    auth_repo["internal/auth/repository/postgres"]

    room_dom["internal/room/domain"]
    room_uc["internal/room/usecase"]
    room_http["internal/room/transport/http (httproom)"]
    room_repo["internal/room/repository/postgres<br/>(+ MembershipQueryAdapter)"]

    ch_dom["internal/channel/domain"]
    ch_uc["internal/channel/usecase"]
    ch_http["internal/channel/transport/http (httpchannel)"]
    ch_repo["internal/channel/repository/postgres"]

    main["cmd/server/main.go"]

    auth_uc --> auth_dom
    auth_repo --> auth_uc
    auth_http --> auth_uc
    auth_http --> auth_mw
    auth_mw --> auth_uc
    auth_mw --> httpx

    room_uc --> room_dom
    room_repo --> room_uc
    room_http --> room_uc
    room_http --> auth_mw
    room_http --> httpx

    ch_uc --> ch_dom
    ch_repo --> ch_uc
    ch_http --> ch_uc
    ch_http --> auth_mw
    ch_http --> httpx

    %% Cross-domain bridge: реализация порта MembershipQuery живёт в room
    room_repo --> ch_uc

    main --> auth_repo
    main --> auth_http
    main --> room_repo
    main --> room_http
    main --> ch_repo
    main --> ch_http
```

### Правила, которые поддерживает граф (по `prompts/Architecture Layers.txt:67-73`)

1. `*/domain` — только stdlib + `github.com/google/uuid`.
2. `*/usecase` — stdlib + `uuid` + свой `*/domain` (никакого чужого `*/domain`).
3. `*/transport/http`, `*/repository/postgres` — могут импортировать свой `usecase` и `domain`, плюс `pkg/httpx`/`pkg/httpx/middleware` (для транспорта) и `pgx/v5`, `sqlc` (для репозитория).
4. `pkg/httpx*` — никаких импортов из `internal/...`.
5. **Кросс-доменное исключение №1 (санкционировано фазой 1.4)**: транспорт любого домена может импортировать `internal/auth/transport/http/middleware` ради `RequireAuth` и `UserIDFromContext`.
6. **Кросс-доменное исключение №2 (вводится PR-2)**: `internal/room/repository/...` может импортировать `internal/channel/usecase` (только интерфейс `MembershipQuery` и его доменные типы) — единственная допустимая «обратная» ссылка между доменами, потому что реализация порта живёт у источника данных. Никаких других пересечений room↔channel нет.

Все эти правила enforced в `arch_test.go` (см. список новых тестов в `04-testing.md`).
