---
parent: ./README.md
view: standards
---

# 07 — Standards Compliance

Соответствие дизайна каждому из восьми стандартов в `prompts/`. Статус ✅ — стандарт применим и соблюдается; ⚠️ — есть оговорка/расхождение, описанное в «Уточнения».

| Стандарт | Статус | Ключевые точки compliance |
|----------|--------|---------------------------|
| `Architecture Layers.txt` | ✅ | Раскладка: cross-cutting middleware → `pkg/httpx/middleware/` (`prompts/Architecture Layers.txt:60-65` — infrastructure / pkg); auth-specific middleware → `internal/auth/transport/http/middleware/` (`prompts/Architecture Layers.txt:46-58` — interface adapters). Граф зависимостей внутрь сохранён (`01-architecture.md`, граф зависимостей модулей): `pkg/httpx` не импортирует `internal/...`; `internal/auth/transport/http/middleware/` импортирует только `usecase`, `domain` и `pkg/httpx`. Composition root в `cmd/server/main.go` (`prompts/Architecture Layers.txt:62-66`). См. ⚠️ Расхождение 1. |
| `Builder.txt` | ✅ N/A | Builder pattern не применяется. У middleware нет сущностей с многими полями; параметры middleware-фабрик передаются явно через сигнатуру. (`prompts/Builder.txt:14-19`: «у структуры 1–3 поля. Проще передать их прямо в конструктор» — у нас 1–2 поля). |
| `Clean architecture.txt` | ✅ | `cmd/server/main.go` — точка склейки middleware-stack (`prompts/Clean architecture.txt:30-38`). Middleware зависят от интерфейсов в `usecase/` (`TokenIssuer`, `Clock`), а не от конкретных адаптеров (`prompts/Clean architecture.txt:29-32`). Cross-cutting примитивы (Recover, Logger, CORS, RequestID) не зависят от auth-домена и могут быть переиспользованы в фазе 2/3. JWT-валидация — деталь, живёт в `internal/auth/repository/jwt/`, не в middleware (`prompts/Clean architecture.txt:51-53`). |
| `Domain Model.txt` | ✅ N/A | Middleware не вводит новых доменных сущностей. `domain.UserID` уже существует с фазы 1.3 (rich-VO с приватным полем и конструктором). Middleware только использует `domain.UserID` для проброса в контекст. |
| `Domain model test.txt` | ✅ N/A | Domain-тестов в этой фиче не вводится. Сущности 1.3 не меняются. |
| `Go style.txt` | ✅ | Go 1.25 (`go.mod:3`). `gofmt`/`goimports` через `make lint` (`Makefile:38-43`). Конфигурация `CORS_ALLOWED_ORIGINS` через env (`prompts/Go style.txt:84-87`, ADR-004). Логирование через `log/slog` (`prompts/Go style.txt:91`). Без логирования PII/токенов (`prompts/Go style.txt:94`, ADR-012). Контекст первым параметром (`prompts/Go style.txt:47-52`). Ранние возвраты в middleware (`prompts/Go style.txt:54-55`). Интерфейсы в потребителе (`Logger` принимает hook-функцию, не интерфейс — оптимально для одного метода). Без новых зависимостей (`prompts/Go style.txt:111-117`, ADR-001). См. ⚠️ Расхождение 2 про `panic`/`recover`. |
| `RepoModel.txt` | ✅ N/A | Middleware не имеет своих репозиториев и SQL-запросов. |
| `Tests Style.txt` | ✅ | Стандартная библиотека `testing` (`prompts/Tests Style.txt:24-26`) — без testify/gomock. `t.Parallel()` в каждом тесте (`prompts/Tests Style.txt:28-30`). Table-driven для валидации заголовков (`prompts/Tests Style.txt:32-49`). Фейки руками (`prompts/Tests Style.txt:75-100`) — `fixedClock`, `slog`-buffer для проверки логов. HTTP-тесты middleware — через `httptest.NewRecorder()` для cross-cutting (без TCP-листенера) и через `httptest.NewServer()` для регрессии 1.3 (`prompts/Tests Style.txt:147-162`). Время и UUID инжектируются (`prompts/Tests Style.txt:55-62`). AAA-структура (`prompts/Tests Style.txt:170-202`). |

