# Project Rupor

Коммуникационная платформа (аналог Discord). Бэкенд на Go, фронтенд на React. На этапе MVP запускается для своих, чтобы собрать обратную связь. В перспективе — публичный сервис. Язык интерфейса: русский. В перспективе — добавление английского.

# TEAM GIT POLICY (MANDATORY)

This repository uses STRICT commit discipline.


Claude is NOT allowed to:
- push / pull / fetch
- change branches, HEAD, index, stash
- modify .git directory
- create or edit commits automatically

Claude is ONLY allowed to:
- run git
- run shell commands
- commit / amend
- generate commit message TEXT
- review diffs
- explain git commands
- suggest actions

Any attempt to execute git or shell commands is a violation.

## Структура проекта

```
project-rupor/
├── cmd/server/          # Точка входа, composition root
├── internal/
│   ├── auth/            # Регистрация, логин, JWT (access + refresh)
│   │   ├── domain/              # Сущности, value objects, доменные ошибки
│   │   ├── usecase/             # Сценарии + интерфейсы зависимостей
│   │   ├── transport/http/      # HTTP-хендлеры, DTO запросов/ответов
│   │   └── repository/postgres/ # Реализация репозиториев через sqlc
│   ├── user/            # Профили пользователей
│   ├── room/            # Комнаты (аналог серверов Discord), роли, инвайты
│   ├── channel/         # Каналы внутри комнат (text / voice)
│   ├── chat/            # Сообщения, история, пагинация
│   └── voice/           # Сигнальный сервер для WebRTC (SDP/ICE)
├── pkg/
│   └── websocket/       # WebSocket hub, подписки на каналы, рассылка событий
├── migrations/          # SQL-миграции (golang-migrate)
├── config/              # Загрузка конфигурации из env-переменных
├── web/                 # React + Vite фронтенд
├── prompts/             # Промпты для агентов (архитектура и т.п.)
├── docker-compose.yml   # PostgreSQL + Redis + сервер
├── Makefile             # run, test, migrate, lint, build
└── go.mod
```

Каждый домен в `internal/` следует Clean Architecture и делится на подпапки `domain/`, `usecase/`, `transport/`, `repository/`. Структура одинакова для всех доменов. Подробные правила слоёв и направления зависимостей — в `prompts/Architecture Layers.txt`.

## Стек

- **Go 1.25+** — `chi` (роутинг), `sqlc` (генерация кода из SQL), `golang-migrate` (миграции)
- **PostgreSQL** — единственная БД, хранит всё: пользователей, комнаты, каналы, сообщения
- **WebSocket** — `nhooyr.io/websocket`, реалтайм чат и сигнализация голоса
- **WebRTC** — P2P аудио через браузерный API, сервер только сигнализирует
- **React + Vite + Zustand** — фронтенд в `web/`
- **Docker Compose** — локальная и продакшн среда

## Команды

```bash
make run          # Запуск сервера
make test         # Тесты
make lint         # golangci-lint
make migrate-up   # Применить миграции
make migrate-down # Откатить миграцию
make build        # Сборка бинарника
make dc-up       # Docker Compose up
```

## API

Все эндпоинты под префиксом `/api/v1/`. Авторизация через JWT в заголовке `Authorization: Bearer <token>`. WebSocket подключение: `/api/v1/ws?token=<jwt>`.

## Конфигурация

Через переменные окружения:
- `DATABASE_URL` — строка подключения к PostgreSQL
- `JWT_SECRET` — секрет для подписи токенов
- `SERVER_PORT` — порт HTTP-сервера (по умолчанию 8080)

## Правила разработки

- Язык общения и комментариев в коде — русский
- Не добавлять библиотеки без согласования
- Не менять API-контракты без согласования
- Предпочитать простые явные решения
- Все функции должны быть тестируемыми, предпочитать чистые функции
- SQL через `sqlc`, не использовать ORM
- Конфигурация только через env, не через файлы

## Правила разработки с ai
1. `research_codebase + промт`(в .thoughts создается файл с результатами ресерча)
2. `design_feature + название тикета + .thoughts/название ресерча`