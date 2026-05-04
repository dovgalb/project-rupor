---
date: 2026-05-04
researcher: Claude
commit: 714e58b
branch: feature/1-project-structure
research_question: "Текущее состояние структуры проекта project_rupor (аналог Discord, монорепо). Какие папки/точки входа существуют, что зафиксировано в стандартах, чего нет."
---

# Исследование: Текущая структура проекта Rupor

## Резюме

Репозиторий находится в стадии «pre-code»: код приложения отсутствует. На уровне корня лежит единственный артефакт Go — `go.mod` с пустым списком зависимостей (`go.mod:1-3`, модуль `github.com/dovgalb/project-rupor`, Go 1.25). Ни одной директории из плановой структуры (`cmd/`, `internal/`, `pkg/`, `migrations/`, `config/`, `web/`) нет, как и `Makefile`, `docker-compose.yml`, корневого `.gitignore`.

Существующая часть монорепо — это «методологическая обвязка»: восемь стандартов в `prompts/` (Clean Architecture, доменная модель, билдеры, тесты, стиль Go, маппинг репозиториев), три слэш-команды в `.claude/commands/` (`research_codebase`, `design_feature`, `implement_backend`), один сабагент `codebase-researcher`, и общий план MVP в `.claude/plans/general_plan.md`. Корневые `CLAUDE.md` и `README.MD` фиксируют целевую архитектуру и процесс работы с агентами, но описанная в `CLAUDE.md` структура (`cmd/server/`, `internal/<домен>/{domain,usecase,transport,repository}/`, `pkg/websocket/` и т.д.) на ФС отсутствует.

Текущая ветка — `feature/1-project-structure`, последний коммит `714e58b add gitignore` (добавлен только `.idea/.gitignore`). Файл `tasks/poject_structure.txt` содержит формулировку задачи: «это проект — аналог дискорда, я планирую его сделать монорепой, мне нужно добавить структуру проекта (папки, точки входа и т.д)». В `.thoughts/research/` до этого исследования файлов не было.

## Детальные результаты

### 1. Корень репозитория

- **Расположение**: `/Users/bogdanserbatov/Documents/pets/project_rupor/`
- **Содержимое (видимое)**:
  - `CLAUDE.md` — project instructions (целевая структура и стек)
  - `README.MD` — описание процесса (3 шага через слэш-команды агентов)
  - `go.mod` — `go.mod:1` `module github.com/dovgalb/project-rupor`; `go.mod:3` `go 1.25`; зависимостей нет
  - `init_project` — однострочная заметка: «нужно сделать Структура проекта, Docker Compose (PostgreSQL), миграции» (`init_project:1`)
  - `prompts/` — 8 файлов стандартов (см. раздел 4)
  - `.claude/` — конфигурация агентов и команд (см. раздел 3)
  - `.thoughts/research/` — пустая директория для ресерчей (создана `Apr 7 19:00`)
  - `tasks/poject_structure.txt` — описание текущего тикета (`tasks/poject_structure.txt:1-2`)
  - `.idea/` — настройки IDE (частично tracked: `.idea/.gitignore`)
- **Чего НЕТ**:
  - `cmd/`, `internal/`, `pkg/`, `migrations/`, `config/`, `web/`
  - `Makefile`, `docker-compose.yml`
  - корневого `.gitignore`
  - `go.sum` (зависимостей нет)
  - `sqlc.yaml`, `golangci.yml`, `.editorconfig`
  - тестов или любого `.go`-файла

### 2. Состояние Git

- **Ветка**: `feature/1-project-structure` (текущая), `main`, `remotes/origin/main`
- **HEAD**: `714e58b`
- **История коммитов**:
  - `f3813e4 init commit`
  - `ff4a36c init prompts and agents`
  - `12134bb init prompts and agents`
  - `714e58b add gitignore`
- **Tracked-файлы (`git ls-files`)** — 18 шт.:
  - `.claude/agents/codebase-researcher.md`
  - `.claude/commands/design_feature.md`
  - `.claude/commands/implement_backend.md`
  - `.claude/commands/research_codebase.md`
  - `.claude/plans/general_plan.md`
  - `.idea/.gitignore`
  - `CLAUDE.md`
  - `README.MD`
  - `go.mod`
  - `init_project`
  - `prompts/Architecture Layers.txt`
  - `prompts/Builder.txt`
  - `prompts/Clean architecture.txt`
  - `prompts/Domain Model.txt`
  - `prompts/Domain model test.txt`
  - `prompts/Go style.txt`
  - `prompts/RepoModel.txt`
  - `prompts/Tests Style.txt`
