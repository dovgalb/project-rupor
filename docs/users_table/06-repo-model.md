---
parent: ./README.md
view: repository
---

# Repo Model — таблица `users`

Документ описывает схему таблицы `users`, которую создаёт миграция `0002_users`, и фиксирует, как она будет маппиться на будущий домен `auth` (задача 1.3 — вне scope этой фичи).

## DDL (целевая схема)

```sql
-- migrations/0002_users.up.sql

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

```sql
-- migrations/0002_users.down.sql

DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS citext;
```

## Сводка по столбцам

| Столбец | Тип | NULL | Default | Ограничения | Назначение |
|---------|-----|------|---------|-------------|------------|
| `id` | `uuid` | NOT NULL | `gen_random_uuid()` | PRIMARY KEY (`users_pkey`) | Стабильный идентификатор пользователя |
| `email` | `citext` | NOT NULL | — | UNIQUE (`users_email_key`) | Email для входа; уникальность case-insensitive |
| `password_hash` | `text` | NOT NULL | — | — | Хеш пароля (bcrypt в задаче 1.3) |
| `username` | `text` | NOT NULL | — | UNIQUE (`users_username_key`), CHECK длины 3..32 (`users_username_length_check`) | Отображаемое имя пользователя |
| `created_at` | `timestamptz` | NOT NULL | `now()` | — | Момент регистрации (UTC) |

## Имена объектов БД (фиксируется этой фичей)

| Объект | Имя | Тип |
|--------|-----|-----|
| Таблица | `users` | table |
| Первичный ключ | `users_pkey` | constraint (PK) |
| Уникальность email | `users_email_key` | constraint (UNIQUE) |
| Уникальность username | `users_username_key` | constraint (UNIQUE) |
| Длина username | `users_username_length_check` | constraint (CHECK) |
| Индекс под `users_email_key` | `users_email_key` | b-tree (создаётся автоматически под UNIQUE) |
| Индекс под `users_username_key` | `users_username_key` | b-tree (создаётся автоматически под UNIQUE) |

`users_email_key` — это имя, которое явно ожидается в маппинге unique-violation на `domain.ErrEmailAlreadyTaken` (`prompts/RepoModel.txt:138-140`).

## Соответствие `prompts/RepoModel.txt` (задача 1.3 — для справки)

Маппинг ниже носит информационный характер и фиксирует контракт между этой миграцией и будущим репозиторием `internal/auth/repository/postgres/`. В рамках задачи 1.2 Go-код не пишется.

| Поле будущего домена `auth.User` | Столбец БД | Конверсия (планируемая в 1.3) |
|----------------------------------|------------|-------------------------------|
| `id domain.UserID` (VO над `uuid.UUID`) | `id uuid` | `domain.NewUserID(row.ID)` ↔ `user.ID().UUID()` |
| `email domain.Email` (VO) | `email citext` | `string(row.Email)` → `domain.NewEmail(...)`; `email.String()` → `citext`-параметр |
| `passwordHash domain.PasswordHash` (или `string`) | `password_hash text` | прямое присваивание; домен принимает уже захешированное значение от `PasswordHasher` |
| `username domain.Username` (VO) | `username text` | `domain.NewUsername(row.Username)` ↔ `username.String()` |
| `createdAt time.Time` | `created_at timestamptz` | прямое значение (`row.CreatedAt` уже в `time.Time` UTC) |

Поля `status` и `deleted_at` не предусмотрены — см. `03-decisions.md`, решения 8–9.

## Соответствие стандартам

| Стандарт | Применимость к миграции | Статус |
|----------|-------------------------|--------|
| `prompts/Architecture Layers.txt` | Миграция = внешний слой (`Architecture Layers.txt:60-65`); не импортирует ничего из `domain/`/`usecase/`/`transport/`. | ✅ |
| `prompts/RepoModel.txt` | Имена ограничений согласованы с маппингом ошибок БД → доменные ошибки (`RepoModel.txt:138-140`). | ✅ |
| `prompts/Clean architecture.txt` | DDL не нарушает направление зависимостей (миграция — самая внешняя точка). | ✅ N/A на DDL |
| `prompts/Domain Model.txt`, `Builder.txt`, `Go style.txt`, `Tests Style.txt`, `Domain model test.txt` | Применимы к Go-коду; миграция — чистый SQL. | N/A |

## sqlc Queries

В рамках задачи 1.2 запросов sqlc **не добавляем**. `sqlc.yaml` (`sqlc.yaml:1-2`) остаётся пустым. Настройка `sqlc.yaml` и SQL-запросы (`InsertUser`, `GetUserByID`, `GetUserByEmail`) — часть задачи 1.3 «Домен auth» (`general_plan.md:110`).