## Уточнения

### Расхождение 1: новый верхний пакет `pkg/httpx/`

`prompts/Architecture Layers.txt:60-65` упоминает `pkg/` как часть infrastructure-слоя для драйверов и адаптеров. До фазы 1.4 единственный подкаталог `pkg/` — `pkg/websocket/` (пустой `.gitkeep`).

Эта фича вводит **второй** подпакет в `pkg/`: `pkg/httpx/` (с подпакетом `middleware/`). Обоснование (`03-decisions.md`, ADR-003):
- Cross-cutting HTTP-обвязка (recover/logger/cors/requestid) не зависит от auth-домена и в фазе 2/3 будет переиспользована для `internal/room/`, `internal/channel/`, WebSocket-handshake. Естественное место — `pkg/`.
- Имя `httpx` отражает то, что пакет ПРЕДОСТАВЛЯЕТ — расширения для `net/http` (`prompts/Go style.txt:25-28`).
- `pkg/websocket/` останется пустым до фазы 3 (`general_plan.md:142-144`); добавление `pkg/httpx/` рядом — естественное расширение по тому же принципу.

Это **не отклонение** от стандарта, а корректное использование `pkg/` для cross-cutting инфраструктуры.

### Расхождение 2: использование `recover()` в `Recover`-middleware

`prompts/Go style.txt:44`: «`panic` — запрещён в продовом коде, допустим только в composition root при невозможности стартовать и в тестах». `recover()` в стандарте напрямую не упоминается, но семантически — это парная операция, которая нужна **только если кто-то паникует**.

В этой фиче `Recover`-middleware вводит **единственное место**, где `recover()` используется в продовом коде. Обоснование:
- `Recover` ловит panic от любых нижестоящих middleware/handler — это страховка для сценария, когда баг проникает в прод (например, nil pointer dereference в новом хендлере фазы 2).
- Без `Recover` panic в любом hander уронит весь HTTP-server-процесс (стандартное поведение `net/http.Server`).
- Альтернативы (best-effort error handling без recover) не работают: panic — exit-механика стека, она не может быть пойманы через `if err != nil`.
- Наш Recover **не маскирует** panic: логирует через `slog.Error` со stack trace, request_id, method, path — после этого panic становится видимым в логах с полным контекстом.

Это **корректная интерпретация** стандарта: production-handler-ы НЕ должны паниковать (правило соблюдено), но платформа должна быть устойчива к багам. `Recover`-middleware — это страховочная сетка, а не разрешение панике.

Тест `TestRecover_PassesAbortHandler` дополнительно гарантирует, что namespace-defined panic-ситуации (например, `http.ErrAbortHandler`) проходят насквозь.

### Расхождение 3: middleware принимает функции вместо интерфейсов

`prompts/Go style.txt:69-74`: «Маленькие интерфейсы лучше больших». Middleware-фабрики Recover/Logger принимают функцию-параметр (а не интерфейс с одним методом):

```go
func Recover(logger *slog.Logger) func(http.Handler) http.Handler
func Logger(logger *slog.Logger, hook func(ctx) []slog.Attr) func(http.Handler) http.Handler
func RequestID(uuidGen func() string) func(http.Handler) http.Handler
```

Альтернатива (интерфейс с одним методом) — overengineering для тривиальных операций:
- `uuidGen` — это `func() string`. Интерфейс `UUIDStringGenerator { New() string }` — лишний boilerplate.
- `hook` — это `func(ctx context.Context) []slog.Attr`. Интерфейс `LogAttrProvider { Attrs(ctx) []slog.Attr }` — то же самое.
- `slog.Logger` — это уже структура из stdlib, не интерфейс. Использование struct в качестве зависимости допустимо для stdlib-типов.

Это **компактный идиоматичный Go-код** в духе `prompts/Go style.txt:69-74` («Не делать интерфейс «на всякий случай»»).

