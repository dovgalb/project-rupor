---
date: 2026-05-12
feature: PR-3_1_realtime_chat
status: draft
research: ./research.md
---

# PR-3_1 — Реалтайм-чат — Документы дизайна

## Бизнес-контекст

Платформа Rupor — Discord-аналог. Каркас комнат и каналов готов, но текстовый канал пока не умеет ни принимать, ни хранить, ни доставлять сообщения. Без чата платформа неотличима от пустой админки: пользователь может создать комнату, пригласить друга, добавить text-канал и не может ничего там написать. Это блокирует MVP-цикл «свои пользователи → обратная связь».

Фаза 3 решает три задачи одновременно: (1) сохраняет сообщения в БД, чтобы при переподключении и обновлении страницы переписка восстанавливалась; (2) доставляет сообщения подписанным клиентам в реальном времени по WebSocket; (3) транслирует событие о новом участнике комнаты (`member.joined`), готовя протокольную базу для будущих realtime-фич (voice signaling, presence, typing).

Технически фаза вводит первый persistent WebSocket-канал в проекте и инвертирует архитектурную привычку: до этого все взаимодействия были HTTP request/response, теперь появляется серверный push с собственной аутентификацией, подписками и broadcast'ом. Все архитектурные решения этой фазы создают шаблон для последующих realtime-доменов (voice, presence).

## Критерии приёмки

1. Зарегистрированный пользователь, состоящий в комнате, может отправить текстовое сообщение в text-канал и увидеть его моментально у себя и у других подписанных клиентов.
2. Сообщение сохраняется в Postgres и доступно через REST `GET /api/v1/channels/{id}/messages?before=<message_id>&limit=<n>` с курсорной пагинацией.
3. Сообщение не принимается, если пользователь не является членом комнаты, либо канал не существует, либо канал — voice (запись только в text-каналы).
4. Длина текста сообщения — `1..4000` символов после `TrimSpace`; пустой / только пробельный / превышающий лимит текст отклоняется доменной ошибкой.
5. WebSocket-эндпоинт `/api/v1/ws?token=<jwt>` принимает только клиентов с валидным access-токеном. Истёкший / некорректный токен → HTTP 401 до апгрейда.
6. Клиент явно подписывается на канал событием `{"type":"subscribe","channel_id":"..."}`. Подписка отклоняется, если клиент не член комнаты канала.
7. Все WS-клиенты, состоящие в комнате, при `JoinByCode` нового члена получают событие `{"type":"member.joined","data":{...}}` без необходимости отдельной подписки.
8. Хаб корректно завершает соединения: при `context.Done()` сервера все коннекты закрываются с close-кодом `1001 (going away)`; при разрыве соединения клиента подписки очищаются.
9. Тесты:
   - Unit-тесты `chat/domain` (Message, MessageText, MessageID) — все инварианты.
   - Unit-тесты `chat/usecase` (`SendMessage`, `ListMessages`) — happy path + все доменные ошибки.
   - Integration-тесты `chat/repository/postgres` против реальной БД (с `TEST_DATABASE_URL`).
   - HTTP-тесты `chat/transport/http` через `httptest.NewServer`.
   - Тесты hub'а (`pkg/websocket`) — регистрация, подписка, broadcast, отключение.
10. Архитектурные тесты в `arch_test.go` запрещают `internal/chat/domain` импортировать что-либо кроме stdlib + uuid; `internal/chat/usecase` — только stdlib + uuid + `chat/domain`; `pkg/websocket` — никаких `internal/*`.

## Документы

| Файл | Разрез | Описание |
|---|---|---|
| [01-architecture.md](./01-architecture.md) | Logical | C4 диаграммы L1 → L2 → L3, зависимости модулей |
| [02-behavior.md](./02-behavior.md) | Process | DFD + sequence diagrams (по 1 на use case) |
| [03-decisions.md](./03-decisions.md) | Decision | Решения, альтернативы, риски, open questions |
| [04-testing.md](./04-testing.md) | Quality | Тестовая стратегия, coverage mapping, состав тест-кейсов |
| [05-events.md](./05-events.md) | Domain events | `message.new`, `member.joined`, контракт publish |
| [06-repo-model.md](./06-repo-model.md) | Repo | Маппинг `Message` ↔ Postgres-row, sqlc-запросы, миграция |
| [07-standards.md](./07-standards.md) | Standards | Compliance с `prompts/*` |
| [08-api-contract.md](./08-api-contract.md) | API | REST + WebSocket контракт |
| [research.md](./research.md) | Baseline | Сводка ресерча — фактическое состояние кодовой базы |
| [plan/](./plan/) | Code plan | Создаётся после утверждения дизайна |
