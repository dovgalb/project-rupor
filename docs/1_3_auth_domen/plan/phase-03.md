---
phase: 3
name: internal/auth/usecase — ports и скелеты use case'ов
layer: usecase
depends_on: phase-02
plan: ./README.md
---

# Phase 3: `internal/auth/usecase/` — ports + конструкторы

## Цель

Зафиксировать контракт слоя use case: интерфейсы зависимостей (порты) и пустые скелеты четырёх use case'ов с конструкторами. Бизнес-логика `Execute()` — фаза 6. Цель этой фазы — стабильные сигнатуры, на которые опрутся фазы 4 (postgres-адаптер реализует `UserRepository` / `RefreshTokenRepository`), 5 (jwt/bcrypt-адаптеры реализуют `TokenIssuer` / `PasswordHasher`) и 7 (HTTP-хендлеры зависят от `*RegisterUser` и т.д.).

## Контекст

- Точные сигнатуры портов — `../01-architecture.md:243-271`.
- Use case'ы — `../01-architecture.md:230-241`. Каждая структура имеет один публичный метод `Execute(ctx, input) (output, error)`.
- В этой фазе use case-методы возвращают `errors.New("usecase: not implemented yet")` или `panic("not implemented")`. Тестов на use case не пишем — они появятся в фазе 6 вместе с реализацией. Сейчас сборка должна быть зелёной.
- Импорты пакета `usecase` ограничены: stdlib + `internal/auth/domain` + `github.com/google/uuid`.

## Файлы для создания

Все — в `internal/auth/usecase/`.

### `ports.go`

Точное содержимое — `../01-architecture.md:243-271`:

```go
package usecase

import (
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/auth/domain"
)

type UserRepository interface {
    Save(ctx context.Context, u *domain.User) error
    FindByID(ctx context.Context, id domain.UserID) (*domain.User, error)
    FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error)
}

type RefreshTokenRepository interface {
    Save(ctx context.Context, t *domain.RefreshToken) error
    FindByHash(ctx context.Context, h domain.TokenHash) (*domain.RefreshToken, error)
    Rotate(ctx context.Context, oldHash domain.TokenHash, newToken *domain.RefreshToken, revokedAt time.Time) error
}

type PasswordHasher interface {
    Hash(p domain.Password) (domain.PasswordHash, error)
    Verify(h domain.PasswordHash, p domain.Password) error
}

type TokenIssuer interface {
    IssueAccess(userID domain.UserID, now time.Time) (token string, expiresAt time.Time, err error)
    VerifyAccess(token string, now time.Time) (domain.UserID, error)
}

type Clock interface {
    Now() time.Time
}

type UUIDGenerator interface {
    New() uuid.UUID
}

type RandomBytes interface {
    Read(n int) ([]byte, error)
}
```

Всё — в одном файле, по правилу «один файл на тематический набор интерфейсов» (`prompts/Architecture Layers.txt:39-43`).

### Skeletons use case'ов

#### `register_user.go`

```go
package usecase

import (
    "context"
    "errors"
    "time"

    "github.com/dovgalb/project-rupor/internal/auth/domain"
)

type RegisterUserInput struct {
    Email    string
    Username string
    Password string
}

type RegisterUserOutput struct {
    UserID    string
    Email     string
    Username  string
    CreatedAt time.Time
}

type RegisterUser struct {
    users  UserRepository
    hasher PasswordHasher
    clock  Clock
    uuids  UUIDGenerator
}

func NewRegisterUser(users UserRepository, hasher PasswordHasher, clock Clock, uuids UUIDGenerator) *RegisterUser {
    return &RegisterUser{users: users, hasher: hasher, clock: clock, uuids: uuids}
}

func (u *RegisterUser) Execute(ctx context.Context, in RegisterUserInput) (RegisterUserOutput, error) {
    _ = ctx
    _ = in
    _ = domain.ErrInvalidEmail // ссылка на пакет, иначе go vet warning об неиспользованном импорте
    return RegisterUserOutput{}, errors.New("usecase: register_user: not implemented")
}
```

Заглушка `Execute` помечена очевидной ошибкой; в фазе 6 будет полностью переписана. Импорт `domain` пока «висит» — оставить как есть, чтобы `go build` не ругался на unused import (или сразу убрать, и добавить вместе с реализацией в фазе 6 — что чище).

**Чистый вариант (предпочитаемый):** заглушка без импорта `domain`. Импорт добавится в фазе 6.

```go
func (u *RegisterUser) Execute(ctx context.Context, in RegisterUserInput) (RegisterUserOutput, error) {
    return RegisterUserOutput{}, errors.New("usecase: register_user: not implemented")
}
```

#### `login_user.go`

