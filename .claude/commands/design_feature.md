---
name: design-feature
description: Проектирование фичи по C4 (multi-file by view) + DFD + Sequence, архитектурное ревью, согласование с человеком и план реализации по фазам
argument-hint: [feature-name] [description or ticket link]
---

# Проектирование фичи — C4 (multi-file by view) → План кода

Ты — старший программный архитектор. Проектируешь фичу по модели C4, разложенной на разрезы (вдохновлено Kruchten 4+1), с обязательным согласованием с человеком на каждом этапе.

**Базовый принцип:** сначала проектируем ЧТО и ЗАЧЕМ, потом — КАК. Никакого планирования кода до утверждения архитектуры.

---

## Этап 0: Понять задачу

### 0.1 Разбор аргументов

- `$ARGUMENTS[0]` — имя фичи (slug, используется как имя директории, например `voice-channel-mute`)
- `$ARGUMENTS[1+]` — описание фичи / требования / ссылка на тикет

Если аргументов нет, спросить:

```
Пожалуйста, укажи:
1. Имя фичи (slug, например "voice-channel-mute")
2. Описание фичи или ссылку на тикет
```

### 0.2 Прочитать запрос на фичу

- Прочитать описание / тикет, переданный пользователем
- Понять **бизнес-цель** — какую проблему решаем
- Зафиксировать критерии приёмки
- Определить scope: backend / frontend / БД / realtime / инфраструктура

### 0.3 Прочитать стандарты проекта

Прочитать ВСЕ стандарты в `promts/`:
- `Architecture Layers.txt`
- `Clean architecture.txt`
- `Domain Model.txt`
- `Builder.txt`
- `RepoModel.txt`
- `Go style.txt`
- `Tests Style.txt`
- `Domain model test.txt`

### 0.4 Изучить реальную структуру кодовой базы

Прежде чем что-либо проектировать, обнаружить реальные паттерны проекта.

Точки входа для изучения:
1. `cmd/server/main.go` — composition root, цепочка инициализации, регистрация всех зависимостей
2. `cmd/server/` — порядок init, регистрация роутов через `chi.Router`
3. `internal/<domain>/domain/` — все сущности, VO, билдеры, доменные ошибки, доменные события
4. `internal/<domain>/usecase/` — все сценарии, интерфейсы зависимостей (порты)
5. `internal/<domain>/transport/http/` — HTTP-хендлеры, маппинг ошибок, регистрация роутов
6. `internal/<domain>/repository/postgres/` — sqlc-запросы и реализации репозиториев
7. `migrations/` — существующие таблицы, индексы
8. `pkg/websocket/` — устройство hub'а, формат событий, подписки

Для каждого затронутого домена зафиксировать (со ссылками `файл:строка`):
- Структуру слоёв и пакетов
- Использование Builder pattern (3-фазный порядок)
- Паттерны Value Objects
- Паттерны маппинга в репозитории (`NewFromEntity` / `ToEntity` / `Restore`)
- Паттерны доменных событий (если есть)

### 0.5 Решить, нужен ли ресерч

Ресерч (этап 1) нужен, когда:
- Не знаешь текущую архитектуру затронутых модулей
- Фича касается незнакомых частей кодовой базы
- Существуют реализации, которые могут быть переиспользованы
- Точки интеграции неясны
- Нужно проверить ranges кодов ошибок

### 0.6 Определить контекст фичи

Решить, какие **условные документы** понадобятся (см. 2.3):

| Условие | Условный документ |
|---------|-------------------|
| Фича порождает доменные события | `05-events.md` |
| Backend-фича с сущностями | `06-repo-model.md` |
| Backend-фича (всегда, если есть код на Go) | `07-standards.md` |
| Фича выставляет REST-эндпоинты | `08-api-contract.md` |

Realtime через WebSocket / WebRTC документируется в `02-behavior.md` как часть процессного view, отдельный файл не создаём.

---

## Этап 1: Ресерч (опционально)

Если ресерч нужен (см. 0.5), запустить **2-3 параллельные задачи** через subagent `codebase-researcher` (Task tool, `subagent_type: "codebase-researcher"`). Если такого subagent нет в проекте — использовать `/research_codebase`.

### Задача ресерча 1: Архитектурный анализ

