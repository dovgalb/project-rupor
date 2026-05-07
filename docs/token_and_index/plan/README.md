---
date: 2026-05-07
feature: token_and_index
design: ../README.md
status: draft
---

# План кода: token_and_index

## Overview

Реализуется ровно одна вещь — миграция `0003_refresh_tokens` (up + down). Никакого Go-кода, никаких изменений в `sqlc.yaml`, `cmd/server/main.go` или `internal/`. Целевая схема и решения по типам — в `../06-repo-model.md` и `../03-decisions.md`.

Подпункт «индексы на email/username» из `.claude/plans/general_plan.md:103` уже закрыт миграцией `0002_users` и в эту фичу не входит (см. `../README.md`, «Бизнес-контекст»).

## Phase Strategy

**Bottom-up, одна фаза.** Тот же подход, что и в `docs/users_table/plan/README.md:14-16`. Миграция — фундаментальный артефакт самого нижнего слоя инфраструктуры (внешний слой по `prompts/Architecture Layers.txt:60-65`). Делить на несколько фаз нечего: up и down — пара, которая верифицируется одной операцией (`migrate-up` затем `migrate-down`).

## Phases

| # | Фаза | Слой | Зависимости | Status |
|---|------|------|-------------|--------|
| 1 | [Создать миграцию `0003_refresh_tokens`](./phase-01.md) | migrations | `0001_init` (`pgcrypto`) и `0002_users` (таблица `users`) уже накатаны | ☐ |

## File Map

### New Files

- `migrations/0003_refresh_tokens.up.sql` — `CREATE TABLE refresh_tokens (...)` со всеми ограничениями + `CREATE INDEX refresh_tokens_user_id_idx`. Полный DDL см. `../06-repo-model.md`, раздел «DDL (целевая схема)».
- `migrations/0003_refresh_tokens.down.sql` — `DROP TABLE IF EXISTS refresh_tokens` (FK и индексы уходят вместе с таблицей; никаких расширений эта миграция не подключает).

### Modified Files

Нет. Composition root, конфигурация, `sqlc.yaml`, доменные пакеты, `Makefile` — не трогаем.

## DI Integration

Не применимо. Миграция применяется внешним инструментом (`golang-migrate`) во время `make migrate-up` / `make init-project` (`Makefile:54-58,69-90`). Цепочка инициализации Go-приложения (`cmd/server/main.go`) в этой фазе не меняется.

## Error Codes

Не применимо. Миграция не порождает доменных ошибок и не публикует HTTP-кодов. Маппинг unique-violation `refresh_tokens_token_hash_key` → доменная ошибка — задача 1.3 (`.claude/plans/general_plan.md:107-110`).

## Success Criteria

- [ ] `make migrate-up` на чистой БД (после `make dc-up`) проходит без ошибок и доходит до `0003_refresh_tokens`.
- [ ] `psql -c "\d refresh_tokens"` показывает таблицу со столбцами `id, user_id, token_hash, expires_at, created_at, revoked_at` ровно тех типов, что в `../06-repo-model.md` (раздел «Сводка по столбцам»).
- [ ] `psql -c "\d refresh_tokens"` показывает ограничения с именами `refresh_tokens_pkey`, `refresh_tokens_user_id_fkey`, `refresh_tokens_token_hash_key`, `refresh_tokens_token_hash_length_check`.
- [ ] `psql -c "\d refresh_tokens"` показывает индексы `refresh_tokens_pkey`, `refresh_tokens_token_hash_key`, `refresh_tokens_user_id_idx` (ровно три, без партиальных).
- [ ] FK `refresh_tokens_user_id_fkey` имеет действие `ON DELETE CASCADE` (проверяется через `pg_constraint.confdeltype = 'c'`).
- [ ] `make migrate-down` удаляет таблицу `refresh_tokens` целиком (вместе с её индексами и FK); состояние БД возвращается к состоянию после `0002_users`. Расширения `pgcrypto` и `citext` остаются.
- [ ] Повторный `make migrate-up` проходит идемпотентно.
- [ ] `git diff --name-only` показывает изменения только в `migrations/0003_refresh_tokens.up.sql`, `migrations/0003_refresh_tokens.down.sql` и в документах `docs/token_and_index/`.
- [ ] Все 14 критериев приёмки из `../README.md` выполнены.
