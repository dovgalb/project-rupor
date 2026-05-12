---
parent: ./README.md
---

# 07 — Standards Compliance

Матрица соответствия каждому документу из `prompts/`. Знак ✅ — стандарт соблюдён; ⚠️ — есть осознанное расхождение с обоснованием.

| Стандарт | Статус | Ключевые точки compliance |
|---|---|---|
| `Architecture Layers.txt` | ✅ | Зависимости направлены строго внутрь. `chat/domain` импортирует только stdlib + uuid. `chat/usecase` импортирует stdlib + uuid + `chat/domain`. Транспорт и репозиторий зависят от usecase + domain. Кросс-доменная связь (`room/repository/postgres` → `chat/usecase`) — единственное санкционированное исключение, явно зафиксировано в `arch_test.go` и в § D-01 решений. `pkg/websocket` НЕ импортирует `internal/*`. WS-handler в `internal/chat/transport/ws/` (а не в `pkg/`) сохраняет правило «pkg не знает про internal». |
| `Clean architecture.txt` | ✅ | Доменные правила (валидация текста, инвариант "только text-канал принимает сообщения") живут в `chat/domain` и `chat/usecase` соответственно. Технические детали (HS256-токен, sqlc-генерация, nhooyr-фрейм) в адаптерах. Интерфейсы (`MessageRepository`, `MembershipQuery`, `Broadcaster`) объявлены в `chat/usecase` (consumer-owned interfaces). |
| `Domain Model.txt` | ✅ | `Message` — rich entity с приватными полями, конструкторы `NewMessage`/`ReconstructMessage`, геттеры без сеттеров. Без бизнес-методов в MVP (нет редактирования). VO: `MessageID`, `MessageText`, `ChannelID`, `UserID`, `RoomID` — все с валидацией в конструкторе, иммутабельны, сравниваются по значению. Доменные ошибки в `chat/domain/errors.go` через `errors.New(...)`. Связи между сущностями — по id (`Message.ChannelID()`, `Message.AuthorID()`), без вложенных объектов. |
| `Builder.txt` | ✅ | Builder используется ТОЛЬКО в тестах. В продовом коде — явные конструкторы. Test builder для `Message` будет в `internal/chat/domain/testing_builders_test.go` (по аналогии с `internal/channel/domain/testing_builders_test.go`): принимает `*testing.T`, дефолты валидны, chainable `With*`-методы, `Build()` падает через `t.Fatalf`. |
| `RepoModel.txt` | ✅ | См. `06-repo-model.md`. Repo model отдельной структуры (`messageRow`) не вводится — `db.Message` подходит one-to-one. Маппинг через свободные функции `messageRowToDomain` / `domainToInsertMessageParams`. VO разворачиваются на границе адаптера, не утекают в `domain/`. Маппер чистый, без I/O. `pgx.ErrNoRows` → доменная ошибка в `ChannelOf` (`ErrChannelNotFound`). Unique-violation обработка не нужна — на `messages` нет уникальных constraint'ов кроме PK; повторный UUID-конфликт логируется как INTERNAL. |
| `Go style.txt` | ✅ | Go 1.25 (как и весь проект). `gofmt`/`goimports`/`golangci-lint` через `make lint`. Именование — `MessageRepository`, `SendMessage`, `MessageText` (без `I*`, без `*Interface`). Ошибки через `fmt.Errorf("...: %w", err)`. `context.Context` первым параметром в I/O-методах. `panic` запрещён в продовом коде (используется только при ошибке настройки в composition root, как сейчас). Горутины в hub'е имеют явного владельца (closeFn). |
| `Tests Style.txt` | ✅ | Только stdlib `testing`. Ручные fakes (без mockgen/testify). `t.Parallel()` везде, где безопасно. AAA с пустыми строками. Хелперы вызывают `t.Helper()`. Время и UUID детерминированы через инжекцию. Сравнения ошибок — `errors.Is`. Тесты HTTP — через `httptest.NewServer` с реальным chi. Интеграция БД — build-tag `integration`, `TEST_DATABASE_URL`. Подробности — `04-testing.md`. |
| `Domain model test.txt` | ✅ | Domain-тесты в `chat/domain` — чёрный ящик (`package domain_test`). Без моков, без БД, без HTTP. Table-driven для `MessageText` (валидные/невалидные кейсы). `errors.Is` для проверок. Покрытие всех веток конструктора `NewMessage` и всех инвариантов VO. Никаких `time.Now()` / `uuid.New()` внутри тестируемого кода — всё через инжектируемые `Clock` и `UUIDGenerator` на уровне usecase, а в domain просто принимается готовое значение. |

