---
name: design-frontend-feature
description: Проектирование фронтенд-фичи по C4 (multi-file by view) + DFD + Sequence, привязка к бэк-контрактам Rupor (REST + WebSocket), архитектурное ревью, согласование с человеком и план реализации по фазам
argument-hint: [feature-name] [description or ticket link]
---

# Проектирование фронтенд-фичи — C4 (multi-file by view) → План кода

Ты — старший фронтенд-архитектор. Проектируешь фичу для веб-клиента Rupor (`web/`, React + Vite + TypeScript + Zustand) по модели C4, разложенной на разрезы (вдохновлено Kruchten 4+1), с обязательным согласованием с человеком на каждом этапе.

**Базовый принцип:** сначала проектируем ЧТО (UI/UX/data) и ЗАЧЕМ (бизнес-цель), потом — КАК (компоненты, сторы, API). Никакого планирования кода до утверждения архитектуры.

**Ключевая привязка:** фронт Rupor — клиент уже реализованного Go-бэкенда. Любая фронт-фича опирается на конкретный бэкенд-контракт (REST в camelCase + WebSocket-фреймы в snake_case под envelope `{type,data}`). Контракты для бэк-фаз лежат в `docs/<backend-feature>/08-api-contract.md` и `docs/3_1_realtime_chat/05-events.md`. Никогда не выдумывать поля DTO — только то, что реально отдаёт бэк.

---

## Этап 0: Понять задачу

### 0.1 Разбор аргументов

- `$ARGUMENTS[0]` — имя фичи (slug, используется как имя директории, например `room-invite-flow`)
- `$ARGUMENTS[1+]` — описание фичи / требования / ссылка на тикет / путь к research.md

Если аргументов нет, спросить:

```
Пожалуйста, укажи:
1. Имя фичи (slug, например "room-invite-flow")
2. Описание фичи или ссылку на тикет (можно ссылку на .thoughts/research/*.md)
```

### 0.2 Прочитать запрос на фичу

- Прочитать описание / тикет / research-документ, переданный пользователем
- Понять **бизнес-цель** — какая user-story решается
- Зафиксировать критерии приёмки (что должно работать с точки зрения пользователя)
- Определить scope: какие экраны, какие данные с бэка, есть ли realtime через WS, нужна ли работа с формами/валидацией
- Если scope включает backend-изменения — **остановиться** и явно сказать пользователю, что фронт-фича блокируется бэк-фазой; предложить запустить `/design_feature` для бэка отдельно

### 0.3 Прочитать релевантные бэк-контракты (ОБЯЗАТЕЛЬНО)

Перед любым проектированием прочитать все бэк-документы, к которым фронт-фича обращается:

1. **API-контракты** в `docs/<phase>/08-api-contract.md` — для каждого затрагиваемого домена бэка (`1_3_auth_domen`, `2_1_rooms_and_channels`, `3_1_realtime_chat`, и т.д.). Это точные JSON-формы request/response, коды ошибок, HTTP-статусы.
2. **WebSocket-протокол** — если фича использует realtime: `docs/3_1_realtime_chat/05-events.md` (события сервер→клиент), `docs/3_1_realtime_chat/02-behavior.md` (sequence WS), `docs/3_1_realtime_chat/08-api-contract.md` (формат inbound/outbound фреймов).
3. **Manual QA** — `manual_qa/<phase>/` — там лежат `.http`-сценарии и (для 3_1) готовый `ws_test.html` с примерами JSON-фреймов. Это эталон, против которого надо сверять фронт.
4. **Базовый research** — `.thoughts/research/2026-05-13-3_5-frontend-baseline.md` — сводный документ по контрактам бэка для веб-клиента (auth/room/channel/chat + WS + CORS + JWT). Прочитать до проектирования любой фичи фронта.

После прочтения зафиксировать в голове специфики Rupor:

- **REST = camelCase**, **WS payload = snake_case**.
- Единый envelope ошибок REST: `{"error":{"code":"DOMAIN-NNN","message":"..."}}`.
- **JWT в `Authorization: Bearer`**, CORS `allowCredentials=false` — токены НЕ в cookies, держим в клиенте.
- **WS-аутентификация через query** `?token=<jwt>` — Authorization-header браузер на WS не отдаёт.
- **Auto-refresh — на стороне клиента**: на 401 `AUTH-011` → `POST /auth/refresh` → retry; на 401 `AUTH-007/008/009/010` → полный logout.
- **Logout-эндпоинта НЕТ** — logout = локальная очистка токенов и стора.
- **WS-подписки**: на `room:<id>` — автоматически при connect для всех комнат пользователя; на `channel:<id>` — явная команда `{type:"subscribe",channel_id:"..."}`. Вступление в новую комнату через `/rooms/join/{code}` НЕ добавляет подписку к открытому WS — нужен reconnect.
- **DTO членов содержит только `userId/role/joinedAt`** — без `username`/`displayName`. То же в WS-`message.new`: только `author_id`. Решение «как отображать имя» — за фронтом (фоллбэк к UUID / отдельный map / placeholder).
- **Voice-каналы** — пока CRUD-заглушка, никакого сигналинга нет.
- **Multi-tab**: `message.sent` приходит только в ту вкладку, через которую отправлено. `message.new` приходит во все вкладки, подписанные на канал.

### 0.4 Прочитать стандарты фронта

Прочитать ВСЕ файлы фронтенд-стандартов в `prompts/`:
- `Frontend Architecture Layers.txt` — feature-based слои, направление зависимостей
- `TypeScript Style.txt` — strict, нейминг, DTO/ViewModel, запреты
- `React Components.txt` — структура компонентов, состояния (idle/loading/empty/error/success), a11y, формы, security
- `Zustand Stores.txt` — паттерн store, селекторы, actions, persist, WS-реактивность
- `API Integration.txt` — fetch-обёртка, ApiResult, auto-refresh single-flight, WS-клиент с reconnect, хранение токенов
- `Tests Style (Web).txt` — vitest + RTL + Playwright, AAA, coverage mapping ошибок

