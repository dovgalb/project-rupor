---
parent: ./README.md
view: quality
---

# 04 — Testing (Quality View)

Все тесты используют стандартную библиотеку `testing` (`prompts/Tests Style.txt:24-26`), `t.Parallel()` везде, без testify/gomock. Фейки пишутся руками рядом с тестом (`prompts/Tests Style.txt:75-100`).

## Coverage Mapping

Каждый код ошибки из `02-behavior.md` и `08-api-contract.md` имеет покрытие тестом.

| Use Case / Компонент | Условие | Code | Тест |
|----------------------|---------|------|------|
| `domain.NewEmail` | пустой / без `@` / >254 / пробелы | `ErrInvalidEmail` | `TestNewEmail_Invalid` |
| `domain.NewEmail` | trim+lowercase нормализация | — | `TestNewEmail_NormalizesAndValidates` |
| `domain.NewUsername` | <3 / >32 / спец. символы | `ErrInvalidUsername` | `TestNewUsername_Invalid` |
| `domain.NewUsername` | trim, regex `^[a-zA-Z0-9_-]+$` | — | `TestNewUsername_Valid` |
| `domain.NewPassword` | <8 / >72 байт | `ErrInvalidPassword` | `TestNewPassword_Invalid` |
| `domain.NewPasswordHash` | пустая строка | `ErrInvalidPasswordHash` | `TestNewPasswordHash_Empty` |
| `domain.NewUserID` | zero-uuid | `ErrInvalidUserID` | `TestNewUserID_Zero` |
| `domain.NewTokenHash` | длина ≠ 32 байт | `ErrInvalidTokenHash` | `TestNewTokenHash_WrongLength` |
| `domain.NewRefreshToken` | `expiresAt ≤ createdAt` | `ErrInvalidRefreshTokenExpiration` | `TestNewRefreshToken_BadExpiration` |
| `domain.RefreshToken.Revoke` | первый вызов | — | `TestRefreshToken_Revoke_Success` |
| `domain.RefreshToken.Revoke` | повторный вызов | `ErrRefreshTokenAlreadyRevoked` | `TestRefreshToken_Revoke_Twice` |
| `domain.RefreshToken.IsActive` | active / revoked / expired | — | `TestRefreshToken_IsActive` (table-driven) |
| `RegisterUser` | happy path | — | `TestRegisterUser_Success` |
| `RegisterUser` | невалидный email | AUTH-001 | `TestRegisterUser_InvalidEmail` |
| `RegisterUser` | невалидный username | AUTH-002 | `TestRegisterUser_InvalidUsername` |
| `RegisterUser` | невалидный password | AUTH-003 | `TestRegisterUser_InvalidPassword` |
| `RegisterUser` | email занят | AUTH-004 | `TestRegisterUser_EmailTaken` |
| `RegisterUser` | username занят | AUTH-005 | `TestRegisterUser_UsernameTaken` |
| `LoginUser` | happy path | — | `TestLoginUser_Success` |
| `LoginUser` | email невалиден / не найден / пароль не совпал | AUTH-006 | `TestLoginUser_InvalidCredentials` (table-driven по 3 кейсам) |
| `LoginUser` | dummy-bcrypt при ErrUserNotFound | — | `TestLoginUser_TimingDoesNotLeakUserExistence` |
| `RefreshAccess` | happy path | — | `TestRefreshAccess_Success` |
| `RefreshAccess` | токен не найден / неверный base64 / неверная длина | AUTH-007 | `TestRefreshAccess_NotFound` (table-driven) |
| `RefreshAccess` | токен отозван | AUTH-008 | `TestRefreshAccess_Revoked` |
| `RefreshAccess` | токен истёк | AUTH-009 | `TestRefreshAccess_Expired` |
| `RefreshAccess` | race: параллельный refresh с тем же токеном | AUTH-008 | `TestRefreshAccess_RaceReturnsRevoked` (через mock-rotate) |
| `GetCurrentUser` | happy path | — | `TestGetCurrentUser_Success` |
| `GetCurrentUser` | userID не найден | mapped to AUTH-010 | `TestGetCurrentUser_NotFound` |
| `MeHandler` | нет Authorization | AUTH-010 | `TestMeHandler_NoAuth` |
| `MeHandler` | Authorization не Bearer | AUTH-010 | `TestMeHandler_BadScheme` |
| `MeHandler` | подпись неверная | AUTH-010 | `TestMeHandler_InvalidSignature` |
| `MeHandler` | exp < now | AUTH-011 | `TestMeHandler_TokenExpired` |
| `TokenIssuer.IssueAccess`+`VerifyAccess` | round-trip | — | `TestTokenIssuer_RoundTrip` |
| `TokenIssuer.VerifyAccess` | `alg: none` | `ErrAccessTokenInvalid` | `TestTokenIssuer_RejectsAlgNone` |
| `TokenIssuer.VerifyAccess` | подписан другим секретом | `ErrAccessTokenInvalid` | `TestTokenIssuer_RejectsWrongSecret` |
| `TokenIssuer.VerifyAccess` | невалидный sub-claim | `ErrAccessTokenInvalid` | `TestTokenIssuer_RejectsBadSub` |
| `TokenIssuer.VerifyAccess` | exp < now | `ErrAccessTokenExpired` | `TestTokenIssuer_RejectsExpired` |
| `PasswordHasher.Hash`+`Verify` | round-trip | — | `TestPasswordHasher_RoundTrip` |
| `PasswordHasher.Verify` | mismatch | `ErrInvalidCredentials` | `TestPasswordHasher_VerifyMismatch` |
| `UserRepository.Save` | duplicate email | `ErrEmailAlreadyTaken` | `TestUserRepository_Save_DuplicateEmail` (integration) |
| `UserRepository.Save` | duplicate username | `ErrUsernameAlreadyTaken` | `TestUserRepository_Save_DuplicateUsername` (integration) |
| `UserRepository.FindByID` / `FindByEmail` | not found | `ErrUserNotFound` | `TestUserRepository_FindByX_NotFound` (integration) |
| `RefreshTokenRepository.Rotate` | atomic happy path | — | `TestRefreshTokenRepository_Rotate_Success` (integration) |
| `RefreshTokenRepository.Rotate` | race: уже отозван | `ErrRefreshTokenRevoked` | `TestRefreshTokenRepository_Rotate_AlreadyRevoked` (integration) |
| `config.Load` | `JWT_ACCESS_TTL` не парсится / out of range | CONFIG-004 | `TestConfig_Load_InvalidAccessTTL` |
| `config.Load` | `JWT_REFRESH_TTL` не парсится / out of range / меньше access | CONFIG-005 | `TestConfig_Load_InvalidRefreshTTL` |
| `RegisterHandler` | malformed JSON body | AUTH-012 | `TestRegisterHandler_MalformedBody` |
| `LoginHandler` / `RefreshHandler` | malformed JSON body | AUTH-012 | `TestLoginHandler_MalformedBody`, `TestRefreshHandler_MalformedBody` |

