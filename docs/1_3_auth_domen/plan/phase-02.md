---
phase: 2
name: internal/auth/domain — Entities, VO, Errors
layer: domain
depends_on: phase-01
plan: ./README.md
---

# Phase 2: `internal/auth/domain/`

## Цель

Реализовать доменный слой `auth`: две сущности (`User`, `RefreshToken`), 7 VO, 19 sentinel-ошибок и unit-тесты к ним. После фазы — пакет `domain` собирается, импортирует только stdlib + `github.com/google/uuid`, тесты `go test ./internal/auth/domain/...` зелёные (`../04-testing.md:65-115`).

## Контекст

- Раскладка пакета и сигнатуры — `../01-architecture.md:170-228`.
- Все правила доменного слоя: `prompts/Domain Model.txt:24-50`. Ключевые: приватные поля, конструкторы `New*` / `Reconstruct*`, инварианты в конструкторах, нет I/O, нет логирования.
- Тесты — чёрный ящик `package domain_test` (`prompts/Domain model test.txt:67`).
- В этой фазе **запрещены** импорты вне stdlib и `uuid`. Это проверяется фазой 8 (smoke), но проще не нарушать сразу.

## Файлы для создания

Все — в `internal/auth/domain/`.

### `errors.go`

19 sentinel-ошибок через `errors.New` (`prompts/Domain Model.txt:60-65`). Точный список и имена — `../01-architecture.md:227-228`. Группировать по теме (валидация / repo / token); внутри группы — по алфавиту, так удобнее искать.

```go
package domain

import "errors"

// Валидация VO/Entity
var (
    ErrInvalidEmail                  = errors.New("auth: invalid email")
    ErrInvalidUsername               = errors.New("auth: invalid username")
    ErrInvalidPassword               = errors.New("auth: invalid password")
    ErrInvalidPasswordHash           = errors.New("auth: invalid password hash")
    ErrInvalidUserID                 = errors.New("auth: invalid user id")
    ErrInvalidRefreshTokenID         = errors.New("auth: invalid refresh token id")
    ErrInvalidTokenHash              = errors.New("auth: invalid token hash")
    ErrInvalidRefreshTokenExpiration = errors.New("auth: refresh token expiration must be after creation")
    ErrInvalidCreatedAt              = errors.New("auth: created_at must not be zero")
)

// Бизнес-правила User/RefreshToken
var (
    ErrEmailAlreadyTaken          = errors.New("auth: email already taken")
    ErrUsernameAlreadyTaken       = errors.New("auth: username already taken")
    ErrUserNotFound               = errors.New("auth: user not found")
    ErrInvalidCredentials         = errors.New("auth: invalid credentials")
    ErrRefreshTokenNotFound       = errors.New("auth: refresh token not found")
    ErrRefreshTokenRevoked        = errors.New("auth: refresh token revoked")
    ErrRefreshTokenExpired        = errors.New("auth: refresh token expired")
    ErrRefreshTokenAlreadyRevoked = errors.New("auth: refresh token already revoked")
    ErrAccessTokenInvalid         = errors.New("auth: access token invalid")
    ErrAccessTokenExpired         = errors.New("auth: access token expired")
)
```

### Value objects

#### `user_id.go`, `refresh_token_id.go`

Шаблон (для `UserID`):

```go
package domain

import "github.com/google/uuid"

type UserID struct{ value uuid.UUID }

func NewUserID(raw uuid.UUID) (UserID, error) {
    if raw == uuid.Nil {
        return UserID{}, ErrInvalidUserID
    }
    return UserID{value: raw}, nil
}

func (id UserID) UUID() uuid.UUID { return id.value }
func (id UserID) String() string  { return id.value.String() }
func (id UserID) IsZero() bool    { return id.value == uuid.Nil }
```

`RefreshTokenID` — копия с заменой имени и ошибки.

`NewUserID` возвращает `(UserID, error)` (а не просто `UserID`), потому что zero-uuid — невалидное состояние и его нужно отлавливать в маппере и use case'е (`prompts/Domain Model.txt:31-35`).

#### `email.go`

```go
type Email struct{ value string }

func NewEmail(raw string) (Email, error) {
    s := strings.ToLower(strings.TrimSpace(raw))
    if s == "" || len(s) > 254 {
        return Email{}, ErrInvalidEmail
    }
    if !emailRe.MatchString(s) {
        return Email{}, ErrInvalidEmail
    }
    return Email{value: s}, nil
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func (e Email) String() string { return e.value }
```