- **Untracked**: `.idea/inspectionProfiles/`, `.idea/modules.xml`, `.idea/project_rupor.iml`, `.idea/vcs.xml`, `tasks/`
- **Коммит `714e58b`** добавил только `.idea/.gitignore` (9 строк), корневого `.gitignore` он не создавал

### 3. Каталог `.claude/`

- **Расположение**: `.claude/`
- **Структура**:
  ```
  .claude/
  ├── agents/codebase-researcher.md
  ├── commands/design_feature.md
  ├── commands/implement_backend.md
  ├── commands/research_codebase.md
  └── plans/general_plan.md
  ```

#### 3.1 `.claude/agents/codebase-researcher.md`
- **Расположение**: `.claude/agents/codebase-researcher.md:1-37`
- **Описание**: декларация сабагента (`name: codebase-researcher`, `model: sonnet 4.6`). Описывает правила исследования: только описывать существующее, обязательные ссылки `файл:строка`, читать файлы целиком.
- **Формат вывода** определяется в `.claude/agents/codebase-researcher.md:22-37` (Резюме / Результаты / Ссылки на код).

#### 3.2 `.claude/commands/research_codebase.md`
- **Расположение**: `.claude/commands/research_codebase.md:1-116`
- **Описание**: слэш-команда `/research_codebase`. Запускает 2–4 параллельных задачи через сабагент `codebase-researcher`, синтезирует результаты, сохраняет ресерч в `.thoughts/research/YYYY-MM-DD-название-темы.md`.

#### 3.3 `.claude/commands/design_feature.md`
- **Расположение**: `.claude/commands/design_feature.md:1-738`
- **Описание**: слэш-команда `/design_feature`. Описывает 5-этапный процесс проектирования: понять задачу → ресерч (опционально) → дизайн по C4 (multi-file) → architect review → утверждение → план реализации по фазам (`docs/{feature-name}/plan/phase-NN.md`).
- **Артефакты дизайна**: `docs/{feature-name}/{README.md, 01-architecture.md, 02-behavior.md, 03-decisions.md, 04-testing.md, [05-events.md, 06-repo-model.md, 07-standards.md, 08-api-contract.md], plan/}` (`.claude/commands/design_feature.md:182-202`).
- **Точки входа для изучения** при дизайне (`.claude/commands/design_feature.md:53-68`): `cmd/server/main.go`, `internal/<domain>/{domain,usecase,transport/http,repository/postgres}/`, `migrations/`, `pkg/websocket/`. На момент исследования эти пути в репо отсутствуют.

#### 3.4 `.claude/commands/implement_backend.md`
- **Расположение**: `.claude/commands/implement_backend.md:1-888`
- **Описание**: слэш-команда `/implement_backend`. Lead-агент создаёт команду через `TeamCreate`, спавнит четырёх ревьюеров (`rv-build`, `rv-arch`, `rv-sec`, `rv-plan`) и backend-имплементеров, оркестрирует mob-цикл по фазам плана. Каждая фаза проходит 4 quality gates параллельно.
- **Стек, зафиксированный командой** (`.claude/commands/implement_backend.md:7`): Go 1.25+, chi, sqlc, golang-migrate, PostgreSQL, `nhooyr.io/websocket`. Слои: `internal/{домен}/{domain,usecase,transport/http,repository/postgres}`.

#### 3.5 `.claude/plans/general_plan.md`
- **Расположение**: `.claude/plans/general_plan.md:1-119`
- **Описание**: общий план MVP. Содержит ER-схему сущностей `User / Room / Channel / Message / Membership / InviteCode` (`.claude/plans/general_plan.md:5-25`), 5 фаз разработки (`.claude/plans/general_plan.md:27-66`), стек (`.claude/plans/general_plan.md:67-79`), список REST-эндпоинтов (`.claude/plans/general_plan.md:81-109`), формат WebSocket-событий (`.claude/plans/general_plan.md:111-119`).
- **Фаза 1 «Фундамент + Авторизация»** (`.claude/plans/general_plan.md:29-34`) явно перечисляет: «Структура проекта, Docker Compose (PostgreSQL), миграции» как первый пункт.

### 4. Каталог `prompts/` — стандарты проекта

Восемь файлов на русском, описывающих обязательные правила. Эти стандарты — единственный источник архитектурных ограничений для будущего кода.