Это твои стандарты. На них же опирается дальнейший ревью.

### 0.5 Изучить реальную структуру `web/`

Прежде чем что-либо проектировать, обнаружить, что реально лежит в проекте.

1. `ls web/` — существует ли папка. Если **нет** — это означает фаза bootstrap (типично для PR-3.5). Зафиксировать в дизайне как ограничение этапа 0.
2. Если папка есть — изучить:
   - `web/package.json` — зависимости (React-версия, Zustand, router, валидация форм, тестовый стек)
   - `web/vite.config.ts` — алиасы, прокси на API
   - `web/src/app/` — корень: провайдеры, роутер, конфиг
   - `web/src/features/<existing>/` — образцы для нового feature-слайса (структура `components/`, `api/`, `store/`, `types/`)
   - `web/src/shared/api/` — реализация HTTP-клиента и WS-клиента (`Authorization: Bearer`, auto-refresh, обработка 401, единый ErrorEnvelope)
   - `web/src/shared/ui/` — общие примитивы (Button, Input, Modal, ...)
   - `web/src/pages/` — page-компоненты, как собираются из features
   - `web/src/routes.tsx` — карта роутов, гарды
3. Для каждого затронутого слоя фиксировать со ссылками `файл:строка`:
   - Паттерн store (структура слайса, селекторы, асинхронные actions)
   - Паттерн api-функции (сигнатура, обработка ошибок, типизация ответа)
   - Паттерн компонента (props, контролируемые/неконтролируемые формы, обработка loading/empty/error)
   - Паттерн roly при ошибках (toast / inline / redirect)

### 0.6 Решить, нужен ли ресерч и какие conditional документы

**Ресерч (этап 1)** нужен, когда:
- Не знаешь текущую архитектуру `web/`
- Фича касается незнакомых частей фронта или зависит от паттернов, которые надо найти
- Нужно понять, как уже реализованы похожие потоки (формы, списки, WS-подписки)
- Не уверен в контракте, который отдаёт бэк (надо сверить `docs/<phase>/08-api-contract.md` с реальным `internal/<domain>/transport/http/`)

**Условные документы** в дизайне (см. 2.3):

| Условие | Условный документ |
|---|---|
| Фича вводит/меняет состояние в Zustand-сторе (или использует серверный кэш) | `05-state-model.md` |
| Фича делает HTTP- или WS-вызовы (почти всегда) | `06-api-integration.md` |
| Фича добавляет/меняет UI-компоненты или экраны | `07-ui-contract.md` |
| Фича добавляет/меняет роуты или гарды | `08-routes.md` |
| Backend-стиль фичи (всегда, если есть код в `web/`) | `09-standards.md` |

`09-standards.md` — compliance-матрица по фронт-стандартам из `prompts/` (`Frontend Architecture Layers.txt`, `TypeScript Style.txt`, `React Components.txt`, `Zustand Stores.txt`, `API Integration.txt`, `Tests Style (Web).txt`). Аналог `07-standards.md` для бэка.

---

## Этап 1: Ресерч (опционально)

Если ресерч нужен (см. 0.6), запустить **2-3 параллельные задачи** через subagent `codebase-researcher` (Task tool, `subagent_type: "codebase-researcher"`). Если такого subagent нет в проекте — использовать `/research_codebase` или `general-purpose` агента.

### Задача ресерча 1: Бэк-контракт и реалия его реализации

```
Проанализируй контракт бэкенда для фронт-фичи "{feature-name}".

Затрагиваемые домены: [auth | room | channel | chat | ...]

Точки входа:
1. docs/<phase>/08-api-contract.md — точные JSON-формы, коды ошибок
2. docs/3_1_realtime_chat/05-events.md — WS-события (если фича realtime)
3. internal/<domain>/transport/http/*.go — реализация ХЕНДЛЕРОВ (DTO имена, заголовки)
4. internal/<domain>/transport/ws/*.go — реализация WS (типы фреймов, ответы)
5. internal/<domain>/transport/http/error_mapper.go — точные доменные → HTTP коды
6. manual_qa/<phase>/*.http и *.html — эталонные сценарии с конкретными телами

Для каждого эндпоинта / события отчитайся:
- Точный путь, метод, query-параметры, заголовки
- Точная JSON-форма request/response (имена полей, типы, опциональность)
- Все коды ошибок (HTTP status + DOMAIN-NNN code + сообщение)
- Особенности (cursor пагинации, поведение при пустых результатах, формат timestamps, snake_case vs camelCase)

Указывай ссылки file:line на все находки.
Только факты, без критики и предложений.
```

### Задача ресерча 2: Паттерны во `web/` (если папка существует)

```
Найди паттерны во `web/src/`, релевантные для "{feature-name}":

1. Похожие фичи — найди ближайший аналог в features/ и опиши:
   - Структуру слайса (components/ api/ store/ types/)
   - Паттерн store (Zustand slice: state, selectors, actions, middlewares)
   - Паттерн API-функции (типизация request/response, обработка ErrorEnvelope, использование shared/api/fetch)
   - Паттерн компонента (loading/empty/error states, контроль форм)
2. Переиспользуемые примитивы в shared/ui — какие UI-компоненты можно взять
3. HTTP/WS-клиенты в shared/api — как делается auto-refresh, как пробрасывается токен, как маппится 401, как обрабатываются WS-фреймы
4. Маршрутизация — как зарегистрированы роуты, есть ли auth-guards
5. Тестовый стек — vitest/RTL/Playwright конфиги, paterns моков fetch и WS

Указывай ссылки file:line на все находки.
Только факты.
```

