---
parent: ./README.md
---

# 07 — Standards Compliance

Соответствие дизайна и предполагаемой реализации каждому из восьми стандартов в `prompts/`. Статус `[OK]` — стандарт применим и соблюдается; `[N/A]` — стандарт не применим к этой фазе (например, нет доменных сущностей); `[WARN]` — есть оговорка/отклонение, описанное в «Уточнения».

| Стандарт | Статус | Ключевые точки compliance |
|----------|--------|---------------------------|
| `Architecture Layers.txt` | [OK] | Раскладка `internal/<домен>/{domain,usecase,transport/http,repository/postgres}/` точно по `prompts/Architecture Layers.txt:46-58`. Терминология `usecase/` (не `service/`) — `prompts/Architecture Layers.txt:120`. Граф зависимостей в `01-architecture.md` направлен внутрь по `prompts/Architecture Layers.txt:67-74`. Composition root в `cmd/server/main.go` по `prompts/Architecture Layers.txt:62-66`. |
| `Builder.txt` | [N/A] | В фазе 1.1 нет доменных сущностей — Builder применять негде. Стандарт активируется с Фазы 1.2 (`auth.User`). Отклонений нет. |
| `Clean architecture.txt` | [OK] | `cmd/server/main.go` будет единственной точкой склейки зависимостей по `prompts/Clean architecture.txt:30-38`. Правило зависимостей внутрь — выполнено (граф зависимостей в `01-architecture.md`). Никаких импортов `domain ← infrastructure` нет, потому что доменов ещё нет. |
| `Domain Model.txt` | [N/A] | Нет доменных сущностей. Все требования (приватные поля, инварианты, value objects, конструкторы `New`/`Reconstruct`) активируются с первой сущностью в Фазе 1.2. Отклонений нет. |
| `Domain model test.txt` | [N/A] | Нет доменных тестов в этой фазе. |
| `Go style.txt` | [OK] | Go 1.25 (`go.mod:3`) — соответствует требованию `Go 1.22+` (`prompts/Go style.txt:7`). `gofmt`/`golangci-lint` обязательны (`prompts/Go style.txt:8-9`) — Makefile target `lint` и CI gate. Конфигурация только через env (`prompts/Go style.txt:84-87`) — ADR-004. Нет ORM (`prompts/Go style.txt:97-101`) — ADR-005. Логирование через `log/slog` (`prompts/Go style.txt:91`) — `cmd/server/main.go`. `panic` запрещён в продовом коде, кроме composition root (`prompts/Go style.txt:44`) — `main()` может `panic` при невозможности стартовать (например, на не-валидной конфигурации, хотя предпочтительный путь — `os.Exit(1)` после `log.Error`). |
| `RepoModel.txt` | [N/A] | Нет репозиториев и доменных моделей в этой фазе. Стандарт активируется с Фазы 1.2 (репозиторий `auth.UserRepository`). |
| `Tests Style.txt` | [OK] | Стандартная библиотека `testing` (`prompts/Tests Style.txt:24-26`) — без testify. `t.Parallel()` в каждом тесте (`prompts/Tests Style.txt:28-30`). Table-driven для `config.Load` (`prompts/Tests Style.txt:32-49`). Детерминизм: `Lookuper` инжектируется, нет зависимости от `os.Setenv` (`prompts/Tests Style.txt:204-211`). AAA-структура с пустыми строками (`prompts/Tests Style.txt:170-202`). |

## Уточнения

### Расхождение 1: `panic` vs `log.Fatal` в `main()`

`prompts/Go style.txt:44` разрешает `panic` в composition root. `prompts/Go style.txt:39` запрещает паники в продовом коде. `cmd/server/main.go` — это composition root, но привычная Go-практика — `log.Fatalf` или `slog.Error + os.Exit(1)`. В дизайне выбран **`slog.Error + os.Exit(1)`** для возможности структурированного логирования (через `slog.Logger`, иначе `log.Fatalf` идёт в дефолтный `log` без структуры). Это не противоречит стандарту — `panic` разрешён, но не обязателен.

### Расхождение 2: пустые папки и `.gitkeep`

`prompts/Go style.txt` не упоминает пустые папки. Создание 24 папок-заглушек с `.gitkeep` — это product-decision (ADR-001), а не отклонение от стандарта.

### Расхождение 3: `archive/arch_test.go` под `t.Skip`

`prompts/Tests Style.txt:226-228` запрещает `t.Skip(...)` без обоснования. Все архитектурные тесты в этой фазе помечены `t.Skip("no internal packages yet — activated in Phase 1.2")` — обоснование в комментарии, в плане кода фиксируется задача снять `t.Skip` при добавлении первой сущности в `internal/`.

### Расхождение 4: `health_test.go` для `cmd/server/`

`prompts/Tests Style.txt:120-129` рекомендует HTTP-тесты как интеграционные через `httptest.NewServer` поверх собранного роутера. В фазе 1.1 health-handler — это одна функция без use case, поэтому интеграционный путь избыточен. Используется `httptest.NewRecorder` + прямой вызов `healthHandler(w, r)` (стандартный Go-паттерн). Это не противоречит `prompts/Tests Style.txt`, но является адаптацией к минимальности фазы.

### Расхождение 5: graceful-shutdown тест опциональный

`prompts/Tests Style.txt:204-208` запрещает `time.Sleep` без причины и сетевые тесты в обычном прогоне. Тест `TestServer_GracefulShutdown` либо помечается `//go:build integration`, либо `t.Skip` по умолчанию. Запускается отдельным шагом в CI или manual. Альтернатива — не писать его вовсе и проверить graceful-shutdown в manual QA. Решение откладывается до этапа плана.

## Pre-flight чек перед коммитом (по `prompts/Go style.txt:119-126`)

Скелет должен проходить эти команды без правок:

1. `gofmt -l .` — пустой вывод
2. `go vet ./...` — exit 0
3. `golangci-lint run` — exit 0, 0 issues
4. `go test ./... -race -count=1` — все 10–11 тестов зелёные
5. `go build ./...` — exit 0
6. `git diff --check` — нет whitespace-ошибок

CI workflow `.github/workflows/ci.yml` выполняет шаги 1, 3, 4, 5 автоматически. Шаги 2 и 6 — локально перед коммитом (можно вынести в pre-commit hook отдельной фичей).