Regex простой, по `../03-decisions.md` OQ-8. Lowercase обязателен — для consistency с БД-уровнем `citext` (OQ-7).

#### `username.go`

```go
type Username struct{ value string }

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func NewUsername(raw string) (Username, error) {
    s := strings.TrimSpace(raw)
    if l := len(s); l < 3 || l > 32 {
        return Username{}, ErrInvalidUsername
    }
    if !usernameRe.MatchString(s) {
        return Username{}, ErrInvalidUsername
    }
    return Username{value: s}, nil
}

func (u Username) String() string { return u.value }
```

Длина 3..32 совпадает с `users_username_length_check` (`migrations/0002_users.up.sql:12`). Regex и обоснование — `../03-decisions.md` ADR-007.

#### `password.go`

```go
type Password struct{ value string }

func NewPassword(raw string) (Password, error) {
    if l := len(raw); l < 8 || l > 72 {
        return Password{}, ErrInvalidPassword
    }
    return Password{value: raw}, nil
}

// String возвращает raw-пароль для передачи в bcrypt-адаптер.
// Не использовать для логирования и сериализации.
func (p Password) String() string { return p.value }
```

Trim **не делаем** — пробелы могут быть значимыми (`../01-architecture.md:217`). Лимит 72 — bcrypt input limit (`prompts` ADR-012).

`String()` — единственный геттер, подразумевается, что вызывается только адаптером bcrypt. JSON-сериализация Password не предусмотрена; полей в DTO `password` мапится через прямой raw-`string` из request body, после чего создаётся `domain.Password`.

#### `password_hash.go`

```go
type PasswordHash struct{ value string }

func NewPasswordHash(raw string) (PasswordHash, error) {
    if raw == "" {
        return PasswordHash{}, ErrInvalidPasswordHash
    }
    return PasswordHash{value: raw}, nil
}

func (h PasswordHash) String() string { return h.value }
```

Формат bcrypt-хеша (`$2a$NN$...`) валидируется самим `bcrypt.CompareHashAndPassword` в адаптере (`../03-decisions.md` OQ-5). Здесь только non-empty.

#### `token_hash.go`

```go
type TokenHash struct{ value [32]byte }

func NewTokenHash(raw []byte) (TokenHash, error) {
    if len(raw) != 32 {
        return TokenHash{}, ErrInvalidTokenHash
    }
    var h TokenHash
    copy(h.value[:], raw)
    return h, nil
}

func (h TokenHash) Bytes() [32]byte    { return h.value }
func (h TokenHash) Equal(o TokenHash) bool { return h.value == o.value }
```

`Bytes()` возвращает массив (не slice) — это не позволяет вызывающему мутировать внутреннее состояние. Маппер sql берёт `b := h.Bytes(); b[:]` — копия → slice (`../06-repo-model.md:92-100`).

### Entity `user.go`

```go
type User struct {
    id           UserID
    email        Email
    username     Username
    passwordHash PasswordHash
    createdAt    time.Time
}

func NewUser(id UserID, email Email, username Username, hash PasswordHash, createdAt time.Time) (*User, error) {
    if id.IsZero() {
        return nil, ErrInvalidUserID
    }
    if createdAt.IsZero() {
        return nil, ErrInvalidCreatedAt
    }
    return &User{id: id, email: email, username: username, passwordHash: hash, createdAt: createdAt}, nil
}

func ReconstructUser(id UserID, email Email, username Username, hash PasswordHash, createdAt time.Time) (*User, error) {
    return NewUser(id, email, username, hash, createdAt)
}

func (u *User) ID() UserID                 { return u.id }
func (u *User) Email() Email               { return u.email }
func (u *User) Username() Username         { return u.username }
func (u *User) PasswordHash() PasswordHash { return u.passwordHash }
func (u *User) CreatedAt() time.Time       { return u.createdAt }
```

`ReconstructUser` сейчас совпадает с `NewUser`, но вынесен **отдельно** — для двух целей:
1. Будущая дивергенция (например, `ReconstructUser` пропустит инвариант, если в схему добавится поле, валидное в БД, но невалидное на фронте).
2. Удовлетворить требование `prompts/Domain Model.txt:43-50` — два явных конструктора.

