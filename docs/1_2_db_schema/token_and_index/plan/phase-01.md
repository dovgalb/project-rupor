---
phase: 1
name: Создать миграцию 0003_refresh_tokens
layer: migrations
depends_on: none
plan: ./README.md
---

# Phase 1: Создать миграцию `0003_refresh_tokens`

## Цель

Добавить в `migrations/` пару SQL-файлов, создающих таблицу `refresh_tokens` со всеми ограничениями и явным индексом по `user_id`, и парный down-файл, корректно откатывающий up. После применения этой миграции БД готова к разработке refresh-флоу в домене `auth` (задача 1.3, отдельный PR).

## Контекст

- На проекте уже накатаны миграции `0001_init` (`migrations/0001_init.up.sql:1-4` — расширение `pgcrypto`, даёт `gen_random_uuid()`) и `0002_users` (`migrations/0002_users.up.sql:1-13` — таблица `users` с PK `id uuid`).
- Таблица `users` — обязательная зависимость FK `refresh_tokens.user_id → users(id)`. Порядок миграций `0001 → 0002 → 0003` обеспечивается префиксами; `golang-migrate` накатывает по возрастанию (`Makefile:54-58`).
- Расширение `citext`, подключённое `0002_users`, в этой миграции **не используется и не трогается**.
- Целевой DDL и обоснование типов — в `../06-repo-model.md`. Решения по альтернативам и рискам — в `../03-decisions.md`.
- Go-код, `sqlc.yaml`, `cmd/server/main.go`, `internal/` в этой фазе **не трогаются**.

## Файлы для создания

### `migrations/0003_refresh_tokens.up.sql`

**Назначение:** создать таблицу `refresh_tokens` со всеми ограничениями и явный b-tree индекс по `user_id`.

**Детали реализации:**

- **Никаких `CREATE EXTENSION`** — необходимые расширения (`pgcrypto`) уже присутствуют из `0001_init`. `citext` для refresh-токенов не нужен.
- `CREATE TABLE refresh_tokens` со столбцами в порядке, зафиксированном в `../06-repo-model.md` («Сводка по столбцам»). Полный DDL см. там же в разделе «DDL (целевая схема)» — фаза должна воспроизвести его дословно.
- `id uuid PRIMARY KEY DEFAULT gen_random_uuid()` — имя PK дефолтное `refresh_tokens_pkey` (см. `../03-decisions.md`, решение 4).
- `user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE` — FK с каскадным удалением. Имя FK дефолтное `refresh_tokens_user_id_fkey` (см. `../03-decisions.md`, решения 5–6). Каскад критичен: удаление пользователя обязано инвалидировать все его сессии.
- `token_hash bytea NOT NULL UNIQUE` — UNIQUE даёт автоматический b-tree индекс с дефолтным именем `refresh_tokens_token_hash_key` (см. `../03-decisions.md`, решение 7). Никакого отдельного `CREATE UNIQUE INDEX` поверх — это создало бы два одинаковых индекса.
- `CONSTRAINT refresh_tokens_token_hash_length_check CHECK (octet_length(token_hash) = 32)` — фиксирует длину sha256 на уровне БД (см. `../03-decisions.md`, решение 3). Имя CHECK задаётся явно для предсказуемости.
- `expires_at timestamptz NOT NULL` — **без default** (см. `../03-decisions.md`, решение 11). TTL — политика приложения, задаётся в задаче 1.3.
- `created_at timestamptz NOT NULL DEFAULT now()` — согласован с `users.created_at` (см. `../03-decisions.md`, решение 12).
- `revoked_at timestamptz NULL` — **единственный nullable** столбец (см. `../03-decisions.md`, решение 9). NULL = активный, не-NULL = отозван.
- После `CREATE TABLE` — отдельный `CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens(user_id);` (см. `../03-decisions.md`, решение 8). Postgres **не индексирует FK-столбец автоматически**; индекс обязателен под logout-all и под сам каскад.
- Имя `refresh_tokens_token_hash_key` критично — оно будет ожидаться в задаче 1.3 для маппинга unique-violation в доменную ошибку, по аналогии с `users_email_key` (`prompts/RepoModel.txt:138-140`).
- Никаких партиальных индексов (`WHERE revoked_at IS NULL`), триггеров, функций, политик RLS — они не входят в scope (см. `../03-decisions.md`, решение 10).
- Файл начать с короткого комментария-заголовка в стиле `0002_users.up.sql:1-2` (`-- 0003_refresh_tokens.up.sql` + одна строка описания на русском).

### `migrations/0003_refresh_tokens.down.sql`

**Назначение:** полностью откатить эффект up-миграции — удалить таблицу `refresh_tokens`.