#### 4.1 `prompts/Architecture Layers.txt`
- **Расположение**: `prompts/Architecture Layers.txt:1-124` (10 806 байт)
- **Содержание**: правила Clean Architecture в адаптации под Go. Слои (`prompts/Architecture Layers.txt:14-66`): Domain → UseCase → Interface Adapters (transport/http, repository/postgres) → Frameworks & Drivers (`cmd/server`, `pkg`, `config`, `migrations`).
- **Правила импорта** (`prompts/Architecture Layers.txt:67-74`): `domain` импортирует только stdlib; `usecase` импортирует только `domain`; `transport`/`repository` импортируют `usecase` и `domain`; `cmd/server` собирает граф зависимостей.
- **Терминология** (`prompts/Architecture Layers.txt:118-123`): `usecase/` (не `service/`), интерфейсы — по роли (`UserRepository`, `PasswordHasher`, `Clock`), DTO use case — `RegisterUserInput`/`Output`, транспортные DTO — `RegisterUserRequest`/`Response`.

#### 4.2 `prompts/Clean architecture.txt`
- **Расположение**: `prompts/Clean architecture.txt:1-89` (7 516 байт)
- **Содержание**: концептуальный справочник «зачем» Clean Architecture. Базовые принципы (`prompts/Clean architecture.txt:9-15`), правило зависимостей с диаграммой (`prompts/Clean architecture.txt:17-26`), различение бизнес-правил и технических деталей (`prompts/Clean architecture.txt:40-53`), Use case vs Domain (`prompts/Clean architecture.txt:55-60`).

#### 4.3 `prompts/Domain Model.txt`
- **Расположение**: `prompts/Domain Model.txt:1-178` (10 350 байт)
- **Содержание**: правила построения rich domain model в `internal/<домен>/domain/`. Что лежит в domain (`prompts/Domain Model.txt:14-21`), конструкторы с инвариантами (`prompts/Domain Model.txt:32-50`), value objects (`prompts/Domain Model.txt:54-75`), методы как бизнес-операции (`prompts/Domain Model.txt:79-99`), два конструктора `NewXxx` / `ReconstructXxx` (`prompts/Domain Model.txt:124-129`), доменные ошибки (`prompts/Domain Model.txt:131-150`).

#### 4.4 `prompts/Builder.txt`
- **Расположение**: `prompts/Builder.txt:1-159` (7 886 байт)
- **Содержание**: правила паттерна Builder. Ограничен тестовыми фабриками и редкой композицией. Builder живёт в тест-пакете (`prompts/Builder.txt:23-28`), дефолты валидны (`prompts/Builder.txt:30-39`), `Build()` вызывает реальный конструктор и падает через `t.Fatalf` (`prompts/Builder.txt:50-67`), принимает `*testing.T` (`prompts/Builder.txt:67-71`).

#### 4.5 `prompts/RepoModel.txt`
- **Расположение**: `prompts/RepoModel.txt:1-176` (10 172 байт)
- **Содержание**: правила маппинга `domain ↔ хранилище`. Три типа структур (`prompts/RepoModel.txt:13-19`): доменная сущность, sqlc row, repo model. Маппинг через `domain.ReconstructXxx(...)` (`prompts/RepoModel.txt:79-83`), две функции `domainToRow`/`rowToDomain` (`prompts/RepoModel.txt:85-89`), мап `sql.ErrNoRows` → `domain.ErrXxxNotFound` (`prompts/RepoModel.txt:152-159`).

#### 4.6 `prompts/Go style.txt`
- **Расположение**: `prompts/Go style.txt:1-126` (10 159 байт)
- **Содержание**: расширения к Effective Go и Google Go Style Guide. Запрет ORM (`prompts/Go style.txt:97-101`), только sqlc; конфигурация только через env (`prompts/Go style.txt:84-87`); логирование через `log/slog` (`prompts/Go style.txt:89-94`); `panic` запрещён в продовом коде кроме composition root (`prompts/Go style.txt:44`); запрет на новые зависимости без согласования (`prompts/Go style.txt:111-117`).

#### 4.7 `prompts/Tests Style.txt`
- **Расположение**: `prompts/Tests Style.txt:1-228` (12 595 байт)
- **Содержание**: 5 уровней тестов (`prompts/Tests Style.txt:13-21`), стандартная библиотека `testing` без testify (`prompts/Tests Style.txt:24-26`), `t.Parallel()` везде (`prompts/Tests Style.txt:28-30`), фейки руками вместо моков (`prompts/Tests Style.txt:75-100`), интеграция репозиториев против реального PostgreSQL (`prompts/Tests Style.txt:120-124`).