Важно: VO передаются в конструктор уже валидными — за их валидацию отвечают конструкторы VO. Конструктор Entity проверяет только инварианты, специфичные для Entity (zero-id, zero-createdAt).

### Entity `refresh_token.go`

```go
type RefreshToken struct {
    id        RefreshTokenID
    userID    UserID
    tokenHash TokenHash
    expiresAt time.Time
    createdAt time.Time
    revokedAt time.Time // zero value = активный
}

func NewRefreshToken(id RefreshTokenID, userID UserID, hash TokenHash, expiresAt, createdAt time.Time) (*RefreshToken, error) {
    if userID.IsZero() {
        return nil, ErrInvalidUserID
    }
    if createdAt.IsZero() {
        return nil, ErrInvalidCreatedAt
    }
    if !expiresAt.After(createdAt) {
        return nil, ErrInvalidRefreshTokenExpiration
    }
    return &RefreshToken{
        id: id, userID: userID, tokenHash: hash,
        expiresAt: expiresAt, createdAt: createdAt,
    }, nil
}

func ReconstructRefreshToken(id RefreshTokenID, userID UserID, hash TokenHash, expiresAt, createdAt, revokedAt time.Time) (*RefreshToken, error) {
    t, err := NewRefreshToken(id, userID, hash, expiresAt, createdAt)
    if err != nil {
        return nil, err
    }
    t.revokedAt = revokedAt
    return t, nil
}

func (t *RefreshToken) Revoke(now time.Time) error {
    if !t.revokedAt.IsZero() {
        return ErrRefreshTokenAlreadyRevoked
    }
    t.revokedAt = now
    return nil
}

func (t *RefreshToken) IsActive(now time.Time) bool  { return t.revokedAt.IsZero() && now.Before(t.expiresAt) }
func (t *RefreshToken) IsRevoked() bool              { return !t.revokedAt.IsZero() }
func (t *RefreshToken) IsExpired(now time.Time) bool { return !now.Before(t.expiresAt) }
// геттеры ID/UserID/TokenHash/ExpiresAt/CreatedAt/RevokedAt — банальные
```

State-machine — `../01-architecture.md:299-314`.

## Файлы тестов

Структура — по `../04-testing.md:65-115`. По одному test-файлу на конструктивную единицу:
- `email_test.go`, `username_test.go`, `password_test.go`, `password_hash_test.go`, `user_id_test.go`, `refresh_token_id_test.go`, `token_hash_test.go`
- `user_test.go`, `refresh_token_test.go`

Все — `package domain_test`, `t.Parallel()` в каждом тесте, table-driven где удобно (Email, Username, Password, TokenHash, RefreshToken.IsActive).

Test helpers (`mustEmail`, `mustUsername`, …) — в `testing_fixtures_test.go`. Каждый — с `t.Helper()` (`prompts/Domain model test.txt:71-86`).

## Ключевые решения

- Все VO — value-типы (`struct`-по-значению), кроме `Password` и `PasswordHash` — тоже value (но не сериализуются).
- Сравнение VO `==` — работает out-of-the-box для `string`/`uuid.UUID`. Для `TokenHash` явный `Equal` — массивы `[32]byte` сравнимы оператором `==`, но метод даёт читаемость.
- `revokedAt time.Time` (а не `*time.Time` или `sql.NullTime`) — zero-value семантика. На границе с БД маппер раскладывает в `pgtype.Timestamptz` (`../06-repo-model.md:67`).
- Никаких бизнес-методов у `User` (`../01-architecture.md:185`).

## Verification

- [ ] `go build ./internal/auth/domain/...` — без ошибок.
- [ ] `go test ./internal/auth/domain/... -race -count=1` — все 16 тестов зелёные.
- [ ] `make lint` зелёный (особое внимание: нет TODO, нет неиспользованных импортов, нет публичных полей у entity/VO).
- [ ] `grep -rE "import" internal/auth/domain/*.go | grep -vE "(testing|domain_test|stdlib_packages|google/uuid)"` — пусто (никаких лишних импортов, кроме stdlib и uuid). Точная проверка — фаза 8.
- [ ] Размер каждого `.go` файла — в пределах 100 строк (мера читаемости; не жёсткая, но если файл больше — вероятно, можно разбить).