```
Проанализируй архитектуру модулей, затрагиваемых фичей "{feature-name}".

Точки входа:
1. cmd/server/main.go — composition root, цепочка инициализации, все зарегистрированные зависимости
2. cmd/server/ — порядок init, регистрация роутов через chi.Router, middleware
3. internal/<domain>/domain/ — список всех сущностей, VO, билдеров, состояний
4. internal/<domain>/usecase/ — список всех сценариев, их зависимости (порты)
5. internal/<domain>/transport/http/ — список хендлеров, маппинг ошибок
6. internal/<domain>/repository/postgres/ — sqlc-запросы, реализации

Для каждого релевантного компонента отчитайся:
- Структура слоёв и пакетов
- Использование Builder pattern (3-фазный порядок)
- Паттерны Value Objects
- Паттерны маппинга в репозитории (NewFromEntity / ToEntity / Restore)
- Паттерны доменных событий (если есть)

Указывай ссылки file:line на все находки.
Сохрани находки — только факты, без критики и предложений.
```

### Задача ресерча 2: Поиск паттернов

```
Найди паттерны, релевантные для "{feature-name}" в internal/ и pkg/:

1. Похожие фичи — найди ближайший аналог и опиши его полную структуру
2. Переиспользуемые компоненты в pkg/ — общие примитивы, утилиты
3. API-паттерны — регистрация роутов через chi, формат request/response, структура error response
4. Паттерны тестирования — структура test suite, mocks (через интерфейсы), stubs
5. Диапазоны кодов ошибок — просканируй ВСЕ коды ошибок, чтобы построить карту используемых диапазонов и найти следующий свободный

Указывай ссылки file:line на все находки.
Сохрани находки — только факты.
```

### Задача ресерча 3: Анализ интеграций

```
Проанализируй точки интеграции для "{feature-name}":

1. Внешние сервисы — HTTP-клиенты, очереди, сторонние API (если есть)
2. Event-driven коммуникация — WebSocket hub в pkg/websocket, доменные события, обработчики
3. Общие типы и контракты — интерфейсы, экспортируемые в другие модули
4. Затрагиваемые таблицы PostgreSQL — существующие таблицы, индексы, миграции
5. Auth / middleware — как применяется JWT-авторизация к роутам

Указывай ссылки file:line на все находки.
Сохрани находки — только факты.
```

### Сохранение ресерча

После завершения всех subagent-ов создать директорию документов и собрать findings:

```bash
mkdir -p docs/{feature-name}
```

Сохранить в `docs/{feature-name}/research.md` со следующей структурой (frontmatter + разделы):

- Frontmatter: `date`, `feature`
- `## Резюме` — 2-3 абзаца: что существует, что релевантно, ключевые паттерны
- `## Структура проекта (обнаружено)` — порядок init chain, паттерн роутера, паттерн сущностей, паттерн репозитория, используемые диапазоны кодов ошибок, следующий свободный диапазон
- `## Обзор архитектуры` — текущая архитектура затронутых модулей
- `## Существующие паттерны` — найденные паттерны со ссылками file:line
- `## Точки интеграции` — где новая фича подключается к существующему коду

---

## Этап 2: Дизайн (multi-file, view-based)

Создать архитектурный дизайн как **отдельные документы по разрезам** (вдохновлено Kruchten 4+1):
- **Logical View** = C4 (структура)
- **Process View** = DFD + Sequences (поведение)
- **Decision View** = ADRs + Risks (обоснование)
- **Quality View** = Testing (верификация)

### 2.1 Структура вывода

```
docs/{feature-name}/
├── README.md             — Index + Business Context + Acceptance Criteria
├── 01-architecture.md    — C4 L1 + L2 + L3 + Module Dependencies (Logical View)
├── 02-behavior.md        — DFD + Sequence Diagrams (Process View)
├── 03-decisions.md       — Design Decisions + Risks + Open Questions (Decision View)
├── 04-testing.md         — Testing Strategy + Test Cases (Quality View)
├── 05-events.md          — Доменные события (условный)
├── 06-repo-model.md      — Модель репозитория и sqlc (условный, backend)
├── 07-standards.md       — Compliance матрица (условный, backend)
├── 08-api-contract.md    — REST-контракт (условный, при наличии HTTP-эндпоинтов)
├── research.md           — Ресерч (из этапа 1, если был)
└── plan/                 — План кода (создаётся после утверждения дизайна, этап 5)
    ├── README.md
    ├── phase-01.md
    ├── phase-02.md
    └── phase-NN.md
```

