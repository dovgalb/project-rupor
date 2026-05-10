# Manual QA: 1_4 auth middleware

HTTP-запросы для ручного тестирования middleware-stack:
RequestID, Recover, Logger, CORS, RequireAuth.

Формат `.http` — совместим с VS Code REST Client и JetBrains HTTP Client.

## Подготовка стенда

```bash
make dc-up && make migrate-up && make run
```

В `.env` (или ENV) должны быть выставлены:
- `JWT_SECRET=...`
- `DATABASE_URL=...`
- `CORS_ALLOWED_ORIGINS=http://localhost:5173` (опционально, дефолт)

## Файлы

- `00_smoke.http` — общая прогонка через middleware-stack: health-endpoint, проверка `X-Request-ID` в каждом ответе.
- `01_cors_preflight.http` — preflight + cross-origin запросы; whitelisted и denied origins.
- `02_request_id.http` — клиентский `X-Request-ID`: приём валидного, отвержение CRLF/non-ASCII/слишком длинного.
- `03_require_auth.http` — все 401-сценарии RequireAuth: без токена, не-Bearer, пустой токен, невалидная подпись, истёкший токен, lowercase `bearer`.

## Переменные

В каждом файле наверху объявлены `@host` и whitelisted `@origin`. Меняйте по необходимости.

В `03_require_auth.http` для happy-path-проверки нужен валидный access-токен — получите его через `manual_qa/1_3_auth/00_flow.http` (register + login) и подставьте в `@access_token`.
