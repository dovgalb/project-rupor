---
parent: ./README.md
view: quality
---

# 04 — Testing (Quality View)

## Особенность фазы

В этой фиче нет доменных сущностей, нет use case'ов, нет репозиториев. Полноценная пирамида тестов из `prompts/Tests Style.txt:13-21` пока неприменима. Тестируются только два компонента: **загрузчик конфигурации** (`config/`) и **главная функция и health-handler** (`cmd/server/`). Плюс — архитектурный smoke на корректность раскладки папок и направления импортов, который будет «расти вместе» с проектом.

## Coverage Mapping

| Компонент | Сценарий | Код ошибки | Тест |
|---|---|---|---|
| `config.Load` | Все три переменные заданы корректно | — | `TestConfig_Load_Valid` |
| `config.Load` | `JWT_SECRET` пустой/отсутствует | CONFIG-001 | `TestConfig_Load_MissingJWTSecret` |
| `config.Load` | `DATABASE_URL` пустой/отсутствует | CONFIG-002 | `TestConfig_Load_MissingDatabaseURL` |
| `config.Load` | `SERVER_PORT` не парсится в int | CONFIG-003 | `TestConfig_Load_InvalidPortFormat` |
| `config.Load` | `SERVER_PORT` < 1 или > 65535 | CONFIG-003 | `TestConfig_Load_PortOutOfRange` |
| `config.Load` | `SERVER_PORT` отсутствует — берётся default 8080 | — | `TestConfig_Load_DefaultPort` |
| `cmd/server` `healthHandler` | `GET /api/v1/health` возвращает 200 + корректный JSON | — | `TestHealthHandler_Returns200` |
| `cmd/server` `gracefulShutdown` | По SIGINT сервер завершается за ≤5с | — | `TestServer_GracefulShutdown` (smoke, опционально) |

## Модуль `config/` — Test Cases

### `config.Config` + `config.Load` (6 тестов)

| Тест | Что проверяет |
|---|---|
| `TestConfig_Load_Valid` | Happy path: `JWT_SECRET=secret`, `DATABASE_URL=postgres://...`, `SERVER_PORT=8080` → возвращает `*Config{ServerPort:8080, DatabaseURL:"postgres://...", JWTSecret:"secret"}`, `nil` ошибка |
| `TestConfig_Load_DefaultPort` | `SERVER_PORT` не задан, остальные заданы → `cfg.ServerPort == 8080`, `nil` ошибка |
| `TestConfig_Load_MissingJWTSecret` | `JWT_SECRET` отсутствует → возвращает `nil, ValidationError{Field:"JWT_SECRET", Reason:"required"}`, `errors.Is(err, ErrConfigInvalid)` == true |
| `TestConfig_Load_MissingDatabaseURL` | `DATABASE_URL` отсутствует → возвращает `nil, ValidationError{Field:"DATABASE_URL", Reason:"required"}` |
| `TestConfig_Load_InvalidPortFormat` | `SERVER_PORT="abc"` → возвращает `ValidationError{Field:"SERVER_PORT", Reason:"must be integer"}` |
| `TestConfig_Load_PortOutOfRange` | Table-driven по `0`, `-1`, `65536`, `100000` → каждый возвращает `ValidationError{Field:"SERVER_PORT", Reason:"must be in 1..65535"}` |

### Stubs / Mocks

- **`mapLookuper map[string]string`** — реализация `config.Lookuper`, возвращает `(value, true)` если ключ есть в мапе, `("", false)` иначе. Живёт в `config/config_test.go` рядом с тестами. Никаких `os.Setenv` — `prompts/Tests Style.txt:204-211` запрещает зависимость тестов от глобального env.

Никаких других моков в этой фазе нет — `config.Load` чистая функция от `Lookuper`.

## Модуль `cmd/server/` — Test Cases

### `healthHandler` (1 тест)

| Тест | Что проверяет |
|---|---|
| `TestHealthHandler_Returns200` | `httptest.NewRecorder()` + вызов `healthHandler(w, r)` → `w.Code == 200`, `w.Body == {"status":"ok"}\n`, `Content-Type == "application/json"` |

### `gracefulShutdown` (опциональный smoke-тест, 1 тест)

| Тест | Что проверяет |
|---|---|
| `TestServer_GracefulShutdown` | Запустить `runServer(ctx, cfg)` в горутине → отправить активный запрос (`time.Sleep`-handler) → отменить ctx → `runServer` возвращается за ≤5с, активный запрос дотягивает ответ |

