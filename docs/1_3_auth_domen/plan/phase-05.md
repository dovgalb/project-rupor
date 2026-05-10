---
phase: 5
name: internal/auth/repository/{jwt,bcrypt}
layer: adapter
depends_on: phase-01, phase-03
plan: ./README.md
---

# Phase 5: `internal/auth/repository/jwt/` + `internal/auth/repository/bcrypt/`

## Цель

Реализовать два мелких адаптера:
- `repository/jwt/` — `TokenIssuer` через `github.com/golang-jwt/jwt/v5` (HS256).
- `repository/bcrypt/` — `PasswordHasher` через `golang.org/x/crypto/bcrypt`.

После фазы — оба удовлетворяют портам из `usecase`, `make test` зелёный (6 + 3 = 9 unit-тестов).

Фазы 4 и 5 независимы — могут идти параллельно. Объединены в один план через зависимость от фазы 1 (deps) и фазы 3 (порты).

## Контекст

- Сигнатуры портов — `../01-architecture.md:262-267` (`TokenIssuer`, `PasswordHasher`).
- Уязвимости и их митигация — `../03-decisions.md`, риски «alg: none» и «токен подписан другим секретом». Тесты на это обязательны.
- Bcrypt cost = 10 в проде, 4 в тестах (`../03-decisions.md` ADR-012). Cost — параметр конструктора.

## `internal/auth/repository/jwt/`

### `token_issuer.go`

```go
package jwt

import (
    "errors"
    "fmt"
    "time"

    jwtv5 "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"

    "github.com/dovgalb/project-rupor/internal/auth/domain"
)

type TokenIssuer struct {
    secret    []byte
    accessTTL time.Duration
}

func NewTokenIssuer(secret []byte, accessTTL time.Duration) *TokenIssuer {
    return &TokenIssuer{secret: secret, accessTTL: accessTTL}
}

func (i *TokenIssuer) IssueAccess(userID domain.UserID, now time.Time) (string, time.Time, error) {
    expiresAt := now.Add(i.accessTTL)
    claims := jwtv5.RegisteredClaims{
        Subject:   userID.String(),
        IssuedAt:  jwtv5.NewNumericDate(now),
        ExpiresAt: jwtv5.NewNumericDate(expiresAt),
    }
    token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
    signed, err := token.SignedString(i.secret)
    if err != nil {
        return "", time.Time{}, fmt.Errorf("jwt: sign: %w", err)
    }
    return signed, expiresAt, nil
}

func (i *TokenIssuer) VerifyAccess(token string, now time.Time) (domain.UserID, error) {
    parser := jwtv5.NewParser(
        jwtv5.WithValidMethods([]string{"HS256"}),
        jwtv5.WithTimeFunc(func() time.Time { return now }),
    )
    var claims jwtv5.RegisteredClaims
    _, err := parser.ParseWithClaims(token, &claims, func(t *jwtv5.Token) (any, error) {
        return i.secret, nil
    })
    if err != nil {
        if errors.Is(err, jwtv5.ErrTokenExpired) {
            return domain.UserID{}, domain.ErrAccessTokenExpired
        }
        return domain.UserID{}, fmt.Errorf("%w: %v", domain.ErrAccessTokenInvalid, err)
    }
    raw, err := uuid.Parse(claims.Subject)
    if err != nil {
        return domain.UserID{}, domain.ErrAccessTokenInvalid
    }
    id, err := domain.NewUserID(raw)
    if err != nil {
        return domain.UserID{}, domain.ErrAccessTokenInvalid
    }
    return id, nil
}
```

Ключевые точки:
- `WithValidMethods([]string{"HS256"})` — закрывает атаку `alg: none` и `alg: RS256` (атакующий использует public key как HMAC secret). Без этой строки — критическая уязвимость.
- `WithTimeFunc(func() time.Time { return now })` — детерминированная проверка `exp` относительно переданного `now`, без `t.Sleep` в тестах (`prompts/Tests Style.txt:55-62`).
- Маппинг `jwtv5.ErrTokenExpired` → `domain.ErrAccessTokenExpired` отдельной веткой; всё остальное — обобщённо в `ErrAccessTokenInvalid` (включая bad signature, malformed, wrong alg).

### `token_issuer_test.go`

6 тестов по `../04-testing.md:235-244`. Все — `package jwt_test`, `t.Parallel()`. Использовать одну общую `secret = []byte("test-secret-1234567890")` и `wrongSecret = []byte("other-secret-0987654321")`.