## Модуль `internal/auth/domain/` — Test Cases

Чёрный ящик: `package domain_test` (`prompts/Domain model test.txt:67`).

### `User` Entity (3 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestNewUser_Success` | На валидных входах возвращает `*User`, поля доступны через геттеры. |
| `TestNewUser_InvalidUserID` | `id.IsZero()` → `ErrInvalidUserID`, объект не создаётся. |
| `TestNewUser_InvalidCreatedAt` | `createdAt.IsZero()` → `ErrInvalidCreatedAt`. |
| `TestReconstructUser_Success` | Конструктор для репозитория возвращает идентичную сущность с тем же id и createdAt. |

### `RefreshToken` Entity (5 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestNewRefreshToken_Success` | Активный токен создаётся, `IsActive(now) == true`. |
| `TestNewRefreshToken_BadExpiration` | `expiresAt ≤ createdAt` → `ErrInvalidRefreshTokenExpiration`. |
| `TestRefreshToken_Revoke_Success` | После `Revoke(now)` — `IsRevoked() == true`, `RevokedAt() == now`. |
| `TestRefreshToken_Revoke_Twice` | Повторный `Revoke` → `ErrRefreshTokenAlreadyRevoked`, `RevokedAt()` не изменился. |
| `TestRefreshToken_IsActive` | Table-driven: active / revoked / expired / revoked+expired. |

