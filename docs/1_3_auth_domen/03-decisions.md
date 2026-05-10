---
parent: ./README.md
view: decision
---

# 03 — Decisions (Decision View)

## Решения

| # | Решение | Выбор | Рассмотренные альтернативы | Обоснование (ссылки `файл:строка`) |
|---|---------|-------|----------------------------|------------------------------------|
| ADR-001 | Драйвер PostgreSQL | `github.com/jackc/pgx/v5` + `pgxpool.Pool` | (a) `database/sql` + `lib/pq`; (b) `database/sql` + `pgx/v5/stdlib` | (1) Согласовано с пользователем (вопрос UI). (2) sqlc поддерживает `sql_package: "pgx/v5"` нативно — даёт типизированные методы поверх `pgx`. (3) `*pgconn.PgError` — типизированный объект ошибки с `Code` и `ConstraintName`, идеально подходит для `prompts/RepoModel.txt:159` («Определение типа ошибки БД — через проверку `*pgconn.PgError`»). (4) `pgxpool` — встроенный пул, не нужна обёртка. (5) `lib/pq` в maintenance, не рекомендуется к новым проектам. |
| ADR-002 | JWT-библиотека | `github.com/golang-jwt/jwt/v5` | (a) `github.com/lestrrat-go/jwx/v2`; (b) ручная реализация на `crypto/hmac` | (1) Согласовано с пользователем. (2) Самая популярная Go JWT-либа, минимум транзитивных зависимостей. (3) Достаточно для HS256 + базовые claims (`sub`, `iat`, `exp`). (4) `jwx` тяжелее, нужен только для JWE/JWK — не наш случай. (5) Ручная реализация — лишний код без выигрыша. |
| ADR-003 | UUID-библиотека | `github.com/google/uuid` | (a) Без библиотеки, `[16]byte` или `string` в домене; (b) `gofrs/uuid` | (1) Согласовано с пользователем. (2) `prompts/Domain Model.txt:25` явно допускает `github.com/google/uuid` «если согласовано». (3) Удобный `uuid.Parse(string)` для `me`-хендлера; `uuid.UUID.String()` для сериализации в JSON. (4) Совместимо с pgx-маппингом `pgtype.UUID` ↔ `uuid.UUID`. |
| ADR-004 | TTL access/refresh | env-переменные с дефолтами: `JWT_ACCESS_TTL=15m`, `JWT_REFRESH_TTL=720h` (30d) | (a) Константы в коде (1h / 7d или 15m / 30d); (b) Параметры командной строки | (1) Согласовано с пользователем. (2) Согласуется с уже существующим паттерном `JWT_SECRET` через env (`config/config.go:48-77`). (3) `time.ParseDuration` даёт читаемый формат (`15m`, `720h`), не `int seconds`. (4) Гибкость без пересборки — критично для dev/prod-разных TTL. (5) Defaults — индустриальный standard для refresh-стратегии с ротацией. (6) Валидация: `access > 0`, `access ≤ 1h`, `refresh > access`, `refresh ≤ 90d` — все защиты от опечаток. |
| ADR-005 | Раскладка адаптеров: куда положить JWT-impl и bcrypt-impl | `internal/auth/repository/postgres/`, `internal/auth/repository/jwt/`, `internal/auth/repository/bcrypt/` — все под зонтиком `repository/` | (a) Завести новый слой `internal/auth/security/` или `internal/auth/infrastructure/`; (b) Положить в `pkg/jwt/`, `pkg/bcrypt/` как переиспользуемые; (c) Втащить в `transport/http/` | (1) `prompts/Architecture Layers.txt:46-58` фиксирует раскладку только примерами (`repository/postgres/`), не запрещает других подпапок под `repository/`. (2) Все три — адаптеры для интерфейсов, объявленных в `usecase/`, которые удовлетворяют тот же контракт «реализация порта» (`prompts/Clean architecture.txt:30-38`). (3) Заводить новый слой `security/` — отклонение от структуры `domain.gitkeep`, `usecase.gitkeep`, `transport/http.gitkeep`, `repository/postgres.gitkeep`, физически зафиксированной фазой 1.1 (`docs/1_1_project-structure/03-decisions.md:12`, ADR-001). (4) `pkg/jwt/` сделает инструмент общим, но для MVP нет потребителей вне auth. Когда middleware в фазе 1.4 переиспользует `TokenIssuer.VerifyAccess` — она остаётся внутри `internal/auth/`, потому что это бизнес-граница «токен + userID». |
| ADR-006 | Атомарность ротации refresh | Compound-метод `RefreshTokenRepository.Rotate(ctx, oldHash, *newToken, revokedAt)` с внутренней `pgx.Tx` (`BEGIN; UPDATE; INSERT; COMMIT`) | (a) UnitOfWork как отдельный интерфейс в `usecase/`; (b) Два отдельных вызова без транзакции (UPDATE → INSERT); (c) Хранимая процедура в БД | (1) `prompts/RepoModel.txt:147-150` явно допускает «транзакцию через `BeginTx` на самом репозитории». (2) UnitOfWork — overengineering для одного use case с двумя SQL-операциями; добавим, когда появится 3+ компаунда. (3) Без транзакции — окно неконсистентности: старый токен помечен `revoked`, новый не вставлен, клиент остался без refresh. (4) В `Rotate` UPDATE имеет `WHERE revoked_at IS NULL` → гарантия идемпотентности при race: второй параллельный refresh получит 0 rows affected, мы возвращаем `ErrRefreshTokenRevoked` (см. ADR-010). |
| ADR-007 | Валидация `Username` | regex `^[a-zA-Z0-9_-]+$`, длина 3..32, trim по краям | (a) Только длина без regex; (b) Расширенный regex с unicode-буквами; (c) Lowercase-нормализация | (1) Длина 3..32 совпадает с `users_username_length_check` в БД (`migrations/0002_users.up.sql:12`). (2) ASCII-only — упрощает sql/индексы и убирает класс багов с homoglyph-атаками (например, кирилл. «о» vs лат. «o») в username — критично для платформы, где username может попасть в @-mention в фазе 3. (3) Lowercase не делаем: уникальность username регистрозависимая в БД (`docs/1_2_db_schema/users_table/03-decisions.md:14`, решение 3). Если в будущем потребуется `Bogdan` ≡ `bogdan` — отдельная миграция данных + смена правила в VO. (4) Расширенный unicode откладываем: задача MVP — чтобы работало для русско/англоязычной команды, оба пишут латиницей. |
| ADR-008 | Защита от user-enumeration в Login | Один и тот же ответ `AUTH-006 invalid credentials` для всех трёх случаев: невалидный формат email, email не найден, неправильный пароль; bcrypt-сравнение выполняется даже при `ErrUserNotFound` (с фейковым хешем) | (a) Возвращать конкретные коды (`AUTH-001` для невалидного email, `AUTH-006` для unknown email, `AUTH-006` для wrong password); (b) Без выравнивания времени | (1) OWASP ASVS V2.1.1: ответы на login не должны палить, существует ли учётка. (2) Без dummy-bcrypt разница во времени между «email not found» (~1мс) и «password mismatch» (~50мс при cost=10) выдаёт enumeration. (3) Стоимость dummy-bcrypt — фиксированная, ~50мс на запрос с несуществующим email; в MVP это допустимо. (4) Альтернатива (без выравнивания времени) — известная атака, реализуется в любом сетевом сканере; не закрывать её в auth-домене безответственно. |
| ADR-009 | Авторизация в `/auth/me` без middleware | Хендлер `MeHandler` сам извлекает `Authorization: Bearer`, вызывает `TokenIssuer.VerifyAccess(token, now)`, передаёт UserID в use case | (a) Включить middleware в scope 1.3; (b) Отложить `/auth/me` до фазы 1.4 | (1) `general_plan.md:114-118` явно выносит middleware в задачу 1.4: «JWT-middleware (извлечение `Authorization: Bearer`, валидация, проброс userID в контекст)». (2) `general_plan.md:111` явно включает `/auth/me` в scope 1.3. Это коллизия в плане — выбор минимального компромисса: inline-проверка в хендлере. (3) `TokenIssuer.VerifyAccess` стабилизирован на этапе 1.3 — middleware фазы 1.4 будет вызывать его без изменений сигнатуры. Рефакторинг — перенести 6–8 строк из `me_handler.go` в middleware. (4) Откладывать `/auth/me` нельзя — без него фронтенд (фаза 5) не сможет показать «вы залогинены как X» после refresh страницы. |
| ADR-010 | Семантика `Rotate` при race / уже отозванном токене | UPDATE с `WHERE token_hash = $1 AND revoked_at IS NULL`. Если `affected_rows == 0` — возвращаем `domain.ErrRefreshTokenRevoked`. INSERT нового токена выполняется только при success предыдущего UPDATE. | (a) UPDATE без `revoked_at IS NULL` (перезаписать); (b) SELECT FOR UPDATE → IF active → UPDATE → INSERT | (1) `WHERE revoked_at IS NULL` — естественная защита от race: два параллельных refresh с одним токеном дают двa SQL-tx, второй получит 0 rows. (2) Перезапись `revoked_at` без условия — теряет аудит (timestamp первого revoke). (3) `SELECT FOR UPDATE` — лишний lock; UPDATE+rowcount даёт ту же гарантию атомарно. (4) В sqlc запрос помечается `:execrows` — генерирует метод, возвращающий `affected int64`; репо возвращает `ErrRefreshTokenRevoked` если 0. |
| ADR-011 | Refresh-токен — НЕ JWT, а 32 случайных байта | `crypto/rand.Read(32)` → `base64url.EncodeToString` (длина 43 символа без padding). Хранится только sha256-хеш | (a) Refresh как JWT с другим secret/claims; (b) JTI внутри JWT + таблица отозванных | (1) `docs/1_2_db_schema/token_and_index/03-decisions.md:13`, решение 2: «hash + ротация». (2) Не нужно парсить JWT на каждом refresh — простой `sha256` + одна SELECT. (3) Утечка БД не компрометирует raw-токены (sha256 необратим). (4) JWT-refresh нёс бы exp в подписи — но мы хотим server-side revocation, что несовместимо с stateless подходом. Хранение хеша + revoked_at — единственный способ отозвать. |
| ADR-012 | Bcrypt cost | `bcrypt.DefaultCost = 10` | (a) `bcrypt.MinCost = 4` (только для тестов); (b) cost 12+ для прода | (1) Cost 10 — индустриальный стандарт 2026 года, ~50мс на современном CPU. (2) Cost 12 — ~200мс, заметно для UX, требует асинхронной обработки. Откладываем до появления нагрузки. (3) Cost 4 — для unit-тестов, чтобы тест-сьют не тормозил; в продовом коде запрещён. PasswordHasher принимает cost через конструктор: `bcrypt.NewPasswordHasher(cost int)`; в `cmd/server/main.go` передаётся 10, в тестах — 4. |