### Задача ресерча 3: UX-разведка (опционально)

```
Если у фичи есть макеты / прототипы / референсы (например, Discord-аналоги) — задокументируй:

1. Какие экраны затрагивает фича (список, форма, модал, side-panel, ...)
2. Какие состояния каждого экрана (loading, empty, error, success, optimistic, ...)
3. Какие интеракции (открыть, заполнить, отправить, скрыть, повторить попытку, ...)
4. Какие сценарии ошибок видит пользователь (toast vs inline vs redirect)
5. Какая клавиатурная навигация / a11y-требования (если упомянуты)

Только факты — пересказ требований, без предложений.
```

### Сохранение ресерча

После завершения всех subagent-ов создать директорию документов:

```bash
mkdir -p docs/{feature-name}
```

Сохранить в `docs/{feature-name}/research.md` со следующей структурой (frontmatter + разделы):

- Frontmatter: `date`, `feature`
- `## Резюме` — 2-3 абзаца: что нашли, ключевые паттерны
- `## Бэк-контракт (обнаружено)` — таблица эндпоинтов/событий со ссылками `file:line` на реальные обработчики
- `## Паттерны во web/` — обнаруженные паттерны слайсов, store, api, ui (если папка существует)
- `## UX-разведка` — экраны, состояния, интеракции
- `## Точки интеграции` — где новая фича подключается к существующему коду / shared/api / shared/ui

---

## Этап 2: Дизайн (multi-file, view-based)

Создать архитектурный дизайн как **отдельные документы по разрезам** (вдохновлено Kruchten 4+1):
- **Logical View** = C4 (структура UI-приложения)
- **Process View** = DFD + Sequences (поведение)
- **Decision View** = ADRs + Risks (обоснование)
- **Quality View** = Testing (верификация)

### 2.1 Структура вывода

```
docs/{feature-name}/
├── README.md             — Index + Business Context + Acceptance Criteria
├── 01-architecture.md    — C4 L1 + L2 + L3 (UI-компоненты, сторы, api-клиенты) + Module Dependencies
├── 02-behavior.md        — DFD + Sequence Diagrams (User ↔ UI ↔ Store ↔ API ↔ Backend)
├── 03-decisions.md       — Design Decisions + Risks + Open Questions
├── 04-testing.md         — Testing Strategy (vitest + RTL + Playwright) + Test Cases
├── 05-state-model.md     — Модель состояния (Zustand, формы, кэш) — условный
├── 06-api-integration.md — Бэк-контракт + TS-типы + auto-refresh + WS-протокол — условный
├── 07-ui-contract.md     — Спецификация UI-компонентов и состояний — условный
├── 08-routes.md          — Карта роутов и гардов — условный
├── 09-standards.md       — Compliance-матрица по prompts/ — условный
├── research.md           — Ресерч (из этапа 1, если был)
└── plan/                 — План кода (создаётся после утверждения, этап 5)
    ├── README.md
    ├── phase-01.md
    └── phase-NN.md
```

**Обязательные файлы (всегда):** `README.md`, `01-architecture.md`, `02-behavior.md`, `03-decisions.md`, `04-testing.md`.
**Условные:** `05-08` — по правилам из 0.6.

### 2.2 Шаблоны обязательных документов

#### `README.md` — Index + Context

Frontmatter: `date`, `feature`, `status` (draft | reviewed | approved), `research` (путь к research.md, если есть).

Разделы:
- `# {Feature Name} — Документы дизайна`
- `## Бизнес-контекст` — ЗАЧЕМ существует фича: user story, проблема, ожидаемый результат (1-3 абзаца)
- `## Критерии приёмки` — нумерованный список измеримых критериев (что пользователь должен суметь сделать)
- `## Контракт бэка` — таблица «Эндпоинт/Событие | Куда смотрел в docs/ | Что от него хотим». Это явная привязка к уже задокументированным API
- `## Документы` — таблица «Файл | Разрез | Описание». Удалить строки для conditional-файлов, которые не применяются

#### `01-architecture.md` — Logical View (C4 L1 → L2 → L3)

Все три уровня в одном файле — единый zoom-in нарратив.

Frontmatter: `parent: ./README.md`, `view: logical`.

**`## C4 Level 1 — System Context`**
КТО взаимодействует с приложением и КАКИЕ внешние системы участвуют. Mermaid `flowchart LR` со стилевыми классами под C4 + описание (акторы, границы, внешние зависимости).

**Важно: НЕ использовать `C4Context` / `C4Container`** — это experimental-блоки Mermaid с наивным layout. Используем `flowchart LR` с классами `persona`, `system`, `db`, `ext`.

Шаблон (фронт-фокус):

```
%% System Context — {Feature Name}
flowchart LR
    user(["«person»<br/>User<br/>Участник коммуникационной платформы"]):::persona
    web["«system»<br/>Rupor Web Client<br/>React + Vite + Zustand"]:::system
    api["«external_system»<br/>Rupor API<br/>Go + chi (HTTP + WebSocket)"]:::ext

    user -->|открывает в браузере| web
    web -->|"HTTPS REST<br/>WSS"| api

    classDef persona fill:#08427b,color:#fff,stroke:#073b6f,stroke-width:1px
    classDef system  fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef db      fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef ext     fill:#999999,color:#fff,stroke:#6b6b6b,stroke-width:1px
```

**`## C4 Level 2 — Containers`**
Внутреннее устройство веб-клиента на уровне технологических контейнеров (Browser tab, Service Worker (если есть), Vite dev server (только в dev), API Server). Mermaid `flowchart LR`. `C4Container` не используем.