**Обязательные файлы (всегда):** `README.md`, `01-architecture.md`, `02-behavior.md`, `03-decisions.md`, `04-testing.md`
**Условные файлы:** `05-08`, на основе анализа из 0.6

### 2.2 Шаблоны обязательных документов

#### `README.md` — Index + Context

Frontmatter: `date`, `feature`, `status` (draft | reviewed | approved), `research` (путь к research.md, если есть).

Разделы:
- `# {Feature Name} — Документы дизайна`
- `## Бизнес-контекст` — ЗАЧЕМ существует фича: проблема, потребность, ожидаемый результат (1-3 абзаца)
- `## Критерии приёмки` — нумерованный список измеримых критериев
- `## Документы` — таблица со столбцами «Файл | Разрез | Описание». Удалить строки для файлов, не применимых к фиче

Пример строки таблицы:

```
| [01-architecture.md](./01-architecture.md) | Logical | C4 диаграммы (L1 → L2 → L3), зависимости модулей |
```

#### `01-architecture.md` — Logical View (C4 L1 → L2 → L3)

Все три уровня C4 в одном файле — они образуют единый zoom-in нарратив.

Frontmatter: `parent: ./README.md`, `view: logical`.

Разделы:

**`## C4 Level 1 — System Context`**
КТО взаимодействует с системой и КАКИЕ внешние системы участвуют. Mermaid `flowchart LR` со стилевыми классами под C4 + описание (акторы, границы, внешние зависимости).

**Важно: НЕ использовать `C4Context` / `C4Container`** — это experimental-блоки Mermaid с наивным layout, у них наезжают подписи стрелок и нод. Используем `flowchart LR` с классами `persona`, `system`, `db`, `ext` и стереотипами `«person»` / `«system»` / `«system_db»` / `«external_system»` в подписях нод — рендер стабилен везде (GitHub, VS Code, Obsidian).

Шаблон диаграммы:

```
%% System Context — {Feature Name}
flowchart LR
    user(["«person»<br/>User<br/>Описание"]):::persona
    rupor["«system»<br/>Rupor<br/>Коммуникационная платформа"]:::system
    ext["«external_system»<br/>External System<br/>Описание"]:::ext

    user -->|использует| rupor
    rupor -->|вызывает| ext

    classDef persona fill:#08427b,color:#fff,stroke:#073b6f,stroke-width:1px
    classDef system  fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef db      fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef ext     fill:#999999,color:#fff,stroke:#6b6b6b,stroke-width:1px
```

Если на L1 есть БД, использовать форму цилиндра: `pg[("«system_db»<br/>PostgreSQL 16<br/>Описание")]:::db`.

**`## C4 Level 2 — Containers`**
КАКИЕ контейнеры/процессы участвуют и КАК они общаются. Mermaid `flowchart LR` с `subgraph` под границу системы и теми же стилевыми классами. **`C4Container` не используем** — по той же причине, что и `C4Context`.

Шаблон:

```
%% Container Diagram — {Feature Name}
flowchart LR
    user(["«person»<br/>User"]):::persona
    browser["«container»<br/>Browser<br/>React + Vite + Zustand"]:::ext

    subgraph rupor["Rupor"]
        api["«container»<br/>API Server<br/>Go + chi<br/>HTTP + WebSocket"]:::system
        db[("«container_db»<br/>PostgreSQL<br/>Хранит данные платформы")]:::db
    end

    user -->|открывает UI| browser
    browser -->|"HTTPS / WSS"| api
    api -->|"SQL / sqlc"| db

    classDef persona fill:#08427b,color:#fff,stroke:#073b6f,stroke-width:1px
    classDef system  fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef db      fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef ext     fill:#999999,color:#fff,stroke:#6b6b6b,stroke-width:1px
```

Условные обозначения по классам:
- `persona` (тёмно-синий, форма stadium `(["..."])`) — пользователи, акторы.
- `system` (синий, прямоугольник) — наша система или контейнер внутри неё.
- `db` (синий, цилиндр `[("...")]`) — БД (`system_db` / `container_db`).
- `ext` (серый, прямоугольник) — внешние системы / external контейнеры.

Подписи рёбер делать короткими — длинные технологические аннотации разносить на две строки через `<br/>` внутри label.

Затем перечислить:
- Затрагиваемые контейнеры (`cmd/server`, `internal/<domain>`, `web/`, `migrations/`, `pkg/websocket`)
- Новые контейнеры (обычно нет — Rupor монолит)

