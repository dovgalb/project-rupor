---
parent: ./README.md
view: compliance
---

# 07 — Standards Compliance

Сводная таблица соответствия дизайна стандартам в `prompts/`.

| Стандарт | Статус | Ключевые точки compliance |
|----------|--------|----------------------------|
| `Architecture Layers.txt` | ✅ | Слои `domain ← usecase ← {transport, repository}` соблюдены. Никакой `room/usecase` или `channel/usecase` не импортирует чужой `domain`. Единственная санкционированная кросс-доменная зависимость (room/repo → channel/usecase для реализации `MembershipQuery`) явно зафиксирована в §D-07/D-19, документирована в `01-architecture.md` §«Граф зависимостей», и enforced в `arch_test.go` (`TestArchitecture_RoomRepoMayImplementChannelPort`, см. `04-testing.md`) |
| `Clean architecture.txt` | ✅ | Зависимости направлены строго внутрь. Интерфейсы (`RoomRepository`, `MembershipRepository`, `InviteRepository`, `InviteCodeGenerator`, `ChannelRepository`, `MembershipQuery`) объявлены в слое-потребителе (`usecase/ports.go`), реализации — в адаптерах |
| `Domain Model.txt` | ✅ | Rich domain: приватные поля, конструкторы с инвариантами (`NewRoom`, `NewMembership`, `NewInvite`, `NewChannel`), `Reconstruct*` для репозитория, бизнес-методы (`Membership.Promote/Demote/CanXxx`, `Invite.Revoke`, `Room.TransferOwnership`). Связи между сущностями — по id (`OwnerID`, `RoomID`), не вложенные. Доменные ошибки sentinel в `errors.go`. Никаких `context.Context` в методах domain |
| `Domain model test.txt` | ✅ | Domain-тесты в `package domain_test` (черный ящик), без I/O, без моков, табличные где это уменьшает дублирование (`TestMembership_Permissions_TableDriven`, `TestParseRole_TableDriven`, `TestParseChannelKind_TableDriven`). Покрытие: все ветки конструкторов, все доменные ошибки, граничные значения VO |
| `RepoModel.txt` | ✅ | См. полный чек-лист в `06-repo-model.md` §«Соответствие». Маппинг через `Reconstruct*`, два направления — две функции, ошибки БД мапятся в доменные (`pgx.ErrNoRows`, `23505`), nullable `revoked_at` через `pgtype.Timestamptz`, транзакции через `pgx.Tx + WithTx`. sqlc-структуры наружу `usecase/` не выходят |
| `Builder.txt` | ✅ | Builder-ы только в тестах (`internal/<domain>/domain/testing_builders_test.go` и в `package usecase_test`). Принимают `*testing.T`, `Build()` падает через `t.Fatalf`, дефолты валидные и детерминированные (фиксированный clock 2026-05-11). В продовом коде Builder отсутствует |
| `Go style.txt` | ✅ | gofmt + golangci-lint обязательны (Makefile). Имена пакетов — короткие, по делу: `httproom`, `httpchannel`, `roompg`, `channelpg`, `roomruntime`. Никаких `utils`/`common`. Конструкторы `NewXxx` возвращают `(*Xxx, error)` где есть инварианты. Ошибки оборачиваются `fmt.Errorf("...: %w", err)`, сравнение через `errors.Is`/`errors.As`. `context.Context` первым параметром, никогда не в структурах. Интерфейсы маленькие, в потребителе. SQL только sqlc, никакого ORM. `panic` в продовом коде нет |
| `Tests Style.txt` | ✅ | Уровни: domain → usecase (фейки) → repo (integration build-tag) → HTTP (httptest). `t.Parallel()` всегда. `t.Helper()` в хелперах. Нейминг `TestFunc_Scenario` / `TestType_Method_Scenario`. Время/UUID/rand инжектируются. Никаких `time.Sleep`, никаких глобальных env (только `t.Setenv`). Перед PR: `go test -race ./...`, нет `t.Skip` без обоснования (интеграционные `t.Skip` ссылаются на issue 1.5) |

## Уточнения / расхождения

### У-1. Компромисс «cross-domain repo → usecase»

`prompts/Architecture Layers.txt:73` запрещает: «одна доменная папка импортирует `transport/` или `repository/` другой доменной папки». Импорт чужого `usecase/` явно не упомянут — это серая зона.

В дизайне `internal/room/repository/postgres/membership_query.go` импортирует:
- `internal/channel/usecase` — ради интерфейса `MembershipQuery` и типа `RoleRequirement` (его реализуем);
- `internal/channel/domain` — ради error-sentinel'ов `ErrChannelAccessDenied`, `ErrChannelInsufficientRole` (их возвращаем).

Обоснование: реализация порта обязана знать сигнатуру порта; sentinel-ошибки — часть контракта порта. Это паттерн Hexagonal/Ports & Adapters: адаптер находится в инфраструктурном слое одной системы, реализует порт другой. Альтернатива (channel.usecase импортирует room.domain напрямую) хуже, потому что превращает channel-домен в зависимый от room.

Решение зафиксировано в `03-decisions.md` §D-07, риск — в §«Риски», enforced — в `arch_test.go` через явный `TestArchitecture_RoomRepoMayImplementChannelPort` (см. `04-testing.md`).

### У-2. Дублирование VO `UserID`/`RoomID` между доменами

`prompts/Architecture Layers.txt:69` запрещает domain-импорт за пределами stdlib + `uuid`. Поэтому общего пакета `pkg/idtypes` не делаем (его и так нельзя импортировать из domain).

Каждый домен (`auth`, `room`, `channel`) объявляет свой `UserID` (а `room` и `channel` — ещё и свой `RoomID`). Семантически это нормально: каждый домен ограничивает свой view идентификатора. На границе адаптеров (особенно `MembershipQueryAdapter`) преобразуем через сырой `uuid.UUID`.

Зафиксировано в §D-08 и §«Граф зависимостей» 01-architecture.md.

### У-3. `prompts/Architecture Layers.txt:120` фиксирует терминологию `usecase` (не `service`)

Соблюдаем: все папки и пакеты — `usecase`.

### У-4. Имена интерфейсов

`prompts/Go style.txt` (через research) требует: интерфейсы по роли, без префикса `I`. Соблюдаем: `RoomRepository`, `MembershipRepository`, `InviteRepository`, `InviteCodeGenerator`, `ChannelRepository`, `MembershipQuery`.

### У-5. DTO

`prompts/Architecture Layers.txt:122-123`:
- Use case DTO: `RegisterUserInput`/`Output` — соблюдаем (`CreateRoomInput`/`CreateRoomOutput`, …).
- Транспортные DTO: `RegisterUserRequest`/`Response` — соблюдаем (`createRoomRequest`/`createRoomResponse`, приватные в `dto.go`).

### У-6. Документация фичи

Структура `docs/2_1_rooms_and_channels/` соответствует шаблону, зафиксированному в `docs/1_4_auth_middleware/README.md:58-72`. Файл `05-events.md` сознательно не создаётся (см. README §Out of scope) — нет доменных событий с подписчиками в PR-2.

### У-7. Один Postgres-конфиг под несколько доменов

Прецедента в кодовой базе нет (sqlc.yaml содержит одну запись для auth). Дизайн расширяет массив `sql:` на три записи (auth, room, channel). Это поддерживаемый паттерн sqlc v2 и не требует изменения Makefile (`make sqlc` обрабатывает все блоки). Зафиксировано в §D-15 и `06-repo-model.md` §«sqlc.yaml — расширение».
