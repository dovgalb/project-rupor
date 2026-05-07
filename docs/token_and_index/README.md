---
date: 2026-05-07
feature: token_and_index
status: draft
research: ../../.thoughts/research/2026-05-07-tokens-and-index.md
---

# token_and_index — Документы дизайна

## Бизнес-контекст

Фича закрывает оставшийся подпункт задачи 1.2 «Схема БД» (`.claude/plans/general_plan.md:99-103`) — добавляет SQL-миграцию для таблицы `refresh_tokens`. Без этой таблицы невозможно реализовать задачу 1.3 «Домен auth» в части refresh-флоу: usecase `Refresh` (`general_plan.md:107`), генерацию/валидацию refresh-JWT (`general_plan.md:109`) и хендлер `POST /api/v1/auth/refresh` (`general_plan.md:111`).

Подпункт «Индексы на email/username» (`general_plan.md:103`) уже закрыт миграцией `0002_users` и зафиксирован как ADR в `docs/users_table/03-decisions.md:18` (решение 7) — отдельные индексы поверх `UNIQUE` не создаются, b-tree-индексы `users_email_key` и `users_username_key` появляются автоматически. В рамках этой фичи документация по индексам не дублируется.

Стратегия хранения refresh-токенов выбрана осознанно (см. `03-decisions.md`, решения 1–2): хранение **в PostgreSQL** с **ротацией** и записью **только хеша** (`sha256(token)`). Это даёт серверный отзыв сессий (logout, ban, ротация) и защиту от компрометации БД (по утечке нельзя восстановить активные refresh-токены).

Scope строго ограничен SQL-миграцией. Доменные сущности, value objects, репозиторий и хендлеры — задача 1.3, отдельный PR. На текущей ветке `feature/PR-1_2_tokens_and_index` Go-код не пишется (тот же подход, что и у `docs/users_table/`).

## Критерии приёмки

1. В `migrations/` появляется пара файлов `0003_refresh_tokens.up.sql` и `0003_refresh_tokens.down.sql`.
2. После `make migrate-up` в БД существует таблица `refresh_tokens` со столбцами и типами, описанными в `06-repo-model.md`.
3. Первичный ключ `id` имеет тип `uuid` со значением по умолчанию `gen_random_uuid()` (опирается на `pgcrypto` из `0001_init`).
4. Внешний ключ `user_id → users(id)` объявлен с `ON DELETE CASCADE`; имя ограничения — `refresh_tokens_user_id_fkey` (Postgres-дефолт).
5. Уникальное ограничение по `token_hash` имеет имя `refresh_tokens_token_hash_key` (Postgres-дефолт под `UNIQUE`); индекс под него — `refresh_tokens_token_hash_key` (b-tree). Это имя — часть контракта с будущим маппингом unique-violation в доменную ошибку (`prompts/RepoModel.txt:138-140`).
6. Существует отдельный b-tree-индекс по `user_id` с именем `refresh_tokens_user_id_idx` для запросов «отозвать все токены пользователя» / «список активных сессий».
7. Столбец `token_hash` имеет тип `bytea` и `NOT NULL`; ограничение длины проверяется `CHECK (octet_length(token_hash) = 32)` (sha256 = 32 байта).
8. Столбец `expires_at` — `timestamptz NOT NULL`; default не задаётся (значение приходит из приложения в задаче 1.3).
9. Столбец `created_at` — `timestamptz NOT NULL DEFAULT now()`.
10. Столбец `revoked_at` — `timestamptz NULL` (NULL = активный токен; ненулевое значение = отозванный/ротированный).
11. После `make migrate-down` все объекты, созданные up-миграцией, удаляются полностью; состояние БД эквивалентно состоянию после `0002_users`.
12. `make migrate-up` идемпотентен на чистой БД: последовательность `migrate-down` → `migrate-up` проходит без ошибок.
13. Имена столбцов и таблицы — `snake_case`.
14. Никаких изменений в Go-коде, `sqlc.yaml`, `cmd/server/main.go` и `internal/` в рамках этой фичи нет.

## Документы

| Файл | Разрез | Описание |
|------|--------|----------|
| [03-decisions.md](./03-decisions.md) | Decision | Решения по схеме (хеш vs raw, FK с CASCADE, ротация через `revoked_at`, индексы), альтернативы и риски |
| [06-repo-model.md](./06-repo-model.md) | Repository | DDL таблицы `refresh_tokens`, имена объектов БД, плановый mapping на будущий домен `auth` (1.3), соответствие `prompts/RepoModel.txt` |
| [plan/README.md](./plan/README.md) | Plan | Стратегия фаз, file map, success criteria *(создаётся после approval дизайна)* |
| [plan/phase-01.md](./plan/phase-01.md) | Plan | Создание `migrations/0003_refresh_tokens.up.sql` и `.down.sql` *(создаётся после approval дизайна)* |

### Намеренно опущенные документы шаблона

Следуем тому же подходу, что и `docs/users_table/README.md:40-47` — у DDL-only фичи нет компонентов уровня кода, рантайм-поведения и API-контракта.

- `01-architecture.md` (Logical View, C4) — у DDL-миграции нет новых компонентов/контейнеров: одна таблица внутри уже существующего PostgreSQL-контейнера. Граф зависимостей слоёв (`prompts/Architecture Layers.txt:67-73`) фичей не затрагивается.
- `02-behavior.md` (Process View, DFD/Sequence) — нет рантайм-поведения. Единственный «процесс» — `migrate up`/`down`, описан в `plan/phase-01.md`. Поведение refresh-флоу проектируется в задаче 1.3 (отдельная фича), там и появятся sequence-диаграммы.
- `04-testing.md` (Quality View) — миграции в проекте не покрываются unit-тестами (`Makefile:42-45`, `prompts/Tests Style.txt` касается Go-кода). Верификация — applied/rolled back на локальной БД, см. `plan/phase-01.md`.
- `05-events.md` — нет доменных событий (нет Go-кода).
- `07-standards.md` — соответствие стандартам сведено к одной строке в `03-decisions.md` (миграция → внешний слой по `prompts/Architecture Layers.txt:60-65`, остальные стандарты на чистый DDL не распространяются).
- `08-api-contract.md` — нет HTTP-эндпоинтов.