#### 4.8 `prompts/Domain model test.txt`
- **Расположение**: `prompts/Domain model test.txt:1-134` (8 450 байт)
- **Содержание**: специфика тестов domain-слоя. Без моков, без фреймворков, чёрный ящик через публичный API (`prompts/Domain model test.txt:7-12`), table-driven (`prompts/Domain model test.txt:30-62`), детерминизм (`prompts/Domain model test.txt:92-98`), хелперы с `t.Helper()` (`prompts/Domain model test.txt:72-91`).

### 5. Каталог `tasks/`

- **Расположение**: `tasks/poject_structure.txt:1-2`
- **Содержимое**:
  ```
  project-structure:
  - это проект - аналог дискорда, я планирую его сделать монорепой, мне нужно добавить структуру проекта(папки, точки входа и т.д)
  ```
- **Статус git**: untracked.

### 6. Корневые методические файлы

#### 6.1 `CLAUDE.md` (project instructions)
- **Расположение**: `CLAUDE.md:1-115`
- **Описание**: фиксирует целевую архитектуру и стек. Целевая структура (`CLAUDE.md:5-30`):
  ```
  project-rupor/
  ├── cmd/server/          # точка входа
  ├── internal/
  │   ├── auth/{domain,usecase,transport/http,repository/postgres}/
  │   ├── user/, room/, channel/, chat/, voice/
  ├── pkg/websocket/
  ├── migrations/
  ├── config/
  ├── web/                 # React + Vite фронтенд
  ├── prompts/
  ├── docker-compose.yml
  ├── Makefile
  └── go.mod
  ```
- **Стек** (`CLAUDE.md:33-39`): Go 1.25+, chi, sqlc, golang-migrate, PostgreSQL, nhooyr.io/websocket, WebRTC, React+Vite+Zustand, Docker Compose.
- **Команды Makefile, описанные в документации** (`CLAUDE.md:42-50`): `make run / test / lint / migrate-up / migrate-down / build / dc-up`. Сам `Makefile` отсутствует.
- **API-контракт** (`CLAUDE.md:54`): префикс `/api/v1/`, JWT в `Authorization: Bearer`, WS — `/api/v1/ws?token=<jwt>`.
- **Конфигурация** (`CLAUDE.md:58-61`): `DATABASE_URL`, `JWT_SECRET`, `SERVER_PORT`.

#### 6.2 `README.MD`
- **Расположение**: `README.MD:1-13`
- **Описание**: процесс работы с агентами в 4 шага: `/research_codebase` → `/design_feature` → «напиши план» → `/implement_backend`.

#### 6.3 `go.mod`
- **Расположение**: `go.mod:1-3`
- **Содержание**:
  ```
  module github.com/dovgalb/project-rupor
  go 1.25
  ```
- Раздел `require` отсутствует — внешних зависимостей нет.

#### 6.4 `init_project`
- **Расположение**: `init_project:1`
- **Содержание**: «нужно сделать Структура проекта, Docker Compose (PostgreSQL), миграции».

### 7. Что описано в документации, но отсутствует в репо

Сводно (по сравнению `CLAUDE.md` ↔ фактическая ФС):

| Артефакт | Где упомянут | Существует? |
|---|---|---|
| `cmd/server/main.go` | `CLAUDE.md:7`, `Architecture Layers.txt:62` | Нет |
| `internal/auth/{domain,usecase,transport/http,repository/postgres}/` | `CLAUDE.md:9-13` | Нет |
| `internal/user/`, `room/`, `channel/`, `chat/`, `voice/` | `CLAUDE.md:14-19` | Нет |
| `pkg/websocket/` | `CLAUDE.md:21` | Нет |
| `migrations/` | `CLAUDE.md:22`, `general_plan.md:30` | Нет |
| `config/` | `CLAUDE.md:23`, `Go style.txt:86` | Нет |
| `web/` (React + Vite) | `CLAUDE.md:24`, `general_plan.md:60-65` | Нет |
| `docker-compose.yml` | `CLAUDE.md:26`, `init_project:1` | Нет |
| `Makefile` | `CLAUDE.md:27`, цели в `CLAUDE.md:42-50` | Нет |
| Корневой `.gitignore` | — | Нет (есть только `.idea/.gitignore`) |
| `go.sum` | — | Нет (зависимостей нет) |
| `sqlc.yaml` | `Go style.txt:97-101`, `RepoModel.txt:9` | Нет |
| `manual_qa/` | `implement_backend.md:792-851` | Нет |
| `docs/` (для дизайнов фич) | `implement_backend.md:14-17` | Нет |

