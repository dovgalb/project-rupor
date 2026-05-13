---
date: 2026-05-12
researcher: Claude
commit: 60bb055
branch: PR-3_1-realtime_chat
research_question: "PR-3_1 — Реалтайм-чат: что уже есть в кодовой базе для миграции `messages`, `pkg/websocket/`, WS-эндпоинта `/api/v1/ws?token=<jwt>`, домена `chat`, REST-пагинации `/channels/:id/messages` и проверки прав. Цель — задокументировать текущее состояние, без рекомендаций."
---

# Исследование: PR-3_1 — Реалтайм-чат (baseline кодовой базы)

## Резюме

Кодовая база к коммиту `60bb055` (ветка `PR-3_1-realtime_chat`, после merge PR-2) полностью завершила фазы 1 и 2 общего плана: домены `auth`, `room`, `channel` с тестами, миграциями `0001-0007`, HTTP-роутами под `/api/v1/`, JWT-аутентификацией и middleware-стеком. Фаза 3 ещё не начата — пакет `pkg/websocket/` содержит только `.gitkeep` (`pkg/websocket/.gitkeep`), миграция `0008_messages` отсутствует, домен `chat` не создан.

Для реализации задач фазы 3 уже готовы: (а) интерфейс `usecase.TokenIssuer.VerifyAccess(token, now)` (`internal/auth/usecase/ports.go:29-32`) — переиспользуется для парсинга JWT из query-параметра; (б) кросс-доменный порт `channel/usecase.MembershipQuery.Require(ctx, roomID, userID, RoleRequirement)` (`internal/channel/usecase/ports.go:33-35`) с реализацией `MembershipQueryAdapter` в `internal/room/repository/postgres/membership_query.go:29-53` — переиспользуется для проверки прав на чтение/отправку сообщений в канале; (в) helpers `WithUserID`/`UserIDFromContext` (`internal/auth/transport/http/middleware/contextkeys.go`); (г) общие middleware RequestID/Recover/Logger/CORS, применяемые ко всем `/api/v1/` через chi (`cmd/server/main.go:143-146`).

Для фазы 3 надо: написать миграцию `0008_messages.up.sql` по шаблону `0007_channels.up.sql`; добавить блок sqlc в `sqlc.yaml` для `internal/chat/repository/postgres/`; реализовать домен `chat` по шаблону `internal/channel/domain/` (Channel, ChannelKind, ChannelName); добавить `pkg/websocket/` (hub, регистрация подключений, broadcast); прописать новый authenticator для WebSocket-апгрейда, читающий JWT из query (текущий HTTP-middleware `RequireAuth` читает только `Authorization: Bearer`, см. `internal/auth/transport/http/middleware/auth.go:17-38`); добавить транспортный слой `internal/chat/transport/http/` (REST `GET /channels/:id/messages`) и `internal/chat/transport/ws/` (если выбрана такая раскладка под `transport/ws/`). Архитектурные правила (`arch_test.go`) на `pkg/websocket/` и `internal/chat/...` ещё не настроены и потребуют расширения.

---

## Детальные результаты

### 1. Composition root и роутер

- **Расположение**: `cmd/server/main.go:71-176`
- **Что делает**: создаёт pgxpool, инстанцирует репозитории/usecase-ы, монтирует роуты под `/api/v1`. Глобальные middleware (`RequestID`, `Recover`, `Logger`, `CORS`) применяются до `mux.Route("/api/v1", ...)` (`cmd/server/main.go:143-146`).
- **Деп-инъекции**:
  - `pool := pgxpool.New(ctx, cfg.DatabaseURL())` (`main.go:74`)
  - `queries := db.New(pool)` (`main.go:80`)
  - `issuer := jwtadapter.NewTokenIssuer([]byte(cfg.JWTSecret()), cfg.JWTAccessTTL())` (`main.go:85`)
  - `realClock{}` / `realUUID{}` (`cmd/server/runtime.go:10-16`) и `cryptoRand{}` (`runtime.go:18-26`) — единственные не-тестовые реализации портов
- **Подключение доменов**: `httpauth.RegisterRoutes(...)` (`main.go:150-157`), `httproom.RegisterRoutes(...)` (`main.go:158-168`), `httpchannel.RegisterRoutes(...)` (`main.go:169-175`). Для chat подобный вызов пока отсутствует.
- **Graceful shutdown**: `signal.NotifyContext` + `srv.Shutdown(ctx, 5s)` (`main.go:185-213`).