Этот тест помечен `t.Skip("integration smoke")` по умолчанию или живёт под build-tag `//go:build integration`. На CI запускается отдельным шагом `go test -tags=integration ./cmd/...`. Альтернатива — оставить в плане как «factor out at Phase 1.2 если устаканится».

### Stubs / Mocks

- В тестах `cmd/server/` моков нет — это compositional root. Тестируется только handler-уровень через `httptest`.

## Архитектурный smoke

В этой фиче ни одного импорта между слоями нет — нечего нарушать. Однако чтобы зафиксировать гарантии на будущее, добавляется один из двух механизмов:

**Вариант A (выбран):** один `arch_test.go` в корне или в `cmd/server/`, использующий `golang.org/x/tools/go/packages` для проверки правил импорта:
- `internal/<домен>/domain/` импортирует только stdlib (whitelist)
- `internal/<домен>/usecase/` импортирует только stdlib + `internal/<домен>/domain/`
- `internal/<домен>/{transport,repository}/` не импортирует другой `internal/<X>/{transport,repository}/`

**Вариант B (отвергнут):** настраивать линтер `depguard`/`gomod-check`. Сложнее, требует кастомной конфигурации.

В фазе 1.1 этот тест помечен `t.Skip("no internal packages yet")` и активируется в Фазе 1.2, когда появится первый код в `internal/auth/`. Решение по варианту A зафиксировано в плане кода (Phase 1 — задел инфраструктуры, реальная активация в Фазе 1.2).

| Тест | Что проверяет |
|---|---|
| `TestArchitecture_DomainImports` | Никакой `.go`-файл в `internal/<X>/domain/` не импортирует пакеты вне whitelist. На фазе 1.1 — `t.Skip("no domain code yet")`. |
| `TestArchitecture_UseCaseImports` | `internal/<X>/usecase/` импортирует только `internal/<X>/domain/` и stdlib. На фазе 1.1 — `t.Skip`. |
| `TestArchitecture_NoCrossDomainTransport` | Нет `internal/<X>/transport/...` импортирующего `internal/<Y>/...` где X != Y. На фазе 1.1 — `t.Skip`. |

## Repo Model Round-Trip Tests

Не применимо к фазе 1.1 — нет доменных сущностей, нет repo моделей. Раздел появится с первым доменом.

## Integration Tests (репозитории)

Не применимо к фазе 1.1 — нет репозиториев. Раздел появится с первым доменом.

## Smoke на инфраструктуру

Не юнит-тесты, а ручные / CI-скрипты, обозначенные в `08-api-contract.md` Test Plan:

| Шаг | Команда | Ожидание |
|---|---|---|
| Сборка | `go build ./...` | Exit 0, бинарник создаётся |
| Линт | `golangci-lint run` | Exit 0, 0 issues |
| Юнит | `go test ./... -race -count=1` | Все тесты `config_test.go` и `health_test.go` зелёные |
| Compose up | `docker compose up -d postgres` | Контейнер `Up`, healthcheck `healthy` за ≤30с |
| Migrate | `make migrate-up` | Exit 0, в БД появляется `schema_migrations` с `version=1, dirty=false` |
| Migrate down | `make migrate-down` | Exit 0, `schema_migrations.version=NULL` или строка удалена |
| Run + curl | `make run &; curl localhost:$PORT/api/v1/health; kill %1` | `200 {"status":"ok"}`, сервер завершается за ≤5с |

CI выполняет первые три шага автоматически. Остальные — manual QA, документируются в `manual_qa/project-structure/test-flow.md` после реализации.

## Test Count Summary

| Модуль | Entity | Builder | UseCase | VO | Repo Model | Integration | Total |
|--------|--------|---------|---------|-----|------------|-------------|-------|
| `config/` | 0 | 0 | 0 | 0 | 0 | 0 | 6 |
| `cmd/server/` | 0 | 0 | 0 | 0 | 0 | 1 (`healthHandler`) | 1–2 (graceful shutdown опционален) |
| `internal/<домены>/` | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| Архитектурный smoke | — | — | — | — | — | 0 (всё `t.Skip`) | 3 (skip-каркас) |
| **Итого** | 0 | 0 | 0 | 0 | 0 | 1 | **10–11** |

Цель — не «много тестов», а правильный каркас: `config/` строго протестирован (он будет читаться всеми будущими фичами), `healthHandler` зафиксирован контрактом, архитектурные правила записаны в `arch_test.go` и активируются с первым доменом.
