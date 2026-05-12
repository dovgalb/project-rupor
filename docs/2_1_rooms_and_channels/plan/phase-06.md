---
phase: 6
name: Room repository (postgres)
layer: repository
depends_on: [phase-01, phase-04, phase-05]
plan: ./README.md
---

# Phase 6: Room repository postgres (3 репозитория + MembershipQueryAdapter + queries + integration tests)

## Цель

Реализовать `internal/room/repository/postgres/`: SQL-запросы, адаптеры репозиториев, маппинг row↔domain, базу для генерации инвайт-кодов, и **MembershipQueryAdapter** — реализацию порта `channel/usecase.MembershipQuery` (единственная санкционированная кросс-доменная связь).

## Контекст

После Phase 01 уже есть таблицы и пустой пакет `internal/room/repository/postgres/db/` (sqlc-сгенерированный). После Phase 04 есть порты `RoomRepository`, `MembershipRepository`, `InviteRepository`, `InviteCodeGenerator`. После Phase 05 есть порт `channel/usecase.MembershipQuery` — мы реализуем его здесь.

Маппинг полей и SQL-запросов полностью описан в `../06-repo-model.md`. Эталон стиля — `internal/auth/repository/postgres/` (см. `../research.md` §«Repository postgres + sqlc»).

## Файлы для создания

### SQL queries (для sqlc)

#### `internal/room/repository/postgres/queries/rooms.sql`
Содержимое — точно как в `../06-repo-model.md` §«rooms.sql». 4 запроса: `InsertRoom`, `GetRoomByID`, `ListRoomsByMember`, `DeleteRoomByID`.

#### `internal/room/repository/postgres/queries/room_members.sql`
4 запроса: `InsertRoomMember`, `GetRoomMember`, `ListRoomMembers`, `DeleteRoomMember`. См. `../06-repo-model.md` §«room_members.sql».

#### `internal/room/repository/postgres/queries/invites.sql`
4 запроса: `InsertInvite`, `RevokeActiveInvitesByRoom`, `GetActiveInviteByCode`, `GetActiveInviteByRoom`. См. `../06-repo-model.md` §«invites.sql».

После сохранения этих файлов запустить `make sqlc` — обновится `internal/room/repository/postgres/db/`.

### Утилиты

#### `internal/room/repository/postgres/pgerr.go`
По образцу `internal/auth/repository/postgres/pgerr.go:9-17`:

```go
package postgres  // alias импорта в main: roompg

import (
    "errors"

    "github.com/jackc/pgx/v5/pgconn"
)

func isUniqueViolation(err error, constraint string) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23505" && pgErr.ConstraintName == constraint
    }
    return false
}

func isForeignKeyViolation(err error, constraint string) bool {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        return pgErr.Code == "23503" && pgErr.ConstraintName == constraint
    }
    return false
}
```

### Mapper

#### `internal/room/repository/postgres/mapper.go`
Четыре пары функций (Entity ↔ row). Полные сигнатуры — в `../06-repo-model.md` §«Маппинг сущность ↔ модель БД».

Например для Room:

```go
func roomRowToDomain(row db.Room) (*domain.Room, error) {
    id, err := domain.NewRoomID(row.ID)
    if err != nil {
        return nil, fmt.Errorf("room row: id: %w", err)
    }
    ownerID, err := domain.NewUserID(row.OwnerID)
    if err != nil {
        return nil, fmt.Errorf("room row: owner_id: %w", err)
    }
    name, err := domain.NewRoomName(row.Name)
    if err != nil {
        return nil, fmt.Errorf("room row: name: %w", err)
    }
    return domain.ReconstructRoom(id, ownerID, name, row.CreatedAt)
}

func domainToInsertRoomParams(r *domain.Room) db.InsertRoomParams {
    return db.InsertRoomParams{
        ID:        r.ID().UUID(),
        OwnerID:   r.OwnerID().UUID(),
        Name:      r.Name().String(),
        CreatedAt: r.CreatedAt(),
    }
}
```

Аналогично для Membership, Invite. Для `Invite` — обработка nullable `revoked_at` через `pgtype.Timestamptz`:

```go
func inviteRowToDomain(row db.Invite) (*domain.Invite, error) {
    // ... парсинг id, roomID, code, createdBy
    var revokedAt time.Time
    if row.RevokedAt.Valid {
        revokedAt = row.RevokedAt.Time
    }
    return domain.ReconstructInvite(id, roomID, code, createdBy, row.CreatedAt, revokedAt)
}
```

