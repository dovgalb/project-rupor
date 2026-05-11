---
phase: 4
name: config CORS_ALLOWED_ORIGINS
layer: infra
depends_on: none
plan: ./README.md
---

# Phase 4: `config/` — `CORS_ALLOWED_ORIGINS`

## Цель

Расширить `config.Config` env-переменной `CORS_ALLOWED_ORIGINS` (CSV), добавить геттер `CORSAllowedOrigins() []string`, валидацию каждого origin как URL и код ошибки `CONFIG-006`.

## Контекст

Существующий `config/config.go` (`config/config.go:1-211`) уже содержит:
- Структуру `Config` с приватными полями (`config/config.go:36-42`).
- Функции `loadPort` / `loadAccessTTL` / `loadRefreshTTL` (`config/config.go:103-210`) — паттерн, по которому добавляется `loadCORSAllowedOrigins`.
- `ValidationError{Code, Field, Reason}` + `ErrConfigInvalid` (`config/config.go:20-34`).
- Существующие коды: `CONFIG-001..CONFIG-005`. Следующий свободный — `CONFIG-006`.

Эта фаза **технически независима** от фаз 1–3: меняется только `config/`. Может выполняться параллельно. В линейном порядке делается перед фазой 5 (там используется `cfg.CORSAllowedOrigins()`).

## Файлы для модификации

### `config/config.go`

**Что меняется:**

1. Добавить константу default:
   ```go
   const defaultCORSAllowedOrigin = "http://localhost:5173"
   ```
   Рядом с `defaultServerPort` (`config/config.go:11-18`).

2. Добавить поле в `Config`:
   ```go
   type Config struct {
       serverPort         int
       databaseURL        string
       jwtSecret          string
       jwtAccessTTL       time.Duration
       jwtRefreshTTL      time.Duration
       corsAllowedOrigins []string  // НОВОЕ
   }
   ```
   `config/config.go:36-42`.

3. Добавить геттер:
   ```go
   func (c *Config) CORSAllowedOrigins() []string {
       // Возвращаем копию, защита от мутации извне.
       cp := make([]string, len(c.corsAllowedOrigins))
       copy(cp, c.corsAllowedOrigins)
       return cp
   }
   ```
   После `JWTRefreshTTL()` (`config/config.go:48`).

4. В функции `Load` добавить шаг:
   ```go
   func Load(l Lookuper) (*Config, error) {
       // ... существующие шаги (jwt, dbURL, port, accessTTL, refreshTTL) ...

       corsOrigins, err := loadCORSAllowedOrigins(l)
       if err != nil {
           return nil, err
       }

       return &Config{
           serverPort:         port,
           databaseURL:        dbURL,
           jwtSecret:          jwt,
           jwtAccessTTL:       accessTTL,
           jwtRefreshTTL:      refreshTTL,
           corsAllowedOrigins: corsOrigins,  // НОВОЕ
       }, nil
   }
   ```
   После строки `config/config.go:92` (после `loadRefreshTTL`).

5. Добавить функцию `loadCORSAllowedOrigins`:
   ```go
   func loadCORSAllowedOrigins(l Lookuper) ([]string, error) {
       raw, ok := l.Lookup("CORS_ALLOWED_ORIGINS")
       if !ok || strings.TrimSpace(raw) == "" {
           return []string{defaultCORSAllowedOrigin}, nil
       }

       parts := strings.Split(raw, ",")
       origins := make([]string, 0, len(parts))
       for _, p := range parts {
           p = strings.TrimSpace(p)
           if p == "" {
               continue
           }
           if err := validateOrigin(p); err != nil {
               return nil, ValidationError{
                   Code:   "CONFIG-006",
                   Field:  "CORS_ALLOWED_ORIGINS",
                   Reason: fmt.Sprintf("invalid origin %q: %s", p, err.Error()),
               }
           }
           origins = append(origins, p)
       }

       if len(origins) == 0 {
           return nil, ValidationError{
               Code:   "CONFIG-006",
               Field:  "CORS_ALLOWED_ORIGINS",
               Reason: "must contain at least one origin",
           }
       }

       return origins, nil
   }

   func validateOrigin(s string) error {
       u, err := url.Parse(s)
       if err != nil {
           return err
       }
       if u.Scheme != "http" && u.Scheme != "https" {
           return fmt.Errorf("scheme must be http or https, got %q", u.Scheme)
       }
       if u.Host == "" {
           return fmt.Errorf("host is empty")
       }
       if u.Path != "" && u.Path != "/" {
           return fmt.Errorf("must not contain path, got %q", u.Path)
       }
       if u.RawQuery != "" || u.Fragment != "" {
           return fmt.Errorf("must not contain query or fragment")
       }
       return nil
   }
   ```
   В конец файла, после `loadRefreshTTL` (после `config/config.go:210`).