Ключевой тест `TestTokenIssuer_VerifyAccess_RejectsAlgNone`: вручную сформировать токен с заголовком `{"alg":"none"}` и пустой подписью; убедиться, что `VerifyAccess` возвращает `ErrAccessTokenInvalid`.

## `internal/auth/repository/bcrypt/`

### `password_hasher.go`

```go
package bcrypt

import (
    "errors"
    "fmt"

    "golang.org/x/crypto/bcrypt"

    "github.com/dovgalb/project-rupor/internal/auth/domain"
)

type PasswordHasher struct {
    cost int
}

func NewPasswordHasher(cost int) *PasswordHasher {
    return &PasswordHasher{cost: cost}
}

func (h *PasswordHasher) Hash(p domain.Password) (domain.PasswordHash, error) {
    raw, err := bcrypt.GenerateFromPassword([]byte(p.String()), h.cost)
    if err != nil {
        return domain.PasswordHash{}, fmt.Errorf("bcrypt: hash: %w", err)
    }
    return domain.NewPasswordHash(string(raw))
}

func (h *PasswordHasher) Verify(stored domain.PasswordHash, p domain.Password) error {
    err := bcrypt.CompareHashAndPassword([]byte(stored.String()), []byte(p.String()))
    if err == nil {
        return nil
    }
    if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
        return domain.ErrInvalidCredentials
    }
    return fmt.Errorf("bcrypt: verify: %w", err)
}
```

Заметим: при невалидном hash-формате (не bcrypt-строка) bcrypt вернёт `bcrypt.ErrHashTooShort` или подобную ошибку — мы оборачиваем как «не invalid-credentials, а bug». Use case (фаза 6) расценит это как 500.

### `password_hasher_test.go`

3 теста по `../04-testing.md:248-254`. Cost = `bcrypt.MinCost` (4) для скорости. Структура:
- `TestPasswordHasher_RoundTrip` — `Hash(p) → h; Verify(h, p) → nil`.
- `TestPasswordHasher_VerifyMismatch` — `Verify(h, "wrongpass") → ErrInvalidCredentials`.
- `TestPasswordHasher_VerifyInvalidHash` — `Verify(NewPasswordHash("not-a-bcrypt-hash"), p)` → wrapped error, **не** `ErrInvalidCredentials`. Проверка через `!errors.Is(err, domain.ErrInvalidCredentials) && err != nil`.

## Compile-check

Каждый адаптер — со своим `compile_check_test.go`:

```go
// internal/auth/repository/jwt/compile_check_test.go
package jwt_test

import (
    "github.com/dovgalb/project-rupor/internal/auth/repository/jwt"
    "github.com/dovgalb/project-rupor/internal/auth/usecase"
)

var _ usecase.TokenIssuer = (*jwt.TokenIssuer)(nil)
```

```go
// internal/auth/repository/bcrypt/compile_check_test.go
package bcrypt_test

import (
    "github.com/dovgalb/project-rupor/internal/auth/repository/bcrypt"
    "github.com/dovgalb/project-rupor/internal/auth/usecase"
)

var _ usecase.PasswordHasher = (*bcrypt.PasswordHasher)(nil)
```

## Ключевые решения

- Имя пакета `jwt` совпадает с `github.com/golang-jwt/jwt/v5` — поэтому в коде используем алиас `jwtv5` для импорта library, а наш собственный пакет имеет короткое имя `jwt`. Это ОК для Go-стиля (`prompts/Go style.txt`).
- Аналогично для `bcrypt` — имя пакета совпадает с `golang.org/x/crypto/bcrypt`. Чтобы избежать коллизии, импорт библиотеки использует имя по-умолчанию (`bcrypt`), а наша структура называется `PasswordHasher` — конфликта нет.
- `domain.UserID{}` (zero) не возвращаем в success — у `NewUserID` zero-uuid невалиден, поэтому при ошибке возвращаем zero-VO + ошибку; вызывающий обязан проверить ошибку, не VO.

## Verification

- [ ] `go build ./internal/auth/repository/jwt/... ./internal/auth/repository/bcrypt/...` — зелёный.
- [ ] `go test ./internal/auth/repository/jwt/... ./internal/auth/repository/bcrypt/... -race -count=1` — все 9 тестов зелёные.
- [ ] `make lint` зелёный.
- [ ] `git diff --name-only` ограничен `internal/auth/repository/jwt/**`, `internal/auth/repository/bcrypt/**`.
- [ ] Алгоритм атаки `alg: none` явно покрыт тестом и тест зелёный.
