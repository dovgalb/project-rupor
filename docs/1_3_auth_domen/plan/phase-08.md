---
phase: 8
name: cmd/server composition root + arch smoke-test
layer: infrastructure
depends_on: phase-04, phase-05, phase-07
plan: ./README.md
---

# Phase 8: Composition root + архитектурный smoke-тест

## Цель

Финальная фаза. Собрать граф зависимостей в `cmd/server/main.go`: подключить `pgxpool.Pool`, sqlc-`*db.Queries`, репозитории, hasher, token-issuer, use case'ы и зарегистрировать роуты. Активировать архитектурный smoke-тест, который запрещён ранее. После фазы — `make run` поднимает сервер, эндпоинты `/api/v1/auth/*` отвечают; `make test` зелёный, включая arch_test.

## Контекст

- Граф composition root — `../01-architecture.md:316-370`.
- `pkg/websocket` пуст — никаких dependency не вносим, кроме auth.
- Архитектурный smoke в `docs/1_1_project-structure/04-testing.md:64-80` помечен `t.Skip` — снимаем `Skip` и расширяем whitelist.
- Существующий `cmd/server/main.go:48-91` уже умеет graceful shutdown по сигналам — не ломаем эту логику.

## Файлы для модификации

### `cmd/server/main.go`

Изменения в `run(cfg, logger)`. Точки вмешательства:

**1. Перед `mux := chi.NewRouter()`** — создаём `pgxpool.Pool`, инфраструктурные адаптеры, use case'ы:

```go
ctx := context.Background()
pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
if err != nil {
    return fmt.Errorf("pgxpool.New: %w", err)
}
defer pool.Close()

queries := db.New(pool)
userRepo := postgres.NewUserRepository(queries)
refreshRepo := postgres.NewRefreshTokenRepository(pool)

hasher := bcrypt.NewPasswordHasher(10) // prod cost
issuer := jwtimpl.NewTokenIssuer([]byte(cfg.JWTSecret()), cfg.JWTAccessTTL())

clock := realClock{}
uuids := realUUID{}
rand := cryptoRand{}

// dummy hash для timing-safe Login (см. ../03-decisions.md ADR-008)
dummyPwd, _ := domain.NewPassword("dummy_password_for_timing_safety")
dummyHash, err := hasher.Hash(dummyPwd)
if err != nil {
    return fmt.Errorf("init dummy hash: %w", err)
}

registerUC := usecase.NewRegisterUser(userRepo, hasher, clock, uuids)
loginUC := usecase.NewLoginUser(userRepo, refreshRepo, hasher, issuer, clock, uuids, rand, cfg.JWTRefreshTTL(), dummyHash)
refreshUC := usecase.NewRefreshAccess(refreshRepo, issuer, clock, uuids, rand, cfg.JWTRefreshTTL())
meUC := usecase.NewGetCurrentUser(userRepo)
```

**2. В `mux.Route("/api/v1", ...)`** — после `r.Get("/health", ...)`:

```go
httpauth.RegisterRoutes(r, httpauth.Deps{
    Register:    registerUC,
    Login:       loginUC,
    Refresh:     refreshUC,
    Me:          meUC,
    TokenIssuer: issuer,
    Clock:       clock,
})
```

**3. После `srv.Shutdown(...)`** (см. `cmd/server/main.go:85`) — `pool.Close()` уже выполнится через `defer`. Проверить порядок: pgx документирует, что `pool.Close()` блокирует до завершения активных запросов. Если `srv.Shutdown` уже подождал http-запросы (они держат pool-коннекшены), `pool.Close()` отработает быстро.

**4. Inline-структуры реальных порт-импл** (см. `../03-decisions.md` OQ-9) — добавить в `main.go` (или вынести в `cmd/server/runtime.go`):

```go
type realClock struct{}
func (realClock) Now() time.Time { return time.Now().UTC() }

type realUUID struct{}
func (realUUID) New() uuid.UUID { return uuid.New() }

type cryptoRand struct{}
func (cryptoRand) Read(n int) ([]byte, error) {
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        return nil, err
    }
    return b, nil
}
```

Импорты: `crypto/rand` (alias не нужен; конфликтует с локальным именем переменной `rand` выше — поэтому переменную лучше переименовать в `randSrc` или в `randomBytes`). Финальный выбор имени — на этапе реализации.

