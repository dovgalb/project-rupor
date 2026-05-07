---
date: 2026-05-07
researcher: Claude
commit: 5d17cf8
branch: feature/PR-1_2_tokens_and_index
research_question: "tokens_and_index — миграция таблицы refresh_tokens (если хранятся в БД) и индексы на email/username"
---

# Исследование: миграция `refresh_tokens` и индексы на `email`/`username`

## Резюме

Исследование охватывает два пункта плана из `.claude/plans/general_plan.md:102-103`. По состоянию на коммит `5d17cf8` (ветка `feature/PR-1_2_tokens_and_index`):

1. **Индексы на `email`/`username` — уже реализованы** в миграции `0002_users` (`migrations/0002_users.up.sql:6-13`). Postgres автоматически создаёт b-tree индексы под уникальные ограничения; имена индексов — `users_email_key` и `users_username_key`. Решение «не дублировать `CREATE INDEX` поверх `UNIQUE`» зафиксировано как ADR (`docs/users_table/03-decisions.md:18`, решение 7) и подтверждено в repo-model (`docs/users_table/06-repo-model.md:50-54`).

2. **Миграция `refresh_tokens` — не начата**. В репозитории нет ни SQL-файла, ни кода, ни ADR/repo-model для этой таблицы. В плане она помечена как опциональная — `general_plan.md:102`: «таблица `refresh_tokens` (если хранятся в БД)». Сопутствующие задачи (usecase `Refresh`, генерация/валидация JWT access+refresh, хендлер `/auth/refresh`, endpoint `POST /api/v1/auth/refresh`) тоже не начаты. Никаких решений о схеме хранения refresh-токенов (поля, TTL, ротация, JTI vs hash) пока не принято.

Ветка названа .`feature/PR-1_2_tokens_and_index`, что объединяет оба подпункта плана 1.2 в один PR; на момент исследования в неё уже включена миграция `0002_users` (закрывает пункт про индексы), а миграция `refresh_tokens` ещё не написана.

## Детальные результаты

### 1. Индексы на `email` / `username` — текущая реализация

- **Расположение DDL**: `migrations/0002_users.up.sql:6-13`
- **Содержимое up** (целиком):
  ```sql
  -- 0002_users.up.sql
  -- Создание таблицы пользователей с ограничениями.

  CREATE EXTENSION IF NOT EXISTS citext;

  CREATE TABLE users (
      id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
      email         citext      NOT NULL UNIQUE,
      password_hash text        NOT NULL,
      username      text        NOT NULL UNIQUE,
      created_at    timestamptz NOT NULL DEFAULT now(),
      CONSTRAINT users_username_length_check CHECK (char_length(username) BETWEEN 3 AND 32)
  );
  ```
- **Откат**: `migrations/0002_users.down.sql:1-5`
  ```sql
  -- 0002_users.down.sql
  -- Откат: удалить таблицу users и расширение citext.

  DROP TABLE IF EXISTS users;
  DROP EXTENSION IF EXISTS citext;
  ```
- **Какие индексы создаются**:
  - `users_pkey` — b-tree под `PRIMARY KEY (id)` (создаётся PostgreSQL автоматически).
  - `users_email_key` — b-tree под `UNIQUE` на столбце `email` (`migrations/0002_users.up.sql:8`).
  - `users_username_key` — b-tree под `UNIQUE` на столбце `username` (`migrations/0002_users.up.sql:10`).
- **Дополнительных `CREATE INDEX` в миграции нет**. Поиск `grep -rni "CREATE INDEX" migrations/` пуст.
- **Отдельных функциональных индексов (`lower(email)` и т.п.) нет** — case-insensitive уникальность обеспечивается типом `citext` (см. ADR ниже).
- **Имена индексов фиксируются** этой миграцией. `users_email_key` именно в таком виде ожидается в `prompts/RepoModel.txt:138-140` для маппинга unique-violation на `domain.ErrEmailAlreadyTaken`.