#### `internal/room/repository/postgres/mapper_test.go`
Round-trip тесты (см. `../04-testing.md` §«Repo Model Round-Trip Tests»):
- `TestRoomMapper_RoundTrip_AllFieldsPreserved`
- `TestMembershipMapper_RoundTrip_AllRoles` (табличный по owner/admin/member)
- `TestInviteMapper_RoundTrip_Active`
- `TestInviteMapper_RoundTrip_Revoked`

Стратегия: построить Entity через builder/конструктор → `domainToXxxRow` → построить fake `db.Xxx` структуру с теми же полями → `xxxRowToDomain` → сравнить getter'ы.

### Repositories

#### `internal/room/repository/postgres/room_repository.go`

```go
type RoomRepository struct {
    pool *pgxpool.Pool  // нужен для транзакции SaveWithOwner
}

func NewRoomRepository(pool *pgxpool.Pool) *RoomRepository
```

Методы:

- **`SaveWithOwner(ctx, room, owner)`** — транзакция, по образцу `internal/auth/repository/postgres/refresh_token_repository.go:50-86` (`Rotate`):
  ```
  tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
  defer tx.Rollback(ctx)  // безопасно после Commit
  q := db.New(tx)
  q.InsertRoom(ctx, domainToInsertRoomParams(room))
  q.InsertRoomMember(ctx, domainToInsertMembershipParams(owner))
    // на unique violation room_members_one_owner_per_room → fmt.Errorf wrap
    //   (это означает баг логики; usecase не должен вызывать SaveWithOwner для существующей room)
  tx.Commit(ctx)
  ```
- **`FindByID(ctx, id)`** — `q.GetRoomByID(ctx, id.UUID())`. На `pgx.ErrNoRows` → `domain.ErrRoomNotFound`.
- **`ListByMember(ctx, userID)`** — `q.ListRoomsByMember(ctx, userID.UUID())`, маппинг каждой row в `usecase.RoomWithRole{Room, Role}` (роль парсится через `domain.ParseRole`).
- **`Delete(ctx, id)`** — `q.DeleteRoomByID(ctx, id.UUID())`. `:execrows`; если 0 → `domain.ErrRoomNotFound`.

#### `internal/room/repository/postgres/membership_repository.go`

```go
type MembershipRepository struct {
    q *db.Queries
}

func NewMembershipRepository(q *db.Queries) *MembershipRepository
```

Методы:
- **`FindByPair(ctx, roomID, userID)`** — `q.GetRoomMember(ctx, ...)`. На `pgx.ErrNoRows` → `domain.ErrNotMember`.
- **`ListByRoom(ctx, roomID)`** — `q.ListRoomMembers(ctx, roomID.UUID())`. Маппинг row→domain через `membershipRowToDomain`.
- **`Add(ctx, m)`** — `q.InsertRoomMember(ctx, ...)`. На `isUniqueViolation(err, "room_members_pkey")` → `domain.ErrAlreadyMember`. На `isUniqueViolation(err, "room_members_one_owner_per_room")` → `fmt.Errorf("membership repo: owner already exists: %w", err)` (баг логики, INTERNAL).

#### `internal/room/repository/postgres/invite_repository.go`

```go
type InviteRepository struct {
    pool *pgxpool.Pool  // транзакция нужна
}

func NewInviteRepository(pool *pgxpool.Pool) *InviteRepository
```

Методы:

- **`RegenerateActive(ctx, invite)`** — транзакция:
  ```
  tx := pool.BeginTx(ctx, ...)
  defer tx.Rollback(ctx)
  q := db.New(tx)
  q.RevokeActiveInvitesByRoom(ctx, RoomID, now)
    // now берётся из invite.CreatedAt() — это и есть момент генерации
  err := q.InsertInvite(ctx, domainToInsertInviteParams(invite))
  if isUniqueViolation(err, "invites_active_code") {
      return usecase.ErrInviteCodeCollision  // для ретрая в usecase
  }
  if err != nil { return wrap }
  tx.Commit(ctx)
  ```
  Импорт `usecase.ErrInviteCodeCollision` — это `internal/room/usecase`, что НЕ нарушает архитектуру (repo может импортировать usecase).
- **`FindActiveByCode(ctx, code)`** — `q.GetActiveInviteByCode(ctx, code.String())`. На `pgx.ErrNoRows` → `domain.ErrInviteNotFound`.

### MembershipQueryAdapter (кросс-доменная связь)