### 2. JWT и identity

- **Интерфейс TokenIssuer** (`internal/auth/usecase/ports.go:29-32`):
  ```go
  type TokenIssuer interface {
      IssueAccess(userID domain.UserID, now time.Time) (token string, expiresAt time.Time, err error)
      VerifyAccess(token string, now time.Time) (domain.UserID, error)
  }
  ```
- **Имплементация** `internal/auth/repository/jwt/token_issuer.go`:
  - `IssueAccess` (lines 23-36): HS256, `claims.Subject = userID.String()`, плюс `IssuedAt`/`ExpiresAt`
  - `VerifyAccess` (lines 38-62): парсит токен с `WithValidMethods([]string{"HS256"})`, `WithTimeFunc(now)`, извлекает Subject как UUID, возвращает `domain.UserID` или `domain.ErrAccessTokenExpired` / `domain.ErrAccessTokenInvalid`.
- **`domain.UserID`** — value object из uuid (`internal/auth/domain/user_id.go`): методы `String()`, `UUID()`, `IsZero()`.
- **Поток данных**: `JWT-строка → TokenIssuer.VerifyAccess(now) → domain.UserID → WithUserID(ctx, uid)`. Контракт не привязан к HTTP, может вызываться из любого транспорта.

### 3. HTTP-аутентификационный middleware

- **`internal/auth/transport/http/middleware/auth.go:17-38`** — `RequireAuth(issuer, clock)` возвращает `func(http.Handler) http.Handler`.
- **Источник токена**: только заголовок `Authorization: Bearer <token>` (`auth.go:20-25`). Префикс — константа `bearerPrefix = "Bearer "` (line 13).
- **Ошибки**:
  - 401 + `AUTH-010 "access token invalid"` — если нет заголовка/неверный формат/невалидный токен
  - 401 + `AUTH-011 "access token expired"` — для `domain.ErrAccessTokenExpired` (lines 40-45)
- **Формат ответа об ошибке** — `httpx.WriteJSONError(w, status, code, msg)` (`pkg/httpx/jsonerror.go:18-22`):
  ```json
  { "error": { "code": "...", "message": "..." } }
  ```
- **Где применяется**: только в группах роутов внутри `RegisterRoutes` соответствующих доменов (например, `internal/auth/transport/http/routes.go:27` для `/auth/me`), не глобально на `/api/v1`.

### 4. Context keys для user_id

- **Файл**: `internal/auth/transport/http/middleware/contextkeys.go`
  - `type userIDKey struct{}` (line 9) — приватный ключ
  - `WithUserID(ctx, id) context.Context` (lines 12-14)
  - `UserIDFromContext(ctx) (domain.UserID, bool)` (lines 17-24)
- **Использование вне middleware**: в access-логгере `cmd/server/main.go:135-141` через hook логгера; в HTTP-хендлерах доменов для извлечения авторизованного userID.

### 5. Глобальные middleware (`pkg/httpx/middleware/`)

Подключение в `cmd/server/main.go:143-146`:
1. `RequestID(uuidGen)` (`pkg/httpx/middleware/requestid.go`) — генерирует/принимает `X-Request-ID`, кладёт в контекст через `httpx.WithRequestID` (`pkg/httpx/contextkeys.go`).
2. `Recover(logger)` (`pkg/httpx/middleware/recover.go`) — ловит panic, 500 `INTERNAL`, перебрасывает `http.ErrAbortHandler`.
3. `Logger(logger, userIDHook)` (`pkg/httpx/middleware/logger.go`) — структурный access-log через slog. Hook добавляет `user_id` если он в контексте.
4. `CORS(cfg.CORSAllowedOrigins(), false)` (`pkg/httpx/middleware/cors.go`) — whitelist без wildcard, без credentials, методы `GET/POST/PUT/DELETE/OPTIONS`, разрешённые заголовки `Authorization, Content-Type, X-Request-ID`.

### 6. Домен channel

