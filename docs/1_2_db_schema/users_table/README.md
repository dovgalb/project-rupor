---
date: 2026-05-05
feature: users_table
status: draft
research: ../../.thoughts/research/2026-05-05-users-table-migration.md
---

# users_table — Документы дизайна

## Бизнес-контекст

Создаём базовую таблицу `users` — первое доменное хранилище в проекте. Без неё невозможно начинать задачу 1.3 «Домен auth» (`.claude/plans/general_plan.md:105-112`): регистрация, логин и выпуск JWT требуют персистентного места хранения учётной записи.

Scope строго ограничен задачей 1.2 из плана MVP (`.claude/plans/general_plan.md:99-103`): одна SQL-миграция с полями `id, email, password_hash, username, created_at` и индексами на `email`/`username`. Доменный слой, value objects, репозиторий и хендлеры — отдельная задача 1.3, отдельный PR. На текущей ветке `feature/PR-1_2-users-table` Go-код не пишется.

Соответственно, эта фича — чистая инфраструктура (внешний слой по `prompts/Architecture Layers.txt:60-65`). Она не имеет рантайм-поведения, HTTP-эндпоинтов, доменных событий и UI. Применяется через `make migrate-up` / откатывается `make migrate-down`.

## Критерии приёмки

1. В `migrations/` появляется пара файлов `0002_users.up.sql` и `0002_users.down.sql`.
2. После `make migrate-up` в БД существует таблица `users` со столбцами `id, email, password_hash, username, created_at` ровно с типами и ограничениями, описанными в `06-repo-model.md`.
3. Уникальное ограничение по `email` имеет имя `users_email_key` (соответствие `prompts/RepoModel.txt:138-140`).
4. Уникальное ограничение по `username` имеет имя `users_username_key`.
5. Первичный ключ `id` имеет тип `uuid` со значением по умолчанию `gen_random_uuid()` и опирается на расширение `pgcrypto`, подключённое в `0001_init`.
6. Расширение `citext` явно подключается в up-миграции и явно отключается в down-миграции.
7. После `make migrate-down` все объекты, созданные up-миграцией, удаляются полностью (таблица и расширение `citext`); состояние БД эквивалентно состоянию после `0001_init`.
8. `make migrate-up` идемпотентен на чистой БД: повторный прогон от начала после `migrate-down` проходит без ошибок.
9. Имена столбцов и таблицы — `snake_case`; все столбцы `NOT NULL` (опциональных полей в этой таблице нет).
10. Никаких изменений в Go-коде, `sqlc.yaml`, `cmd/server/main.go` и `internal/` в рамках этой фичи нет.

## Документы

| Файл | Разрез | Описание |
|------|--------|----------|
| [03-decisions.md](03-decisions.md) | Decision | Решения по схеме (типы, имена ограничений, расширения), альтернативы и риски |
| [06-repo-model.md](06-repo-model.md) | Repository | DDL таблицы `users`, mapping на будущий домен `auth`, соответствие `prompts/RepoModel.txt` |
| [plan/README.md](plan/README.md) | Plan | Стратегия фаз, file map, success criteria |
| [plan/phase-01.md](plan/phase-01.md) | Plan | Создание `migrations/0002_users.up.sql` и `.down.sql` |

### Намеренно опущенные документы шаблона

- `01-architecture.md` (Logical View, C4) — у DDL-миграции нет компонентов, контейнеров и graph зависимостей помимо одной таблицы в существующем контейнере PostgreSQL.
- `02-behavior.md` (Process View, DFD/Sequence) — нет рантайм-поведения; единственный «процесс» — `migrate up`/`down`, описан в `plan/phase-01.md`.
- `04-testing.md` (Quality View) — миграции в проекте не покрываются unit-тестами (см. `Makefile:42-45`, `prompts/Tests Style.txt` касается Go-кода). Верификация — applied/rolled back на локальной БД, см. `plan/phase-01.md`.
- `05-events.md` — нет доменных событий.
- `07-standards.md` — соответствие стандартам сведено к одной строке в `03-decisions.md` (миграция → внешний слой по `Architecture Layers.txt`, остальные стандарты на DDL не распространяются).
- `08-api-contract.md` — нет HTTP-эндпоинтов.