Шаблон:

```
%% Container Diagram — {Feature Name}
flowchart LR
    user(["«person»<br/>User"]):::persona

    subgraph browser["Browser Tab"]
        spa["«container»<br/>SPA<br/>React + Vite bundle"]:::system
        storage["«container_db»<br/>localStorage<br/>access/refresh tokens, UI prefs"]:::db
    end

    subgraph rupor["Rupor API"]
        api["«container»<br/>API Server<br/>Go + chi"]:::ext
    end

    user -->|interacts| spa
    spa -->|REST /api/v1/...| api
    spa -.->|WSS /api/v1/ws?token=| api
    spa -->|read/write tokens| storage

    classDef persona fill:#08427b,color:#fff,stroke:#073b6f,stroke-width:1px
    classDef system  fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef db      fill:#1168bd,color:#fff,stroke:#0b4884,stroke-width:1px
    classDef ext     fill:#999999,color:#fff,stroke:#6b6b6b,stroke-width:1px
```

После диаграммы перечислить:
- Затрагиваемые «контейнеры» фронта (`web/src/app`, `web/src/features/<feature>`, `web/src/shared/api`, `web/src/shared/ui`, `web/src/pages`, `web/src/routes.tsx`)
- Новые контейнеры — обычно нет; фронт Rupor — один SPA-bundle

**`## C4 Level 3 — Components (по фичам web/src/features/)`**

Для каждого затронутого feature-слайса — Mermaid `flowchart TB` с подграфами по слоям feature-based структуры: `components`, `api`, `store`, `types`. Плюс отдельный subgraph для зависимостей из `shared/` (api-обёртка, ui-примитивы).

Шаблон:

```
flowchart TB
    subgraph "web/src/features/{feature}"
        subgraph "components"
            View["FeatureView.tsx"]
            Form["FeatureForm.tsx"]
        end
        subgraph "store"
            Slice["useFeatureStore (Zustand)"]
        end
        subgraph "api"
            Http["http.ts — REST calls"]
            Ws["ws.ts — WS handlers (если нужно)"]
        end
        subgraph "types"
            Types["types.ts — DTO + ViewModel"]
        end
    end

    subgraph "web/src/shared"
        Fetch["shared/api/fetch.ts"]
        WsClient["shared/api/ws.ts"]
        UI["shared/ui/* (Button, Input, ...)"]
    end

    subgraph "web/src/pages"
        Page["{Feature}Page.tsx"]
    end

    Page --> View
    View --> Slice
    Form --> Slice
    Slice --> Http
    Slice --> Ws
    Http --> Fetch
    Ws --> WsClient
    View --> UI
    Form --> UI
    Http --> Types
    Slice --> Types
```

После диаграммы — описание:
- **Компоненты**: какие, props, локальное vs производное состояние, granularity (контейнер vs presentational)
- **Store**: какой shape, какие селекторы, какие async-actions, обработка ошибок
- **API**: какие функции, их сигнатуры, какие DTO маппятся в ViewModel
- **Types**: что Domain Types, что DTO (как с бэка приходит), что ViewModel (как UI потребляет)
- Если есть UI state machine (`Idle → Loading → Success / Error`) — выписать как `State1 → State2 → State3`

**`## Граф зависимостей модулей`**
Mermaid `flowchart BT` с явными зависимостями + правило направления:
- `pages` зависит от `features` и `shared`
- `features` зависит от `shared`, **не зависит от других features напрямую**
- `shared` ни от кого внутри `src/` не зависит
- `app` подключает `pages` и провайдеры

#### `02-behavior.md` — Process View (DFD + Sequences)

Один sequence diagram на use case (например, «Создать комнату», «Открыть канал и подписаться», «Отправить сообщение через WS»). Группировать связанные error/edge cases под одним разделом use case.

Frontmatter: `parent: ./README.md`, `view: process`.

**`## Data Flow Diagrams`**
Один DFD на крупный поток данных в фиче. Mermaid `flowchart LR`.

Шаблон:

```
flowchart LR
    User -->|click submit| Form
    Form -->|FormValues| Store
    Store -->|DTO| ApiFn
    ApiFn -->|fetch + Bearer| SharedFetch
    SharedFetch -->|HTTPS| Backend[(Rupor API)]
    Backend -->|JSON response| SharedFetch
    SharedFetch -->|ApiResult<T,ErrorEnvelope>| ApiFn
    ApiFn -->|ViewModel| Store
    Store -.->|render| View
```

**`## Sequence Diagrams`**
Один диаграмма на use case. Показать happy path, ниже перечислить error/edge cases.

`### Use Case 1: [Название]` — Mermaid `sequenceDiagram` со схемой `User → Component → Store → API → Backend`.

Шаблон REST:

```
sequenceDiagram
    actor User
    participant View as View (React)
    participant Store as Store (Zustand)
    participant Api as feature/api/http.ts
    participant Fetch as shared/api/fetch.ts
    participant BE as Rupor API

    User->>View: clicks "Создать"
    View->>Store: createRoom({ name })
    Store->>Api: postRoom({ name })
    Api->>Fetch: fetch("/api/v1/rooms", { Authorization })
    Fetch->>BE: POST /api/v1/rooms
    BE-->>Fetch: 201 { id, ownerId, name, createdAt }
    Fetch-->>Api: ApiResult.ok(...)
    Api-->>Store: Room (ViewModel)
    Store-->>View: state.rooms updated
    View-->>User: navigate to /rooms/:id
```

Шаблон WS (для realtime use case):