- **Структура `Channel`** (`internal/channel/domain/channel.go:7-49`): поля `id (ChannelID)`, `roomID (RoomID)`, `name (ChannelName)`, `kind (ChannelKind)`, `createdAt (time.Time)`.
- **Конструкторы**: `NewChannel(...)` с валидацией, `ReconstructChannel(...)` для гидратации из БД (lines 13-49).
- **`ChannelKind`** (`internal/channel/domain/channel_kind.go`): `ChannelKindText=0`, `ChannelKindVoice=1`; методы `IsValid()`, `String()` (`"text"`/`"voice"`), `ParseChannelKind(string)` (lowercase only).
- **`ChannelName`** (`internal/channel/domain/channel_name.go`): длина 1-64 после `TrimSpace`, запрет управляющих символов и категории `Cf` (anti-spoofing).
- **Доменные ошибки** (`internal/channel/domain/errors.go`):
  - валидация: `ErrInvalidChannelID`, `ErrInvalidRoomID`, `ErrInvalidChannelKind`, `ErrInvalidCreatedAt`
  - бизнес: `ErrChannelNotFound`, `ErrChannelNameAlreadyTaken`, `ErrChannelAccessDenied`, `ErrChannelInsufficientRole`

### 7. Домен room и мембершип

- **`Membership`** (`internal/room/domain/membership.go`): поля `roomID`, `userID`, `role`, `joinedAt`. Конструкторы `NewMembership`, `ReconstructMembership`. Методы повышения/понижения роли: `Promote()`, `Demote()` (запрет на демоут owner — `ErrCannotDemoteOwner`).
- **Можно/нельзя** (методы Membership, lines 65-96): `CanReadRoom/Members/Channels` — true для любой валидной роли; `CanCreateChannel/DeleteChannel/GenerateInvite` — admin/owner; `CanDeleteRoom` — owner; `CanKick(target)` — иерархично.
- **`Role`** (`internal/room/domain/role.go`): `RoleMember=0`, `RoleAdmin=1`, `RoleOwner=2`. `IsValid()`, `String()`, `ParseRole(string)`.
- **Бизнес-ошибки** (`internal/room/domain/errors.go`): `ErrNotMember`, `ErrAlreadyMember`, `ErrInsufficientRole`, `ErrCannotDemoteOwner`.

### 8. Кросс-доменный порт MembershipQuery

- **Объявление** в `internal/channel/usecase/ports.go:18-35`:
  ```go
  type RoleRequirement int
  const (
      RoleAnyMember    RoleRequirement = iota
      RoleAdminOrOwner
      RoleOwnerOnly
  )
  type MembershipQuery interface {
      Require(ctx context.Context, roomID domain.RoomID, userID domain.UserID, req RoleRequirement) error
  }
  ```
- **Контракт ошибок** (`ports.go:27-32`):
  - `nil` — роль достаточна
  - `domain.ErrChannelAccessDenied` — не член комнаты
  - `domain.ErrChannelInsufficientRole` — член, но роль ниже
  - обёрнутая `fmt.Errorf` — техническая
- **Имплементация**: `internal/room/repository/postgres/membership_query.go:29-66` — `MembershipQueryAdapter`:
  - Внутри вызывает sqlc-метод `GetRoomMember(roomID, userID)` (`internal/room/repository/postgres/queries/room_members.sql:5-8`)
  - На `pgx.ErrNoRows` → `chdom.ErrChannelAccessDenied`
  - Парсит `row.role` через `room.domain.ParseRole`
  - Функция `satisfies(req, role)` проверяет соответствие требования (lines 55-66)
- **Архитектурное исключение**: `arch_test.go:259-280` (`TestArchitecture_RoomRepoMayImplementChannelPort`) специально разрешает `internal/room/repository/postgres/` импортировать `channel/usecase` и `channel/domain` — единственная санкционированная связь между доменами на уровне адаптера.
- **Где используется**: `CreateChannel` (`internal/channel/usecase/create_channel.go:56`, `RoleAdminOrOwner`), `ListChannels` (`internal/channel/usecase/list_channels.go:38`, `RoleAnyMember`), `DeleteChannel` (`internal/channel/usecase/delete_channel.go:39`, `RoleAdminOrOwner`). Эти три вызова — единственные на текущем коммите.

### 9. SQLC, миграции и шаблон репозитория