### 2. ADR по индексам и связанным решениям

- **Расположение**: `docs/users_table/03-decisions.md`
- **Решение 7 — индексы на `email`/`username`** (`docs/users_table/03-decisions.md:18`):
  > Покрываются автоматически уникальными ограничениями (Postgres создаёт b-tree индекс под каждый `UNIQUE`). План требует «индексы на email/username» (`general_plan.md:103`); `UNIQUE` это требование закрывает. Дублирование `CREATE INDEX` поверх `UNIQUE` создаёт два одинаковых индекса.
- **Решение 2 — тип `email`** (`docs/users_table/03-decisions.md:13`): выбран `citext` вместо `text + UNIQUE INDEX (lower(email))`. Это убирает необходимость в функциональном индексе и сохраняет дефолтное имя `users_email_key`.
- **Решение 3 — тип `username`** (`docs/users_table/03-decisions.md:14`): `text + UNIQUE + CHECK (length BETWEEN 3 AND 32)`. Регистрозависимая уникальность; нормализация (если потребуется) — задача доменного VO в фазе 1.3, не миграции.
- **Решение 6 — имена ограничений** (`docs/users_table/03-decisions.md:17`): дефолтные Postgres-имена (`users_pkey`, `users_email_key`, `users_username_key`).
- **Repo-model сводка** (`docs/users_table/06-repo-model.md:48-54`):
  | Объект | Имя | Тип |
  |--------|-----|-----|
  | Таблица | `users` | table |
  | PK | `users_pkey` | constraint (PK) |
  | Уникальность email | `users_email_key` | constraint (UNIQUE) |
  | Уникальность username | `users_username_key` | constraint (UNIQUE) |
  | Длина username | `users_username_length_check` | constraint (CHECK) |
  | Индекс под `users_email_key` | `users_email_key` | b-tree (под UNIQUE) |
  | Индекс под `users_username_key` | `users_username_key` | b-tree (под UNIQUE) |
- **План фазы 01** (`docs/users_table/plan/phase-01.md:34-35`) явно фиксирует: «никаких отдельных `CREATE INDEX` — индексы под уникальные ограничения создаются Postgres автоматически».

### 3. Миграция `refresh_tokens` — текущее состояние

- **SQL-файлов для `refresh_tokens` в репозитории нет.**
  Поиск `find migrations/ -type f` возвращает только: `0001_init.up.sql`, `0001_init.down.sql`, `0002_users.up.sql`, `0002_users.down.sql`.
- **Упоминаний `refresh_token`/`refresh_tokens` в Go-коде, sqlc-конфиге и SQL-файлах нет.**
  `grep -rni "refresh"` по `*.go`, `*.sql`, `*.md` находит совпадения только в плановой/документной части:
  - `.claude/plans/general_plan.md:47` — endpoint `POST /api/v1/auth/refresh` (контракт API).
  - `.claude/plans/general_plan.md:102` — пункт «Миграция: таблица `refresh_tokens` (если хранятся в БД)», статус `[ ]`.
  - `.claude/plans/general_plan.md:107` — usecase `Refresh` в `internal/auth/usecase/`, статус `[ ]`.
  - `.claude/plans/general_plan.md:109` — генерация и валидация JWT (access + refresh), статус `[ ]`.
  - `.claude/plans/general_plan.md:111` — хендлер `POST /auth/refresh`, статус `[ ]`.
  - `.claude/plans/general_plan.md:166` — refresh-флоу на фронте (фаза 5), статус `[ ]`.
  - `CLAUDE.md:33` — комментарий о домене `auth` (Регистрация, логин, JWT access + refresh).
  - `.thoughts/research/2026-05-05-users-table-migration.md:105,139` — предыдущий research отметил, что задача `refresh_tokens` — отдельный пункт и в его scope не входила.