```go
type LoginUserInput struct {
    Email    string
    Password string
}

type LoginUserOutput struct {
    AccessToken      string
    RefreshToken     string
    AccessExpiresAt  time.Time
    RefreshExpiresAt time.Time
}

type LoginUser struct {
    users    UserRepository
    refresh  RefreshTokenRepository
    hasher   PasswordHasher
    issuer   TokenIssuer
    clock    Clock
    uuids    UUIDGenerator
    rand     RandomBytes
    refreshTTL time.Duration
}

func NewLoginUser(users UserRepository, refresh RefreshTokenRepository, hasher PasswordHasher, issuer TokenIssuer, clock Clock, uuids UUIDGenerator, rand RandomBytes, refreshTTL time.Duration) *LoginUser {
    return &LoginUser{users: users, refresh: refresh, hasher: hasher, issuer: issuer, clock: clock, uuids: uuids, rand: rand, refreshTTL: refreshTTL}
}

func (u *LoginUser) Execute(ctx context.Context, in LoginUserInput) (LoginUserOutput, error) {
    return LoginUserOutput{}, errors.New("usecase: login_user: not implemented")
}
```

Конструктор принимает `refreshTTL time.Duration` — это политика приложения, инжектится из конфига (`cfg.JWTRefreshTTL()`) на уровне `cmd/server/main.go`. Аналогично у `RefreshAccess`.

`accessTTL` инкапсулирован внутри `TokenIssuer.IssueAccess` (передаётся в его конструктор) — потому что подписать JWT с `exp = now + accessTTL` лучше там, где формируются claims.

#### `refresh_access.go`

```go
type RefreshAccessInput struct {
    RefreshToken string
}

type RefreshAccessOutput = LoginUserOutput // та же структура (см. ../08-api-contract.md:196-204)

type RefreshAccess struct {
    refresh    RefreshTokenRepository
    issuer     TokenIssuer
    clock      Clock
    uuids      UUIDGenerator
    rand       RandomBytes
    refreshTTL time.Duration
}

func NewRefreshAccess(refresh RefreshTokenRepository, issuer TokenIssuer, clock Clock, uuids UUIDGenerator, rand RandomBytes, refreshTTL time.Duration) *RefreshAccess {
    return &RefreshAccess{refresh: refresh, issuer: issuer, clock: clock, uuids: uuids, rand: rand, refreshTTL: refreshTTL}
}

func (u *RefreshAccess) Execute(ctx context.Context, in RefreshAccessInput) (RefreshAccessOutput, error) {
    return RefreshAccessOutput{}, errors.New("usecase: refresh_access: not implemented")
}
```

Type alias `RefreshAccessOutput = LoginUserOutput` явно фиксирует семантику «refresh возвращает то же, что login» (`../08-api-contract.md:196`).

#### `get_current_user.go`

```go
type GetCurrentUserInput struct {
    UserID string // UUID-строка из claim sub
}

type GetCurrentUserOutput struct {
    UserID    string
    Email     string
    Username  string
    CreatedAt time.Time
}

type GetCurrentUser struct {
    users UserRepository
}

func NewGetCurrentUser(users UserRepository) *GetCurrentUser {
    return &GetCurrentUser{users: users}
}

func (u *GetCurrentUser) Execute(ctx context.Context, in GetCurrentUserInput) (GetCurrentUserOutput, error) {
    return GetCurrentUserOutput{}, errors.New("usecase: get_current_user: not implemented")
}
```

## Ключевые решения

- Sigatures в `ports.go` — финальные. Их меняет только редизайн дизайн-документов, не реализация в фазе 6.
- Use case-структуры с публичным методом `Execute` — единый паттерн (`prompts/Architecture Layers.txt:39-43`).
- TTL — параметр конструктора use case, не глобальное состояние пакета.
- Output-типы зеркалят response DTO в `../08-api-contract.md`, но **не разделяют** их с DTO напрямую — DTO живут в `transport/http/` и могут вводить json-теги. Use case возвращает чистые типы без внешних зависимостей.
- В `RegisterUserOutput` поле `UserID` — `string`, а не `domain.UserID`: на границе use case ↔ transport мы возвращаем уже сериализуемые значения; transport не разворачивает VO.

## Verification

- [ ] `go build ./internal/auth/usecase/...` — без ошибок.
- [ ] `go vet ./internal/auth/usecase/...` — чисто.
- [ ] Импорты в каждом файле: только `context`, `errors`, `time` (stdlib) + `github.com/google/uuid` (только в `ports.go`) + `internal/auth/domain` (нигде в скелетах фазы 3, появится в фазе 6). Файлы skeleton'ов в этой фазе **не импортируют domain** — это намеренно, чтобы импорты появлялись вместе с тем, что их использует.
- [ ] Тестов в этой фазе нет — `go test ./internal/auth/usecase/...` сообщает «no test files» и это OK.
- [ ] `make lint` зелёный.
