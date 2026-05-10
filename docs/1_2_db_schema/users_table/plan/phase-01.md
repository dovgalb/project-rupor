---
phase: 1
name: Создать миграцию 0002_users
layer: migrations
depends_on: none
plan: ./README.md
---

# Phase 1: Создать миграцию `0002_users`

## Цель

Добавить в `migrations/` пару SQL-файлов, создающих таблицу `users` со всеми ограничениями, и парный down-файл, корректно откатывающий up. После применения этой миграции БД готова к разработке домена `auth` (задача 1.3, отдельный PR).

## Контекст

- На проекте уже накатана миграция `0001_init` (`migrations/0001_init.up.sql:1-4`), подключающая `pgcrypto`. Её результат (`gen_random_uuid()`) используется в этой фазе для дефолта `id`.
- Каталог `migrations/` управляется `golang-migrate` через `Makefile:54-58`. Новые миграции попадают в обработку автоматически по номеру в имени.
- Целевой DDL и обоснование типов — в `../06-repo-model.md`. Решения по альтернативам и рискам — в `../03-decisions.md`.
- Go-код, `sqlc.yaml`, `cmd/server/main.go`, `internal/` в этой фазе **не трогаются**.

## Файлы для создания

### `migrations/0002_users.up.sql`

**Назначение:** создать таблицу `users` и все её ограничения; обеспечить наличие расширения `citext` (`pgcrypto` уже присутствует из `0001_init`).

**Детали реализации:**

- В первой инструкции — `CREATE EXTENSION IF NOT EXISTS citext;`. Идемпотентно, безопасно для повторного прогона на dev-БД.
- Затем `CREATE TABLE users` со столбцами в порядке, зафиксированном в `../06-repo-model.md` («Сводка по столбцам»). Полный DDL см. там же в разделе «DDL (целевая схема)» — фаза должна воспроизвести его дословно.
- Все столбцы — `NOT NULL`. Опциональных полей в таблице нет (см. `../03-decisions.md` решения 8–9).
- Имена ограничений оставляем дефолтными для PK/UNIQUE (`users_pkey`, `users_email_key`, `users_username_key`); CHECK именуется явно — `users_username_length_check` (см. `../06-repo-model.md` «Имена объектов БД»).
- Имя `users_email_key` критично — оно напрямую ожидается в `prompts/RepoModel.txt:138-140` для маппинга unique-violation в доменную ошибку.
- Никаких отдельных `CREATE INDEX` — индексы под уникальные ограничения создаются Postgres автоматически (см. `../03-decisions.md`, решение 7).
- Никаких триггеров, функций, политик RLS — они не входят в scope.
- Файл начать с короткого комментария-заголовка в стиле `0001_init.up.sql:1-2` (`-- 0002_users.up.sql` + одна строка описания на русском).

### `migrations/0002_users.down.sql`

**Назначение:** полностью откатить эффект up-миграции — удалить таблицу `users` и расширение `citext`.

**Детали реализации:**

- `DROP TABLE IF EXISTS users;` — `IF EXISTS` для устойчивости к повторному прогону.
- `DROP EXTENSION IF EXISTS citext;` — расширение принадлежит этой миграции и должно уходить вместе с ней (см. `../03-decisions.md`, решение 10).
- Порядок: сначала `DROP TABLE`, потом `DROP EXTENSION` — иначе Postgres откажется удалять расширение, на котором висит тип столбца.
- Расширение `pgcrypto` **не трогать** — оно принадлежит миграции `0001_init` и удаляется только её собственным down-файлом.
- Файл начать с короткого комментария-заголовка в стиле `0001_init.down.sql:1-2`.

## Файлы для модификации

Нет.

## Ключевые решения

- Тип `id` = `uuid` с `gen_random_uuid()` (см. `../03-decisions.md`, решение 1) — опирается на расширение `pgcrypto` из `0001_init`.
- `email` = `citext` (см. `../03-decisions.md`, решение 2) — отсюда требование `CREATE EXTENSION citext` в этой же миграции.
- `created_at` = `timestamptz NOT NULL DEFAULT now()` (см. `../03-decisions.md`, решение 5) — приложение в фазе 1.3 сможет не передавать поле в INSERT.
- Имена ограничений дефолтные (см. `../03-decisions.md`, решение 6) — критично для совместимости с будущим маппингом ошибок.
- Поля `status` и `deleted_at` намеренно отсутствуют (см. `../03-decisions.md`, решения 8–9).

## Verification

Запускается локально на машине разработчика, без CI/тестов:

- [ ] `make dc-up` (если БД ещё не поднята).
- [ ] `make migrate-up` — проходит без ошибок, в логах `golang-migrate` появляется `0002_users` как применённая.
- [ ] `psql "$DATABASE_URL" -c '\d users'` показывает 5 столбцов с типами `uuid, citext, text, text, timestamp with time zone`, все `NOT NULL`, дефолты у `id` и `created_at` присутствуют.
- [ ] `psql "$DATABASE_URL" -c "SELECT conname FROM pg_constraint WHERE conrelid = 'users'::regclass ORDER BY conname;"` возвращает ровно: `users_email_key`, `users_pkey`, `users_username_key`, `users_username_length_check`.
- [ ] `psql "$DATABASE_URL" -c "INSERT INTO users (email, password_hash, username) VALUES ('a@b.test', 'hash', 'alice');"` — успех; `id` и `created_at` проставляются автоматически.
- [ ] Повторный INSERT с тем же `email` падает по `users_email_key`; INSERT с тем же `username` — по `users_username_key`.
- [ ] INSERT с `username = 'ab'` падает по `users_username_length_check`; с `username = 'abc'` — успех.
- [ ] INSERT с `email = 'A@B.TEST'` после первого `'a@b.test'` падает по `users_email_key` (case-insensitive за счёт `citext`).
- [ ] `make migrate-down` — таблица и расширение `citext` исчезают; `\dx` больше не показывает `citext`, `\dt` не показывает `users`. `pgcrypto` остаётся.
- [ ] Повторный `make migrate-up` после down проходит идемпотентно.
- [ ] `git diff --name-only` ограничивается двумя файлами в `migrations/` и документами в `docs/users_table/`.