## Ссылки на код

- `go.mod:1` — `module github.com/dovgalb/project-rupor`
- `go.mod:3` — `go 1.25`
- `CLAUDE.md:5-30` — целевая структура монорепо
- `CLAUDE.md:33-39` — целевой стек
- `CLAUDE.md:42-50` — целевые команды Makefile
- `CLAUDE.md:54-56` — формат API
- `CLAUDE.md:58-61` — env-переменные
- `init_project:1` — заметка с задачей фундамента
- `tasks/poject_structure.txt:1-2` — формулировка текущего тикета
- `README.MD:1-13` — процесс работы с агентами
- `.claude/plans/general_plan.md:5-25` — ER-схема сущностей MVP
- `.claude/plans/general_plan.md:27-66` — 5 фаз MVP
- `.claude/plans/general_plan.md:81-109` — список REST-эндпоинтов
- `.claude/plans/general_plan.md:111-119` — формат WS-событий
- `.claude/agents/codebase-researcher.md:1-4` — frontmatter сабагента (sonnet 4.6)
- `.claude/commands/research_codebase.md:1-116` — процесс ресерча
- `.claude/commands/design_feature.md:182-202` — структура артефактов дизайна
- `.claude/commands/implement_backend.md:7` — стек, ожидаемый имплементером
- `prompts/Architecture Layers.txt:14-66` — слои Clean Architecture
- `prompts/Architecture Layers.txt:67-74` — правила импорта между слоями
- `prompts/Architecture Layers.txt:118-123` — терминология (`usecase/`, имена интерфейсов, DTO)
- `prompts/Domain Model.txt:14-21` — что лежит в domain-пакете
- `prompts/Domain Model.txt:124-129` — `New` vs `Reconstruct` конструкторы
- `prompts/Go style.txt:84-87` — конфигурация только через env
- `prompts/Go style.txt:97-101` — только sqlc, никаких ORM
- `prompts/Go style.txt:111-117` — запрещённые без согласования действия
- `prompts/RepoModel.txt:79-83` — маппинг через `domain.ReconstructXxx`
- `prompts/Tests Style.txt:13-21` — пять уровней тестов

## Архитектурные наблюдения

- **Stage репозитория**: bootstrap. Существуют только `go.mod` и метаданные (CLAUDE.md, prompts, .claude). Нет ни одного `.go`-файла, нет миграций, нет `Makefile`/`docker-compose.yml`.
- **Поток процесса разработки** (фактически записан в `README.MD:1-13` и в `.claude/commands/`): `/research_codebase` → `/design_feature {feature} {research-path}` → «напиши план» → `/implement_backend {plan-path}`. Сейчас выполняется первый шаг для тикета `project-structure`.
- **Целевой архитектурный паттерн** (зафиксирован в `prompts/Architecture Layers.txt:14-74`): Clean Architecture в адаптации под Go с правилом импорта `domain ← usecase ← transport/repository ← cmd/server`. Терминология слоя сценариев — `usecase/`.
- **Целевой стек хранилища**: PostgreSQL + sqlc + golang-migrate (`prompts/Go style.txt:97-101`, `CLAUDE.md:33-39`). ORM запрещены прямым правилом.
- **Целевой стек реалтайма**: WebSocket через `nhooyr.io/websocket` (`CLAUDE.md:35`, `.claude/commands/implement_backend.md:7`), общий хаб в `pkg/websocket/`.
- **Доменная модель MVP** (`.claude/plans/general_plan.md:5-25`): корневые сущности `User`, `Room` (агрегат с членством и ролями owner/admin/member, инвайт-кодом), `Channel` (text/voice), `Message`. Связи между сущностями — по id (`prompts/Domain Model.txt:152-156`).
- **Текущая ветка `feature/1-project-structure`** соответствует тикету `tasks/poject_structure.txt`, описывающему задачу добавления структуры монорепо. Untracked-папка `tasks/` и untracked `.idea/*` присутствуют, ни одной заготовки кода под структуру в индексе или в working tree нет.