- **`sqlc.yaml`** (`sqlc.yaml`): три отдельных `sql:` блока (auth/room/channel), все используют `engine: postgresql`, `sql_package: pgx/v5`, `package: db`, `out: internal/<домен>/repository/postgres/db`. Отключены `emit_interface`, `emit_json_tags`, `emit_db_tags`, `emit_pointers_for_null_types`, `emit_prepared_queries`. Type overrides: `uuid` → `github.com/google/uuid.UUID`, `timestamptz` → `time.Time`. Для chat понадобится 4-й аналогичный блок.
- **Миграции** (`migrations/`): пары `NNNN_<имя>.{up|down}.sql`. Текущий максимум — `0007_channels.{up|down}.sql`; следующая — `0008_messages.{up|down}.sql`.
- **Шаблон миграции таблицы** (`migrations/0007_channels.up.sql`):
  ```sql
  CREATE TABLE channels (
      id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
      room_id     uuid        NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
      name        text        NOT NULL,
      kind        text        NOT NULL,
      created_at  timestamptz NOT NULL DEFAULT now(),
      CONSTRAINT channels_name_length_check CHECK (char_length(name) BETWEEN 1 AND 64),
      CONSTRAINT channels_kind_check CHECK (kind IN ('text', 'voice')),
      CONSTRAINT channels_room_id_name_key UNIQUE (room_id, name)
  );
  CREATE INDEX channels_room_id_idx ON channels(room_id);
  ```
  Конвенции: PK `uuid DEFAULT gen_random_uuid()`, FK с `ON DELETE CASCADE`, `created_at timestamptz NOT NULL DEFAULT now()`, именованные `CONSTRAINT` (включая UNIQUE и CHECK на длину/перечисление), отдельный индекс на FK. Soft-delete не используется.
- **Шаблон sqlc-запросов** (`internal/channel/repository/postgres/queries/channels.sql`):
  ```sql
  -- name: InsertChannel :exec
  INSERT INTO channels (id, room_id, name, kind, created_at) VALUES ($1, $2, $3, $4, $5);

  -- name: ListChannelsByRoom :many
  SELECT id, room_id, name, kind, created_at FROM channels
  WHERE room_id = $1 ORDER BY created_at ASC;

  -- name: DeleteChannelInRoom :execrows
  DELETE FROM channels WHERE id = $1 AND room_id = $2;
  ```
  Поддерживаются `:exec`, `:one`, `:many`, `:execrows`. Параметры позиционные.
- **Шаблон репозитория** (`internal/channel/repository/postgres/channel_repository.go`):
  - Конструктор `NewChannelRepository(pool *pgxpool.Pool) *ChannelRepository` (lines 10-18) держит `q *db.Queries`.
  - `Save(ctx, *domain.Channel) error`: вызывает `r.q.InsertChannel(ctx, domainToInsertChannelParams(ch))`; на unique-violation `channels_room_id_name_key` → `domain.ErrChannelNameAlreadyTaken`; иначе оборачивает через `fmt.Errorf("postgres: insert channel: %w", err)` (lines 23-31).
  - `ListByRoom(ctx, roomID)`: получает rows, маппит через `channelRowToDomain` (lines 33-50).
  - `DeleteInRoom(ctx, channelID, roomID)`: проверяет `affected==0 → domain.ErrChannelNotFound` (lines 52-68).
- **Маппер** (`internal/channel/repository/postgres/mapper.go`): функции `channelRowToDomain(db.Channel) (*domain.Channel, error)` и `domainToInsertChannelParams(*domain.Channel) db.InsertChannelParams`.
- **Обработка ошибок Postgres** (`internal/channel/repository/postgres/pgerr.go`): `isUniqueViolation(err, constraintName) bool` сравнивает `pgErr.Code == "23505"` и имя constraint.
- **Compile-check** (`internal/channel/repository/postgres/compile_check_test.go`):
  ```go
  var _ chuc.ChannelRepository = (*pg.ChannelRepository)(nil)
  ```
- **Интеграционные хелперы** (`internal/channel/repository/postgres/integration_helpers_test.go`): `dbConn(t)` читает `TEST_DATABASE_URL`, `truncate(t, pool)` делает `TRUNCATE ... CASCADE`, `seedRoom(t, pool, roomID, ownerID)` вставляет user+room+owner-membership через прямой SQL.

### 10. Архитектурные правила (arch_test.go)

`arch_test.go:1-418` парсит non-test файлы каждого пакета и проверяет, что список импортов укладывается в whitelist.

