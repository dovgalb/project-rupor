---
parent: ./README.md
view: decision
---

# 03 — Decisions (Decision View)

ADR + риски + open questions.

## Решения

| # | Решение | Выбор | Рассмотренные альтернативы | Обоснование |
|---|---------|-------|-----------------------------|-------------|
| D-01 | Структура документации | Один комплект `docs/2_1_rooms_and_channels/` на оба домена | Раздельные `2_1_rooms` + `2_2_channels` | Согласовано с пользователем: единая фича PR-2, общие sequences (join → list channels) проще читать в одном комплекте |
| D-02 | Хранение ролей в БД | `text` колонка с `CHECK(role IN ('owner','admin','member'))` | Postgres ENUM-тип `room_role` | В кодовой базе нет прецедента ENUM (см. `migrations/0002_users.up.sql`, `0003_refresh_tokens.up.sql`); CHECK проще мигрировать (расширение значений = новая миграция без `ALTER TYPE`); парсинг в домен через `ParseRole(string)` уже спроектирован одинаково для обоих вариантов |
| D-03 | Хранение типа канала | `text` колонка с `CHECK(kind IN ('text','voice'))` | Postgres ENUM `channel_kind` | Те же причины, что в D-02; добавление `announcement` или `forum` будет миграцией текста, а не `ALTER TYPE` |
| D-04 | Гарантия одного активного инвайта на комнату | Partial unique index `CREATE UNIQUE INDEX invites_active_room ON invites(room_id) WHERE revoked_at IS NULL` + транзакционный `RegenerateActive` | Только app-level контроль (без БД-инварианта) | БД-инвариант защищает от race-условий между параллельными `RegenerateInvite` (см. UC-R6 edge cases в `02-behavior.md`); код RegenerateActive остаётся честно атомарным |
| D-05 | Глобальная уникальность активного инвайт-кода | Partial unique index `invites_active_code ON invites(code) WHERE revoked_at IS NULL` | Без уникальности: разрешить коллизии и решать ambiguity на чтении | Иначе `JoinByCode` пришлось бы как-то выбирать одну из нескольких комнат с одинаковым активным кодом, либо отвергать. Partial unique гарантирует, что активный код однозначен |
| D-06 | Каскадное удаление | `ON DELETE CASCADE` на FK для `room_members.room_id`, `invites.room_id`, `channels.room_id` | Soft-delete, либо ручной DELETE в репозитории | `auth/refresh_tokens` уже использует `ON DELETE CASCADE` для `user_id` (`migrations/0003_refresh_tokens.up.sql:7`) — следуем установленному паттерну. Для MVP soft-delete переусложнение |
| D-07 | Channel — отдельный домен с портом MembershipQuery | `internal/channel/` с собственным `usecase/ports.go::MembershipQuery`; реализация — `internal/room/repository/postgres/membership_query.go` | (a) Channel как часть room-агрегата; (b) channel.usecase напрямую импортирует room.domain | Согласовано с пользователем. Соответствует уже существующей структуре `internal/channel/` (см. research). Единственная кросс-доменная связь — реализация порта в room (Adapter pattern), импорт чужого `usecase` из своего `repository` допустим (см. §D-08) |
| D-08 | Дублирование `UserID`/`RoomID` value-object между `auth/domain`, `room/domain`, `channel/domain` | Каждый домен объявляет свой VO — обёртку над `uuid.UUID` | Общий `pkg/idtypes/userid.go` | По `prompts/Architecture Layers.txt:69` `domain/` импортирует только stdlib + `uuid`. `pkg/...` запрещён. Дублирование в три файла по 16 строк — приемлемая цена за честную изоляцию |
| D-09 | `ListMembers` отдаёт только `userId`/`role`/`joinedAt`, без username/email | Без JOIN на `auth.users` | Кросс-доменный JOIN либо хранение denormalized username в `room_members` | Кросс-доменный JOIN нарушил бы изоляцию room-репозитория. Denormalization создаёт rasписание `auth.users.username` ↔ `room_members.cached_username`. Профили будут отдаваться отдельным эндпоинтом `internal/user/` (фаза 5) |
| D-10 | Префикс кодов ошибок per-domain | `ROOM-001..009`, `CHANNEL-001..007` (с возможностью `errors.Is(err, ErrChannelAccessDenied)` без импорта room.domain) | Единый префикс `RUPOR-NNN`, либо channel переиспользует `ROOM-NNN` | Auth уже использует `AUTH-NNN` — установленный паттерн. Domain-isolation: каждый домен мапит ТОЛЬКО свои ошибки, не импортирует чужой error_mapper. Adapter `MembershipQueryAdapter` транслирует `room.domain.ErrNotMember` → `channel.domain.ErrChannelAccessDenied` на границе |
| D-11 | Транзакция `CreateRoom` (room + owner-membership) | `RoomRepository.SaveWithOwner(ctx, room, ownerMembership)` — транзакция инкапсулирована в репозитории; принимает `*pgxpool.Pool` | (a) Отдельный `UnitOfWork` интерфейс в usecase; (b) Две отдельные операции без транзакции | Следуем паттерну `RefreshTokenRepository.Rotate` (`internal/auth/repository/postgres/refresh_token_repository.go:50-86`). Без транзакции возможен «осиротевший» room без owner — нарушение инварианта системы (комната без владельца) |
| D-12 | Транзакция `RegenerateInvite` | `InviteRepository.RegenerateActive(ctx, roomID, code, createdBy, now)` — транзакция в репозитории; принимает `*pgxpool.Pool` | App-level: `Revoke` потом `Save` без транзакции | Без транзакции в окне между UPDATE и INSERT можно прочитать состояние «нет активного кода». Транзакция + partial unique index — двойная защита |
| D-13 | Идемпотентность `RegenerateInvite` | Один эндпоинт `POST /rooms/:id/invite` отвечает за «выпустить» и «отозвать-перевыпустить» | Два эндпоинта: `POST /invite` и `DELETE /invite` | UI-первоначальная задача — «получить ссылку». Семантика «отозвать» в MVP — это просто перевыпуск (старый невалиден). Один эндпоинт упрощает фронт |
| D-14 | Формат инвайт-кода | 8 символов, Crockford base32 (32 символа без `I/L/O/U`), upper-case | UUID, nanoid, bcrypt-подобный | Согласовано с пользователем. ~10^12 пространство достаточно для MVP, читается голосом, не путается визуально |
| D-15 | Sqlc layout | Один общий `sqlc.yaml`, массив `sql:` с тремя записями: auth, room, channel | (a) Один общий `queries/` каталог; (b) Несколько отдельных `sqlc.yaml` | sqlc v2 поддерживает массив `sql:` нативно. Изоляция по доменам: `internal/<domain>/repository/postgres/queries/`, генерация в `internal/<domain>/repository/postgres/db/` |
| D-16 | Уникальность имени канала в комнате | `UNIQUE(room_id, name)` | Без уникальности (Discord-стиль допускает дубликаты) | Уникальность резко упрощает UX (нет двух «#general» в одной комнате) и доменные инварианты тестов; снять ограничение позже легче, чем ввести |
| D-17 | TransferOwnership как доменный метод без HTTP | Метод `Room.TransferOwnership(...)` существует, но эндпоинта нет | Не добавлять метод вообще | Rich domain — методы должны выражать бизнес-операции, даже если транспорт временно их не вызывает. Это упрощает добавление эндпоинта в будущей фазе и тестируется в domain-тестах. Цена — две лишние строки в entity |
| D-18 | Хранение `revoked_at` для инвайта | `timestamptz NULL` (zero == активен) | Отдельный bool `is_active` | Совпадает с паттерном `auth.refresh_tokens.revoked_at` (`migrations/0003_refresh_tokens.up.sql:9`). NULL/время совмещают флаг и аудит-метку в одном поле |
| D-19 | Маппинг `ErrChannelAccessDenied` ↔ `ErrChannelInsufficientRole` | Адаптер `MembershipQueryAdapter` транслирует `room.domain.ErrNotMember` → `channel.domain.ErrChannelAccessDenied`, а недостаток роли (`!CanXxx`) → `ErrChannelInsufficientRole` | Channel.usecase различает только «доступ запрещён» одной ошибкой | Раздельные коды ошибок (CHANNEL-006/007) дают фронту понять «не состоите в комнате» vs «состоите, но не админ» — это критично для UX |
| D-20 | Архитектурные тесты для room/channel | Расширить `arch_test.go` на `internal/room/...` и `internal/channel/...` (новые `Test*` функции) | Не enforce'ить — полагаться на ревью | `arch_test.go` уже enforce'ит правила для auth (`arch_test.go:61-201`). Новые домены без тех же правил быстро дрейфуют |

## Риски и митигация

| Риск | Влияние | Митигация |
|------|---------|-----------|
| Гонка двух `RegenerateInvite` создаст две активные строки в `invites` | High | Partial unique index `invites_active_room` + транзакция `RegenerateActive` (D-04, D-12) |
| Коллизия 8-символьных кодов при росте числа активных инвайтов | Low | На MVP-объёмах (<10^4 активных) вероятность коллизии ~10^-8 при первом броске. До 3 ретраев в `RegenerateInvite`. Если в будущем вырастет — добавить 10-символьную версию или сменить алфавит |
| Кросс-доменный импорт `room/repo → channel/usecase` нарушит arch_test | High | Явно прописать исключение в новом `TestArchitecture_RoomRepoMayImplementChannelPort` (см. `04-testing.md`). Документировать в `01-architecture.md` §«Граф зависимостей» |
| Owner случайно удаляет комнату → каскад сносит каналы и инвайты без подтверждения | Medium | API возвращает `204` — фронт обязан показывать confirm-диалог. На бэке двойного подтверждения не вводим (MVP). Зафиксировано в `08-api-contract.md` |
| Передача невалидного `roomID` в `/rooms/{roomID}/channels/{channelID}` (UUID валидный, но комната не существует) | Low | `INSERT` упадёт по FK violation → `INTERNAL 500`. На MVP допустимо. Улучшение — preflight-check `roomID exists` (доп. SQL); отложено |
| `internal/user/` пуст, `ListMembers` отдаёт «голые» userId | Medium | Документировано в D-09. Фронт обращается к auth-эндпоинту `/auth/me` для своего профиля; список профилей других пользователей появится в фазе 5 |
| Зависимость PR-2 от непокрытых интеграционных тестов фазы 1.5 | Low | Интеграционные `t.Skip(... see issue 1.5)` уже есть в auth. PR-2 включит свои интеграционные тесты под тем же build-tag, активацию руками `make test-integration` (см. `04-testing.md`) |
| Channel.domain параллельно объявляет `RoomID` — возможна путаница в коде | Low | Тесты domain-уровня проверяют `roomID.UUID()` round-trip; компилятор заставит явно конвертировать между `room.RoomID` и `channel.RoomID` через `uuid.UUID` (нет неявного приведения). Это минимальная цена за изоляцию |

## Open Questions

- [x] **Q1.** Slug папки документации? — **A:** `2_1_rooms_and_channels` (D-01).
- [x] **Q2.** Формат инвайт-кодов? — **A:** 8 символов Crockford base32, multi-use, без TTL, owner может revoke (D-13, D-14).
- [x] **Q3.** Полномочия `admin`? — **A:** всё кроме DELETE room и transfer ownership.
- [x] **Q4.** Channel — отдельный домен или вложенный? — **A:** отдельный с портом MembershipQuery (D-07).
- [ ] **Q5.** Стоит ли добавить `kick member` эндпоинт уже в PR-2? Доменно метод `Membership.CanKick` есть. Текущий план — отложить эндпоинт до фазы модерации; домен и тесты на kick не делаем (out of scope).
- [ ] **Q6.** Приватные каналы / role-overrides внутри канала? Out of scope MVP — все каналы публичны для всех members.
- [ ] **Q7.** Лимиты на количество комнат на пользователя / каналов в комнате? Не вводим в PR-2; добавим позже на основании метрик.