### Value Objects (8 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestNewEmail` | Table-driven: валидные с trim+lowercase, невалидные (пустая, без `@`, пробелы, >254), сравнение `Email == Email`. |
| `TestNewUsername` | Table-driven: валидные (3..32 ASCII alnum/_/-), невалидные (короткий, длинный, пробелы, `+`, `@`, кириллица). |
| `TestNewPassword` | Table-driven: 8..72 байт OK, <8 → `ErrInvalidPassword`, >72 → то же. |
| `TestNewPasswordHash_Empty` | Пустая строка → `ErrInvalidPasswordHash`. |
| `TestNewUserID_Zero` | `uuid.Nil` → `ErrInvalidUserID`. |
| `TestNewRefreshTokenID_Zero` | `uuid.Nil` → `ErrInvalidRefreshTokenID`. |
| `TestNewTokenHash_WrongLength` | Table-driven: `len < 32`, `len > 32` → `ErrInvalidTokenHash`. `len == 32` — OK. |
| `TestTokenHash_Equal` | Два одинаковых хеша равны; разные — нет. |

### Test Helpers

`internal/auth/domain/testing_fixtures_test.go` (либо в `*_test.go` для соответствующих сущностей):

```go
func mustEmail(t *testing.T, raw string) domain.Email {
    t.Helper()
    e, err := domain.NewEmail(raw)
    if err != nil { t.Fatalf("mustEmail(%q): %v", raw, err) }
    return e
}
// аналогично mustUsername, mustPasswordHash, mustTokenHash
```

`prompts/Domain model test.txt:71-86` — обязателен `t.Helper()`.

## Модуль `internal/auth/usecase/` — Test Cases

Чёрный ящик: `package usecase_test`. Фейковые реализации интерфейсов в `internal/auth/usecase/fakes_test.go` (или рядом с каждым тестом).

### Фейки (`fakes_test.go`)

```go
type fakeUserRepo struct {
    byID    map[string]*domain.User
    byEmail map[string]*domain.User
    saved   []*domain.User
    saveErr error
}
func (r *fakeUserRepo) Save(ctx context.Context, u *domain.User) error { ... }
func (r *fakeUserRepo) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) { ... }
func (r *fakeUserRepo) FindByEmail(ctx context.Context, e domain.Email) (*domain.User, error) { ... }

type fakeRefreshRepo struct {
    byHash    map[[32]byte]*domain.RefreshToken
    saved     []*domain.RefreshToken
    rotated   []*domain.RefreshToken
    rotateErr error
}

type fakeHasher struct {
    hashes map[domain.Password]domain.PasswordHash // pre-populated
    verifyErr error
}

type fakeIssuer struct {
    issued []domain.UserID
    accessTTL time.Duration
    issueErr error
    verifyErr error
}

type fixedClock struct{ now time.Time }
type fixedUUID struct{ next []uuid.UUID }
type fixedRand struct{ next [][]byte }
```

Все фейки — структуры с публичными полями, `prompts/Tests Style.txt:75-100`.

### `RegisterUser` (6 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestRegisterUser_Success` | Happy path: вызов `Hash`, `New()`, `Now()`, `Save` — в правильном порядке. Repo содержит сохранённого user. Output совпадает с входом. |
| `TestRegisterUser_InvalidEmail` | `input.Email = "not-email"` → `ErrInvalidEmail`. Repo пустой, hasher не вызван. |
| `TestRegisterUser_InvalidUsername` | `input.Username = "ab"` → `ErrInvalidUsername`. |
| `TestRegisterUser_InvalidPassword` | `input.Password = "1234"` → `ErrInvalidPassword`. Hasher не вызван. |
| `TestRegisterUser_EmailTaken` | `repo.Save` возвращает `ErrEmailAlreadyTaken` → use case проксирует. |
| `TestRegisterUser_UsernameTaken` | `repo.Save` возвращает `ErrUsernameAlreadyTaken` → use case проксирует. |