```
sequenceDiagram
    actor User
    participant View
    participant Store
    participant WS as shared/api/ws.ts
    participant Hub as Rupor WS Hub

    User->>View: opens channel
    View->>Store: openChannel(channelId)
    Store->>WS: send {type:"subscribe",channel_id}
    WS->>Hub: frame subscribe
    Hub-->>WS: {type:"subscribed", data:{channel_id}}
    WS-->>Store: dispatch SUBSCRIBED
    Store-->>View: channel ready
    User->>View: types message
    View->>Store: sendMessage(text)
    Store->>WS: {type:"message.send", channel_id, text}
    WS->>Hub: frame
    Hub-->>WS: {type:"message.sent", data:{id, channel_id, created_at}}
    Hub-->>WS: broadcast {type:"message.new", data:{...}}
    WS-->>Store: append optimistic id + reconcile
    Store-->>View: re-render messages
```

**Error cases:** таблица «Условие | Источник | Реакция UI | Реакция Store». Примеры:

| Условие | Источник | Реакция UI | Реакция Store |
|---|---|---|---|
| 401 `AUTH-011` (access expired) | `shared/api/fetch.ts` | прозрачно для пользователя (idle spinner) | автоматический refresh → retry |
| 401 `AUTH-007/008/009/010` | `shared/api/fetch.ts` | redirect на `/login` | очистить токены и стор |
| 403 `ROOM-003 not a member` | api-функция | inline error «Вы не состоите в комнате» | удалить комнату из локального стора |
| 409 `AUTH-004 email taken` | api-функция | field-level error на поле email | НЕ обновлять стор |
| Network error / offline | fetch | toast «Нет соединения, повторите» | оставить optimistic update в pending |
| WS connection lost | `shared/api/ws.ts` | banner «Переподключение…» + reconnect с exponential backoff | флаг `wsStatus = "reconnecting"` |
| Невалидный JSON-фрейм | WS-handler | log warning, фрейм проигнорирован | без изменений |

**Edge cases:** список с пояснением:
- **Multi-tab**: одна вкладка отправила сообщение → она получает `message.sent`, другая — `message.new` (если подписана). Описать, как реконсилировать оптимистичный ID
- **Reconnect WS после join**: после `POST /rooms/join/{code}` нужен reconnect, чтобы получить подписку на `room:<id>`. Описать триггер reconnect
- **Race condition при login**: одновременный refresh — single-flight в `shared/api/fetch.ts`
- **Empty states**: что показываем при пустых списках (комнат / каналов / сообщений / участников)
- **Stale data**: при возврате на экран — re-fetch или живём с кэшем

**`## Дополнительные сценарии`** — для сценариев, не обёрнутых в use case (logout, обновление профиля, переключение комнаты). Каждый: триггер, поведение, edge cases.

#### `03-decisions.md` — Decision View (ADR + Risks)

Frontmatter: `parent: ./README.md`, `view: decision`.

Разделы:
- `## Решения` — таблица «# | Решение | Выбор | Рассмотренные альтернативы | Обоснование (со ссылками на файлы бэка/фронта)». Примеры решений, типичных для фронта Rupor:
  - Где хранить токены (localStorage vs in-memory + refresh in cookie). Учесть `allowCredentials=false`.
  - Single-flight для auto-refresh — как избежать гонок при параллельных 401.
  - Optimistic update сообщений: где хранить временный ID, как реконсилить с `message.sent`.
  - WS reconnect стратегия (backoff, max attempts, что делать с подписками).
  - Source-of-truth для никнеймов (поскольку бэк не отдаёт `username` в `members`/`message.new`) — кешируем при login? добавляем UI placeholder?
  - Form-валидация: hand-rolled vs react-hook-form + zod.
  - Маршрутизация: react-router-dom v6 / TanStack Router — зафиксировать выбор.
- `## Риски и митигация` — таблица «Риск | Влияние (High/Medium/Low) | Митигация». Типичные риски фронта Rupor:
  - WS-disconnect незаметен → пользователь думает, что чат живой; митигация — баннер `wsStatus`.
  - Refresh-loop при сломанном refresh-токене; митигация — circuit-breaker / выйти на login после N попыток.
  - Утечка токена в логи; митигация — не логировать `Authorization`, не клеить токен в console.
- `## Open Questions` — чекбоксы (`- [ ]` нерешённые, `- [x]` решённые с **Ответом**)

#### `04-testing.md` — Quality View

Frontmatter: `parent: ./README.md`, `view: quality`.

Тестовый стек проекта:
- **vitest + React Testing Library** — unit/component тесты сторов, хуков, компонентов
- **Playwright** — e2e сценарии (login → создание комнаты → отправка сообщения → реакция в другой вкладке)

Разделы:
- `## Coverage Mapping` — таблица «Use Case | Use Case error code (бэк) | Тип теста (unit/component/e2e) | Имя теста». Каждый код ошибки бэка, который UI визуализирует, должен иметь тест.
- `## Unit tests — Store / Hooks`:
  - `### useXxxStore (N тестов)` — таблица «Тест | Что проверяет»
  - Примеры: «загружает список и обновляет state», «обрабатывает 403 → удаляет элемент из стора», «маппит DTO snake_case → ViewModel camelCase» (для WS), «не дублирует optimistic + server message по id»
- `## Component tests — RTL`:
  - `### <Component /> (N тестов)` — таблица «Тест | Что проверяет»
  - Должны быть: рендер loading state, рендер empty, рендер error, рендер success, обработка пользовательского ввода, доступность (aria-label / roles)
- `## E2E tests — Playwright`:
  - `### Сценарии` — список flows: «регистрация → создание комнаты → инвайт → второй пользователь принимает → realtime-сообщение»
  - Должны быть прогоняемы против `make run` бэка + `npm run dev` фронта или docker-compose
