---
phase: 1
name: Зависимости, config, sqlc.yaml
layer: infrastructure
depends_on: none
plan: ./README.md
---

# Phase 1: Зависимости, config, sqlc.yaml

## Цель

Подготовить инфраструктурный фундамент: добавить четыре прямые зависимости в `go.mod`, расширить `config/` под TTL access/refresh, ввести `sqlc.yaml` и `make sqlc` цель. После фазы — `go build ./...` зелёный, `config_test.go` зелёный, `make sqlc` ничего не генерирует (нет `queries/*.sql`), но конфигурация валидна.

## Контекст

- В `go.mod` сейчас одна прямая зависимость — `github.com/go-chi/chi/v5` (`go.mod:1-5`). Добавляем ровно 4 (`../README.md:44`).
- В `config/config.go` уже есть паттерн `Lookuper` + `ValidationError` + коды `CONFIG-001..003` (`config/config.go:1-103`). Дополняем по той же форме.
- `Makefile` уже содержит цели `run/test/lint/build/migrate-up/migrate-down` (`Makefile:1-58`). Добавляем `sqlc`.
- `sqlc.yaml` сейчас отсутствует — это первая фича, требующая sqlc.

## Файлы для модификации

### `go.mod`, `go.sum`

Добавить ровно следующие прямые зависимости (`../03-decisions.md` ADR-001..ADR-003):
- `github.com/jackc/pgx/v5` — драйвер + `pgxpool` + `pgconn.PgError` + `pgtype` (актуальная стабильная мажорная версия v5.x).
- `github.com/golang-jwt/jwt/v5` — JWT HS256.
- `github.com/google/uuid` — UUID для domain/usecase.
- `golang.org/x/crypto` — bcrypt (подпакет `bcrypt`).

Команда: `go get github.com/jackc/pgx/v5 github.com/golang-jwt/jwt/v5 github.com/google/uuid golang.org/x/crypto && go mod tidy`. После — проверить, что `go.mod` содержит ровно 5 строк в `require` (chi + 4 новых) на верхнем уровне; всё транзитивное — в блоке `require ( ... // indirect )`.

### `config/config.go`

Добавить поля и геттеры, **сохраняя порядок и стиль** существующих:

```go
type Config struct {
    serverPort     int
    databaseURL    string
    jwtSecret      string
    jwtAccessTTL   time.Duration
    jwtRefreshTTL  time.Duration
}

func (c *Config) JWTAccessTTL() time.Duration  { return c.jwtAccessTTL }
func (c *Config) JWTRefreshTTL() time.Duration { return c.jwtRefreshTTL }
```

В `Load(...)` после успешного парсинга `port` добавить вызовы `loadAccessTTL(l)` и `loadRefreshTTL(l, accessTTL)` (refresh валидируется относительно access, поэтому передаём как параметр). Возвращаемый `*Config` пополняется новыми полями.

#### `loadAccessTTL`

- ENV: `JWT_ACCESS_TTL`. Default: `15 * time.Minute` (`../03-decisions.md` ADR-004).
- Если переменной нет или пустая — вернуть default, без ошибки.
- Если есть — `time.ParseDuration(raw)`. Ошибка → `ValidationError{Code: "CONFIG-004", Field: "JWT_ACCESS_TTL", Reason: fmt.Sprintf("must be a duration, got %q", raw)}`.
- После парсинга проверить `d > 0 && d <= time.Hour` (защита от опечаток, `../03-decisions.md` ADR-004 п.6). Нарушение → `CONFIG-004` с понятным `Reason`.

#### `loadRefreshTTL`

- ENV: `JWT_REFRESH_TTL`. Default: `720 * time.Hour` (30 дней).
- Парсинг и null-handling — аналогично access.
- Дополнительные проверки: `d > 0`, `d > accessTTL`, `d <= 90 * 24 * time.Hour`. Все нарушения → `ValidationError{Code: "CONFIG-005", Field: "JWT_REFRESH_TTL", Reason: ...}`.

### `config/config_test.go`

Добавить 4 теста по `../04-testing.md:313-322`. Использовать существующий паттерн `mapLookuper` (если уже есть в файле — переиспользовать; если нет — описать локально). Тесты:

- `TestConfig_Load_DefaultAccessTTL` — нет `JWT_ACCESS_TTL` → `cfg.JWTAccessTTL() == 15*time.Minute`.
- `TestConfig_Load_DefaultRefreshTTL` — нет `JWT_REFRESH_TTL` → `cfg.JWTRefreshTTL() == 720*time.Hour`.
- `TestConfig_Load_InvalidAccessTTL` — table-driven по входам `"abc"`, `"-1m"`, `"0s"`, `"2h"`; ожидаем `errors.Is(err, config.ErrConfigInvalid)` и `verr.Code == "CONFIG-004"`.
- `TestConfig_Load_InvalidRefreshTTL` — table-driven: `"abc"`, `"-1h"`, `"2400h"` (>90d), `"1m"` (< access default 15m); ожидаем `CONFIG-005`.

В каждом тесте `t.Parallel()` (`prompts/Tests Style.txt:24-26`).

### `Makefile`

Добавить цель **до** существующих `lint` и после `migrate-down`:

```makefile
sqlc:
	sqlc generate
.PHONY: sqlc
```

Никаких docker-обёрток, никаких `go run` — sqlc предполагается установленным локально (`prompts` не запрещают prerequisite). Цель будет no-op до фазы 4 (нет `queries/*.sql`).

## Файлы для создания

### `sqlc.yaml`

Точное содержимое — `../06-repo-model.md:267-283`:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations"
    queries: "internal/auth/repository/postgres/queries"
    gen:
      go:
        sql_package: "pgx/v5"
        package: "db"
        out: "internal/auth/repository/postgres/db"
        emit_interface: false
        emit_json_tags: false
        emit_db_tags: false
        emit_pointers_for_null_types: false
        emit_prepared_queries: false
```

В фазе 1 каталоги `queries/` и `db/` ещё не существуют. `sqlc generate` без `queries/*.sql` либо no-op, либо сообщит «no queries» — оба варианта приемлемы; ошибки конфига проверим в фазе 4.

## Ключевые решения

- Драйвер pgx/v5 (`../03-decisions.md` ADR-001).
- JWT — golang-jwt/jwt/v5 (ADR-002), UUID — google/uuid (ADR-003).
- TTL через env с дефолтами 15m/720h (ADR-004).
- Защитные верхние границы `access ≤ 1h`, `refresh ≤ 90d` — приоритет защиты от опечаток над гибкостью.

## Verification

- [ ] `go mod tidy && go build ./...` — без ошибок; `go.mod` содержит 5 прямых зависимостей.
- [ ] `make test ./config/...` — зелёный, включая 4 новых теста.
- [ ] `make lint` зелёный.
- [ ] `git diff --name-only` ограничен `go.mod`, `go.sum`, `config/config.go`, `config/config_test.go`, `Makefile`, `sqlc.yaml`.
- [ ] `make sqlc` отрабатывает (либо warning «no queries found», либо успех — но не падение).