### `LoginUser` (6 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestLoginUser_Success` | Happy path: возвращает access+refresh, в `refreshRepo` появилась запись с правильным userID и hash, hash = sha256(rawRefresh). |
| `TestLoginUser_InvalidCredentials_BadEmailFormat` | input email `"x"` → `ErrInvalidCredentials` (не `ErrInvalidEmail`!). |
| `TestLoginUser_InvalidCredentials_UserNotFound` | `repo.FindByEmail` → `ErrUserNotFound`; use case вызывает `hasher.Verify` (dummy для timing) и возвращает `ErrInvalidCredentials`. |
| `TestLoginUser_InvalidCredentials_WrongPassword` | `hasher.Verify` → `ErrMismatchedHashAndPassword`; use case возвращает `ErrInvalidCredentials`. |
| `TestLoginUser_DummyHashCalledOnUserNotFound` | Проверка таймингового выравнивания: `fakeHasher.verifyCalls` инкрементируется даже при `ErrUserNotFound`. |
| `TestLoginUser_RandReadFails` | `fakeRand.Read` возвращает ошибку → use case возвращает обёрнутую ошибку, `refreshRepo` пуст, `hasher.Verify` уже вызван (порядок). |

### `RefreshAccess` (6 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestRefreshAccess_Success` | Happy path: возвращает новый access+refresh, в `refreshRepo` зафиксирован `Rotate(oldHash, new, now)`. |
| `TestRefreshAccess_NotFound` | Table-driven: невалидный base64, длина после декода ≠ 32, `FindByHash` → `ErrRefreshTokenNotFound`. Все три → `ErrRefreshTokenNotFound`. |
| `TestRefreshAccess_Revoked` | `repo.FindByHash` возвращает токен с `revokedAt != zero` → `ErrRefreshTokenRevoked`. |
| `TestRefreshAccess_Expired` | `repo.FindByHash` возвращает токен с `expiresAt < now` → `ErrRefreshTokenExpired`. |
| `TestRefreshAccess_RaceReturnsRevoked` | `Rotate` возвращает `ErrRefreshTokenRevoked` (race) → use case проксирует. |
| `TestRefreshAccess_TokenIssuerFails` | `issuer.IssueAccess` → ошибка → use case возвращает обёрнутую ошибку, `Rotate` НЕ вызывается. |

### `GetCurrentUser` (3 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestGetCurrentUser_Success` | Happy path: возвращает поля из user'а. |
| `TestGetCurrentUser_BadUUID` | `input.UserID = "not-uuid"` → `ErrInvalidUserID`. |
| `TestGetCurrentUser_NotFound` | `repo.FindByID` → `ErrUserNotFound`; use case проксирует. |

## Модуль `internal/auth/repository/postgres/` — Test Cases

Интеграционные тесты против **реального PostgreSQL** (`prompts/Tests Style.txt:120-145`). Build tag `//go:build integration` — запускаются отдельно, не блокируют `make test`.

В фазе 1.3 интеграционные тесты могут быть упрощены или помечены `t.Skip("integration: requires Postgres")` с TODO к фазе 1.5 (`general_plan.md:120-123`). Минимально необходимое — round-trip тесты маппинга.

### `UserRepository` (integration, 5 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestUserRepository_Save_RoundTrip` | Save → FindByID → возвращённая сущность совпадает по всем полям (через геттеры; `cmp.Diff` опционально). |
| `TestUserRepository_Save_DuplicateEmail` | Сохранили user, повторный Save с тем же email → `ErrEmailAlreadyTaken`. |
| `TestUserRepository_Save_DuplicateUsername` | То же с username → `ErrUsernameAlreadyTaken`. |
| `TestUserRepository_FindByEmail_CaseInsensitive` | Сохранили `User@Example.COM` (после `Email` нормализации — `user@example.com`); FindByEmail с любым регистром даёт ту же сущность (citext-уровень). |
| `TestUserRepository_FindByID_NotFound` | Random UUID → `ErrUserNotFound`. |

### `RefreshTokenRepository` (integration, 4 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestRefreshTokenRepository_Save_RoundTrip` | Save → FindByHash → совпадает по всем полям. |
| `TestRefreshTokenRepository_FindByHash_NotFound` | `ErrRefreshTokenNotFound`. |
| `TestRefreshTokenRepository_Rotate_Atomic` | Сохранили активный токен → Rotate(old, new) → старый имеет `revokedAt != null`, новый существует и активен; всё в одной tx. |
| `TestRefreshTokenRepository_Rotate_AlreadyRevoked` | Сохранили токен, помечен revoked. Rotate → `ErrRefreshTokenRevoked`, нового токена в БД нет. |