- **ADR/repo-model для `refresh_tokens` отсутствуют.**
  Каталог `docs/` содержит только два «комплекта» документации: `docs/project-structure/` (инфраструктурный) и `docs/users_table/`. Папки `docs/refresh_tokens/` нет; в существующих документах нет ни структуры, ни решений по таблице, ни выбранной стратегии (хранить хеш токена или JTI, ротировать или нет, TTL, привязка к user_id, device_id и т.п.).
- **Слой `internal/auth/` пуст.**
  Все четыре подкаталога содержат только `.gitkeep`:
  - `internal/auth/domain/.gitkeep`
  - `internal/auth/usecase/.gitkeep`
  - `internal/auth/repository/postgres/.gitkeep`
  - `internal/auth/transport/http/.gitkeep`
- **`sqlc.yaml`** (`sqlc.yaml:1-2`):
  ```yaml
  version: "2"
  sql: []
  ```
  Конфиг sqlc по-прежнему пуст; никаких queries для refresh_tokens (или users) не сгенерировано.
- **Связь с задачей JWT (1.3).** План явно ставит refresh-флоу в задаче 1.3 (`general_plan.md:107,109,111`). Текущая ветка `feature/PR-1_2_tokens_and_index` тематически закрывает только инфраструктурную часть схемы БД (1.2): добавить таблицу для хранения refresh-токенов, если решено хранить их в БД. Само решение «хранить ли в БД» в коде/документах не зафиксировано — формулировка плана условная: «если хранятся в БД».

### 4. Состояние ветки и Git

- **Текущая ветка**: `feature/PR-1_2_tokens_and_index`.
- **HEAD**: `5d17cf8` — `feat(migrations): добавить миграцию 0002_users`. Это коммит, добавивший `migrations/0002_users.up.sql` и `migrations/0002_users.down.sql` (закрыл пункт про users и индексы).
- **Незакоммиченные изменения** (на старте сессии):
  - Modified: `.claude/plans/general_plan.md`, `README.MD`.
  - Untracked: `tasks/`, `test.http`.
- **Главная ветка**: `main`. Последний мерж в `main` — `d971527` (мерж PR `feature/1-project-structure`).
- **Предыдущий research-документ по 1.2** (`.thoughts/research/2026-05-05-users-table-migration.md`) делался на ветке `feature/PR-1_2-users-table` до того, как миграция `0002_users` была написана; задача `refresh_tokens` в его scope не входила.

### 5. Инфраструктура миграций (контекст для будущей `refresh_tokens`)

Без изменений со времени предыдущего research:
- **Расположение SQL-миграций**: `migrations/`, формат имени `NNNN_<slug>.up.sql` / `NNNN_<slug>.down.sql` (4-значный префикс с ведущими нулями). Следующий свободный номер — `0003`.
- **`Makefile:54-58`**:
  ```make
  migrate-up:
      migrate -path migrations -database "$(DATABASE_URL)" up
  migrate-down:
      migrate -path migrations -database "$(DATABASE_URL)" down 1
  ```
- **`Makefile:60-61`**: `sqlc generate` (но конфиг пуст, генерировать нечего).
- **`docker-compose.yml:1-18`**: единственный сервис `postgres:16-alpine`, БД `rupor`, креды `rupor/rupor`. Redis в compose отсутствует.
- **`.env.example:1-8`**: `DATABASE_URL`, `JWT_SECRET`, `SERVER_PORT` — без отдельных переменных под refresh-токены.
- **Расширения БД**: `pgcrypto` (включается `0001_init`, даёт `gen_random_uuid()`), `citext` (включается `0002_users`, нужен только для `email`).

## Ссылки на код