- **`internal/<домен>/domain/`**: только stdlib + `github.com/google/uuid` (`arch_test.go:74-93, 216-235, 316-335`).
- **`internal/<домен>/usecase/`**: только stdlib + `google/uuid` + `internal/<домен>/domain` (`arch_test.go:95-115, 237-257, 337-362`). Для channel дополнительно ЗАПРЕЩЕНЫ импорты `internal/room/*` (line 337-362).
- **`internal/<домен>/repository/postgres/`**: изолирован, не импортирует `transport/`, `auth/`, чужие домены (`arch_test.go:117-143, 398-417`). Исключение — `room/repository/postgres/` может импортировать `channel/usecase` и `channel/domain` (`arch_test.go:259-280`).
- **`internal/<домен>/transport/http/`**: stdlib + uuid + chi + собственные domain/usecase + `auth/{domain, usecase, transport/http/middleware}` + `pkg/httpx` (`arch_test.go:145-160, 282-314, 364-396`). Можно использовать `RequireAuth` из auth-домена.
- **`pkg/httpx`**: НЕ импортирует ничего из `internal/...` (`arch_test.go:162-187`).
- **`internal/auth/transport/http/middleware`**: единственное исключение — может импортировать `internal/auth/{domain, usecase}` и `pkg/httpx` (`arch_test.go:189-214`).
- На `pkg/websocket/` и на `internal/chat/...` правил пока нет — потребует расширения arch_test.go при появлении этих пакетов.

### 11. Стиль тестов

- **Domain**: чёрный ящик (`*_test` package), без моков, t.Parallel, t.Helper, table-driven где сценарии равноправны (см. `internal/channel/domain/channel_test.go`, `channel_kind_test.go`, `channel_name_test.go`).
- **Usecase**: ручные fakes в `*_test.go` (`internal/channel/usecase/fakes_test.go`): `fakeChannelRepo`, `fakeMembershipQuery`, `fixedClock`, `fixedUUID`. Поддерживается инжекция ошибок (`saveErr`, `requireErr`). Используется SUT-паттерн (`newCreateChannelSUT`) с builder-методами (`internal/channel/usecase/create_channel_test.go:15-48`). Без testify / gomock — проверки через `errors.Is` и `t.Fatalf`.
- **HTTP-транспорт**: `httptest.NewServer(chi.Router)` + реальный usecase + fake-репо (`internal/channel/transport/http/setup_test.go`). Реальный `jwtadapter.NewTokenIssuer` для генерации тестовых токенов. В fakes под HTTP — `sync.Mutex` из-за параллельного исполнения (`setup_test.go:98-152`).
- **Repository (Postgres)**: build-tag `integration`, `TEST_DATABASE_URL`, `t.Skip` если не задан; `truncate ... CASCADE` перед каждым тестом.

### 12. Конфигурация (env)

`config/config.go:39-282`:

| Env | По умолчанию | Валидация | Код |
|---|---|---|---|
| `JWT_SECRET` | — | required, непустая строка | CONFIG-001 |
| `DATABASE_URL` | — | required, непустая строка | CONFIG-002 |
| `SERVER_PORT` | 8080 | 1..65535 | CONFIG-003 |
| `JWT_ACCESS_TTL` | 15m | >0, <= 1h | CONFIG-004 |
| `JWT_REFRESH_TTL` | 720h (30d) | >0, > access, <= 2160h (90d) | CONFIG-005 |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | comma-separated, http(s) + host, без path/query | CONFIG-006 |

Никаких env-переменных для WebSocket (max message size, read/write deadlines, ping interval) сейчас нет.

### 13. Состояние `pkg/websocket/`

- Папка `pkg/websocket/` существует, но содержит только `.gitkeep` (`pkg/websocket/.gitkeep`, 0 байт).
- Зависимость `nhooyr.io/websocket` (упомянутая в `general_plan.md:35`) **не добавлена** в `go.mod` (`go.mod:5-19` — только `chi/v5`, `golang-jwt/v5`, `google/uuid`, `pgx/v5`, `golang.org/x/crypto`). Подключение библиотеки потребует согласования с пользователем (правило из `CLAUDE.md`).

### 14. Состояние домена `chat`

- Папка `internal/chat/` не существует. Никаких файлов, тестов, миграций.
- В `cmd/server/main.go` нет импорта `chat`-пакетов.
- В `sqlc.yaml` нет блока для chat.
- В `arch_test.go` нет правил для chat-пакетов.

