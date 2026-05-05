---
parent: ./README.md
---

# 08 — HTTP API Contract

В фазе 1.1 единственный публичный эндпоинт — `GET /api/v1/health`. Контракт фиксируется здесь, чтобы будущие фазы (`auth`, `room` и т.д.) могли расширять префикс `/api/v1/` без конфликтов и чтобы клиенты (CI smoke, мониторинг) могли стабильно его опрашивать.

## GET /api/v1/health

**Назначение:** проверка живости HTTP-сервера. НЕ проверяет связность с PostgreSQL (см. ADR-006 в `03-decisions.md`).

### Request

```
Method: GET
Path:   /api/v1/health
Headers:
  (нет обязательных)
Query params:
  (нет)
Body:
  (нет)
```

Авторизация **не требуется** — health-check публичный по соглашению. Auth-middleware на этот путь не применяется.

### Response 200 OK

**Headers:**
```
Content-Type: application/json; charset=utf-8
```

**Body:**
```json
{
  "status": "ok"
}
```

Точный формат:
- `status` — строка, всегда `"ok"`. Перечень допустимых значений в этой фазе ограничен одним вариантом. В будущих фазах может появиться `"degraded"` (например, БД недоступна) — это будет breaking change и потребует обновления контракта.

### Error responses

| Status | Error Code | Body | Когда |
|--------|-----------|------|-------|
| 405 | — | `{"error":"method not allowed"}` | Любой метод кроме GET (POST/PUT/DELETE на `/api/v1/health`). Обрабатывается chi-дефолтом. |
| 404 | — | `404 page not found\n` (text/plain, дефолт chi) | Любой путь, не зарегистрированный в роутере. |

В фазе 1.1 health-handler не возвращает 5xx, потому что не делает зависимостей. Если в будущем добавим pre-flight check (Phase 1.2+), 503 появится с отдельным error code.

### Поведение

- Хендлер не блокируется ни на чём — синхронный `w.WriteHeader(200)` + `w.Write(body)`.
- Логирование: на каждом запросе пишется access-log через middleware `chi/middleware.Logger` или своя реализация на `slog`. Решение — взять `slog`-обёртку, чтобы не тащить дополнительную зависимость (`chi/middleware` входит в `chi/v5` и не требует отдельного импорта, но логи у него — стандартный `log`, не `slog`). Open question — в Q9 ниже.
- Метрики: в этой фазе не вводим. Появятся в отдельной фиче «observability».

### Test Plan

| Шаг | Команда | Ожидание |
|-----|---------|----------|
| Сервер запущен | `make run` (background) | `slog` пишет `INFO server started addr=:8080` |
| Базовый GET | `curl -i http://localhost:8080/api/v1/health` | `HTTP/1.1 200 OK`, `Content-Type: application/json...`, тело `{"status":"ok"}` |
| Метод не разрешён | `curl -i -X POST http://localhost:8080/api/v1/health` | `HTTP/1.1 405 Method Not Allowed` |
| Несуществующий путь | `curl -i http://localhost:8080/api/v1/nope` | `HTTP/1.1 404 Not Found` |
| Health под нагрузкой | `for i in $(seq 1 100); do curl -s http://localhost:8080/api/v1/health > /dev/null; done` | Все 100 запросов 200, время каждого <50мс |
| Graceful shutdown | `make run` → `kill -TERM $!` → ожидание | Сервер завершается за ≤5с, активные запросы дотягивают ответ |

Эти шаги входят в `manual_qa/project-structure/test-flow.md`, который создаётся после реализации.

## Резерв префикса `/api/v1/`

Фиксируется как соглашение для будущих фаз (`general_plan.md:81-109`):
- `/api/v1/auth/*` — Фаза 1.2 (auth)
- `/api/v1/rooms/*`, `/api/v1/rooms/:id/channels/*` — Фаза 2 (rooms/channels)
- `/api/v1/channels/:id/messages` — Фаза 3 (chat REST)
- `/api/v1/ws` — Фаза 3 (WebSocket)
- `/api/v1/health` — Фаза 1.1 (эта фича)

В фазе 1.1 регистрируется один маршрут. Группировка через `r.Route("/api/v1", ...)` будет добавлена сразу — это даёт нулевой overhead и предотвращает рассинхрон префиксов в разных доменах.

## Open Questions (для API)

- [ ] **Q9.** Использовать `chi/middleware.Logger` (стандартный `log`) или свою `slog`-обёртку для access-log? Текущий выбор — `slog`-обёртку, чтобы все логи приложения были в одном формате.
- [ ] **Q10.** Возвращать ли в `/api/v1/health` версию приложения (`{"status":"ok","version":"abc123"}`) с git-commit-sha через `-ldflags`? Текущий выбор — нет; добавим, когда появится production-деплой и понадобится подтверждение версии.
