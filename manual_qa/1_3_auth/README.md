# Manual QA: 1_3 auth

HTTP-запросы для ручного тестирования эндпоинтов `/api/v1/auth/*`.

Формат `.http` — совместим с VS Code REST Client и JetBrains HTTP Client.

## Подготовка стенда

```bash
make dc-up && make migrate-up && make run
```

## Файлы

- `00_flow.http` — happy-path сценарий end-to-end (register → login → me → refresh → me с новым access). Запускать по порядку, токены подставляются автоматически.
- `01_register.http` — POST /register: успех, дубликаты, валидация, malformed body, неверный метод.
- `02_login.http` — POST /login: успех, плохие credentials, malformed body.
- `03_refresh.http` — POST /refresh: успех, повторный со старым (revoked), invalid base64, malformed.
- `04_me.http` — GET /me: с валидным Bearer, без заголовка, неверная схема, битая подпись, истёкший токен.
- `99_health.http` — GET /health.

## Переменные

В каждом файле наверху объявлены `@host` и тестовые `email/username/password`. Меняйте по необходимости.

В `00_flow.http` access/refresh-токены захватываются через `# @name` (VS Code REST Client) и подставляются в следующие запросы. В JetBrains HTTP Client синтаксис захвата другой — для JetBrains пользоваться `01..04_*.http` и копировать токены вручную из ответа.