#### `internal/room/repository/postgres/membership_query.go`
Реализация `internal/channel/usecase.MembershipQuery`. Полная заготовка — в `../06-repo-model.md` §«MembershipQueryAdapter». Краткая структура:

```go
package postgres  // тот же пакет roompg, что и room repos

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"

    chdom "github.com/dovgalb/project-rupor/internal/channel/domain"
    chuc "github.com/dovgalb/project-rupor/internal/channel/usecase"
    roomdom "github.com/dovgalb/project-rupor/internal/room/domain"
    "github.com/dovgalb/project-rupor/internal/room/repository/postgres/db"
)

type MembershipQueryAdapter struct {
    q *db.Queries
}

func NewMembershipQueryAdapter(q *db.Queries) *MembershipQueryAdapter {
    return &MembershipQueryAdapter{q: q}
}

func (a *MembershipQueryAdapter) Require(
    ctx context.Context,
    roomID chdom.RoomID,
    userID chdom.UserID,
    req chuc.RoleRequirement,
) error {
    row, err := a.q.GetRoomMember(ctx, db.GetRoomMemberParams{
        RoomID: roomID.UUID(),
        UserID: userID.UUID(),
    })
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return chdom.ErrChannelAccessDenied
        }
        return fmt.Errorf("membership query: %w", err)
    }
    role, perr := roomdom.ParseRole(row.Role)
    if perr != nil {
        return fmt.Errorf("membership query: parse role: %w", perr)
    }
    if !satisfies(role, req) {
        return chdom.ErrChannelInsufficientRole
    }
    return nil
}

func satisfies(role roomdom.Role, req chuc.RoleRequirement) bool {
    switch req {
    case chuc.RoleAnyMember:
        return true
    case chuc.RoleAdminOrOwner:
        return role == roomdom.RoleAdmin || role == roomdom.RoleOwner
    case chuc.RoleOwnerOnly:
        return role == roomdom.RoleOwner
    default:
        return false
    }
}
```

### InviteCodeGenerator

#### `internal/room/repository/postgres/code_gen.go`

```go
package postgres

import (
    "fmt"
    "io"

    "github.com/dovgalb/project-rupor/internal/room/domain"
)

const inviteCodeLength = 8

type Base32CodeGen struct {
    rand io.Reader  // crypto/rand reader; в композиции — cmd/server/runtime.go::cryptoRand
}

func NewBase32CodeGen(rand io.Reader) *Base32CodeGen {
    return &Base32CodeGen{rand: rand}
}

func (g *Base32CodeGen) New() (domain.InviteCode, error) {
    alphabet := domain.InviteCodeAlphabet()
    buf := make([]byte, inviteCodeLength)
    if _, err := io.ReadFull(g.rand, buf); err != nil {
        return domain.InviteCode{}, fmt.Errorf("code gen: read rand: %w", err)
    }
    out := make([]byte, inviteCodeLength)
    for i, b := range buf {
        out[i] = alphabet[int(b)%len(alphabet)]
    }
    return domain.NewInviteCode(string(out))
}
```

**Замечание о `b % len(alphabet)`**: распределение слегка смещено (modulo bias), но для 8 символов из 32-символьного алфавита смещение незначимо для UUID-подобной задачи (см. `../03-decisions.md` §D-14, риск низкий). Альтернатива (rejection sampling) усложнила бы код; принимаем простой вариант.

### Compile-check assertions

#### `internal/room/repository/postgres/compile_check_test.go`

```go
package postgres_test

import (
    chuc "github.com/dovgalb/project-rupor/internal/channel/usecase"
    pg "github.com/dovgalb/project-rupor/internal/room/repository/postgres"
    roomuc "github.com/dovgalb/project-rupor/internal/room/usecase"
)

var (
    _ roomuc.RoomRepository       = (*pg.RoomRepository)(nil)
    _ roomuc.MembershipRepository = (*pg.MembershipRepository)(nil)
    _ roomuc.InviteRepository     = (*pg.InviteRepository)(nil)
    _ roomuc.InviteCodeGenerator  = (*pg.Base32CodeGen)(nil)
    _ chuc.MembershipQuery        = (*pg.MembershipQueryAdapter)(nil)
)
```

Это компилируемый файл, без рантайм-тестов — гарантирует совместимость интерфейсов.

### Integration tests (build tag `integration`)

По образцу `internal/auth/repository/postgres/integration_helpers_test.go:1-35`. Все файлы начинаются с `//go:build integration`.