### Repo Model Round-Trip Tests

| Тест | Что проверяет |
|------|---------------|
| `TestUserMapping_RoundTrip_AllFieldsPreserved` | Domain → InsertParams → INSERT → SELECT → row → Domain. Все поля сохранены. |
| `TestRefreshTokenMapping_RoundTrip_AllFieldsPreserved` | То же для `RefreshToken`, включая `revokedAt = zero` ↔ NULL и `revokedAt != zero` ↔ timestamptz. |

## Модуль `internal/auth/repository/jwt/` — Test Cases

### `TokenIssuer` (6 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestTokenIssuer_RoundTrip` | IssueAccess(uid, now) → token, exp; VerifyAccess(token, now+1m) → uid. |
| `TestTokenIssuer_VerifyAccess_RejectsAlgNone` | Подделанный токен с `alg: none` → `ErrAccessTokenInvalid`. |
| `TestTokenIssuer_VerifyAccess_RejectsWrongSecret` | Токен подписан другим secret → `ErrAccessTokenInvalid`. |
| `TestTokenIssuer_VerifyAccess_RejectsExpired` | Токен с `exp < now` → `ErrAccessTokenExpired`. |
| `TestTokenIssuer_VerifyAccess_RejectsBadSub` | `sub` не парсится в UUID → `ErrAccessTokenInvalid`. |
| `TestTokenIssuer_VerifyAccess_RejectsTampered` | Подмена payload → подпись не сходится → `ErrAccessTokenInvalid`. |

## Модуль `internal/auth/repository/bcrypt/` — Test Cases

### `PasswordHasher` (3 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestPasswordHasher_RoundTrip` | Hash(p) → h; Verify(h, p) → nil. Cost = 4 в тестах для скорости. |
| `TestPasswordHasher_VerifyMismatch` | Verify(h, "wrong") → `ErrInvalidCredentials`. |
| `TestPasswordHasher_VerifyInvalidHash` | Verify(empty, p) → wrapped error (не `ErrInvalidCredentials`). |

## Модуль `internal/auth/transport/http/` — Test Cases

`httptest.NewServer` поверх собранного `chi.Router` (`prompts/Tests Style.txt:147-162`). Реальные use case'ы (не моки use case'ов!), фейковые репозитории/hasher/issuer.

### `RegisterHandler` (5 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestRegisterHandler_Success` | POST /auth/register с валидным body → 201, JSON соответствует `08-api-contract.md` (id — uuid-формат, createdAt — RFC3339). |
| `TestRegisterHandler_MalformedBody` | Не-JSON / unexpected EOF → 400 + `AUTH-012`. |
| `TestRegisterHandler_InvalidEmail` | email `"x"` → 400 + `AUTH-001`. |
| `TestRegisterHandler_EmailTaken` | Repo возвращает `ErrEmailAlreadyTaken` → 409 + `AUTH-004`. |
| `TestRegisterHandler_UnsupportedMethod` | GET /auth/register → 405. |

### `LoginHandler` (4 теста)

| Тест | Что проверяет |
|------|---------------|
| `TestLoginHandler_Success` | 200 + JSON {accessToken, refreshToken, accessExpiresAt, refreshExpiresAt}. |
| `TestLoginHandler_MalformedBody` | 400 + AUTH-012. |
| `TestLoginHandler_InvalidCredentials` | Любая ошибка из use case → 401 + AUTH-006 (table-driven). |
| `TestLoginHandler_InternalError` | Сбой rand → 500 + INTERNAL. |

### `RefreshHandler` (5 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestRefreshHandler_Success` | 200 + JSON. |
| `TestRefreshHandler_MalformedBody` | 400 + AUTH-012. |
| `TestRefreshHandler_NotFound` | 401 + AUTH-007. |
| `TestRefreshHandler_Revoked` | 401 + AUTH-008. |
| `TestRefreshHandler_Expired` | 401 + AUTH-009. |

### `MeHandler` (5 тестов)

| Тест | Что проверяет |
|------|---------------|
| `TestMeHandler_Success` | Bearer-токен валиден → 200 + JSON. |
| `TestMeHandler_NoAuth` | Нет заголовка → 401 + AUTH-010. |
| `TestMeHandler_BadScheme` | `Authorization: Basic ...` → 401 + AUTH-010. |
| `TestMeHandler_InvalidSignature` | Подпись неправильная → 401 + AUTH-010. |
| `TestMeHandler_TokenExpired` | exp < now → 401 + AUTH-011. |

