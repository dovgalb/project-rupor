---
phase: 2
name: Repo Skeleton
layer: infrastructure
depends_on: [phase-01]
plan: ./README.md
---

# Phase 2: Repo Skeleton

## Цель

Физически создать дерево директорий монорепо: 24 поддиректории `internal/<домен>/<слой>/.gitkeep` (6 доменов × 4 слоя), `pkg/websocket/.gitkeep`, и initial-миграцию `migrations/0001_init.{up,down}.sql`. После этой фазы `make migrate-up` накатывает миграцию (если поднята БД из Phase 1).

## Контекст

Что произвела предыдущая фаза:
- `Makefile` с целями `migrate-up`/`migrate-down`, ссылающимися на `migrations/`.
- `docker-compose.yml` с `postgres:16-alpine`.
- `sqlc.yaml`, схема которого ссылается на `./migrations`.

Эта фаза заполняет ссылки реальными артефактами.

## Файлы для создания

### 24 файла `.gitkeep` в `internal/<домен>/<слой>/`

**Назначение:** физическая фиксация терминологии слоёв из `prompts/Architecture Layers.txt:46-58` и `CLAUDE.md:8-19` (ADR-001 в `../03-decisions.md`).

**Содержимое каждого файла:** пустой файл, нулевой длины.

**Список доменов** (из `general_plan.md:5-25` и `CLAUDE.md:9-19`):
- `auth` — регистрация, логин, JWT
- `user` — профили пользователей
- `room` — комнаты (аналог Discord-серверов)
- `channel` — каналы внутри комнат
- `chat` — сообщения, история, пагинация
- `voice` — сигнальный сервер для WebRTC

**Список слоёв** (из `prompts/Architecture Layers.txt:14-66`):
- `domain/` — сущности, value objects, доменные ошибки
- `usecase/` — сценарии и интерфейсы зависимостей
- `transport/http/` — HTTP-хендлеры и DTO
- `repository/postgres/` — реализации репозиториев через sqlc

**Полный список файлов (24 шт):**

```
internal/auth/domain/.gitkeep
internal/auth/usecase/.gitkeep
internal/auth/transport/http/.gitkeep
internal/auth/repository/postgres/.gitkeep
internal/user/domain/.gitkeep
internal/user/usecase/.gitkeep
internal/user/transport/http/.gitkeep
internal/user/repository/postgres/.gitkeep
internal/room/domain/.gitkeep
internal/room/usecase/.gitkeep
internal/room/transport/http/.gitkeep
internal/room/repository/postgres/.gitkeep
internal/channel/domain/.gitkeep
internal/channel/usecase/.gitkeep
internal/channel/transport/http/.gitkeep
internal/channel/repository/postgres/.gitkeep
internal/chat/domain/.gitkeep
internal/chat/usecase/.gitkeep
internal/chat/transport/http/.gitkeep
internal/chat/repository/postgres/.gitkeep
internal/voice/domain/.gitkeep
internal/voice/usecase/.gitkeep
internal/voice/transport/http/.gitkeep
internal/voice/repository/postgres/.gitkeep
```

**Команда для создания (для оператора):**
```bash
for d in auth user room channel chat voice; do
  for l in domain usecase transport/http repository/postgres; do
    mkdir -p "internal/$d/$l"
    touch "internal/$d/$l/.gitkeep"
  done
done
```

### `pkg/websocket/.gitkeep`

**Назначение:** заглушка для будущего общего WebSocket hub'а из `general_plan.md:75-78` и `CLAUDE.md:21`. Заполнится в Фазе 3 (текстовый чат) или Фазе 4 (голос) общего плана.

### `migrations/0001_init.up.sql`

**Назначение:** initial-миграция, активирует расширение `pgcrypto` для `gen_random_uuid()` (ADR-009).

**Содержимое:**

```sql
-- 0001_init.up.sql
-- Initial migration: enable pgcrypto for gen_random_uuid().

CREATE EXTENSION IF NOT EXISTS pgcrypto;
```

**Замечания:**
- `IF NOT EXISTS` — миграция идемпотентна на уровне расширения. Это безопасно при повторном применении вручную.
- `pgcrypto` встроено в `postgres:16-alpine`, не требует доустановки пакетов.
- Бизнес-таблиц (`users`, `rooms`, и т.д.) НЕТ — они идут отдельными миграциями `0002_*`, `0003_*` в Фазе 1.2.

### `migrations/0001_init.down.sql`

**Назначение:** парный откат.

**Содержимое:**

```sql
-- 0001_init.down.sql
-- Rollback: drop pgcrypto extension.

DROP EXTENSION IF EXISTS pgcrypto;
```

`IF EXISTS` — откат идемпотентен.

## Файлы для модификации

В этой фазе модификаций существующих файлов нет.

## Ключевые решения

- **ADR-001** (раскладка `internal/<домен>/<слой>/.gitkeep` сразу для всех 6 доменов) — реализуется здесь.
- **ADR-009** (`pgcrypto` вместо `uuid-ossp`) — реализуется здесь.
- **ADR-005** (golang-migrate) — миграция в формате `NNNN_name.{up,down}.sql` — стандарт golang-migrate.

См. `../03-decisions.md` для полного контекста.

## Verification

- [ ] `find internal -name ".gitkeep" | wc -l` == `24`
- [ ] `find internal -type d | sort` показывает дерево из 6 доменов × 4 слоёв (всего 30 директорий: 6 доменных + 24 слоя)
- [ ] `ls pkg/websocket/.gitkeep` — файл существует
- [ ] `ls migrations/0001_init.up.sql migrations/0001_init.down.sql` — оба файла существуют
- [ ] `cat migrations/0001_init.up.sql` содержит `CREATE EXTENSION IF NOT EXISTS pgcrypto`
- [ ] `cat migrations/0001_init.down.sql` содержит `DROP EXTENSION IF EXISTS pgcrypto`
- [ ] `git status` — все 27 новых файлов готовы к add (24 internal + 1 pkg + 2 migrations)
- [ ] После `make dc-up` (из Phase 1): `make migrate-up` — exit 0, в БД применяется миграция версии 1
- [ ] `psql $DATABASE_URL -c "SELECT extname FROM pg_extension WHERE extname='pgcrypto'"` возвращает `pgcrypto`
- [ ] `psql $DATABASE_URL -c "SELECT version, dirty FROM schema_migrations"` возвращает `1, false`
- [ ] `make migrate-down` — exit 0, расширение `pgcrypto` удаляется (или остаётся, если БД его использует — это не ошибка для `IF EXISTS`)
- [ ] `make migrate-up` повторно после `make migrate-down` — снова exit 0 (идемпотентность)
- [ ] Все слои у каждого домена тождественны: `for d in auth user room channel chat voice; do ls internal/$d; done` показывает одинаковое дерево

## Что НЕ делает эта фаза

- Не создаёт `config/`, `cmd/server/`, никакого Go-кода.
- Не создаёт бизнес-таблицы (users, rooms, ...) — отдельные миграции в Фазе 1.2.
- Не создаёт sqlc-блоки per-domain — они появятся при первом домене.
- Не активирует архитектурный smoke тест — он будет в Фазе 1.2 (см. `../04-testing.md`).