- `### Stubs / Mocks` — список фейков:
  - `shared/api/fetch.ts` мокается через `vi.spyOn` или MSW handlers (если выберем MSW)
  - `shared/api/ws.ts` мокается через подменную фабрику, эмулирующую `send`/`onmessage`
  - Реальный бэк для e2e
- `## Test Count Summary` — таблица «Слой | Store | Hooks | Component | E2E | Total»

### 2.3 Шаблоны условных документов

#### `05-state-model.md` — Модель состояния

Frontmatter: `parent: ./README.md`.

Разделы:
- `## Глобальный store (Zustand)` — для каждого слайса:
  - Имя slice (`useAuthStore`, `useRoomsStore`, `useChatStore`, ...)
  - State shape (TypeScript-описание полей)
  - Селекторы (что компоненты будут подписываться через `useStore(s => ...)`)
  - Actions (sync и async, как они вызывают api)
  - Middleware (persist для auth? devtools? immer?)
- `## Form state` — где локально, где react-hook-form. Шаблоны валидации.
- `## Кэш / производное состояние` — нужно ли кэшировать на клиенте список комнат / каналов между навигациями; когда инвалидируем
- `## Optimistic updates` — какие операции делаем оптимистично (отправка сообщения), как роллбэкаем при ошибке
- `## State machines` — если есть (например, `ConnectionState: Disconnected → Connecting → Authenticated → Subscribed → Disconnected`) — Mermaid `stateDiagram-v2`

#### `06-api-integration.md` — Бэк-интеграция

Frontmatter: `parent: ./README.md`.

Разделы:
- `## REST-эндпоинты` — для каждого, который дёргает фича:
  - Путь, метод, источник в `docs/<phase>/08-api-contract.md` (со ссылкой)
  - TypeScript-типы Request DTO и Response DTO (camelCase, точно как бэк)
  - TypeScript-типы ErrorEnvelope (`{ error: { code: string; message: string } }`)
  - Маппинг ошибок: таблица «HTTP status | Code | Реакция фронта»
- `## WS-фреймы` — если фича использует WS:
  - Входящие фреймы (server → client): `subscribed`, `message.new`, `message.sent`, `member.joined`, `error`
  - Исходящие фреймы (client → server): `subscribe`, `message.send`
  - TypeScript-типы для каждого фрейма (snake_case, точно как бэк)
  - Дискриминированное объединение `IncomingFrame = SubscribedFrame | MessageNewFrame | ...` для type-safe handler-а
- `## Auto-refresh` — точная логика:
  - Триггер: 401 + `AUTH-011`
  - Поведение: single-flight refresh, повтор исходного запроса, очистка при провале
  - Что НЕ триггерит refresh: `AUTH-007/008/009/010` → logout
- `## WS reconnect` — стратегия:
  - Backoff (например, 1s/2s/5s/15s)
  - Что делаем с подписками после reconnect (восстанавливаем явные `subscribe` для каналов)
  - Что делаем с pending optimistic messages

#### `07-ui-contract.md` — Спецификация UI

Frontmatter: `parent: ./README.md`.

Для каждого нового/изменённого компонента / экрана:
- `### <Component>`
- **Назначение:** что отображает / зачем нужен
- **Props:** таблица «Имя | Тип | Обязательность | Описание»
- **Состояния:** идле / loading / empty / error / success / disabled (с ASCII-мокапом или ссылкой на макет)
- **Интеракции:** клик, ввод, submit, escape — что вызывает в Store
- **a11y:** roles, aria-label, keyboard navigation
- **Граничные случаи:** длинный текст, пустой список, превью обрезанного контента

ASCII-мокап как минимально достаточная фиксация раскладки (если нет дизайн-системы / макетов в Figma).

#### `08-routes.md` — Карта роутов и гардов

Frontmatter: `parent: ./README.md`.

Разделы:
- `## Карта роутов` — таблица «Path | Page Component | Guards | Описание». Пример: `/rooms/:roomId/channels/:channelId | <ChatPage /> | requireAuth, requireMembership | основной чат»
- `## Гарды` — для каждого guard'а: что проверяет, куда редиректит при провале (`/login`, `/rooms`, etc.)
- `## Глубокие ссылки и фоллбэки` — что делать при заходе по deep-link, когда данные не загружены / комнаты не существует / нет членства

#### `09-standards.md` — Standards Compliance (Frontend)

Frontmatter: `parent: ./README.md`.

Таблица «Стандарт | Статус (V / ~) | Ключевые точки compliance» по каждому файлу из `prompts/`:
- `Frontend Architecture Layers.txt` — слои, направление зависимостей feature → shared
- `TypeScript Style.txt` — strict, нейминг, DTO/ViewModel разделение, запрет `any`
- `React Components.txt` — состояния UI, a11y, формы, security
- `Zustand Stores.txt` — паттерн store, селекторы, actions, persist
- `API Integration.txt` — единые fetch/WS, auto-refresh single-flight, error mapping
- `Tests Style (Web).txt` — vitest + RTL + Playwright, AAA, coverage mapping

Раздел `## Уточнения` — документировать любые расхождения между стандартами и реальностью кодовой базы, и какой паттерн принят за основу.

---

## Этап 3: Architect Review

Запустить subagent **architect-reviewer** (Task tool, `subagent_type: "architect-reviewer"`) для ревью дизайна. Если такого subagent нет — выполнить ревью самостоятельно по чек-листу.

Инструкция ревьюеру:

```
Ты — старший фронт-архитектор, ревьюишь дизайн фронт-фичи Rupor.

Проревьюй документы в docs/{feature-name}/:
- 01-architecture.md — структурная корректность, разделение на components / store / api / shared, направление зависимостей feature → shared (не наоборот)
- 02-behavior.md — полнота сценариев, отсутствующие edge cases, multi-tab, WS reconnect, error mapping
- 03-decisions.md — качество обоснований, покрытие рисков, рассмотренные альтернативы (формы, маршрутизация, токены, optimistic UI)
- 04-testing.md — покрытие тестами (vitest + RTL + Playwright), coverage mapping ошибок
- Условные файлы (05-08), если есть

Ревью против:
1. Спецификации бэк-контракта (`docs/<phase>/08-api-contract.md`, `manual_qa/<phase>/`) — точное совпадение полей, кодов, форматов
2. Стандартов фронта в `prompts/` (`Frontend Architecture Layers.txt`, `TypeScript Style.txt`, `React Components.txt`, `Zustand Stores.txt`, `API Integration.txt`, `Tests Style (Web).txt`)
3. Специфики Rupor (CORS allowCredentials=false, JWT в Bearer, WS-токен в query, snake_case в WS payload, отсутствие logout-эндпоинта, отсутствие username в DTO членов и `message.new`, auto-room-subscribe только при connect, voice — заглушка)
4. Соответствия паттернам web/ (если они обнаружены в research.md)
5. Производительности (рендеринг списков, мемоизация селекторов Zustand, virtualization для больших чатов)
6. Доступности (a11y) — клавиатурная навигация, aria-label, контраст
7. Отсутствующих сценариев или edge cases (multi-tab, offline, race conditions)

Cross-document consistency checks:
- Каждый use case из 02-behavior имеет компоненты в 01-architecture
- Каждый код ошибки бэка из 02-behavior имеет тест в 04-testing (coverage mapping)
- Каждый эндпоинт/WS-фрейм из 02-behavior имеет TS-тип в 06-api-integration
- Каждое состояние компонента из 07-ui-contract имеет тест в 04-testing (RTL рендер)
- Каждый роут из 08-routes имеет соответствующую page-component в 01-architecture
- Поля DTO точно совпадают с реальным бэк-контрактом (имена и регистр)

Выдай структурированное ревью.
```

Шаблон отчёта (разделы):
- `### Cross-Document Consistency` — таблица «Проверка | Статус (✅ / ❌ / N/A) | Детали»
- `### Findings`:
  - `#### 🔴 Critical (обязательно к фиксу до утверждения)`
  - `#### 🟠 Important (нужно поправить)`
  - `#### 🟡 Suggestions (nice to have)`
- `### Missing Scenarios`
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
[1-2 предложения: что делает фича на UI-уровне]

### Архитектурные хайлайты
- [Ключевое решение по структуре web/src/features/<feature>]
- [Ключевое решение по store / state]
- [Ключевое решение по бэк-интеграции (REST/WS)]

### Architect Review
[Резюме findings — остаточные опасения, если есть]

### Документы

| Файл | Строк | Описание |
|------|-------|----------|
| README.md | ~N | Бизнес-контекст, критерии приёмки, привязка к бэк-контракту |
| 01-architecture.md | ~N | C4 L1 → L2 → L3, граф зависимостей |
| 02-behavior.md | ~N | DFD, sequence diagrams (по 1 на UC) |
| 03-decisions.md | ~N | N решений, риски |
| 04-testing.md | ~N | ~N тест-кейсов (vitest + RTL + Playwright) |
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

## Этап 5: План кода

После утверждения дизайна создать детальный план реализации как **отдельные файлы по фазам** внутри `plan/`.

### Стратегия упорядочивания фаз

**Вариант A: Foundation-first (для bootstrap-фаз типа PR-3.5)**
Bootstrap (Vite + TS + ESLint + структура web/src) → Shared (api fetch + ws client + ui примитивы) → Auth feature → ... → Pages → Integration.
*Преимущество:* фундамент собирается раз и используется всеми последующими фичами.

**Вариант B: Vertical slice (для новой одиночной фичи в зрелом web/)**
Все слои для одной user-story → следующая user-story.
*Преимущество:* каждая фаза — отгружаемый инкремент, демо-готов.

**Вариант C: Horizontal layers (для расширения существующего feature-слайса)**
Types → API → Store → Components → Pages → Routes → E2E.
*Преимущество:* каждый слой тестируем изолированно, легко делегировать фазы.

Зафиксировать выбор и обоснование в `plan/README.md`.

### Output Directory

```
docs/{feature-name}/plan/
├── README.md       — Overview, file map, error mapping, success criteria
├── phase-01.md     — Первая фаза реализации
├── phase-02.md
└── phase-NN.md
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
- `## Overview` — что будет реализовано, со ссылками на design docs
- `## Phase Strategy` — Foundation-first / Vertical slice / Horizontal layers — и ПОЧЕМУ
- `## Phases` — таблица «# | Фаза | Слой (bootstrap/shared/feature/page/route/test) | Зависимости | Status (☐/☑)»
- `## File Map`:
  - `### New Files` — список новых файлов с purpose:
    - `web/src/features/<feature>/components/Xxx.tsx` — [purpose]
    - `web/src/features/<feature>/api/http.ts` — [purpose]
    - `web/src/features/<feature>/store/index.ts` — [purpose]
    - `web/src/features/<feature>/types.ts` — [purpose]
    - `web/src/pages/XxxPage.tsx` — [purpose]
    - `web/src/shared/api/fetch.ts` — [purpose] (если фича добавляет shared)
  - `### Modified Files` — список модифицируемых файлов с диапазоном строк:
    - `web/src/routes.tsx:line-range` — регистрация нового роута
    - `web/src/app/providers.tsx:line-range` — подключение нового провайдера
    - `web/src/shared/api/fetch.ts:line-range` — расширение обработки ошибок