## Уточнения и осознанные расхождения

### Уточнение 1: Имя метода маппинга

`prompts/RepoModel.txt:46` упоминает `userRow.toDomain()` как метод receiver'а. В проекте уже используется обратный паттерн — свободные функции (`channelRowToDomain(row)`, см. `internal/channel/repository/postgres/mapper.go:1-15`). Следуем существующему паттерну проекта, а не примеру из стандарта. Это соответствует правилу 15 skill: «Соответствие реальным паттернам проекта».

### Уточнение 2: `ReconstructMessage` vs `NewMessage`

`prompts/Domain Model.txt:124-129` рекомендует два различных конструктора: `NewUser` (создание новой сущности) и `ReconstructUser` (восстановление из БД). В существующем `internal/channel/domain/channel.go:41-49` `ReconstructChannel` — алиас `NewChannel`. Следуем существующему паттерну: `ReconstructMessage` будет точным алиасом `NewMessage`, поскольку у `Message` инварианты создания и восстановления совпадают (id, channelID, authorID, text, createdAt — всё передаётся снаружи в обоих случаях).

### Уточнение 3: Снэйк_кейс в WS-payload vs camelCase в REST

См. `05-events.md` § Кодирование payload. В WS — snake_case (`channel_id`, `created_at`), в REST — camelCase (`roomId`, `createdAt`, см. `internal/channel/transport/http/dto.go:16-22`). Это осознанное расхождение между двумя транспортами. Альтернатива (единый camelCase) — обсуждать в фазе 3.2.

### Уточнение 4: Логирование токена в WS-handshake

`prompts/Go style.txt:94` запрещает логирование токенов и секретов. Стандартный `pkg/httpx/middleware.Logger` логирует URL целиком (`logger.go:13-47`). Для пути `/api/v1/ws` URL содержит `?token=<jwt>`. Решение: в фазе 3 добавить в `pkg/httpx/middleware.Logger` опцию `URLSanitizer` или жёсткий фильтр, маскирующий значение query-параметра `token` при логировании. Покрывается тестом `Logger_StripsTokenFromWSPath` (см. `04-testing.md`). Это потребует **изменения существующего пакета `pkg/httpx/middleware`** — отмечается как отдельная задача фазы (см. план кода).

### Уточнение 5: Запрет добавления библиотек без согласования

`prompts/Go style.txt:111-114`: «Запрещено добавлять новые зависимости в go.mod без согласования». Зависимость `nhooyr.io/websocket` согласована пользователем явно через AskUserQuestion 2026-05-12 (ответ зафиксирован в `03-decisions.md` § D-12).

### Уточнение 6: Структура папок WS

`prompts/Architecture Layers.txt:49` упоминает `internal/<домен>/transport/ws/` как место для WS-обработчиков. Этот паттерн полностью совпадает с принятым решением (`03-decisions.md` § D-15). Подтверждение из стандарта.

### Уточнение 7: Тесты publish-стороны use case

Use case `SendMessage` не должен зависеть от того, успешен publish или нет. Тест `TestSendMessage_PublishFails_ReturnsSuccess` (`04-testing.md`) явно проверяет, что panic / error в `Broadcaster.PublishToChannel` не пробрасывается клиенту и не откатывает Save. Это соответствует решению § D-08 о best-effort.

---

## Итог

Все ключевые стандарты соблюдены. Расхождения с буквой стандарта (`Reconstruct` как алиас, свободные функции мапперов) — это следование уже устоявшимся паттернам проекта, что предписано правилом 15 skill `/design_feature`. Любое отклонение от существующих паттернов было бы более серьёзным нарушением, чем отклонение от примеров в `prompts/`.
