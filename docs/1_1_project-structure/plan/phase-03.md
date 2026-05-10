---
phase: 3
name: Config Package
layer: application
depends_on: [phase-01]
plan: ./README.md
---

# Phase 3: Config Package

## Цель

Создать пакет `config/` с типом `Config`, интерфейсом `Lookuper`, функцией `Load(Lookuper) (*Config, error)` и тестами. Все три ENV-переменные (`SERVER_PORT`, `DATABASE_URL`, `JWT_SECRET`) валидируются по схеме из `../03-decisions.md` ADR-004 и `../02-behavior.md` UC1 (Error cases).

После этой фазы `go test ./config/...` зелёный.

## Контекст

Что произвели предыдущие фазы:
- Phase 1: `.golangci.yml`, `Makefile` (`make test`/`make lint`), `.env.example` со списком переменных.

Эта фаза не зависит от Phase 2 (нет общих файлов). Может выполняться параллельно с Phase 2, но в плане поставлена после неё для линейного порядка ревью.

Эта фаза не модифицирует `go.mod` — пакет `config/` использует только stdlib (`os`, `strconv`, `errors`, `fmt`).

## Файлы для создания

### `config/config.go`

**Назначение:** тип `Config`, интерфейс `Lookuper`, реализация `OsLookuper`, типизированная ошибка `ValidationError`, sentinel `ErrConfigInvalid`, функция `Load(Lookuper) (*Config, error)`.

**Детали реализации:**

```go
// Package config загружает и валидирует конфигурацию приложения из переменных окружения.
//
// Источник env инжектируется через интерфейс Lookuper, чтобы тесты не зависели от os.Setenv
// (см. prompts/Tests Style.txt — детерминизм).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const defaultServerPort = 8080

// ErrConfigInvalid — sentinel-ошибка для всех ошибок валидации конфигурации.
// Транспорт/main мапит её в exit 1 + сообщение в stderr.
var ErrConfigInvalid = errors.New("config: invalid")

// ValidationError описывает конкретное нарушение в одной env-переменной.
// Реализует error и обёрнута через ErrConfigInvalid (errors.Is).
type ValidationError struct {
	Code   string // например "CONFIG-001"
	Field  string // например "JWT_SECRET"
	Reason string // человекочитаемая причина
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config: %s: %s (%s)", e.Field, e.Reason, e.Code)
}

func (e ValidationError) Is(target error) bool {
	return target == ErrConfigInvalid
}

// Config — иммутабельная конфигурация приложения.
// Все поля приватные, чтобы не было соблазна мутировать в рантайме.
// Доступ — через геттеры.
type Config struct {
	serverPort  int
	databaseURL string
	jwtSecret   string
}

func (c *Config) ServerPort() int     { return c.serverPort }
func (c *Config) DatabaseURL() string { return c.databaseURL }
func (c *Config) JWTSecret() string   { return c.jwtSecret }

// Lookuper абстрагирует источник env-переменных.
// В проде — OsLookuper, в тестах — фейковый mapLookuper.
type Lookuper interface {
	Lookup(key string) (string, bool)
}

// OsLookuper читает переменные через os.LookupEnv.
type OsLookuper struct{}

func (OsLookuper) Lookup(key string) (string, bool) {
	return os.LookupEnv(key)
}

// Load читает конфигурацию через l и валидирует.
// При ошибке возвращает ValidationError, оборачиваемую через errors.Is(err, ErrConfigInvalid).
func Load(l Lookuper) (*Config, error) {
	jwt, ok := l.Lookup("JWT_SECRET")
	if !ok || jwt == "" {
		return nil, ValidationError{
			Code:   "CONFIG-001",
			Field:  "JWT_SECRET",
			Reason: "required",
		}
	}

	dbURL, ok := l.Lookup("DATABASE_URL")
	if !ok || dbURL == "" {
		return nil, ValidationError{
			Code:   "CONFIG-002",
			Field:  "DATABASE_URL",
			Reason: "required",
		}
	}

	port, err := loadPort(l)
	if err != nil {
		return nil, err
	}

	return &Config{
		serverPort:  port,
		databaseURL: dbURL,
		jwtSecret:   jwt,
	}, nil
}

func loadPort(l Lookuper) (int, error) {
	raw, ok := l.Lookup("SERVER_PORT")
	if !ok || raw == "" {
		return defaultServerPort, nil
	}

	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, ValidationError{
			Code:   "CONFIG-003",
			Field:  "SERVER_PORT",
			Reason: fmt.Sprintf("must be integer, got %q", raw),
		}
	}

	if port < 1 || port > 65535 {
		return 0, ValidationError{
			Code:   "CONFIG-003",
			Field:  "SERVER_PORT",
			Reason: fmt.Sprintf("must be in 1..65535, got %d", port),
		}
	}

	return port, nil
}
```

