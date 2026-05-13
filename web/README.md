# Rupor Web Client

MVP веб-клиент Rupor (React + TypeScript + Vite). Подробный дизайн — `../docs/3_5_frontend/`, план — `../docs/3_5_frontend/plan/`.

## Требования

- Node.js LTS (>=20)
- npm 10+
- Запущенный бэк на `http://localhost:8080` (см. корневой `Makefile`: `make run`).

## Установка

```bash
npm install
```

## Команды

| Команда                    | Что делает                                         |
|----------------------------|----------------------------------------------------|
| `npm run dev`              | Запускает Vite dev-сервер на `http://localhost:5173`. Прокси `/api/v1` → `localhost:8080`. |
| `npm run typecheck`        | Проверка типов (`tsc --noEmit`).                   |
| `npm run lint`             | ESLint (`--max-warnings=0`).                       |
| `npm run lint:fix`         | ESLint + автофиксы.                                |
| `npm run test`             | Unit/component-тесты (vitest).                     |
| `npm run test:watch`       | Vitest в watch-режиме.                             |
| `npm run build`            | Production-сборка в `dist/`.                       |
| `npm run preview`          | Локальный preview production-сборки.               |
| `npm run e2e`              | Playwright E2E (после Phase 11).                   |

## Структура

См. `prompts/Frontend Architecture Layers.txt` (feature-based + shared).

```
src/
├── app/         — корень, провайдеры, глобальные стили
├── pages/       — page-компоненты
├── features/    — фичевые слайсы (auth, rooms, channels, chat)
├── shared/      — api/, ui/, lib/
└── routes.tsx   — карта роутов
```