## Архитектурный smoke-тест

Активирует `t.Skip` из фазы 1.1 (`docs/1_1_project-structure/04-testing.md:64-80`).

Файл: `arch_test.go` в корне (или `internal/arch_test.go`). Использует `golang.org/x/tools/go/packages` (если разрешено добавить в test-deps; иначе — простой обход `go/parser` + `go/ast`).

**Альтернатива без новых зависимостей**: использовать stdlib `go/build` или `golang.org/x/tools/go/packages` уже зарегистрировано как стандартный Go-инструмент (входит в Go SDK). Согласовать на этапе implementation.

| Тест | Проверка |
|------|----------|
| `TestArchitecture_DomainImports` | Все импорты в `internal/auth/domain/*.go` принадлежат whitelist: stdlib + `github.com/google/uuid`. |
| `TestArchitecture_UseCaseImports` | `internal/auth/usecase/*.go` импортирует только stdlib + `github.com/google/uuid` + `github.com/dovgalb/project-rupor/internal/auth/domain`. |
| `TestArchitecture_RepositoriesIsolated` | `internal/auth/repository/{postgres,jwt,bcrypt}/` не импортируют `internal/auth/transport/...` и не друг друга. |

## `config/` — Test Cases (расширение)

Новые тесты к существующим (`config/config_test.go:1-147`):

| Тест | Что проверяет |
|------|---------------|
| `TestConfig_Load_DefaultAccessTTL` | `JWT_ACCESS_TTL` отсутствует → `cfg.JWTAccessTTL() == 15*time.Minute`. |
| `TestConfig_Load_DefaultRefreshTTL` | `JWT_REFRESH_TTL` отсутствует → `cfg.JWTRefreshTTL() == 720*time.Hour`. |
| `TestConfig_Load_InvalidAccessTTL` | Table-driven: `"abc"`, `"-1m"`, `"2h"` (>1h) → `CONFIG-004`. |
| `TestConfig_Load_InvalidRefreshTTL` | Table-driven: `"abc"`, `"-1h"`, `"100d"`/`"2400h"` (>90d), `"1m"` (< access) → `CONFIG-005`. |

## Test Count Summary

| Модуль | Domain | Builder | UseCase | VO | Repo Model | Integration | HTTP | Total |
|--------|--------|---------|---------|----|-----------|-------------|------|-------|
| `internal/auth/domain` | 8 (User+RT) | — | — | 8 | — | — | — | 16 |
| `internal/auth/usecase` | — | — | 21 | — | — | — | — | 21 |
| `internal/auth/repository/postgres` | — | — | — | — | 2 | 9 | — | 11 |
| `internal/auth/repository/jwt` | — | — | 6 | — | — | — | — | 6 |
| `internal/auth/repository/bcrypt` | — | — | 3 | — | — | — | — | 3 |
| `internal/auth/transport/http` | — | — | — | — | — | — | 19 | 19 |
| `config/` (расширение) | — | — | 4 | — | — | — | — | 4 |
| Архитектурный smoke | — | — | — | — | — | 3 | — | 3 |
| **ИТОГО** | **8** | — | **34** | **8** | **2** | **12** | **19** | **83** |

(Builder pattern не применяется в этой фиче — для двух сущностей с конструкторами на 4–5 полей он избыточен по `prompts/Builder.txt:14-19`. Если в фазе 2 (`room`) появятся сущности с 8+ полями — там будет builder.)

## Стратегия запуска

- `make test` (`Makefile:35-36`) — запускает unit-тесты domain/usecase/jwt/bcrypt/transport-http с фейками. **Это default**.
- `go test -tags=integration ./internal/auth/repository/postgres/... -count=1` — запускает интеграционные тесты с реальным Postgres. Требует `make dc-up` и `make migrate-up`. Может выполняться вручную или отдельным CI-job.
- `go test ./... -race -count=1` — race detector.
- В CI (`.github/workflows/ci.yml`) сейчас запускается без integration-tag; интеграционные тесты будут добавлены в задаче 1.5 (`general_plan.md:122`).
