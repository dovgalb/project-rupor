---
phase: 10
name: Docker integration
layer: bootstrap
depends_on: [phase-09]
plan: ./README.md
---

# Phase 10: Docker integration

## Цель

Упаковать фронт в production-ready Docker-образ (nginx:alpine + статика), добавить сервис `web` в `docker-compose.yml`. После фазы — `docker compose up -d` поднимает `postgres + web` (api остаётся через `make run` — Hybrid вариант, см. `plan/README.md` секция «Открытое решение по api в compose»). Nginx раздаёт SPA и проксирует `/api/v1` (включая WS-upgrade) на `host.docker.internal:8080`.

## Контекст

Phase 9 завершила фронт-функциональность. Текущий `docker-compose.yml` содержит только Postgres. Контракт прокси — `../08-routes.md` секция «Прокси и nginx», `../06-api-integration.md` секция «Прокси Vite (dev)».

**Решение по api-сервису (см. `plan/README.md`):** Hybrid — `web` контейнер проксирует на `host.docker.internal:8080`, api остаётся локально через `make run`. Dockerize api — отдельная задача после PR-3.5.

## Файлы для создания

### `web/Dockerfile`

Multi-stage build:

```dockerfile
# === Stage 1: builder ===
FROM node:lts-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci

COPY . .
RUN npm run build

# === Stage 2: runtime ===
FROM nginx:alpine AS runtime

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
```

### `web/nginx.conf`

```nginx
server {
  listen 80;
  server_name _;

  root /usr/share/nginx/html;
  index index.html;

  # SPA fallback — все unknown пути → index.html
  location / {
    try_files $uri $uri/ /index.html;
  }

  # Прокси REST + WebSocket upgrade на api
  location /api/v1/ {
    proxy_pass http://host.docker.internal:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    # WS timeout
    proxy_read_timeout 600s;
    proxy_send_timeout 600s;
  }

  # Security headers
  add_header X-Frame-Options "DENY" always;
  add_header X-Content-Type-Options "nosniff" always;
  add_header Referrer-Policy "strict-origin-when-cross-origin" always;

  # Gzip
  gzip on;
  gzip_types text/plain text/css application/javascript application/json image/svg+xml;
  gzip_min_length 1024;
}
```

**Note:** `host.docker.internal` работает на Docker Desktop (macOS/Windows). На Linux в `docker-compose.yml` нужно `extra_hosts: ["host.docker.internal:host-gateway"]`.

### `web/.dockerignore`

```
node_modules
dist
coverage
playwright-report
test-results
.env*
*.log
.git
.idea
.vscode
e2e
**/*.test.*
**/*.spec.*
```

## Файлы для модификации

### `docker-compose.yml`

Добавляем сервис `web`:

```yaml
services:
  postgres:
    # ... существующий блок без изменений
  
  web:
    build:
      context: ./web
      dockerfile: Dockerfile
    container_name: rupor-web
    ports:
      - "${WEB_PORT:-3000}:80"
    extra_hosts:
      - "host.docker.internal:host-gateway"  # для Linux
    depends_on:
      - postgres
    restart: unless-stopped
```

`WEB_PORT` по умолчанию 3000 (8080 уже занят api).

`depends_on: postgres` — формально web не зависит от postgres напрямую, но указываем для порядка start. Реальная зависимость — на api, который запускается через `make run` (Hybrid).

### `Makefile`

Добавить targets:

```makefile
web-install:
	npm --prefix web install

web-dev:
	npm --prefix web run dev

web-build:
	npm --prefix web run build

web-test:
	npm --prefix web run test -- --run

web-lint:
	npm --prefix web run lint

web-typecheck:
	npm --prefix web run typecheck

web-e2e:
	npm --prefix web run e2e
```

И добавить эти таргеты в `.PHONY` объявление в начале файла.

Также рассмотреть: `dc-up` теперь поднимает и `postgres`, и `web`. Если пользователь хочет только postgres — оставляем `dc-up` как есть (compose up -d поднимает всё определённое в файле). Альтернатива — `dc-up-db` для отдельного варианта; решается в реализации.

### `.env.example`

Добавить (если ещё нет):
```
WEB_PORT=3000
```

### `web/package.json` (если ещё не было)

Убедиться, что `scripts.build` чистый: `vite build` (без артефактов dev).

## Ключевые решения

- **D-16** Vite dev proxy — остаётся для разработки.
- **D-17** Production: multi-stage Dockerfile (node:lts-alpine builder → nginx:alpine runtime). Реализовано.
- **Hybrid api**: web проксирует на host. Принято в `plan/README.md`. Dockerize api — отдельная фаза после MVP.

## Verification

- [ ] `docker compose build web` — собирается без ошибок.
- [ ] `docker compose up -d` поднимает postgres + web.
- [ ] `docker compose logs web` — nginx стартует чисто.
- [ ] Открыть `http://localhost:3000/` (`WEB_PORT`) → SPA загружается, видим `/login`.
- [ ] **Запустить бэк локально** (`make run`) → залогиниться через `http://localhost:3000/` → REST запросы работают (через прокси `host.docker.internal:8080`).
- [ ] WS-апгрейд работает — отправка сообщения проходит, message.new приходит.
- [ ] SPA fallback: открыть `http://localhost:3000/rooms/xxx/channels/yyy` напрямую → nginx отдаёт `index.html`, фронт сам роутит.
- [ ] Security headers присутствуют в ответе (`X-Frame-Options: DENY`, etc.) — проверить `curl -I`.
- [ ] Gzip работает — `curl -H "Accept-Encoding: gzip" -I http://localhost:3000/...js` показывает `Content-Encoding: gzip`.
- [ ] Bundle size в `dist/` — приемлемый (~500KB gz для JS, проверить).
- [ ] `docker compose down` корректно останавливает.
- [ ] `Makefile` targets работают: `make web-build`, `make web-test`, etc.

## Что НЕ делается в этой фазе

- Dockerize api-сервера — отдельная фаза вне PR-3.5.
- HTTPS / TLS — для prod-deploy с настоящим доменом, не для локального docker-compose.
- CDN / static asset caching — после MVP.
- CI/CD pipeline — отдельная фаза.