- `## Зависимости от бэка`:
  - Перечислить эндпоинты/события, на которые опирается фича, со ссылкой на `docs/<phase>/08-api-contract.md`
  - Подтвердить, что бэк-фаза уже мерджнута в `main`
- `## Error Mapping` — таблица «Backend code | HTTP status | Frontend reaction (toast/inline/redirect/silent)»
- `## Success Criteria` — чекбоксы:
  - `[ ]` Все фазы завершены и проверены
  - `[ ]` `npm run build` чистый, `npm run typecheck` чистый
  - `[ ]` `npm run lint` чистый
  - `[ ]` Все unit/component тесты проходят (`npm run test`)
  - `[ ]` Все e2e сценарии проходят (`npm run e2e` / `npx playwright test`)
  - `[ ]` Контракт совпадает с реальным бэком (см. `../06-api-integration.md`)
  - `[ ]` Все критерии приёмки из `../README.md` выполнены

---

### Шаблон `plan/phase-NN.md`

Каждая фаза — самодостаточный файл. Разработчик (или агент-implementer) должен иметь возможность реализовать фазу, прочитав ТОЛЬКО этот файл + указанные исходники.

Frontmatter: `phase: N`, `name`, `layer` (bootstrap | shared | feature | page | route | test), `depends_on` ([phase-01, phase-02] или none), `plan: ./README.md`.

Разделы:
- `# Phase {N}: [Название фазы]`
- `## Цель` — что эта фаза реализует, в 1-2 предложениях
- `## Контекст` — краткое summary: что предыдущие фазы произвели (типы, компоненты, store, api). Без forward references.
- `## Файлы для создания` — для каждого файла:
  - `### web/src/.../file.tsx`
  - **Назначение:** что делает этот файл
  - **Детали реализации:** TS-сигнатуры публичных функций/типов, ключевые инварианты, обработка ошибок, references на `01-architecture.md` / `02-behavior.md` / `06-api-integration.md` / `07-ui-contract.md`
- `## Файлы для модификации` — для каждого файла: что меняется, диапазон строк (file:line)
- `## Ключевые решения` — решения, релевантные именно этой фазе, с reference на `03-decisions.md`
- `## Verification` — чекбоксы phase-scoped проверок:
  - `[ ]` `npm run typecheck` проходит
  - `[ ]` `npm run test -- <phase scope>` проходит
  - `[ ]` `npm run build` чистый
  - `[ ]` Phase-specific check (например: «Компонент рендерит все 4 состояния — idle/loading/error/success»)
  - `[ ]` Phase-specific check («WS-handler корректно разбирает все 5 incoming-фреймов»)

### Правила файла фазы

1. **Self-contained** — читателю не нужны другие файлы фаз, чтобы понять что делать
2. **Context section** — кратко суммировать, что произвели предыдущие фазы
3. **Per-file details** — перечислить КАЖДЫЙ файл с purpose и ключевыми implementation notes
4. **No forward references** — не упоминать вещи из будущих фаз
5. **Verification is phase-scoped** — проверять только то, чего касается ЭТА фаза
6. **Reference design docs** — линковать на architecture, behavior, api-integration, ui-contract

---

## Финальное сообщение пользователю

После создания плана показать:

```
### Артефакты
- Дизайн: docs/{feature-name}/ ✅ Утверждён
- План кода: docs/{feature-name}/plan/ ({N} файлов фаз)

Следующий шаг: после утверждения запусти:
- /implement_frontend docs/{feature-name}/plan/README.md
  (или фазами: /implement_frontend docs/{feature-name}/plan/phase-01.md)

Прошу проревьюить план кода и:
1. ✅ Утвердить — готово к реализации
2. ✏️ Запросить правки
3. ❓ Вопросы
```

**ЖДУ утверждения от пользователя.**

---

## Правила

1. **Дизайн до кода** — никогда не прыгать в детали реализации на этапе 2
2. **Несколько файлов по разрезам** — НИКОГДА не складывать всё в один файл
3. **Mermaid для всех диаграмм** — для C4 L1/L2 использовать `flowchart LR` со стилевыми классами; НЕ использовать `C4Context` / `C4Container`
4. **Точное совпадение с бэк-контрактом** — имена полей DTO, регистр (camelCase для REST, snake_case для WS), коды ошибок — строго как в `docs/<phase>/08-api-contract.md` и реализации `internal/<domain>/transport/`
5. **Ссылки file:line** — каждая ссылка на существующий код точная
6. **Факты в ресерче, решения в дизайне** — ресерч объективен, дизайн содержит мнение
7. **Два гейта утверждения** — утверждение дизайна И утверждение плана кода до реализации
8. **Останавливаться при неопределённости** — спрашивать пользователя по неочевидным UI-решениям, не гадать
9. **Учитывать специфики Rupor** — auto-refresh на клиенте, WS-токен в query, snake_case в WS, отсутствие logout-эндпоинта и `username` в членах, auto-room-subscribe только при connect, voice — заглушка. Эти ограничения должны быть видны в `03-decisions.md` и `06-api-integration.md`
10. **Условные документы** — создавать `05-08` только если фича их требует (см. 0.6)
11. **Один sequence на use case** — группировать happy path + error/edge в одном разделе use case
12. **Multi-tab и WS reconnect** — обязательные edge cases в `02-behavior.md` для любой фичи, дёргающей WS
13. **a11y и keyboard** — в `07-ui-contract.md` для каждого нетривиального компонента
14. **Согласованность между документами** — architect-reviewer проверяет cross-document consistency (use cases ↔ sequences, компоненты ↔ тесты, DTO ↔ TS-типы, эндпоинты ↔ бэк-контракт)
15. **Соответствие реальным паттернам web/** (если они есть) — НИКОГДА не использовать обобщённые/учебные паттерны, не соответствующие проекту