**Что важно:**
- Поля `Config` приватные (по `prompts/Domain Model.txt:106-115`, `prompts/Go style.txt:32-36`). Конфиг не сериализуется и не парсится из JSON, поэтому тегов нет.
- `ValidationError` — типизированная ошибка по `prompts/Go style.txt:38-41`. `Is(target)` метод позволяет `errors.Is(err, ErrConfigInvalid)` без обвязки в `fmt.Errorf("%w", ErrConfigInvalid)` — это упрощает один уровень вложенности.
- Никаких `init()`, никаких глобальных переменных кроме `ErrConfigInvalid` — по `prompts/Go style.txt:104-108`.
- Всё тестируемо: `Load` чистая функция от `Lookuper`.
- `loadPort` приватная — логически часть `Load`, но вынесена для читаемости. Возвращает `(int, error)` — стандарт.

**Соответствие дизайну:**
- `../01-architecture.md` раздел «Компоненты `config/`» — структура соответствует диаграмме.
- `../02-behavior.md` UC1 Error cases — все CONFIG-001/002/003 покрыты.

### `config/config_test.go`

**Назначение:** 6 тестов из `../04-testing.md`, table-driven где уместно.

**Детали реализации:**

```go
package config_test

import (
	"errors"
	"testing"

	"github.com/dovgalb/project-rupor/config"
)

// mapLookuper — фейковая реализация config.Lookuper для тестов.
// Возвращает (value, true) для существующих ключей, ("", false) иначе.
// Не использует os.Setenv — тесты детерминированы и параллелизуемы.
type mapLookuper map[string]string

func (m mapLookuper) Lookup(key string) (string, bool) {
	v, ok := m[key]
	return v, ok
}

func validEnv() mapLookuper {
	return mapLookuper{
		"JWT_SECRET":   "super-secret",
		"DATABASE_URL": "postgres://user:pass@localhost:5432/db?sslmode=disable",
		"SERVER_PORT":  "8080",
	}
}

func TestConfig_Load_Valid(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load(validEnv())
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if cfg.ServerPort() != 8080 {
		t.Fatalf("ServerPort = %d, want 8080", cfg.ServerPort())
	}
	if cfg.DatabaseURL() != "postgres://user:pass@localhost:5432/db?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL())
	}
	if cfg.JWTSecret() != "super-secret" {
		t.Fatalf("JWTSecret = %q", cfg.JWTSecret())
	}
}

func TestConfig_Load_DefaultPort(t *testing.T) {
	t.Parallel()

	env := validEnv()
	delete(env, "SERVER_PORT")

	cfg, err := config.Load(env)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if cfg.ServerPort() != 8080 {
		t.Fatalf("ServerPort = %d, want default 8080", cfg.ServerPort())
	}
}

func TestConfig_Load_MissingJWTSecret(t *testing.T) {
	t.Parallel()

	env := validEnv()
	delete(env, "JWT_SECRET")

	_, err := config.Load(env)
	if !errors.Is(err, config.ErrConfigInvalid) {
		t.Fatalf("err is not ErrConfigInvalid: %v", err)
	}
	var verr config.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err is not ValidationError: %v", err)
	}
	if verr.Code != "CONFIG-001" {
		t.Fatalf("Code = %q, want CONFIG-001", verr.Code)
	}
	if verr.Field != "JWT_SECRET" {
		t.Fatalf("Field = %q, want JWT_SECRET", verr.Field)
	}
}

func TestConfig_Load_MissingDatabaseURL(t *testing.T) {
	t.Parallel()

	env := validEnv()
	delete(env, "DATABASE_URL")

	_, err := config.Load(env)
	var verr config.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err is not ValidationError: %v", err)
	}
	if verr.Code != "CONFIG-002" {
		t.Fatalf("Code = %q, want CONFIG-002", verr.Code)
	}
}

func TestConfig_Load_InvalidPortFormat(t *testing.T) {
	t.Parallel()

	env := validEnv()
	env["SERVER_PORT"] = "not-a-number"

	_, err := config.Load(env)
	var verr config.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("err is not ValidationError: %v", err)
	}
	if verr.Code != "CONFIG-003" {
		t.Fatalf("Code = %q, want CONFIG-003", verr.Code)
	}
	if verr.Field != "SERVER_PORT" {
		t.Fatalf("Field = %q", verr.Field)
	}
}

func TestConfig_Load_PortOutOfRange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		port string
	}{
		{"ноль", "0"},
		{"отрицательный", "-1"},
		{"сразу выше диапазона", "65536"},
		{"сильно выше", "100000"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			env := validEnv()
			env["SERVER_PORT"] = tc.port

			_, err := config.Load(env)
			var verr config.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("port %q: err is not ValidationError: %v", tc.port, err)
			}
			if verr.Code != "CONFIG-003" {
				t.Fatalf("port %q: Code = %q, want CONFIG-003", tc.port, verr.Code)
			}
		})
	}
}
```