### 15. Зафиксированные решения / референсные ресерчи

Предыдущие исследования в `.thoughts/research/` (читать как контекст, не как обязательство):

- `2026-05-11-PR-2-rooms-and-channels.md` — детально описывает решения по rooms/channels, включая обоснование кросс-доменного порта MembershipQuery и архитектурное исключение в `arch_test.go`.
- `2026-05-10-auth-middleware-baseline.md` — baseline middleware-стека: коды ошибок AUTH-010/AUTH-011, формат envelope `{ "error": { code, message } }`, контракт `RequireAuth`.
- `2026-05-11-1_5-tests-baseline.md` — стиль unit/HTTP/integration-тестов.
- `2026-05-07-1_3-auth-domen-baseline.md`, `2026-05-07-tokens-and-index.md`, `2026-05-05-users-table-migration.md`, `2026-05-04-project-structure.md` — фундаментальные документы фазы 1.

---

## Ссылки на код (быстрый индекс)

### JWT и аутентификация
- `internal/auth/usecase/ports.go:29-32` — интерфейс `TokenIssuer`
- `internal/auth/repository/jwt/token_issuer.go:38-62` — `VerifyAccess`
- `internal/auth/transport/http/middleware/auth.go:17-38` — `RequireAuth` (только Bearer)
- `internal/auth/transport/http/middleware/contextkeys.go:9-24` — `userIDKey`, `WithUserID`, `UserIDFromContext`
- `internal/auth/domain/user_id.go` — `domain.UserID`
- `internal/auth/domain/errors.go` — `ErrAccessTokenExpired`, `ErrAccessTokenInvalid`

### Membership и права
- `internal/channel/usecase/ports.go:12-43` — `ChannelRepository`, `RoleRequirement`, `MembershipQuery`, `Clock`, `UUIDGenerator`
- `internal/room/repository/postgres/membership_query.go:29-66` — `MembershipQueryAdapter`
- `internal/room/repository/postgres/queries/room_members.sql:5-8` — `GetRoomMember`
- `internal/room/domain/membership.go:42-96` — методы `Promote/Demote/Can...`
- `internal/room/domain/role.go` — `Role` enum
- `internal/channel/domain/errors.go:16-24` — `ErrChannelAccessDenied`, `ErrChannelInsufficientRole`, `ErrChannelNotFound`, `ErrChannelNameAlreadyTaken`

### Composition root и роутер
- `cmd/server/main.go:71-176` — `run(cfg, logger)`
- `cmd/server/main.go:124-130` — channel composition
- `cmd/server/main.go:132-146` — chi root, middleware-стек
- `cmd/server/main.go:148-176` — монтаж `/api/v1`
- `cmd/server/runtime.go:10-26` — `realClock`, `realUUID`, `cryptoRand`

### Шаблон таблицы / репозитория / тестов (channel — основной референс для chat)
- `migrations/0007_channels.up.sql` — DDL шаблон
- `internal/channel/repository/postgres/queries/channels.sql` — sqlc-запросы шаблон
- `internal/channel/repository/postgres/channel_repository.go` — репозиторий шаблон
- `internal/channel/repository/postgres/mapper.go` — маппер шаблон
- `internal/channel/repository/postgres/pgerr.go` — обработка PG-ошибок
- `internal/channel/repository/postgres/compile_check_test.go` — compile-check
- `internal/channel/repository/postgres/integration_helpers_test.go` — `dbConn`, `truncate`, `seedRoom`
- `internal/channel/usecase/create_channel.go` — usecase шаблон с проверкой `MembershipQuery.Require`
- `internal/channel/usecase/fakes_test.go` — fakes шаблон
- `internal/channel/transport/http/routes.go` + `create_channel_handler.go` + `setup_test.go` — HTTP шаблон

### Архитектура и тесты
- `arch_test.go:74-417` — все правила импортов по слоям и доменам
- `arch_test.go:259-280` — разрешение `room/repository/postgres` импортировать `channel/usecase`
- `arch_test.go:337-362` — запрет `channel/usecase` → `room/*`