6. Добавить импорты:
   - `net/url` (для `url.Parse`).
   - `strings` (для `strings.Split`, `strings.TrimSpace`).

### `config/config_test.go`

**Что меняется:**

Добавить 4 теста (`../04-testing.md:230-236`). Стиль — table-driven там, где много кейсов; обычные `t.Run` где один сценарий.

**Helper в файле тестов** (если ещё не существует):
```go
type mapLookuper map[string]string

func (m mapLookuper) Lookup(key string) (string, bool) {
    v, ok := m[key]
    return v, ok
}
```

(Если уже есть в `config_test.go` фазы 1.3 — переиспользуем; если нет — добавляем рядом с тестом.)

**Минимальный валидный конфиг для тестов** — функция `validBaseEnv()`:
```go
func validBaseEnv() mapLookuper {
    return mapLookuper{
        "JWT_SECRET":   "test-secret",
        "DATABASE_URL": "postgres://localhost/test",
    }
}
```
(Используется в существующих тестах. Если ещё нет — добавляем.)

**Tests:**

1. `TestConfig_Load_DefaultCORS`:
   ```go
   func TestConfig_Load_DefaultCORS(t *testing.T) {
       t.Parallel()
       env := validBaseEnv()
       cfg, err := config.Load(env)
       if err != nil { t.Fatalf("Load: %v", err) }
       got := cfg.CORSAllowedOrigins()
       want := []string{"http://localhost:5173"}
       if !reflect.DeepEqual(got, want) {
           t.Fatalf("CORSAllowedOrigins() = %v, want %v", got, want)
       }
   }
   ```

2. `TestConfig_Load_CSVOrigins`:
   ```go
   func TestConfig_Load_CSVOrigins(t *testing.T) {
       t.Parallel()
       cases := []struct {
           name string
           env  string
           want []string
       }{
           {"single", "http://a.com", []string{"http://a.com"}},
           {"two", "http://a.com,https://b.com:8080", []string{"http://a.com", "https://b.com:8080"}},
           {"with_spaces", "http://a.com , https://b.com", []string{"http://a.com", "https://b.com"}},
           {"trailing_comma", "http://a.com,", []string{"http://a.com"}},
       }
       for _, tc := range cases {
           tc := tc
           t.Run(tc.name, func(t *testing.T) {
               t.Parallel()
               env := validBaseEnv()
               env["CORS_ALLOWED_ORIGINS"] = tc.env
               cfg, err := config.Load(env)
               if err != nil { t.Fatalf("Load: %v", err) }
               got := cfg.CORSAllowedOrigins()
               if !reflect.DeepEqual(got, tc.want) {
                   t.Fatalf("got %v, want %v", got, tc.want)
               }
           })
       }
   }
   ```

3. `TestConfig_Load_EmptyCORS`:
   ```go
   func TestConfig_Load_EmptyCORS(t *testing.T) {
       t.Parallel()
       cases := []struct{ name, env string }{
           {"only_comma", ","},
           {"only_spaces", "   "},
           {"only_commas_and_spaces", " , , "},
       }
       for _, tc := range cases {
           tc := tc
           t.Run(tc.name, func(t *testing.T) {
               t.Parallel()
               env := validBaseEnv()
               env["CORS_ALLOWED_ORIGINS"] = tc.env
               _, err := config.Load(env)
               var verr config.ValidationError
               if !errors.As(err, &verr) {
                   t.Fatalf("err = %v, want ValidationError", err)
               }
               if verr.Code != "CONFIG-006" {
                   t.Fatalf("Code = %q, want CONFIG-006", verr.Code)
               }
               if verr.Field != "CORS_ALLOWED_ORIGINS" {
                   t.Fatalf("Field = %q", verr.Field)
               }
           })
       }
   }
   ```
   Кейс `""` (пустая строка) проверяется отдельно через **отсутствие** `CORS_ALLOWED_ORIGINS` в env — это идёт по ветке default. Если пользователь явно установил `CORS_ALLOWED_ORIGINS=""`, `mapLookuper` вернёт `("", true)`, наша логика `if !ok || strings.TrimSpace(raw) == ""` выберет default. Это корректное поведение — пустая строка эквивалентна отсутствию переменной.