- `migrations/0001_init.up.sql:1-4` — расширение `pgcrypto`.
- `migrations/0001_init.down.sql:1-4` — откат `pgcrypto`.
- `migrations/0002_users.up.sql:1-13` — текущая схема `users`, включая `UNIQUE` на `email` и `username`.
- `migrations/0002_users.up.sql:8` — `email citext NOT NULL UNIQUE` (создаёт индекс `users_email_key`).
- `migrations/0002_users.up.sql:10` — `username text NOT NULL UNIQUE` (создаёт индекс `users_username_key`).
- `migrations/0002_users.down.sql:1-5` — откат: `DROP TABLE users` + `DROP EXTENSION citext`.
- `docs/users_table/03-decisions.md:18` — решение 7: индексы покрываются `UNIQUE`, отдельных `CREATE INDEX` нет.
- `docs/users_table/06-repo-model.md:46-54` — таблица имён объектов БД (включая индексы под `users_email_key` и `users_username_key`).
- `docs/users_table/plan/phase-01.md:34-35` — фиксация «никаких отдельных CREATE INDEX».
- `prompts/RepoModel.txt:138-140` — ожидаемое имя `users_email_key` для маппинга unique-violation в `domain.ErrEmailAlreadyTaken`.
- `.claude/plans/general_plan.md:47` — `POST /api/v1/auth/refresh` в API-контракте.
- `.claude/plans/general_plan.md:102` — пункт миграции `refresh_tokens`, статус `[ ]`.
- `.claude/plans/general_plan.md:107,109,111,166` — все остальные пункты с `refresh`, статус `[ ]`.
- `internal/auth/{domain,usecase,repository/postgres,transport/http}/.gitkeep` — пустые слои домена `auth`.
- `sqlc.yaml:1-2` — пустая конфигурация sqlc (`sql: []`).
- `Makefile:54-58` — команды `migrate-up`/`migrate-down`.
- `docker-compose.yml:1-18` — единственный сервис `postgres:16-alpine`.

## Архитектурные наблюдения

- **Текущее состояние двух подпунктов 1.2**: подпункт «индексы на email/username» закрыт миграцией `0002_users` (`UNIQUE` ⇒ автоматический b-tree); подпункт «миграция `refresh_tokens`» — ещё не реализован, никаких артефактов в репозитории нет.
- **Имена индексов как контракт**: `users_email_key` и `users_username_key` — не косметика, а часть контракта между БД-слоем и доменным слоем (`prompts/RepoModel.txt:138-140`). Любая будущая правка миграции, меняющая имена, потребует обновления маппинга unique-violation на доменные ошибки.
- **`citext` против функционального индекса**: case-insensitive уникальность `email` решена через тип столбца, а не функциональный индекс; это сохраняет дефолтное имя индекса и упрощает sqlc-маппинг (`citext` ↔ Go `string`).
- **Условный характер задачи `refresh_tokens`**: в плане формулировка — «если хранятся в БД» (`.claude/plans/general_plan.md:102`). Принципиальное решение «хранить ли refresh-токены в БД, и если да — какую структуру использовать» в репозитории не зафиксировано; нет ни ADR (`docs/refresh_tokens/03-decisions.md`), ни repo-model.
- **Зависимости миграции `refresh_tokens`**:
  - Внешние ключи: ожидаемая связь `user_id → users(id)` потребует, чтобы `0002_users` была применена раньше (что обеспечивается порядком `0002 < 0003`).
  - `pgcrypto` уже подключён (`0001_init`) — `gen_random_uuid()` доступен без новых `CREATE EXTENSION`.
  - `citext` нужен только для `email`; для refresh-токенов не требуется.
- **Слой**: миграция — внешний/инфраструктурный слой (`prompts/Architecture Layers.txt:60-65`); не зависит от `domain/`/`usecase/`/`transport/`. Это означает, что DDL для `refresh_tokens` можно ввести независимо от того, написан ли код в `internal/auth/usecase/` (1.3).
- **Соответствие плану 1.2 vs 1.3**: миграция `refresh_tokens` относится к 1.2 (схема БД), а usecase/handler/JWT-генерация — к 1.3 (домен auth). Текущая ветка по названию объединяет именно 1.2-часть (`tokens_and_index`).
