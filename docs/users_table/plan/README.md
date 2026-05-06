---
date: 2026-05-05
feature: users_table
design: ../README.md
status: draft
---

# План кода: users_table

## Overview

Реализуется ровно одна вещь — миграция `0002_users` (up + down). Никакого Go-кода, никаких изменений в `sqlc.yaml`, `cmd/server/main.go` или `internal/`. Целевая схема и решения по типам — в `../06-repo-model.md` и `../03-decisions.md`.

## Phase Strategy

**Bottom-up, одна фаза.** Миграция — фундаментальный артефакт самого нижнего слоя инфраструктуры (внешний слой по `prompts/Architecture Layers.txt:60-65`). Делить на несколько фаз нечего: up и down — пара, которая верифицируется одной операцией (`migrate-up` затем `migrate-down`).

## Phases

| # | Фаза | Слой | Зависимости | Status |
|---|------|------|-------------|--------|
| 1 | [Создать миграцию `0002_users`](./phase-01.md) | migrations | `0001_init` (`pgcrypto`) уже накатан | ☐ |

## File Map

### New Files

- `migrations/0002_users.up.sql` — `CREATE EXTENSION citext` + `CREATE TABLE users (...)` со всеми ограничениями; полный DDL см. `../06-repo-model.md`.
- `migrations/0002_users.down.sql` — `DROP TABLE users` + `DROP EXTENSION citext`.

### Modified Files

Нет. Composition root, конфигурация, `sqlc.yaml`, доменные пакеты, `Makefile` — не трогаем.

## DI Integration

Не применимо. Миграция применяется внешним инструментом (`golang-migrate`) во время `make migrate-up`/`make init-project`. Цепочка инициализации Go-приложения (`cmd/server/main.go`) в этой фазе не меняется.

## Error Codes

Не применимо. Миграция не порождает доменных ошибок и не публикует HTTP-кодов. Маппинг unique-violation `users_email_key` → `domain.ErrEmailAlreadyTaken` — задача фазы 1.3 (`.claude/plans/general_plan.md:110`).

## Success Criteria

- [ ] `make migrate-up` на чистой БД (после `make dc-up`) проходит без ошибок и доходит до `0002_users`.
- [ ] `psql -c "\d users"` показывает таблицу `users` со столбцами `id, email, password_hash, username, created_at` ровно тех типов, что в `../06-repo-model.md`.
- [ ] `psql -c "\d users"` показывает ограничения с именами `users_pkey`, `users_email_key`, `users_username_key`, `users_username_length_check`.
- [ ] `make migrate-down` удаляет таблицу `users` и расширение `citext`; состояние БД возвращается к состоянию после `0001_init`.
- [ ] Повторный `make migrate-up` проходит идемпотентно.
- [ ] `git diff` показывает изменения только в `migrations/0002_users.up.sql`, `migrations/0002_users.down.sql` и в документах `docs/users_table/`.
- [ ] Все критерии приёмки из `../README.md` выполнены.