4. `TestConfig_Load_InvalidOrigin`:
   ```go
   func TestConfig_Load_InvalidOrigin(t *testing.T) {
       t.Parallel()
       cases := []struct{ name, value string }{
           {"no_scheme", "localhost:5173"},
           {"missing_colon", "https//missing-colon.com"},
           {"with_path", "http://x.com/path"},
           {"with_query", "http://x.com?foo=bar"},
           {"ftp_scheme", "ftp://x.com"},
           {"empty_host", "http://"},
           {"mixed_valid_invalid", "http://x.com,not-a-url"},
       }
       for _, tc := range cases {
           tc := tc
           t.Run(tc.name, func(t *testing.T) {
               t.Parallel()
               env := validBaseEnv()
               env["CORS_ALLOWED_ORIGINS"] = tc.value
               _, err := config.Load(env)
               var verr config.ValidationError
               if !errors.As(err, &verr) {
                   t.Fatalf("err = %v, want ValidationError", err)
               }
               if verr.Code != "CONFIG-006" {
                   t.Fatalf("Code = %q, want CONFIG-006", verr.Code)
               }
           })
       }
   }
   ```

### `.env`

**Что меняется:** добавить строку:
```
CORS_ALLOWED_ORIGINS=http://localhost:5173
```
В конце файла. Это синхронизирует локальный `.env` с дефолтом и делает его явным для разработчика.

### `.env.example`

**Что меняется:** добавить строку:
```
CORS_ALLOWED_ORIGINS=http://localhost:5173
```
В конце файла. Документирует переменную для других разработчиков.

## Файлы для создания

Никаких. Это расширение существующих файлов.

## Ключевые решения

- **Default `http://localhost:5173`** (`../03-decisions.md`, ADR-004) — Vite dev-port. В `.env` явно прописываем для документирования; код использует этот же default при отсутствии переменной.
- **Wildcard `*` НЕ поддерживается на уровне валидации** — `validateOrigin("*")` упадёт на `url.Parse` (точнее, парсер вернёт `Path="*"`, и проверка `Path != ""` отвергнет). Это закрепляет ADR-004 на уровне config.
- **Геттер возвращает копию слайса** — защита от мутации через `cfg.CORSAllowedOrigins() = nil`. Совпадает с паттерном защитного программирования стандартной либы.
- **Empty string эквивалентна отсутствию переменной** — выбираем default, не падаем. Это удобно: разработчик может закомментировать переменную в `.env`, не получив `CONFIG-006`.
- **Валидация строгая**: только `http`/`https`, без path/query/fragment, обязателен host. Защищает от копипаст-ошибок.

## Verification

- [ ] `go build ./config/...` проходит.
- [ ] `go test ./config/...` проходит — все существующие тесты + 4 новых.
- [ ] `gofmt -l config/` — пустой вывод.
- [ ] `go vet ./config/...` — exit 0.
- [ ] Phase-specific check: `cfg.CORSAllowedOrigins()` возвращает `[]string{"http://localhost:5173"}` при отсутствии env.
- [ ] Phase-specific check: `cfg.CORSAllowedOrigins()` возвращает копию, мутация которой не влияет на следующие вызовы.
- [ ] Phase-specific check: `errors.Is(err, config.ErrConfigInvalid)` работает для `CONFIG-006`.
- [ ] `.env` и `.env.example` содержат `CORS_ALLOWED_ORIGINS=http://localhost:5173`.
- [ ] `git diff --name-only` ограничен `config/config.go`, `config/config_test.go`, `.env`, `.env.example`.
