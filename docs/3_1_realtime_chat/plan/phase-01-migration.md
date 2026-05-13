---
phase: 1
name: migration
layer: migrations
depends_on: none
plan: ./README.md
---

# Phase 1: Миграция БД `0008_messages`

## Цель

Создать таблицу `messages` со всеми constraint'ами и индексами, описанными в [../06-repo-model.md](../06-repo-model.md). После этой фазы БД готова принимать вставки сообщений; репозитория ещё нет.

## Контекст

Текущий максимум миграций — `0007_channels`. Сборка миграций — `golang-migrate` (см. `Makefile:48-52`). Расширение Postgres `pgcrypto` уже подключено в `0001_init.up.sql` (для `gen_random_uuid()`). Существующие таблицы: `users` (`0002`), `refresh_tokens` (`0003`), `rooms` (`0004`), `room_members` (`0005`), `invites` (`0006`), `channels` (`0007`).

Шаблон именования миграции — `NNNN_<имя>.{up|down}.sql`. Обе стороны (up и down) обязательны.

## Файлы для создания

### `migrations/0008_messages.up.sql`

**Назначение:** Создаёт таблицу `messages` для хранения текстовых сообщений в каналах.

**Содержимое — точно как в [../06-repo-model.md §Миграция](../06-repo-model.md#migrations0008_messagesupsql):**

```sql
-- 0008_messages.up.sql
-- Создание таблицы messages (сообщения в text-каналах).

CREATE TABLE messages (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id  uuid        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id   uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT messages_text_length_check CHECK (char_length(text) BETWEEN 1 AND 16000)
);

CREATE INDEX messages_channel_created_id_idx
    ON messages (channel_id, created_at DESC, id DESC);

CREATE INDEX messages_author_id_idx
    ON messages (author_id);
```

**Детали реализации:**
- `id` — UUID v4 (через `gen_random_uuid()` из `pgcrypto`).
- `channel_id` FK на `channels(id)` с `ON DELETE CASCADE` — при удалении канала сообщения исчезают.
- `author_id` FK на `users(id)` с `ON DELETE CASCADE` — при удалении пользователя его сообщения исчезают (политика MVP; «tombstone» не реализуется).
- `text` без явного `COLLATE` (используется default БД — `UTF8`).
- CHECK на длину `1..16000` — это БД-граница (16000 байт ≈ верхняя оценка для 4000 рун UTF-8). Domain-валидация в `MessageText` (1..4000 рун) строже, это defense-in-depth.
- Композитный индекс `(channel_id, created_at DESC, id DESC)` обслуживает курсорную пагинацию `ListMessagesBeforeCursor` без отдельной сортировки.
- Индекс `(author_id)` — для будущих фич («мои сообщения», антиспам); не критичен для фазы 3.1, но дёшев.

### `migrations/0008_messages.down.sql`

**Назначение:** Откат миграции.

```sql
-- 0008_messages.down.sql
DROP TABLE IF EXISTS messages;
```

`DROP TABLE` каскадно удаляет индексы и constraint'ы; явных дополнительных операций не требуется. Расширение `pgcrypto` не трогается — оно общее для всех миграций.

## Файлы для модификации

Нет. Phase-01 — чисто добавление двух SQL-файлов.

## Ключевые решения

- **Зашитая длина 16000 байт в CHECK** — согласовано с [§ D-14](../03-decisions.md) и [`../06-repo-model.md`](../06-repo-model.md). Не путать с domain-лимитом 4000 рун.
- **`ON DELETE CASCADE`** — согласовано с поведением `0007_channels.up.sql:6` и `0005_room_members.up.sql`.
- **Без soft-delete** — фаза 3.1 не вводит редактирование/удаление сообщений (см. [§ D-04](../03-decisions.md)). Колонки `deleted_at`, `edited_at` появятся в более поздней миграции, если фича будет.

## Verification

- [ ] `make migrate-up` применяет миграцию без ошибок.
- [ ] `\d messages` в psql показывает таблицу со всеми колонками, FK, CHECK, индексами.
- [ ] `make migrate-down` откатывает миграцию без ошибок.
- [ ] `make migrate-up` повторно — миграция идемпотентно повторяется (golang-migrate отказывается без явного `force`, это ок).
- [ ] CHECK работает: `INSERT INTO messages (channel_id, author_id, text) VALUES (..., '');` падает с ошибкой constraint.
- [ ] CHECK работает: `INSERT INTO messages (..., text) VALUES (..., REPEAT('x', 16001));` падает.
- [ ] FK CASCADE работает: вставить канал → вставить сообщение → удалить канал → `SELECT count(*) FROM messages WHERE channel_id = ...` возвращает 0.