**`## C4 Level 3 — Components (по модулям internal/)`**
Для каждого затронутого модуля — Mermaid `flowchart TB` с подграфами по слоям Clean Architecture (`domain`, `usecase`, `transport/http`, `repository/postgres`) и стрелками между ними.

Шаблон:

```
flowchart TB
    subgraph "internal/{domain}"
        subgraph "domain"
            Entity["Entity"]
            VO["Value Objects"]
            Events["Domain Events"]
        end
        subgraph "usecase"
            UC["UseCase"]
            Port["«interface» Port"]
        end
        subgraph "transport/http"
            Handler["Handler"]
            DTO["DTO"]
        end
        subgraph "repository/postgres"
            Repo["Repository"]
            Queries["sqlc Queries"]
        end
    end

    Handler --> UC
    UC --> Port
    Repo -.implements.-> Port
    UC --> Entity
    Repo --> Queries
```

После диаграммы — описание сущностей, VO, state machines, ключевых интерфейсов. Если есть state machine — оформить как `State1 → State2 → State3`.

**`## Граф зависимостей модулей`**
Mermaid `flowchart BT` с зависимостями + правило (ограничения направления зависимостей по `promts/Architecture Layers.txt`).

#### `02-behavior.md` — Process View (DFD + Sequences)

Один sequence diagram на use case (не на сценарий). Группировать связанные error/edge cases под одним разделом use case.

Frontmatter: `parent: ./README.md`, `view: process`.

Разделы:

**`## Data Flow Diagrams`**
Один DFD на крупный поток данных в фиче. Mermaid `flowchart LR`.

Шаблон:

```
flowchart LR
    Client -->|HTTP| Handler
    Handler -->|DTO → Entity| UseCase
    UseCase -->|Port| Repo
    Repo -->|sqlc| DB[(PostgreSQL)]
    UseCase -.->|Event| WS[WebSocket Hub]
    WS -->|broadcast| Subscribers
```

**`## Sequence Diagrams`**
Один диаграмма на use case. Показать happy path, ниже перечислить error/edge cases.

`### Use Case 1: [Название]` — Mermaid `sequenceDiagram` со схемой `User → Handler → UseCase → Repo → DB → ...`

Шаблон:

```
sequenceDiagram
    actor User
    participant Handler
    participant UseCase
    participant Repo
    participant DB

    User->>Handler: HTTP Request
    Handler->>UseCase: Execute(params)
    UseCase->>Repo: Find(id)
    Repo->>DB: SELECT ...
    DB-->>Repo: Row
    Repo-->>UseCase: Entity
    UseCase-->>Handler: Result
    Handler-->>User: HTTP Response
```

**Error cases:** таблица «Условие | Код ошибки | HTTP Status | Поведение». Примеры строк:
- Сущность не найдена | DOMAIN-XXX | 404 | Вернуть error
- Невалидное состояние | DOMAIN-XXX | 409 | Вернуть error с текущим состоянием
- Валидация не прошла | DOMAIN-XXX | 400 | Вернуть field-level ошибки

**Edge cases:** список (race conditions → optimistic locking / serializable; timeouts → circuit breaker / fallback; …).

**`## Дополнительные сценарии`** — для сценариев, не обёрнутых в use case. Каждый сценарий: триггер, поведение, edge cases.

#### `03-decisions.md` — Decision View (ADR + Risks)

Frontmatter: `parent: ./README.md`, `view: decision`.

Разделы:
- `## Решения` — таблица «# | Решение | Выбор | Рассмотренные альтернативы | Обоснование (со ссылками file:line как доказательство из кодовой базы)»
- `## Риски и митигация` — таблица «Риск | Влияние (High/Medium/Low) | Митигация»
- `## Open Questions` — чекбоксы (`- [ ]` нерешённые, `- [x]` решённые с **Ответом**)

#### `04-testing.md` — Quality View

Frontmatter: `parent: ./README.md`, `view: quality`.