**Детали реализации:**

- `DROP TABLE IF EXISTS refresh_tokens;` — `IF EXISTS` для устойчивости к повторному прогону. Один statement достаточен: FK, UNIQUE, CHECK, индексы (включая явный `refresh_tokens_user_id_idx`) удаляются Postgres автоматически вместе с таблицей.
- **Никаких `DROP EXTENSION`** — эта миграция не подключала ни `pgcrypto`, ни `citext`. Расширения принадлежат своим миграциям (`0001_init` и `0002_users` соответственно) и удаляются их собственными down-файлами.
- **Никакого `ALTER TABLE users`** — `users` не модифицировалась.
- Файл начать с короткого комментария-заголовка в стиле `0002_users.down.sql:1-2`.

## Файлы для модификации

Нет.

## Ключевые решения

- Хранение `sha256(token)` в `bytea`, не raw-токен и не JTI (см. `../03-decisions.md`, решения 1–3).
- FK `ON DELETE CASCADE` + явный индекс по `user_id` (см. `../03-decisions.md`, решения 5, 8).
- `revoked_at` как nullable timestamptz, не bool (см. `../03-decisions.md`, решение 9). Запрос активных токенов будет: `WHERE revoked_at IS NULL AND expires_at > now()`.
- Отсутствуют поля `device_id` / `user_agent` / `ip` (см. `../03-decisions.md`, решение 13).
- `expires_at` без default (см. `../03-decisions.md`, решение 11) — намеренно требует значение от приложения.

## Verification

Запускается локально на машине разработчика, без CI/тестов:

- [ ] `make dc-up` (если БД ещё не поднята).
- [ ] `make migrate-up` — проходит без ошибок, в логах `golang-migrate` появляется `0003_refresh_tokens` как применённая.
- [ ] `psql "$DATABASE_URL" -c '\d refresh_tokens'` показывает 6 столбцов с типами `uuid, uuid, bytea, timestamp with time zone, timestamp with time zone, timestamp with time zone`. `id`, `user_id`, `token_hash`, `expires_at`, `created_at` — `NOT NULL`; `revoked_at` — `NULL`. Дефолты у `id` (`gen_random_uuid()`) и `created_at` (`now()`) присутствуют. У `expires_at` дефолта нет.
- [ ] `psql "$DATABASE_URL" -c "SELECT conname FROM pg_constraint WHERE conrelid = 'refresh_tokens'::regclass ORDER BY conname;"` возвращает ровно: `refresh_tokens_pkey`, `refresh_tokens_token_hash_key`, `refresh_tokens_token_hash_length_check`, `refresh_tokens_user_id_fkey`.
- [ ] `psql "$DATABASE_URL" -c "SELECT indexname FROM pg_indexes WHERE tablename = 'refresh_tokens' ORDER BY indexname;"` возвращает ровно: `refresh_tokens_pkey`, `refresh_tokens_token_hash_key`, `refresh_tokens_user_id_idx`.
- [ ] `psql "$DATABASE_URL" -c "SELECT confdeltype FROM pg_constraint WHERE conname = 'refresh_tokens_user_id_fkey';"` возвращает `c` (CASCADE).
- [ ] Сценарий happy-path INSERT (предварительно создать строку в `users`):
  ```sql
  INSERT INTO users (email, password_hash, username) VALUES ('a@b.test', 'hash', 'alice') RETURNING id;
  -- подставить полученный id вместо :uid
  INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
  VALUES (':uid', sha256('demo-token'::bytea), now() + interval '30 days');
  ```
  — успех; `id` и `created_at` проставляются автоматически; `revoked_at` остаётся NULL.
- [ ] Повторный INSERT с тем же `token_hash` падает по `refresh_tokens_token_hash_key`.
- [ ] INSERT c `token_hash = sha1('x'::bytea)` (20 байт) падает по `refresh_tokens_token_hash_length_check`.
- [ ] INSERT с `user_id`, которого нет в `users`, падает по `refresh_tokens_user_id_fkey`.
- [ ] `DELETE FROM users WHERE id = ':uid';` каскадно удаляет связанные строки в `refresh_tokens` (после удаления — `SELECT count(*) FROM refresh_tokens WHERE user_id = ':uid'` возвращает 0).
- [ ] `make migrate-down` — таблица `refresh_tokens` исчезает; `\dt` больше её не показывает. Расширения `pgcrypto` и `citext` остаются (`\dx` показывает их).
- [ ] Повторный `make migrate-up` после down проходит идемпотентно.
- [ ] `git diff --name-only` ограничивается двумя файлами в `migrations/` и документами в `docs/token_and_index/`.