#### `internal/room/repository/postgres/integration_helpers_test.go`
- `newTestPool(t *testing.T) *pgxpool.Pool` — берёт `TEST_DATABASE_URL`, `t.Skip` если не задан.
- `truncate(t *testing.T, pool *pgxpool.Pool)` — `TRUNCATE rooms, room_members, invites, channels CASCADE`. Регистрируется в `t.Cleanup`.
- На текущем этапе (до закрытия 1.5) каждый тест начинается с `t.Skip("integration: requires Postgres, see issue 1.5")` — по образцу auth-тестов.

#### `internal/room/repository/postgres/room_repository_integration_test.go`
- `TestRoomRepositoryIntegration_SaveWithOwner_Atomic`
- `TestRoomRepositoryIntegration_Delete_Cascades`
- `TestRoomRepositoryIntegration_ListByMember_OrderedDesc`

#### `internal/room/repository/postgres/membership_repository_integration_test.go`
- `TestMembershipRepositoryIntegration_OneOwnerPartialUnique` — попытка вставить второго owner падает.
- `TestMembershipRepositoryIntegration_AddDuplicate_ReturnsErrAlreadyMember`

#### `internal/room/repository/postgres/invite_repository_integration_test.go`
- `TestInviteRepositoryIntegration_RegenerateActive_RevokesAndInserts`
- `TestInviteRepositoryIntegration_OnlyOneActivePerRoom`
- `TestInviteRepositoryIntegration_FindActiveByCode_IgnoresRevoked`

#### `internal/room/repository/postgres/membership_query_integration_test.go`
- `TestMembershipQueryAdapterIntegration_RequireAdminOrOwner` — табличный по парам (role, req).

## Файлы для модификации

- `internal/room/repository/postgres/.gitkeep` — удалён в Phase 01.
- (`sqlc.yaml` уже расширен в Phase 01.)

## Ключевые решения

- **`pool` против `*db.Queries`** в конструкторах — то, что и в auth: `RoomRepository` и `InviteRepository` принимают `*pgxpool.Pool` (для транзакций), `MembershipRepository` — `*db.Queries` (`../03-decisions.md` §D-11/D-12, шаблон `internal/auth/repository/postgres/refresh_token_repository.go:24-26`).
- **`MembershipQueryAdapter` живёт в room/repo**, а не в channel/repo — потому что данные (`room_members`) принадлежат room. Это honest hexagonal: адаптер на стороне источника данных.
- **`ErrInviteCodeCollision` импортируется из `internal/room/usecase`** — это позволено (repo → usecase).
- **`Base32CodeGen` принимает `io.Reader`** — тестируем с `bytes.Reader`, в проде получает `cmd/server/runtime.go::cryptoRand`. Никаких глобальных rand'ов.
- **Modulo bias accepted** — простота кода важнее теоретической равномерности (ADR-impact низкий).
- **Все integration-тесты под `t.Skip`** до закрытия задачи 1.5 — следуем уже установленному паттерну в auth.

## Verification

- [ ] 3 SQL-файла в `queries/` созданы.
- [ ] `make sqlc` отрабатывает без ошибок; в `db/` появляются `rooms.sql.go`, `room_members.sql.go`, `invites.sql.go`.
- [ ] `pgerr.go`, `mapper.go`, `code_gen.go` созданы.
- [ ] 3 репозитория (`room_repository.go`, `membership_repository.go`, `invite_repository.go`) + `membership_query.go` созданы.
- [ ] `compile_check_test.go` компилируется — это гарантирует, что все 5 типов реализуют свои интерфейсы.
- [ ] Mapper round-trip тесты проходят (`go test ./internal/room/repository/postgres/...`).
- [ ] `go build ./internal/room/repository/postgres/...` чистый.
- [ ] `go vet ./internal/room/repository/postgres/...` чистый.
- [ ] `golangci-lint run ./internal/room/repository/postgres/...` чистый.
- [ ] При поднятом Postgres и `TEST_DATABASE_URL` (если 1.5 уже сделан): `go test -tags=integration ./internal/room/repository/postgres/...` зелёный. Иначе — `t.Skip` срабатывает.
- [ ] Ни один файл не импортирует `internal/channel/transport/...` или `internal/auth/...` (можно `grep -rE "internal/(channel/transport|auth)" internal/room/repository/postgres/*.go`).
- [ ] Файл `membership_query.go` импортирует `channel/usecase` и `channel/domain` — это единственная санкционированная кросс-доменная связь.