### Расхождение 4: `Logger` middleware с auth-зависимым hook

`Logger` живёт в `pkg/httpx/middleware/` (cross-cutting), но логирует `user_id` (auth-specific) через hook.

Обоснование (см. `03-decisions.md`, OQ-1):
- Прямой импорт `Logger` → `internal/auth/transport/http/middleware/` нарушит правило слоёв (`pkg/` → `internal/` запрещено).
- Hook-функция передаётся в `Logger` извне (из `cmd/server/main.go`), и она ссылается на `authmw.UserIDFromContext`.
- В `cmd/server/main.go` это композиция:
  ```go
  userIDHook := func(ctx context.Context) []slog.Attr {
      uid, ok := authmw.UserIDFromContext(ctx)
      if !ok { return nil }
      return []slog.Attr{slog.String("user_id", uid.String())}
  }
  mux.Use(httpxmw.Logger(logger, userIDHook))
  ```
- `Logger` не знает про auth — он лишь вызывает hook и пристёгивает результат к slog-event'у.

Это **корректное соблюдение** правила «зависимость через интерфейс/функцию, инжекция в composition root».

### Расхождение 5: интеграционная природа тестов `RequireAuth`

`prompts/Tests Style.txt:75-100` рекомендует фейки для usecase-зависимостей. Тесты `RequireAuth` используют **реальный** `*jwt.TokenIssuer` (а не fake-issuer).

Обоснование:
- `RequireAuth` — это интеграция middleware с jwt-валидацией; полезность теста — проверить, что цепочка работает совместно (включая format claims, error mapping).
- Фейк `TokenIssuer` (возвращающий `domain.UserID` без реальной криптографии) тестировал бы только flow middleware, не реальную защиту.
- jwt-issuer быстрый: `IssueAccess` ~µs, `VerifyAccess` ~µs. Не замедляет test suite.
- В тесте `TestMeHandler_*` 1.3 уже используется тот же подход (`docs/1_3_auth_domen/04-testing.md`).

Это **прагматичное отклонение** в духе `prompts/Tests Style.txt:147-152` («HTTP-тесты — реальные use case'ы и реальные адаптеры, где возможно»).

## Pre-flight чек перед коммитом (по `prompts/Go style.txt:119-126`)

Реализация фазы 1.4 должна проходить эти команды без правок:

1. `gofmt -l .` — пустой вывод
2. `go vet ./...` — exit 0
3. `golangci-lint run` — exit 0, 0 issues
4. `go test ./... -race -count=1` — все unit-тесты зелёные (~65 тестов; см. `04-testing.md`)
5. `go build ./...` — exit 0
6. `git diff --check` — нет whitespace-ошибок

CI workflow (`.github/workflows/ci.yml`) выполняет шаги 1–5 как сейчас. Никаких новых build-tags не вводится.

## Список новых правил, появляющихся с 1.4 (для ревью на этапе 2 и далее)

- **Все приватные маршруты регистрируются через `chi.Group + r.Use(authmw.RequireAuth(issuer, clock))`** — не через декоратор хендлера, не через глобальный middleware с whitelist путей.
- **userID в контексте — через `authmw.WithUserID/UserIDFromContext`** — не через прямой `context.WithValue` со строковым ключом.
- **JSON-envelope ошибок генерируется через `pkg/httpx.WriteJSONError`** — единый источник истины. `error_mapper.go:writeError` тоже его использует.
- **Глобальные cross-cutting middleware регистрируются в `cmd/server/main.go`** в порядке `RequestID → Recover → Logger → CORS`. Изменения порядка — через ADR.
- **`recover()` допустим только в `pkg/httpx/middleware.Recover`** — больше нигде в продовом коде.
- **`X-Request-ID` принимается и пробрасывается в response** — клиенты могут использовать для коррелятора с upstream-логами.
- **CORS-origins — env-driven через `CORS_ALLOWED_ORIGINS`** — фиксированный whitelist; wildcard `*` запрещён.
- **`pkg/httpx/...` не импортирует `internal/...`** — закрепляется тестом `TestArchitecture_PkgHttpxImports`.