**Что важно:**
- Пакет тестов — `config_test` (чёрный ящик), по `prompts/Domain model test.txt:65-67` и `prompts/Tests Style.txt:117`. Импортируем через путь модуля.
- `t.Parallel()` в каждом тесте и подтесте (`prompts/Tests Style.txt:28-30`).
- `tc := tc` в цикле перед `t.Run` — `prompts/Domain model test.txt:118-122`. В Go 1.22+ `for range` это поправлено, но `tc := tc` лишним не будет — паттерн привычен и совместим с табличными циклами по индексу.
- AAA-структура: arrange (env), act (Load), assert. Для лаконичности в простых тестах все три могут быть подряд.
- `errors.Is(err, config.ErrConfigInvalid)` и `errors.As(err, &verr)` — стандартные пути.
- Никаких моков-генераторов: `mapLookuper` — фейк руками (`prompts/Tests Style.txt:75-100`).
- Никаких `os.Setenv` — детерминизм (`prompts/Tests Style.txt:204-211`).

## Файлы для модификации

В этой фазе модификаций существующих файлов нет.

## Ключевые решения

- **ADR-004** (env без сторонних библиотек) — `os.LookupEnv` через интерфейс. Реализуется здесь.
- **Соответствие `prompts/Domain Model.txt`** — даже для конфигурационной структуры применяем приватные поля + геттеры. Это последовательность стиля по всему проекту.
- **Sentinel + типизированная ошибка** — `errors.Is(err, ErrConfigInvalid)` для категории, `errors.As(err, &ValidationError)` для деталей. Стандарт `prompts/Go style.txt:39-41`.

См. `../03-decisions.md` для полного контекста.

## Verification

- [ ] `go build ./config/...` — exit 0
- [ ] `go test ./config/... -race -count=1 -v` — все 6 тестов зелёные (с подтестами в `TestConfig_Load_PortOutOfRange` — итого 9 пройденных проверок: 5 + 4 подтеста)
- [ ] `golangci-lint run ./config/...` — 0 issues
- [ ] `gofmt -l config/` — пустой вывод
- [ ] `goimports -l -local github.com/dovgalb/project-rupor config/` — пустой вывод
- [ ] Поля `Config` приватные: `grep -E "^\tserverPort|^\tdatabaseURL|^\tjwtSecret" config/config.go | wc -l` == 3
- [ ] `Config` имеет геттеры `ServerPort()`, `DatabaseURL()`, `JWTSecret()`: `grep -c "^func (c \*Config)" config/config.go` ≥ 3
- [ ] `config.go` не импортирует ничего из `internal/` или `cmd/`: `go list -f '{{.Imports}}' ./config` содержит только stdlib
- [ ] Покрытие: `go test ./config -cover` ≥ 90% (на 6 тестах должно быть 100% строк)
- [ ] Каждый код CONFIG-001/002/003 проверяется отдельным тестом (см. coverage mapping в `../04-testing.md`)

## Что НЕ делает эта фаза

- Не использует `os.Setenv` в тестах — пакет тестируется через инжектируемый `Lookuper`.
- Не создаёт `cmd/server/main.go` — это Phase 4.
- Не валидирует формат `DATABASE_URL` (что это PostgreSQL connection string) — упрощение по принципу «явная ошибка от драйвера БД достаточна на этапе подключения». Расширение в CONFIG-004 — будущая работа.
- Не валидирует длину `JWT_SECRET` (минимум N символов для криптостойкости) — расширение в CONFIG-005 — будущая работа.
- Не загружает из `.env`-файла — `direnv`/`source` или ENV-инжекция в системе. ADR-013.
