---
phase: 6
name: internal/auth/usecase — реализация Execute()
layer: usecase
depends_on: phase-02, phase-03
plan: ./README.md
---

# Phase 6: Реализация use case'ов

## Цель

Заполнить тела `Execute(...)` для всех четырёх use case'ов; написать 21 unit-тест с фейками. После фазы — `make test ./internal/auth/usecase/...` зелёный, race-detector чистый.

Зависит от фазы 2 (домен) и фазы 3 (порты). От фазы 4 (postgres) и 5 (jwt/bcrypt) — НЕ зависит, потому что use case'ы тестируются на фейках. Адаптеры нужны только в фазе 8 (composition root).

## Контекст

- Sequence diagrams — `../02-behavior.md` (DFD-1..DFD-4 + sequence для каждого use case'а в `02-behavior.md:84+`).
- Тестовый план — `../04-testing.md:117-199`.
- Маппинг ошибок use case → код ошибки → HTTP — `../08-api-contract.md:32-46`.

## Файлы для модификации

Все — в `internal/auth/usecase/`. Заменяем skeleton'ы из фазы 3.

### `register_user.go` (полная реализация)

Алгоритм (`../02-behavior.md:85-119`):

1. `email, err := domain.NewEmail(in.Email)` — `ErrInvalidEmail` → проксируем.
2. `username, err := domain.NewUsername(in.Username)` — `ErrInvalidUsername` → проксируем.
3. `password, err := domain.NewPassword(in.Password)` — `ErrInvalidPassword` → проксируем.
4. `hash, err := u.hasher.Hash(password)` — любая ошибка → `fmt.Errorf("usecase: register: hash: %w", err)`.
5. `id, err := domain.NewUserID(u.uuids.New())` — теоретически невозможная ошибка (zero-uuid от UUIDGenerator), но если случится — обернуть.
6. `now := u.clock.Now()`.
7. `user, err := domain.NewUser(id, email, username, hash, now)` — обернуть.
8. `if err := u.users.Save(ctx, user); err != nil { return ..., err }` — `domain.ErrEmailAlreadyTaken` / `ErrUsernameAlreadyTaken` пробрасываются как есть.
9. Output:
   ```go
   return RegisterUserOutput{
       UserID:    user.ID().String(),
       Email:     user.Email().String(),
       Username:  user.Username().String(),
       CreatedAt: user.CreatedAt(),
   }, nil
   ```

### `login_user.go`

Алгоритм (`../02-behavior.md` DFD-2 + ADR-008 для timing-safe):

1. Парсим `email`. Если `domain.NewEmail` → ошибка → **возвращаем `domain.ErrInvalidCredentials`** (не `ErrInvalidEmail`!) — защита от user-enumeration (`../03-decisions.md` ADR-008).
2. Парсим `password` через `domain.NewPassword`. Любая ошибка → `domain.ErrInvalidCredentials`.
3. `user, err := u.users.FindByEmail(ctx, email)`:
   - Если `errors.Is(err, domain.ErrUserNotFound)` — **продолжаем выполнение с фейковым hash'ем**, чтобы выровнять timing. После этого вернём `ErrInvalidCredentials`.
   - Если другая ошибка — обернуть и вернуть.
4. Реализация dummy-bcrypt:
   ```go
   var hashToVerify domain.PasswordHash
   userFound := user != nil
   if userFound {
       hashToVerify = user.PasswordHash()
   } else {
       hashToVerify = dummyHash // см. ниже
   }
   if vErr := u.hasher.Verify(hashToVerify, password); vErr != nil {
       return LoginUserOutput{}, domain.ErrInvalidCredentials
   }
   if !userFound {
       return LoginUserOutput{}, domain.ErrInvalidCredentials
   }
   ```
   `dummyHash` — заранее посчитанный bcrypt-хеш фиксированной строки (например, "dummy"), inject через конструктор или генерируется лениво при первом отсутствии user'а. Простейший вариант: пакетная переменная `var dummyHash domain.PasswordHash`, инициализируемая в `init()` через `bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.MinCost)` — но это создаёт зависимость `usecase → bcrypt` (запрещено архитектурно).
   **Решение:** dummyHash инжектится в конструктор `NewLoginUser` ровно одним дополнительным параметром `dummyHash domain.PasswordHash`. В `cmd/server/main.go` он создаётся через тот же `PasswordHasher` (`hasher.Hash(domain.NewPassword("dummy12345").Must())` — обернуть в helper). Это единственный чистый способ удовлетворить и архитектурный smoke, и timing-safety.
5. Issue access: `accessToken, accessExp, err := u.issuer.IssueAccess(user.ID(), now)`.
6. Сгенерировать refresh: `raw, err := u.rand.Read(32)` → если ошибка, return wrapped (`../03-decisions.md` риск «сбой crypto/rand»).
7. `hash := sha256.Sum256(raw)` (импорт `crypto/sha256`); `tokenHash, _ := domain.NewTokenHash(hash[:])`.
8. `refreshExp := now.Add(u.refreshTTL)`.
9. `rtID, _ := domain.NewRefreshTokenID(u.uuids.New())`.
10. `rt, err := domain.NewRefreshToken(rtID, user.ID(), tokenHash, refreshExp, now)`.
11. `if err := u.refresh.Save(ctx, rt); err != nil { return ..., wrapped }`.
12. Кодируем raw в base64url (`base64.RawURLEncoding`) и возвращаем:
    ```go
    return LoginUserOutput{
        AccessToken:      accessToken,
        RefreshToken:     base64.RawURLEncoding.EncodeToString(raw),
        AccessExpiresAt:  accessExp,
        RefreshExpiresAt: refreshExp,
    }, nil
    ```

### `refresh_access.go`

Алгоритм (`../02-behavior.md` DFD-3, sequence для RefreshAccess):

1. `raw, err := base64.RawURLEncoding.DecodeString(in.RefreshToken)` — ошибка → `domain.ErrRefreshTokenNotFound`.
2. Проверка `len(raw) == 32` — иначе `ErrRefreshTokenNotFound`.
3. `hash := sha256.Sum256(raw)`; `tokenHash, _ := domain.NewTokenHash(hash[:])`.
4. `existing, err := u.refresh.FindByHash(ctx, tokenHash)` — `ErrRefreshTokenNotFound` → проксируем; другие ошибки — wrap.
5. `now := u.clock.Now()`.
6. Если `existing.IsRevoked()` → `ErrRefreshTokenRevoked`.
7. Если `existing.IsExpired(now)` → `ErrRefreshTokenExpired`.
8. Сгенерировать новый refresh (как в Login: rand.Read(32) → sha256 → domain.NewRefreshToken). При ошибке rand — wrap.
9. `accessToken, accessExp, err := u.issuer.IssueAccess(existing.UserID(), now)` — wrap.
10. `if err := u.refresh.Rotate(ctx, tokenHash, newRT, now); err != nil { ... }` — `ErrRefreshTokenRevoked` (race) проксируем; другие — wrap.
11. Output (тот же шаблон, что в Login).

### `get_current_user.go`

```go
func (u *GetCurrentUser) Execute(ctx context.Context, in GetCurrentUserInput) (GetCurrentUserOutput, error) {
    raw, err := uuid.Parse(in.UserID)
    if err != nil {
        return GetCurrentUserOutput{}, domain.ErrInvalidUserID
    }
    id, err := domain.NewUserID(raw)
    if err != nil {
        return GetCurrentUserOutput{}, domain.ErrInvalidUserID
    }
    user, err := u.users.FindByID(ctx, id)
    if err != nil {
        return GetCurrentUserOutput{}, err // ErrUserNotFound пробрасывается
    }
    return GetCurrentUserOutput{
        UserID:    user.ID().String(),
        Email:     user.Email().String(),
        Username:  user.Username().String(),
        CreatedAt: user.CreatedAt(),
    }, nil
}
```

## Файлы для создания (тесты)

### `fakes_test.go`

Структуры фейков по `../04-testing.md:121-156`. `package usecase_test` (чёрный ящик). Каждый фейк — простая map'а или slice без потокобезопасности (использование однопоточное в тестах).

Дополнительно: `func newRealHasher() *bcrypt.PasswordHasher { ... }` — НЕ используем; вместо bcrypt в use case-тестах живёт `fakeHasher`, который мапит `Password → PasswordHash` через лукап.

### `register_user_test.go` — 6 тестов (`../04-testing.md:160-169`)

Каждый тест — `t.Parallel()`. Структура: build SUT с фейками → execute → assert.

Helper: `mustOutput := func(t *testing.T, ...) RegisterUserOutput { ... }` (`prompts/Tests Style.txt:75-100`).

### `login_user_test.go` — 6 тестов (`../04-testing.md:171-180`)

Особо важный — `TestLoginUser_DummyHashCalledOnUserNotFound`: использовать `fakeHasher` со счётчиком `verifyCalls`, проверить, что `verifyCalls == 1` даже если user не найден.

### `refresh_access_test.go` — 6 тестов (`../04-testing.md:182-191`)

Table-driven для `TestRefreshAccess_NotFound` (3 кейса: bad base64, wrong length, repo not found).

### `get_current_user_test.go` — 3 теста (`../04-testing.md:193-199`)

## Ключевые решения

- Парсинг и валидация входов происходит **внутри use case** (а не в handler) — handler только декодирует JSON. Это даёт единые правила и одно место для тестирования.
- В Login любая ошибка валидации сводится к `ErrInvalidCredentials` для timing-safety.
- Constant-time bcrypt verify для несуществующих email — обязателен (ADR-008).
- Refresh-token raw → sha256 → стораж только хеш. Raw виден только клиенту (`../03-decisions.md` ADR-011).
- `base64.RawURLEncoding` (без padding) — длина 43 для 32 байт; совпадает с `../08-api-contract.md:155`.
- В use case'ах **запрещён** прямой доступ к `time.Now()` или `crypto/rand`. Только через инжекты `Clock`, `RandomBytes`. Это критично для тестируемости.

## Verification

- [ ] `go build ./internal/auth/usecase/...` — зелёный.
- [ ] `go test ./internal/auth/usecase/... -race -count=1` — все 21 тест зелёные.
- [ ] `make lint` зелёный.
- [ ] Импорты в `usecase/*.go`: только stdlib (`context`, `errors`, `fmt`, `time`, `crypto/sha256`, `encoding/base64`) + `github.com/google/uuid` + `internal/auth/domain`. Никаких импортов на адаптеры. Финальная проверка — фаза 8 (smoke-тест).
- [ ] `git diff --name-only` ограничен `internal/auth/usecase/**`.
- [ ] Каждый из тестов из `../04-testing.md:117-199` фактически написан. Сверить число: 6+6+6+3 = 21.