Разделы:
- `## Coverage Mapping` — таблица «Use Case | Error Code | Тест» (например, `TestUseCase_Condition_ReturnsXXX`)
- `## [Модуль 1] — Test Cases` → `### [Сущность / Компонент] ([N] тестов)` → таблица «Тест | Что проверяет»
- `### Stubs / Mocks` — список фейковых зависимостей и что они возвращают
- `## Repo Model Round-Trip Tests` — таблица (например, `TestXxxModel_RoundTrip_AllFieldsPreserved` — Entity → Model → Entity без потери данных)
- `## Integration Tests (репозитории)` — гоняются против реального PostgreSQL (см. `promts/Tests Style.txt`)
- `## Test Count Summary` — таблица «Модуль | Entity | Builder | UseCase | VO | Repo Model | Integration | Total»

### 2.3 Шаблоны условных документов

#### `05-events.md` — Доменные события

Frontmatter: `parent: ./README.md`.

Для каждого события: имя, таблица полей (Поле | Тип | Описание), где публикуется (usecase), кто подписан (например, `pkg/websocket` для рассылки клиентам), WS-топик (если уходит в websocket).

#### `06-repo-model.md` — Модель репозитория и sqlc (Backend only)

Frontmatter: `parent: ./README.md`.

Разделы:
- `## Маппинг сущность ↔ модель БД` — таблица «Поле сущности (VO) | Поле модели (raw) | Конверсия». Примеры: `Name()` ↔ `Name string` через `.Value()` / `primitives.NewName()`; `Status()` ↔ `Status string` через `.String()` / `StatusFromString()`; `CreatedAt()` ↔ `CreatedAt int64` через `.UnixNano()` / `time.FromUnixNano()`
- `## sqlc Queries (сигнатуры)` — sql-блок с сигнатурами вида:

```sql
-- name: GetXxxByID :one
SELECT ... FROM xxx WHERE id = $1;

-- name: ListXxxByYyy :many
SELECT ... FROM xxx WHERE yyy = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: InsertXxx :exec
INSERT INTO xxx (...) VALUES (...);
```

- `## Миграции` — имена файлов (`migrations/NNNN_create_xxx.up.sql` / `.down.sql`), список индексов с обоснованием, соответствие `promts/RepoModel.txt`

#### `07-standards.md` — Standards Compliance (Backend only)

Frontmatter: `parent: ./README.md`.

Таблица «Стандарт | Статус (✅ / ⚠️) | Ключевые точки compliance» по каждому файлу из `promts/`:
- `Architecture Layers.txt`
- `Builder.txt` (3-фазный порядок Builder)
- `Clean architecture.txt` (направление зависимостей)
- `Domain Model.txt` (инкапсуляция, VO, инварианты)
- `Domain model test.txt`
- `Go style.txt`
- `RepoModel.txt` (маппинг VO ↔ raw model, round-trip)
- `Tests Style.txt`

Раздел `## Уточнения` — документировать любые расхождения между стандартами и существующими паттернами в кодовой базе, и какой паттерн принят за основу.

#### `08-api-contract.md` — HTTP API Contract (если есть REST-эндпоинты)

Frontmatter: `parent: ./README.md`.

Для каждого эндпоинта (`## METHOD /api/v1/path`):

**Request:** код-блок с query params / headers (формат, типы, default-значения, обязательность). Пример:

```
Query params:
  page: int (default 1)
  limit: int (default 20, max 100)
  status: string? (optional, filter by status)
Headers:
  Authorization: Bearer {token}
```

**Response (XXX):** json-блок с ТОЧНОЙ формой ответа (имена полей, типы). Пример:

```json
{
  "items": [
    {
      "id": "uuid-string",
      "title": "string",
      "status": "active|archived",
      "createdAt": "2026-01-01T00:00:00Z",
      "author": {
        "id": "uuid-string",
        "name": "string"
      }
    }
  ],
  "total": 42,
  "page": 1,
  "limit": 20
}
```

**Error responses:** таблица «Status | Error Code | Body | Когда». Примеры:
- 400 | DOMAIN-XXX | `{"error":{"code":"DOMAIN-XXX","message":"..."}}` | Невалидные query params
- 401 | — | `{"error":"unauthorized"}` | Нет / истёк token
- 409 | DOMAIN-YYY | `{"error":{"code":"DOMAIN-YYY","message":"..."}}` | Duplicate
- 500 | — | `{"error":"internal"}` | Внутренняя ошибка сервера

Повторить для каждого эндпоинта с точными JSON-формами.

---

## Этап 3: Architect Review

Запустить subagent **architect-reviewer** (Task tool, `subagent_type: "architect-reviewer"`) для ревью дизайна. Если такого subagent нет в проекте — выполнить ревью самостоятельно по чек-листу ниже.