### Инфраструктура
- `config/config.go:39-282` — конфигурация env
- `pkg/httpx/middleware/{cors,logger,recover,requestid}.go` — глобальные middleware
- `pkg/httpx/jsonerror.go:18-22` — `WriteJSONError`
- `pkg/httpx/contextkeys.go` — `WithRequestID`, `RequestIDFromContext`
- `sqlc.yaml` — три блока генерации (auth/room/channel)
- `docker-compose.yml` — Postgres 16-alpine
- `Makefile` — `run`, `test`, `lint`, `build`, `migrate-up/down`, `dc-up/down`
- `pkg/websocket/.gitkeep` — пустой плейсхолдер
- `go.mod:5-19` — фактический набор зависимостей (`nhooyr.io/websocket` ОТСУТСТВУЕТ)
- `.claude/plans/general_plan.md:141-151` — чек-лист фазы 3

---

## Архитектурные наблюдения

- **Слоистая чистота**. Зависимости направлены строго внутрь (`prompts/Architecture Layers.txt:67-73`). Доменные пакеты импортируют только stdlib + uuid. Усе кросс-доменные связи проходят через явно объявленные порты в usecase-слое. Для chat эта же модель: домен `chat/domain` — чистые сущности, `chat/usecase` объявляет свой `MessageRepository` и переиспользует `channel/usecase.MembershipQuery` (импортируется только если решено централизовать порт; альтернатива — продублировать аналогичный порт в `chat/usecase`).
- **Кросс-доменный порт `MembershipQuery` сейчас живёт в `channel/usecase`**. Его реализация в `room/repository/postgres` зашита в `arch_test.go:259-280` как явное исключение. Для домена chat возможны две раскладки: (а) объявить отдельный порт `chat/usecase.MembershipQuery` (зеркало) и дописать второй адаптер в `room/repository/postgres/`; (б) переиспользовать существующий `channel/usecase.MembershipQuery` напрямую (тогда `chat/usecase` импортирует `channel/usecase`, что потребует расширения arch_test.go). Решение — за дизайн-фазой.
- **JWT в WebSocket**. `RequireAuth` неприменим без модификации, потому что читает только `Authorization`-заголовок. Однако `TokenIssuer.VerifyAccess(token, now)` — чистая функция и переиспользуется любым транспортом. Под WebSocket потребуется отдельный authenticator: парсер `r.URL.Query().Get("token")` → `VerifyAccess` → `WithUserID(ctx, uid)` до апгрейда. Helpers `WithUserID`/`UserIDFromContext` уже подходят.
- **Поток данных для message.send (предположительный)**: WS-фрейм → ws-хендлер → парсинг JSON → проверка `MembershipQuery.Require(roomID, userID, RoleAnyMember)` → `chat/usecase.SendMessage.Execute(input)` → `domain.NewMessage(...)` → `chat/repository/postgres.MessageRepository.Save(msg)` → `pkg/websocket/Hub.Broadcast(channelID, msg)`. Структуру маршрутизации между WS и REST раскладывает дизайн-фаза.
- **REST `GET /channels/:id/messages?before=&limit=`**: курсорная пагинация (по `created_at` или `(created_at, id)`). По шаблону `ListChannelsByRoom` в sqlc, но с `WHERE created_at < $2 LIMIT $3` и `ORDER BY created_at DESC`. Проверка прав — через тот же `MembershipQuery.Require(..., RoleAnyMember)`.
- **Конфигурация WebSocket**. Сейчас нет env-переменных для WS (`config/config.go`). При появлении hub'а вероятно потребуются переменные для размера буфера/таймаутов — добавляются по образцу `loadAccessTTL`/`loadCORSAllowedOrigins` с собственным кодом ошибки `CONFIG-007` и т.д.
- **Тестируемость hub'а**. Шаблон unit-тестов на ручных fakes без моков (`fakes_test.go`-паттерн) допускает тестирование hub'а в виде структуры с инжектируемыми порождающими/закрывающими хуками. Конкретная форма зависит от того, какие интерфейсы будут объявлены в `pkg/websocket/`.
- **arch_test.go должен расти**. При добавлении `pkg/websocket/` и `internal/chat/...` потребуются новые тесты импортов (по шаблону `TestArchitecture_Pkg_Httpx*` и `TestArchitecture_Channel*`).
- **`nhooyr.io/websocket` ещё не подключён**. По правилу `CLAUDE.md` («Не добавлять библиотеки без согласования») подключение этой зависимости — отдельный шаг, требующий согласования.