## Риски и митигация

| Риск | Влияние | Митигация |
|------|---------|-----------|
| Утечка `JWT_SECRET` (например, через `.env` в git) | High | (1) `.env` в `.gitignore` (`docs/1_1_project-structure/03-decisions.md:24`, ADR-013). (2) `cfg.JWTSecret()` не логируется — `prompts/Go style.txt:94`. (3) В проде секрет генерируется вне репозитория. |
| Сбой `crypto/rand.Read` (теоретически) даёт предсказуемый refresh | Critical | `crypto/rand.Read` возвращает ошибку — use case возвращает 500, токены не выпускаются. Никогда не fallback на `math/rand`. |
| Замедление логина из-за dummy-bcrypt при ErrUserNotFound | Low | Cost 10 → ~50мс на запрос — приемлемо для login-эндпоинта. Альтернатива (rate-limiting) — отдельная задача. |
| Раса: два регистрационных запроса с одним email одновременно | Low | `unique_violation` `users_email_key` ловится в репо → `ErrEmailAlreadyTaken` → 409. БД-уровневая защита. |
| Раса: два refresh-запроса с одним токеном | Low | `Rotate` использует `WHERE revoked_at IS NULL` — второй UPDATE даёт 0 rows → `ErrRefreshTokenRevoked` → 401. Клиент перелогинится. |
| Утечка refresh-токена → атакующий получает access | High | Ротация: первое использование украденного токена отзовёт его, при втором использовании получим `AUTH-008`. Реальная reuse-detection (отзыв всех токенов user'а) — отложено в OQ-3. |
| `pgxpool.Pool` не закрылся при graceful shutdown → leaked connections | Low | В `cmd/server/main.go` после `srv.Shutdown(...)` (`cmd/server/main.go:85`) добавляется `pool.Close()`. |
| Алгоритм атаки на JWT: `alg: none` | Critical | `golang-jwt/jwt/v5` v5+ требует явного списка `WithValidMethods([]string{"HS256"})` при парсинге. В `repository/jwt/token_issuer.go` это явно настраивается; тест `TestTokenIssuer_VerifyAccess_RejectsAlgNone` проверяет. |
| Алгоритм атаки: токен подписан другим секретом | Critical | HS256 requires same secret. Тест `TestTokenIssuer_VerifyAccess_RejectsWrongSecret`. |
| `Authorization` header регистр (Bearer vs bearer) | Low | Принимаем только `Bearer ` (CamelCase). При появлении middleware в 1.4 решение пересматривается. |
| Дрейф между timezone в БД и сервере при сравнении `expires_at` | Low | Все timestamps в `timestamptz` (UTC, `migrations/0002_users.up.sql:11`, `migrations/0003_refresh_tokens.up.sql:9-10`). Use case передаёт `clock.Now()` — `time.Now().UTC()` в реальной реализации. |
| Использование `t.Sleep` в тестах для проверки expiration | Low | Все тесты инжектят `Clock`-фейк (`prompts/Tests Style.txt:55-62`). Никаких real-sleep. |
| Логирование пароля или токена | High | Никогда не логируем `password`, `accessToken`, `refreshToken`. В `slog.Error` пишем только `userID` и обёрнутую ошибку без чувствительных значений. |

## Open Questions

- [x] **OQ-1.** Куда складывать JWT-impl и bcrypt-impl — в `repository/`, в `pkg/`, в новый слой?
  - **Ответ:** все под `internal/auth/repository/` (см. ADR-005). Никаких новых верхних слоёв в `internal/auth/`.
- [x] **OQ-2.** Refresh-токен — JWT или random?
  - **Ответ:** random 32 байта (см. ADR-011). Совпадает с решением `docs/1_2_db_schema/token_and_index/03-decisions.md`, решение 2.
- [ ] **OQ-3.** Реакция на reuse-detection (использование уже отозванного refresh).
  - В MVP — только возврат `AUTH-008`.
  - Расширенная стратегия: отзывать ВСЕ refresh-токены этого пользователя (`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`) и логировать как security event. Запрос `RevokeAllRefreshTokensByUser` уже планируется (`docs/1_2_db_schema/token_and_index/06-repo-model.md:97-102`).
  - **Решение откладывается** до фазы 1.5 / 2 (когда появится security-логирование). Открытый вопрос для пользователя.
- [ ] **OQ-4.** Лимит активных refresh-токенов на пользователя.
  - В MVP — нет лимита. Один пользователь может иметь N активных сессий (по числу устройств).
  - Реализация (если потребуется): при INSERT нового refresh — отозвать самый старый, если активных >= N.
  - Аналог в Discord/Slack — лимит 5 устройств. **Откладывается**.
- [x] **OQ-5.** Тип `domain.PasswordHash` — `string` или VO?
  - **Ответ:** VO с приватным `value string`. Конструктор `NewPasswordHash(raw string) (PasswordHash, error)` проверяет non-empty. Это согласуется с `prompts/Domain Model.txt:155-160` («value objects для хеша как обёртка»). Геттер `String()` нужен для передачи в `bcrypt.CompareHashAndPassword`.
- [x] **OQ-6.** Транзакционная стратегия в Login (SELECT user + INSERT refresh).
  - **Ответ:** без транзакции (две независимые SQL-операции). Промежуточный inconsistent-state не возникает: если INSERT упал, новых данных не сохранили, user уже существовал. Клиент получает 500 и повторяет.
- [ ] **OQ-7.** Регистр sensitivity для login по email.
  - В БД email хранится в `citext` (`migrations/0002_users.up.sql:9`) — case-insensitive lookup автоматический.
  - В домене `Email.NewEmail(...)` нормализует в lower-case (см. таблицу VO в `01-architecture.md`).
  - Двойная нормализация — избыточная защита. Можно убрать lowercase в VO, оставив `citext` БД-уровневой защитой. **Решение:** оставляем lowercase в VO для consistency между in-memory и БД-представлениями.
- [ ] **OQ-8.** `Email.NewEmail` regex — насколько строгий?
  - В MVP: `^[^@\s]+@[^@\s]+\.[^@\s]+$` — позволяет почти любой email с `@` и точкой в домене.
  - RFC 5322-compliant regex — over-engineering, отбрасывает валидные edge cases.
  - **Решение:** оставляем простой regex; верификация e-mail (отправка письма) — отдельная фаза.
- [x] **OQ-9.** Где жить `Clock`, `UUIDGenerator`, `RandomBytes`-реализациям?
  - **Ответ:** `internal/auth/usecase/` сам объявляет интерфейсы. Реальные реализации — мини-структуры в `cmd/server/main.go` или маленькие helper-пакеты под `pkg/`. Самый простой вариант — inline-структуры в `cmd/server/main.go`:
    ```go
    type realClock struct{}
    func (realClock) Now() time.Time { return time.Now().UTC() }
    type realUUID struct{}
    func (realUUID) New() uuid.UUID { return uuid.New() }
    type cryptoRand struct{}
    func (cryptoRand) Read(n int) ([]byte, error) {
        b := make([]byte, n)
        _, err := rand.Read(b)
        return b, err
    }
    ```
  - Их 3 штуки, 5 строк каждый — не выносить в отдельный пакет. Если повторно потребуются в `internal/room/usecase/` — там же объявляются как локальные структуры; можно вынести в `pkg/clock/`, `pkg/uuidgen/`, `pkg/rand/` при появлении 3-го потребителя (правило `prompts/Go style.txt:27` «не создавать пакеты ради одной функции»).