Инструкция ревьюеру:

```
Ты — старший программный архитектор, ревьюишь дизайн фичи.

Проревьюй документы в docs/{feature-name}/:
- 01-architecture.md — структурная корректность, разделение слоёв, направление зависимостей
- 02-behavior.md — полнота сценариев, отсутствующие edge cases, покрытие ошибок
- 03-decisions.md — качество обоснований, покрытие рисков, рассмотренные альтернативы
- 04-testing.md — покрытие тестами, coverage mapping
- Условные файлы (05-08), если есть

Ревью против:
1. Стандартов проекта в promts/
2. Принципов Clean Architecture — направление зависимостей, разделение слоёв
3. Правил Domain Model — инкапсуляция, VO, инварианты
4. Согласованности с существующими паттернами кодовой базы (из research.md)
5. Скейлабельность и performance implications
6. Отсутствующих сценариев или edge cases

Cross-document consistency checks:
- Каждая сущность из 01-architecture имеет тест-кейсы в 04-testing
- Каждый use case из 01-architecture имеет sequence diagram в 02-behavior
- Каждый код ошибки из 02-behavior имеет тест в 04-testing (coverage mapping)
- Каждая сущность из 01-architecture имеет запись в 06-repo-model (если есть)
- Каждое поле в repo model 06 покрывает ВСЕ поля сущности из 01-architecture
- Каждый эндпоинт из 02-behavior имеет точные JSON-формы в 08-api-contract (если есть)
- Коды ошибок не конфликтуют с существующими диапазонами (из research.md)
- Каждый переход состояния в 01-architecture имеет sequence в 02-behavior И тест в 04-testing

Выдай структурированное ревью.
```

Шаблон отчёта ревьюера (разделы):

- `### Cross-Document Consistency` — таблица «Проверка | Статус (✅ / ❌ / N/A) | Детали» по каждому пункту чек-листа выше
- `### Findings` с подразделами:
  - `#### 🔴 Critical (обязательно к фиксу до утверждения)` — список «[Файл: описание]»
  - `#### 🟠 Important (нужно поправить)` — список
  - `#### 🟡 Suggestions (nice to have)` — список
- `### Missing Scenarios` — сценарии, не покрытые в 02-behavior.md
- `### Verdict: ✅ READY FOR REVIEW / ⚠️ NEEDS ITERATION`

### Если найдены проблемы

- Поправить findings в **конкретном файле**, где живёт проблема
- Перезапустить architect review, если изменения значимые
- Итерировать до ✅ READY FOR REVIEW

---

## Этап 4: Утверждение дизайна человеком

Показать пользователю:

```
## Дизайн готов к ревью: {Feature Name}

### Резюме
[1-2 предложения: что делает фича]

### Архитектурные хайлайты
- [Ключевое архитектурное решение 1]
- [Ключевое архитектурное решение 2]
- [Ключевое архитектурное решение 3]

### Architect Review
[Резюме findings ревью — есть ли остаточные опасения]

### Документы

| Файл | Строк | Описание |
|------|-------|----------|
| README.md | ~N | Бизнес-контекст, критерии приёмки |
| 01-architecture.md | ~N | C4 L1 → L2 → L3, зависимости модулей |
| 02-behavior.md | ~N | DFD, sequence diagrams (по 1 на UC) |
| 03-decisions.md | ~N | N решений, риски |
| 04-testing.md | ~N | ~N тест-кейсов с coverage mapping |
| ... | | [условные файлы] |

Все находятся: docs/{feature-name}/

Прошу проревьюить документы дизайна и:
1. ✅ Утвердить — переходим к плану кода
2. ✏️ Запросить правки — указать, что менять
3. ❓ Вопросы — задать по конкретным решениям
```

**ЖДУ утверждения от пользователя.** НЕ переходить к плану кода без явного утверждения.

Если пользователь просит правки:
1. Обновить **конкретный файл** с фидбеком (не все файлы)
2. Перезапустить architect review, если правки значимые
3. Снова показать на утверждение

---

## Этап 5: План кода (4-й C — Code Level)

После утверждения дизайна создать детальный план реализации как **отдельные файлы по фазам** внутри директории `plan/`.

### Стратегия упорядочивания фаз

Выбрать порядок фаз, исходя из типа фичи:

**Вариант A: Bottom-up (по умолчанию для большинства фич)**
Migrations → Domain → Repository → UseCase → Transport → Frontend
*Преимущество:* фундамент собирается снизу, каждая фаза тестируема изолированно.

**Вариант B: Adapter-first (для фич, расширяющих существующие сущности новой персистентностью)**
Migrations + Repository → Domain → UseCase → Transport → Frontend
*Преимущество:* модель данных валидирует допущения о хранилище раньше.

**Вариант C: Vertical slice (для фич с независимыми эндпоинтами)**
Все слои для Endpoint 1 → Все слои для Endpoint 2 → ...
*Преимущество:* каждая фаза — отгружаемый инкремент.

Зафиксировать выбранную стратегию и обоснование в `plan/README.md`.

### Output Directory

```
docs/{feature-name}/plan/
├── README.md       — Overview, file map, DI integration, error codes, success criteria
├── phase-01.md     — Первая фаза реализации
├── phase-02.md     — Вторая фаза
└── phase-NN.md     — Один файл на фазу
```

### Зачем разделять файлы?

- Каждая фаза **самодостаточна** — implementer читает ОДИН файл на задание
- Фазы можно ревьюить и утверждать **независимо**
- Лид может отдать единичный файл агенту-implementer без шума
- Прогресс трекается чекбоксом в README, файл фазы остаётся reference-документом

---

### Шаблон `plan/README.md`

Frontmatter: `date`, `feature`, `design: ../README.md`, `status` (draft | approved).

Разделы:

- `# План кода: {Feature Name}`
- `## Overview` — резюме: что будет реализовано, со ссылками на design docs
- `## Phase Strategy` — Bottom-up / Adapter-first / Vertical slice — и ПОЧЕМУ
- `## Phases` — таблица «# | Фаза | Слой | Зависимости | Status (☐/☑)»
- `## File Map`:
  - `### New Files` — список новых файлов с purpose:
    - `internal/{domain}/domain/xxx.go` — [purpose]
    - `internal/{domain}/usecase/xxx.go` — [purpose]
    - `internal/{domain}/transport/http/xxx.go` — [purpose]
    - `internal/{domain}/repository/postgres/queries/xxx.sql` — [purpose]
    - `migrations/NNNN_xxx.up.sql` / `.down.sql` — [purpose]
  - `### Modified Files` — список модифицируемых файлов с диапазоном строк:
    - `cmd/server/main.go:line-range` — регистрация новой зависимости
    - `internal/{domain}/.../existing.go:line-range` — [что меняется]
- `## DI Integration`:
  - **Init chain position:** где в реальной цепочке инициализации (обнаруженной в этапе 0.4)
  - **Composition root changes:** что добавить в `cmd/server/main.go` — ссылка file:line
  - **Initialization order:** шаг за шагом, со ссылками на зависимости
- `## Error Codes`:
  - **Range:** `{PREFIX}-XXX..{PREFIX}-YYY`
  - **Conflict check:** [Verified — нет конфликтов; перечислить используемые ranges из research.md]
  - Таблица «Code | Description | HTTP Status»
- `## Success Criteria` — чекбоксы:
  - `[ ]` Все фазы завершены и проверены
  - `[ ]` Все тесты проходят (`make test`, см. `../04-testing.md`)
  - `[ ]` Все коды ошибок покрыты тестами (см. coverage mapping в `../04-testing.md`)
  - `[ ]` `go build ./...` чистый
  - `[ ]` `make lint` чистый
  - `[ ]` API-контракт совпадает с реализацией (см. `../08-api-contract.md`, если есть)
  - `[ ]` Все критерии приёмки из `../README.md` выполнены

---

### Шаблон `plan/phase-NN.md`

Каждая фаза — самодостаточный файл. Разработчик (или агент-implementer) должен иметь возможность реализовать фазу, прочитав ТОЛЬКО этот файл + указанные исходники.

Frontmatter: `phase: N`, `name`, `layer` (domain | usecase | transport | repository | frontend | migrations), `depends_on` ([phase-01, phase-02] или none), `plan: ./README.md`.

Разделы:

- `# Phase {N}: [Название фазы]`
- `## Цель` — что эта фаза реализует, в 1-2 предложениях
- `## Контекст` — краткое summary: что предыдущие фазы произвели (типы, интерфейсы), без forward references на будущие фазы
- `## Файлы для создания` — для каждого файла:
  - `### internal/{domain}/.../file.go`
  - **Назначение:** что делает этот файл
  - **Детали реализации:** список бизнес-правил, инвариантов, constraint-ов VO, интерфейсов; references на design docs (`01-architecture.md` для структуры, `02-behavior.md` для логики, `08-api-contract.md` для JSON-форм)
