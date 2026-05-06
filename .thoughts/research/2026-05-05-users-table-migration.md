---
date: 2026-05-05
researcher: Claude
commit: d971527
branch: feature/PR-1_2-users-table
research_question: "Миграция: таблица users (id, email, password_hash, username, created_at) — текущее состояние кодовой базы по этой задаче"
---

# Исследование: подготовка к миграции таблицы `users`

## Резюме

Задача из плана MVP (`.claude/plans/general_plan.md:101`) — добавить миграцию для таблицы `users` с полями `id`, `email`, `password_hash`, `username`, `created_at`. На момент исследования миграция **не реализована**: в каталоге `migrations/` существует только одна миграция `0001_init`, которая включает расширение `pgcrypto` и не создаёт никаких таблиц. Каталоги `internal/auth/` и `internal/user/` инициализированы только файлами `.gitkeep` — кода домена/usecase/repository/transport ещё нет.

Инфраструктура для добавления миграции и последующей генерации Go-кода уже на месте: `golang-migrate` подключён через `Makefile` (`migrate-up`/`migrate-down`), `sqlc` подключён через `sqlc.yaml` (но конфигурация пуста: `sql: []`), PostgreSQL поднимается через `docker-compose.yml`. Конвенции по именованию миграций (`NNNN_name.up.sql` / `NNNN_name.down.sql`), маппингу repo→domain и Clean Architecture зафиксированы в `prompts/Architecture Layers.txt` и `prompts/RepoModel.txt`.

Текущая ветка `feature/PR-1_2-users-table` (HEAD `d971527`) — задача 1.2 из плана. Следующий свободный номер миграции — `0002`.

## Детальные результаты

### 1. Существующие миграции

- **Расположение**: `migrations/0001_init.up.sql`, `migrations/0001_init.down.sql`
- **Содержимое up**:
  ```sql
  -- 0001_init.up.sql
  -- Initial migration: enable pgcrypto for gen_random_uuid().
  CREATE EXTENSION IF NOT EXISTS pgcrypto;
  ```
- **Содержимое down**:
  ```sql
  -- 0001_init.down.sql
  -- Rollback: drop pgcrypto extension.
  DROP EXTENSION IF EXISTS pgcrypto;
  ```
- **Описание**: единственная существующая миграция. Подключает `pgcrypto` ради `gen_random_uuid()` — это намёк, что первичные ключи в проекте задумываются как `uuid` со значением по умолчанию `gen_random_uuid()`.
- **Конвенции имени**: формат `NNNN_<slug>.up.sql` / `NNNN_<slug>.down.sql`, четырёхзначный префикс с ведущими нулями. Следующий номер — `0002`.
- **Других SQL-файлов в репозитории нет** (поиск `find . -name "*.sql"` возвращает только эти два файла).

### 2. Инструменты миграций (`Makefile`)

- **Расположение**: `Makefile:54-58`
- **Команды**:
  ```make
  migrate-up:
      migrate -path migrations -database "$(DATABASE_URL)" up

  migrate-down:
      migrate -path migrations -database "$(DATABASE_URL)" down 1
  ```
- **Зависимости**: бинарник `golang-migrate` (проверяется в `init-project`, `Makefile:78`). `DATABASE_URL` берётся из `.env` или дефолта `postgres://rupor:rupor@localhost:5432/rupor?sslmode=disable` (`Makefile:9`).
- **Поток**: `make migrate-up` → читает все файлы в `migrations/` → накатывает в порядке номеров. `make migrate-down` откатывает только последнюю миграцию (`down 1`).
- **Init-flow**: `init-project` (`Makefile:69-90`) поднимает Docker, ждёт PostgreSQL и автоматически вызывает `migrate-up` при первоначальной настройке проекта.

### 3. Состояние `sqlc`

- **Расположение конфига**: `sqlc.yaml`
- **Содержимое** целиком:
  ```yaml
  version: "2"
  sql: []
  ```
- **Описание**: sqlc подключён на уровне версии конфига, но **ни одного блока `sql`** не объявлено. Это значит:
  - Каталог с SQL-запросами под sqlc не создан (поиск каталогов `queries` пуст).
  - Сгенерированных Go-структур (`db.User`, `db.Queries`) пока не существует.
  - Чтобы после добавления миграции вызывать `make sqlc` (`Makefile:60-61`), потребуется заполнить `sql:` блоком (schema/queries/engine/codegen).

### 4. Доменная структура (текущее наполнение)

- **`internal/auth/`**: только пустые подкаталоги:
  - `internal/auth/domain/.gitkeep`
  - `internal/auth/usecase/.gitkeep`
  - `internal/auth/repository/postgres/.gitkeep`
  - `internal/auth/transport/http/.gitkeep`
- **`internal/user/`**: симметрично, только `.gitkeep`-файлы:
  - `internal/user/domain/.gitkeep`
  - `internal/user/usecase/.gitkeep`
  - `internal/user/repository/postgres/.gitkeep`
  - `internal/user/transport/http/.gitkeep`
- **Описание**: ни доменных сущностей `User`, ни value objects (`Email`, `Password`, `Username`), ни репозиториев, ни хендлеров пока не реализовано. Плейсхолдеры подтверждают, что слои разнесены между двумя доменами `auth` и `user` (как и предусмотрено `internal/` в `CLAUDE.md`).

### 5. Подключение к БД и окружение