**5. Импорты пополняются:**
- `github.com/jackc/pgx/v5/pgxpool`
- `github.com/dovgalb/project-rupor/internal/auth/repository/postgres`
- `github.com/dovgalb/project-rupor/internal/auth/repository/postgres/db`
- `github.com/dovgalb/project-rupor/internal/auth/repository/jwt` под алиасом `jwtimpl`
- `github.com/dovgalb/project-rupor/internal/auth/repository/bcrypt`
- `github.com/dovgalb/project-rupor/internal/auth/usecase`
- `github.com/dovgalb/project-rupor/internal/auth/domain`
- `github.com/dovgalb/project-rupor/internal/auth/transport/http` под алиасом `httpauth`
- `github.com/google/uuid`
- `crypto/rand` под алиасом `cryptorand` (или обыграть конфликт имён)

## Файлы для создания

### `arch_test.go` (или `internal/arch_test.go`)

Активирует архитектурный smoke. Позиционирование — корень репо: единственный файл, который может проверять межпакетные импорты без обратных зависимостей.

```go
//go:build !integration

package main_test // или package arch_test, если файл в корне

// Использует golang.org/x/tools/go/packages — НЕ добавляется в go.mod как
// production-зависимость, а только как test-зависимость. Если эта зависимость
// нежелательна, альтернатива — go/parser + filepath.Walk (без новых deps,
// больше boilerplate).
```

Минимум 3 теста (`../04-testing.md:307-311`):

1. `TestArchitecture_DomainImports` — все импорты в `internal/auth/domain/*.go` (исключая `*_test.go`) должны быть в whitelist `{stdlib, github.com/google/uuid}`.
2. `TestArchitecture_UseCaseImports` — `internal/auth/usecase/*.go` (исключая тесты) импортирует только stdlib + `github.com/google/uuid` + `github.com/dovgalb/project-rupor/internal/auth/domain`.
3. `TestArchitecture_RepositoriesIsolated` — пакеты `internal/auth/repository/{postgres,jwt,bcrypt}` не импортируют друг друга и не импортируют `internal/auth/transport/...`.

`stdlib` определяется через `golang.org/x/tools/go/packages` или через простой признак «import path не содержит `.`» (приближённо, но достаточно для проекта без vendored stdlib).

**Решение по реализации:** взять `golang.org/x/tools/go/packages` как test-only dependency (`go get -t golang.org/x/tools`) — ОК, потому что это часть Go SDK ecosystem (ADR/заметка в `../03-decisions.md` нужно дописать после фазы или согласовать с пользователем). Если пользователь против новой зависимости — использовать `go/parser` + `filepath.Walk` (больше кода, меньше зависимостей).

### `docs/1_1_project-structure/04-testing.md`

Снять `t.Skip` в фазе 1.1 — обновить документ, что архитектурный тест активирован в фазе 1.3 (если документ упоминает это явно). **Но** не редактируем доки фазы 1.1 в рамках этой фичи без согласования — лучше добавить ссылку из `arch_test.go` на причину активации, и оставить документ 1.1 как есть.

## Manual QA

10 шагов по `../08-api-contract.md:281-296`. Запуск:

```bash
make dc-up
make migrate-up
make run
```

Затем выполнить шаги 1..10 через `manual_qa/auth/test-flow.http` (если создан в фазе 7) или `curl`. Каждый шаг — ожидаемый response. **Это часть Definition of Done фазы 8.**

## Ключевые решения

- `pgxpool.New` создаётся в `run`, не в глобальной переменной — нет process-wide мутабельного состояния. Closures не нужны.
- `defer pool.Close()` гарантирует освобождение даже при паника-выходе.
- `dummyHash` создаётся один раз при старте — ~50 мс. Оправдано timing-safety (ADR-008).
- Inline real* структуры остаются в `cmd/server/`, не выносятся в `pkg/` — `../03-decisions.md` OQ-9, правило «не создавать пакет ради одной функции».
- Архитектурный тест — единственное место, где разрешена test-only зависимость на `golang.org/x/tools`. Никаких production-имплементаций на этой библиотеке.

## Verification

- [ ] `go build ./...` — зелёный.
- [ ] `make test` — все ~71 unit-теста зелёные, включая arch_test.
- [ ] `make lint` зелёный.
- [ ] `go test ./... -race -count=1` — race-detector чистый.
- [ ] `make dc-up && make migrate-up && make run` — сервер поднимается, в логах нет ошибок.
- [ ] Manual QA из `../08-api-contract.md:281-296`: все 10 шагов дают ожидаемые ответы.
- [ ] При прерывании сервера (Ctrl+C) — graceful shutdown логирует `shutdown complete`, нет panic'ов.
- [ ] `git diff --name-only` ограничен `cmd/server/main.go`, `arch_test.go`, и (если потребовалось) `go.mod`/`go.sum` для `golang.org/x/tools` test-deps.
- [ ] **Все 14 пунктов «Критериев приёмки» из `../README.md:33-47` выполнены.**