- `## Файлы для модификации` — для каждого файла: что меняется, диапазон строк (file:line)
- `## Ключевые решения` — решения, релевантные ИМЕННО ЭТОЙ фазе, с reference на 03-decisions.md
- `## Verification` — чекбоксы phase-scoped проверок:
  - `[ ]` `go build ./...` проходит
  - `[ ]` `go test ./internal/{domain}/...` проходит
  - `[ ]` Phase-specific check (например: «Все поля сущности — Value Objects»)
  - `[ ]` Phase-specific check («Builder следует 3-фазному порядку»)
  - `[ ]` Phase-specific check («Коды ошибок совпадают с 08-api-contract.md»)

### Правила файла фазы

1. **Self-contained** — читателю не нужны другие файлы фаз, чтобы понять что делать
2. **Context section** — кратко суммировать, что произвели предыдущие фазы (типы, интерфейсы)
3. **Per-file details** — перечислить КАЖДЫЙ файл с purpose и ключевыми implementation notes
4. **No forward references** — не упоминать вещи из будущих фаз
5. **Verification is phase-scoped** — проверять только то, чего касается ЭТА фаза
6. **Reference design docs** — линковать на architecture, behavior, API contract для деталей реализации

---

## Финальное сообщение пользователю

После создания плана показать:

```
### Артефакты
- Дизайн: docs/{feature-name}/ ✅ Утверждён
- План кода: docs/{feature-name}/plan/ ({N} файлов фаз)

Следующий шаг: после утверждения запусти:
- Backend: /implement_backend docs/{feature-name}/plan/README.md
- Frontend: /implement_frontend docs/{feature-name}/plan/README.md

Прошу проревьюить план кода и:
1. ✅ Утвердить — готово к реализации
2. ✏️ Запросить правки — указать корректировки
3. ❓ Вопросы — задать по конкретным фазам
```

**ЖДУ утверждения от пользователя.**

---

## Правила

1. **Дизайн до кода** — никогда не прыгать в детали реализации на этапе 2
2. **Несколько файлов по разрезам** — разделять структуру (01), поведение (02), решения (03), тестирование (04). НИКОГДА не складывать всё в один файл
3. **Mermaid для всех диаграмм** — рендерится, версионируется, поддерживает diff. Для C4 L1/L2 использовать `flowchart LR` со стилевыми классами под C4 (см. шаблоны в 01-architecture); НЕ использовать experimental-блоки `C4Context` / `C4Container` — у них наезжают подписи
4. **Ссылки file:line** — каждая ссылка на существующий код содержит точное расположение
5. **Факты в ресерче, решения в дизайне** — ресерч объективен, дизайн содержит мнение
6. **Два гейта утверждения** — утверждение дизайна И утверждение плана кода до реализации
7. **Прочитать ВСЕ стандарты И обнаружить реальную структуру** — прочитать каждый файл в `promts/` И изучить настоящую кодовую базу (`cmd/server/main.go`, цепочка инициализации, роутер, сущности) перед проектированием
8. **Останавливаться при неопределённости** — спрашивать пользователя, не гадать архитектурные решения
9. **C4 zoom-in нарратив** — L1 → L2 → L3 в одном файле (`01-architecture.md`), они рассказывают одну непрерывную историю
10. **Условные файлы** — создавать 05-08 только если фича их требует (см. этап 0.6)
11. **Один sequence на use case** — группировать happy path + ошибочные кейсы + edge cases в одном разделе use case в `02-behavior.md`
12. **Точный API-контракт** — у каждого REST-эндпоинта обязательны точные JSON request/response с именами и типами полей в `08-api-contract.md`
13. **Согласованность между документами** — architect-reviewer проверяет, что все документы корректно ссылаются друг на друга (сущности ↔ тесты, use cases ↔ sequences, эндпоинты ↔ контракт, …)
14. **Проверка конфликтов кодов ошибок** — проверять, что новые коды ошибок не конфликтуют с существующими диапазонами перед назначением
15. **Соответствие реальным паттернам проекта** — обнаруженным на этапе 0.4. НИКОГДА не использовать обобщённые/учебные паттерны, не соответствующие кодовой базе
