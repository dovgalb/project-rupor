---
phase: 1
name: Migrations and sqlc
layer: migrations
depends_on: none
plan: ./README.md
---

# Phase 1: Миграции БД + расширение sqlc.yaml + первичная генерация

## Цель

Создать схему БД для `rooms`, `room_members`, `invites`, `channels`. Расширить `sqlc.yaml` под три домена (auth + room + channel). Сгенерировать пустые пакеты `db` для room и channel — они понадобятся последующим фазам для компиляции репозиториев.

## Контекст

В `migrations/` уже есть три миграции (`0001_init`, `0002_users`, `0003_refresh_tokens`). Нумерация — 4-значная, парные `up`/`down`, стиль зафиксирован в `../06-repo-model.md` §«Миграции».

В `sqlc.yaml` сейчас один блок `sql:` для auth (`sqlc.yaml:1-27`). Расширяем массив на три блока. Все три используют те же overrides (`uuid` → `google/uuid.UUID`, `timestamptz` → `time.Time`).

После этой фазы пакет `internal/room/repository/postgres/db/` будет создан и пуст (queries пока не написаны), как и `internal/channel/repository/postgres/db/`. Это нормально — `db.New(pool)` и пустой `Queries` — то, на чём будут собираться адаптеры.

## Файлы для создания

### `migrations/0004_rooms.up.sql` / `.down.sql`
Содержимое — точно как в `../06-repo-model.md` §«migrations/0004_rooms.up.sql».

Проверки в верификации:
- `pgcrypto` уже подключён `0001_init.up.sql` — `gen_random_uuid()` работает.
- `users(id)` существует с `0002_users.up.sql` — FK резольвится.

### `migrations/0005_room_members.up.sql` / `.down.sql`
Содержимое — точно как в `../06-repo-model.md` §«migrations/0005_room_members.up.sql».

Ключевые элементы:
- Composite PK `(room_id, user_id)`.
- `CONSTRAINT room_members_role_check CHECK (role IN ('owner', 'admin', 'member'))` — без Postgres ENUM, см. `../03-decisions.md` §D-02.
- **Partial unique index `room_members_one_owner_per_room ON room_members(room_id) WHERE role = 'owner'`** — БД-инвариант «ровно один owner на комнату» (`../03-decisions.md` §D-04).
- Индекс `room_members_user_id_idx` — для `ListUserRooms` (UC-R3).

### `migrations/0006_invites.up.sql` / `.down.sql`
Содержимое — точно как в `../06-repo-model.md` §«migrations/0006_invites.up.sql».

Ключевые элементы:
- `code text` с `CHECK (char_length(code) = 8)`.
- **Partial unique index `invites_active_room ON invites(room_id) WHERE revoked_at IS NULL`** — один активный код на комнату (`../03-decisions.md` §D-04).
- **Partial unique index `invites_active_code ON invites(code) WHERE revoked_at IS NULL`** — глобальная уникальность активного кода (`../03-decisions.md` §D-05).
- Обычный индекс `invites_room_id_idx` — для будущих выборок истории кодов.

### `migrations/0007_channels.up.sql` / `.down.sql`
Содержимое — точно как в `../06-repo-model.md` §«migrations/0007_channels.up.sql».

Ключевые элементы:
- `kind text` с `CHECK (kind IN ('text', 'voice'))`.
- `CONSTRAINT channels_room_id_name_key UNIQUE (room_id, name)` — уникальность имени канала в комнате (`../03-decisions.md` §D-16).
- Индекс `channels_room_id_idx` — для `ListChannels`.

## Файлы для модификации

### `sqlc.yaml` — полная замена

Заменить целиком на содержимое из `../06-repo-model.md` §«sqlc.yaml — расширение». Три блока в `sql:` массиве: auth (как сейчас), room, channel. Все три с одинаковыми overrides.

После сохранения запустить `make sqlc` — должны появиться три каталога:
- `internal/auth/repository/postgres/db/` (без изменений — содержимое то же)
- `internal/room/repository/postgres/db/` (новый, пустой каркас: только `db.go` и `models.go` со стурктурами всех таблиц проекта)
- `internal/channel/repository/postgres/db/` (новый, пустой каркас)

**Внимание:** sqlc сгенерирует `models.go` в каждом каталоге со структурами **всех** таблиц схемы (rooms, room_members, invites, channels, users, refresh_tokens), потому что `schema: "migrations"` указывает на общую папку. Это нормально — каждый домен будет использовать только свои queries; неиспользуемые модели не мешают компиляции.

### `internal/room/repository/postgres/.gitkeep` — удалить
### `internal/channel/repository/postgres/.gitkeep` — удалить

## Ключевые решения

- **Без Postgres ENUM** — text + CHECK для role и kind (`../03-decisions.md` §D-02, §D-03). Расширение значений = новая миграция без `ALTER TYPE`.
- **Hard-delete с CASCADE** на FK `room_members.room_id`, `invites.room_id`, `channels.room_id` (`../03-decisions.md` §D-06).
- **Все БД-инварианты в индексах** (one owner, one active invite per room, unique active code globally) — БД защищает то, что также проверяется в коде (`../03-decisions.md` §D-04, §D-05, §D-11, §D-12).
- **Один общий sqlc.yaml** с массивом `sql:` (`../03-decisions.md` §D-15).

## Verification

- [ ] 8 новых файлов миграций созданы (4 пары up/down).
- [ ] `make migrate-up` отрабатывает без ошибок (требует `make dc-up`).
- [ ] `make migrate-down` четыре раза подряд откатывает все 4 новые миграции.
- [ ] `psql $DATABASE_URL -c '\d rooms'`, `\d room_members`, `\d invites`, `\d channels` показывает ожидаемые таблицы и индексы (включая partial unique).
- [ ] `psql $DATABASE_URL -c "INSERT INTO room_members (room_id, user_id, role, joined_at) VALUES ..."` — попытка вставить второго owner в одну комнату падает с `23505`.
- [ ] `psql $DATABASE_URL -c "INSERT INTO invites ..."` — попытка вставить второй активный invite в одну комнату падает с `23505`.
- [ ] `make sqlc` отрабатывает без ошибок.
- [ ] Появились каталоги `internal/room/repository/postgres/db/` и `internal/channel/repository/postgres/db/` со сгенерированными `db.go`/`models.go`.
- [ ] `internal/auth/repository/postgres/db/` не изменился (`git diff` пустой).
- [ ] `go build ./...` чистый — pacquetes `db` импортируемы, даже если пока не используются.
- [ ] `.gitkeep` в `internal/room/repository/postgres/` и `internal/channel/repository/postgres/` удалены.
