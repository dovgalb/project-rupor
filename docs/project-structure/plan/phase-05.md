---
phase: 5
name: Doc & Cleanup
layer: docs
depends_on: [phase-04]
plan: ./README.md
---

# Phase 5: Doc & Cleanup

## Цель

Закрыть suggestions из architect review: добавить раздел quickstart в `README.MD`, удалить избыточный `init_project`. Эта фаза не вводит код, только документацию и cleanup.

## Контекст

Что произвели предыдущие фазы:
- Phase 1–4: рабочий монорепо со всем стеком.
- `make dc-up && make migrate-up && make run && curl /api/v1/health` отрабатывает end-to-end.

Текущее состояние документации:
- `README.MD:1-13` описывает только процесс работы с агентами (`/research_codebase`, `/design_feature`, `/implement_backend`).
- `init_project:1` — заметка «нужно сделать Структура проекта, Docker Compose (PostgreSQL), миграции», после реализации фичи становится избыточной.

## Файлы для модификации

### `README.MD`

**Что меняется:** добавляется новый раздел `## Запуск локально` перед существующим `## процесс запуска` (или в конец — позиция обсуждаема).

**Диапазон строк:** добавление после строки 13. Существующее содержимое (`README.MD:1-13`) не трогается.

**Содержимое нового раздела:**

```markdown
## Запуск локально

Требования:
- Go 1.25+
- Docker + Docker Compose
- `golang-migrate` CLI: `brew install golang-migrate` или `go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest`
- (опционально) `golangci-lint` для `make lint`: `brew install golangci-lint`
- (опционально) `sqlc` для `make sqlc` (понадобится при работе с БД-кодом): `brew install sqlc`

Шаги:

```bash
# 1. Скопировать шаблон env
cp .env.example .env

# 2. Поднять PostgreSQL в Docker
make dc-up

# 3. Накатить миграции
make migrate-up

# 4. Запустить сервер
make run

# В другом терминале:
curl http://localhost:8080/api/v1/health
# {"status":"ok"}
```

Для остановки: `Ctrl+C` (graceful shutdown ≤5с) и `make dc-down`.

Список всех команд: `make help`.

См. также:
- Архитектура и решения: [docs/project-structure/](docs/project-structure/)
- Стандарты кода: [prompts/](prompts/)
```

**Что важно:**
- Раздел самодостаточный — разработчик после клона следует этим командам и получает рабочую среду.
- Команды `brew install` для macOS — пользователь на macOS (`darwin` в env). Если в будущем понадобится поддержка Linux/Windows — добавить альтернативы (см. open question).
- Ссылка на `docs/project-structure/` — single source of truth для деталей.

### `init_project`

**Что меняется:** файл удаляется.

**Команда:** `rm init_project`.

**Обоснование:** содержимое (`init_project:1` — однострочная TODO) после реализации фичи становится историей. История фиксируется в commit-сообщении и в `general_plan.md:30-34` (Фаза 1 пункт «Структура проекта, Docker Compose, миграции»).

## Файлы для создания

В этой фазе новых файлов нет.

## Ключевые решения

- **S-1** из architect review: добавить quickstart — реализуется здесь.
- **S-2** из architect review: удалить `init_project` — реализуется здесь.
- **S-3** из architect review (Q9 как ADR-015 о middleware) — НЕ реализуется в этой фазе. Решение откладывается до Фазы 1.2 (когда появятся реальные защищённые эндпоинты и нужен будет access-log).

См. `../03-decisions.md` Open Questions для контекста.

## Verification

- [ ] `cat README.MD | grep -q "## Запуск локально"` — раздел добавлен
- [ ] `cat README.MD | grep -q "make dc-up"` — команда упомянута
- [ ] `cat README.MD | grep -q "make migrate-up"` — команда упомянута
- [ ] `cat README.MD | grep -q "curl http://localhost:8080/api/v1/health"` — пример curl присутствует
- [ ] `! test -f init_project` — файл удалён
- [ ] `git ls-files | grep -q init_project` — файл больше не tracked после `git add -u`
- [ ] Markdown-рендер `README.MD` валидный (визуальная проверка в редакторе или `markdownlint README.MD` если установлен — не блокирующий)

## Что НЕ делает эта фаза

- Не вводит инструкции по production-деплою — отдельная фича «deploy».
- Не вводит инструкции для Linux/Windows — открытый вопрос (см. ADR-013-комментарий в `../03-decisions.md` — пользовательская ОС macOS, dev-среда задокументирована под неё).
- Не добавляет `CONTRIBUTING.md`, `LICENSE`, `CODE_OF_CONDUCT.md` — отдельные документы, на этапе MVP не требуются.
- Не закрывает Q9 (access-log middleware) как ADR — откладывается на Фазу 1.2.
- Не активирует архитектурный smoke (`arch_test.go`) — тоже Фаза 1.2 (см. `../04-testing.md`).