- **`docker-compose.yml`**: единственный сервис — `postgres:16-alpine` (`docker-compose.yml:1-18`). Дефолтные креды `rupor/rupor`, БД `rupor`, порт 5432, healthcheck через `pg_isready`. **Сервис Redis в текущем compose-файле отсутствует** (план/CLAUDE.md упоминают Redis в перспективе, но в compose его нет).
- **`.env.example`**:
  ```
  DATABASE_URL=postgres://rupor:rupor@localhost:5432/rupor?sslmode=disable
  JWT_SECRET=change-me-in-prod
  SERVER_PORT=8080
  ```
- **Описание**: одна и та же `DATABASE_URL` используется и для миграций (`Makefile:54`), и для приложения. Имя БД фиксировано — `rupor`.

### 6. Конвенции, относящиеся к будущей таблице `users`

Источники: `prompts/Architecture Layers.txt`, `prompts/RepoModel.txt`, `CLAUDE.md`.

- **Слой и расположение SQL**: миграции — внешний слой (`prompts/Architecture Layers.txt:60-65`); живут в `migrations/`. SQL-запросы под sqlc — в адаптере `internal/<домен>/repository/postgres/`.
- **ORM запрещён** (`prompts/Architecture Layers.txt:107`): доступ к БД только через sqlc.
- **Идентификаторы**: в домене — value object `UserID`, в БД — `uuid.UUID` (`prompts/RepoModel.txt:118-122`). Совместимо с `gen_random_uuid()` из миграции `0001_init`.
- **Email**: в домене — value object `domain.Email`, в БД — обычный `string` (`prompts/RepoModel.txt:90-93`).
- **Пример имени уникального ограничения**: `users_email_key` упоминается как ожидаемое имя constraint для маппинга `unique violation → domain.ErrEmailAlreadyTaken` (`prompts/RepoModel.txt:138-140`). Это PostgreSQL-дефолт для `UNIQUE` по столбцу `email`.
- **Поля в примере репо-модели** (`prompts/RepoModel.txt:46-52`) для `userRow`: `id uuid.UUID`, `email string`, `username string`, `createdAt time.Time`, `status string`. Поле `status` упомянуто только в примере маппинга — в плане MVP (`general_plan.md:101`) не фигурирует, требуется в будущей таблице/нет — на момент исследования не зафиксировано.
- **План явно требует** (`general_plan.md:101-103`):
  - Миграцию `users` с полями `id, email, password_hash, username, created_at`.
  - Отдельную миграцию `refresh_tokens` (отдельная задача).
  - Индексы на `email` / `username`.
- **Хеширование паролей**: bcrypt предусмотрен задачей `1.3` (`general_plan.md:108`). Для SQL это означает, что `password_hash` хранится как строка хеша.

### 7. Состояние ветки

- Текущая ветка: `feature/PR-1_2-users-table`.
- HEAD: `d971527` — мерж PR `feature/1-project-structure` в `main` (последний завершённый этап — инфраструктура).
- Незакоммиченные изменения на момент старта сессии: `.claude/plans/general_plan.md` (модификация), новый каталог `tasks/`, файл `test.http` (untracked).

## Ссылки на код

- `migrations/0001_init.up.sql:1-4` — единственная существующая миграция (pgcrypto).
- `migrations/0001_init.down.sql:1-3` — её откат.
- `Makefile:54-58` — команды `migrate-up` / `migrate-down`.
- `Makefile:60-61` — команда `sqlc generate`.
- `Makefile:9` — дефолтный `DATABASE_URL`.
- `sqlc.yaml:1-2` — пустая конфигурация sqlc (`sql: []`).
- `docker-compose.yml:1-18` — единственный сервис postgres:16-alpine.
- `.env.example:1-8` — переменные окружения.
- `internal/auth/{domain,usecase,repository/postgres,transport/http}/.gitkeep` — пустые слои домена `auth`.
- `internal/user/{domain,usecase,repository/postgres,transport/http}/.gitkeep` — пустые слои домена `user`.
- `.claude/plans/general_plan.md:99-103` — задача «Схема БД» с пунктом про таблицу `users`.
- `prompts/Architecture Layers.txt:60-65,107` — расположение миграций и запрет ORM.
- `prompts/RepoModel.txt:46-52` — пример полей `userRow`.
- `prompts/RepoModel.txt:118-122` — конвенция id (uuid в БД, value object в домене).
- `prompts/RepoModel.txt:138-140` — ожидаемое имя constraint `users_email_key`.

## Архитектурные наблюдения

- **Миграции и sqlc разделены**: `migrations/` — источник схемы для `golang-migrate`; `sqlc` (когда будет настроен) будет читать схему отдельно через `schema:` в `sqlc.yaml`. На данный момент sqlc-конфиг ещё не настроен, поэтому добавление миграции `0002_users` само по себе не повлечёт автоматической генерации Go-кода.
- **Поток применения миграции**: `make dc-up` (Postgres) → `make migrate-up` → таблица создаётся в БД `rupor`. Откат — `make migrate-down`.
- **Зависимости от `0001_init`**: расширение `pgcrypto` доступно, поэтому `id uuid PRIMARY KEY DEFAULT gen_random_uuid()` будет работать без дополнительных миграций.
- **Слой**: SQL DDL (миграция) — внешний/инфраструктурный слой; не зависит ни от `domain/`, ни от `usecase/`. Бизнес-инварианты (формат email, длина username) валидируются в `domain/` value objects, но их базовое отражение на уровне БД (например, `UNIQUE`, `NOT NULL`) — задача миграции.
- **Соответствие плану**: на ветке `feature/PR-1_2-users-table` ожидается одна логически связанная единица работы — миграция `0002_users` (up/down) с полями из `general_plan.md:101` и индексами из `general_plan.md:103`. Задача `refresh_tokens` (`general_plan.md:102`) — отдельный пункт, в текущий research-вопрос не входит.
