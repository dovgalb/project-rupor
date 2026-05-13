---
phase: 1
name: Bootstrap & Scaffolding
layer: bootstrap
depends_on: none
plan: ./README.md
---

# Phase 1: Bootstrap & Scaffolding

## Цель

Создать каркас фронт-проекта `web/`: Vite + React 18 + TypeScript (strict) + ESLint + Prettier + структура папок (feature-based) + базовые конфиги. После фазы можно запускать `npm run dev` и видеть пустую страницу-плейсхолдер.

## Контекст

Папки `web/` НЕТ — создаём с нуля. Базируемся на `prompts/Frontend Architecture Layers.txt`, `prompts/TypeScript Style.txt`, ADR D-01 (router=react-router-dom v6), D-13 (feature-based), D-21..D-25 (i18n, тема, toaster, palette).

## Файлы для создания

### `web/package.json`

**Назначение:** манифест Node-проекта.

**Детали реализации:**
- `name: "rupor-web"`, `private: true`, `type: "module"`.
- Scripts: `dev`, `build`, `preview`, `typecheck`, `lint`, `lint:fix`, `test`, `test:watch`, `e2e`.
- Dependencies (фиксированные мажоры — точные версии актуальные на момент реализации):
  - `react@^18`, `react-dom@^18`
  - `react-router-dom@^6`
  - `zustand@^4` (или ^5 если совместимо — проверить changelog persist API)
  - `react-hook-form@^7`, `@hookform/resolvers@^3`, `zod@^3`
- DevDependencies:
  - `vite@^5`, `@vitejs/plugin-react@^4`
  - `typescript@^5`
  - `@types/react@^18`, `@types/react-dom@^18`, `@types/node`
  - `eslint@^8`, `@typescript-eslint/parser`, `@typescript-eslint/eslint-plugin`, `eslint-plugin-react`, `eslint-plugin-react-hooks`, `eslint-plugin-jsx-a11y`
  - `prettier@^3`
  - `vitest@^1`, `@vitest/ui` (optional)
  - `@testing-library/react@^14`, `@testing-library/jest-dom`, `@testing-library/user-event`
  - `jsdom`
  - `@playwright/test@^1` (для e2e в Phase 11)

### `web/vite.config.ts`

**Назначение:** конфиг Vite.

**Детали реализации:**
- `@vitejs/plugin-react`.
- Alias `@` → `./src`.
- `server.port: 5173`, `server.host: true`.
- Proxy `/api/v1` → `http://localhost:8080`, `changeOrigin: true`, `ws: true`. См. `../06-api-integration.md`.
- `build.sourcemap: true` для dev, `false` для prod.
- `css.modules.localsConvention: "camelCaseOnly"`.

### `web/tsconfig.json` + `web/tsconfig.node.json`

**Назначение:** TS-конфиг.

**Детали реализации (`tsconfig.json`):**
- `compilerOptions`:
  - `target: "ES2022"`, `module: "ESNext"`, `moduleResolution: "bundler"`.
  - `strict: true`.
  - `noUncheckedIndexedAccess: true`, `noImplicitOverride: true`, `exactOptionalPropertyTypes: true`, `noFallthroughCasesInSwitch: true` (см. `prompts/TypeScript Style.txt`).
  - `jsx: "react-jsx"`.
  - `paths: { "@/*": ["./src/*"] }`.
  - `types: ["vite/client", "vitest/globals"]`.
- `include: ["src", "vitest.config.ts"]`.

`tsconfig.node.json` для Vite-конфига сам по себе.

### `web/.eslintrc.cjs`

**Детали:**
- `parser: "@typescript-eslint/parser"`.
- Plugins: `@typescript-eslint`, `react`, `react-hooks`, `jsx-a11y`.
- Extends: `eslint:recommended`, `plugin:@typescript-eslint/recommended`, `plugin:react/recommended`, `plugin:react/jsx-runtime`, `plugin:react-hooks/recommended`, `plugin:jsx-a11y/recommended`.
- Rules:
  - `@typescript-eslint/no-explicit-any: "error"`.
  - `no-console: ["warn", { allow: ["warn", "error"] }]`.
  - `react/no-danger: "error"`.
  - `import/no-default-export: "error"` (через `eslint-plugin-import` или встроенно — выбрать).
- `settings: { react: { version: "detect" } }`.

### `web/.prettierrc`

```json
{
  "semi": true,
  "singleQuote": false,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2
}
```

### `web/.gitignore`

```
node_modules
dist
coverage
playwright-report
test-results
.env.local
*.log
```

### `web/index.html`

**Назначение:** HTML-шаблон Vite.

**Детали:**
- `<title>Rupor</title>`.
- `<meta charset="UTF-8" />`, `<meta name="viewport" ...>`.
- `<div id="root"></div>`.
- `<script type="module" src="/src/main.tsx"></script>`.
- `lang="ru"`.

### `web/vitest.config.ts`

**Детали:**
- `defineConfig({ test: { environment: "jsdom", globals: true, setupFiles: ["./src/test-setup.ts"] }})`.
- Использует тот же alias `@`.
- `coverage` (опционально) через `@vitest/coverage-v8`.

### `web/src/test-setup.ts`

**Назначение:** глобальный setup для vitest.

**Детали:**
- `import "@testing-library/jest-dom/vitest"`.
- (после Phase 9) — здесь же `vi.mock` для wsClient singleton, если нужен глобально.

### `web/src/main.tsx`

**Назначение:** entry point — монтирует `<App />`.

```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./app/App";
import "./app/styles/reset.css";
import "./app/styles/theme.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
```

### `web/src/app/App.tsx`

**Назначение:** корень приложения. В этой фазе — заглушка с одним `<div>`.

```tsx
export function App() {
  return <div>Rupor — coming soon</div>;
}
```

В Phase 5 заполнится: `<ErrorBoundary><RouterProvider router={router} /><Toaster /></ErrorBoundary>`.

### `web/src/app/styles/theme.css`

**Назначение:** глобальные CSS-переменные тёмной темы (D-22).

**Содержимое:** см. `../07-ui-contract.md` секция «Тема и CSS». Палитра Discord-like: `--color-bg-primary`, `--color-bg-secondary`, `--color-bg-tertiary`, `--color-text-primary`, `--color-text-muted`, `--color-accent`, `--color-danger`, `--color-success`, `--color-warning`, `--radius-sm/md`, `--font-base`.

### `web/src/app/styles/reset.css`

**Назначение:** нормализация по умолчанию.

**Содержимое:** базовый reset (можно взять `modern-normalize` подходом, без отдельной зависимости — встроить inline):
- `*, *::before, *::after { box-sizing: border-box }`
- `html, body, #root { height: 100% }`
- `body { margin: 0; font-family: var(--font-base); background: var(--color-bg-primary); color: var(--color-text-primary) }`
- `button { font: inherit; cursor: pointer }`
- `input, textarea, select { font: inherit }`
- `a { color: var(--color-accent); text-decoration: none }`
- `img { display: block; max-width: 100% }`

### `web/src/routes.tsx`

**Назначение:** заглушка для роутера (заполнится в Phase 5).

```ts
import { createBrowserRouter } from "react-router-dom";
export const router = createBrowserRouter([
  { path: "*", element: <div>Rupor — coming soon</div> },
]);
```

### `web/src/shared/lib/i18n/ru.ts`

**Назначение:** словарь строк UI (D-23). На MVP — все строки литералы здесь, обращение `import { ru } from "@/shared/lib/i18n/ru"`.

**Детали:**
- Структура: вложенный объект по фичам:
  ```ts
  export const ru = {
    auth: {
      loginTitle: "Войти в Rupor",
      registerTitle: "Создать аккаунт",
      email: "Email",
      username: "Имя пользователя",
      password: "Пароль",
      loginBtn: "Войти",
      registerBtn: "Создать аккаунт",
      logoutBtn: "Выйти",
      invalidCredentials: "Неверный email или пароль",
      // ...
    },
    rooms: {
      myRooms: "Мои комнаты",
      createRoom: "Создать комнату",
      joinByCode: "Войти по коду",
      // ...
    },
    channels: { /* ... */ },
    chat: { /* ... */ },
    common: {
      cancel: "Отмена",
      save: "Сохранить",
      retry: "Повторить",
      // ...
    },
  } as const;
  export type Ru = typeof ru;
  ```
- Сейчас можно заполнить минимально, пополнять по мере необходимости в каждой фазе.

### `web/src/shared/lib/logger.ts`

**Назначение:** единый logger (см. `../06-api-integration.md`).

```ts
const isDev = import.meta.env.DEV;

export const logger = {
  debug: (...args: unknown[]) => { if (isDev) console.log("[debug]", ...args); },
  info:  (...args: unknown[]) => { if (isDev) console.log("[info]", ...args); },
  warn:  (...args: unknown[]) => console.warn(...args),
  error: (...args: unknown[]) => console.error(...args),
};
```

ESLint-правило `no-console` разрешает `warn` и `error` — этот файл единственное исключение для `console.log` (под `if (isDev)`). Можно добавить `// eslint-disable-next-line no-console` локально.

### `web/README.md`

**Назначение:** инструкция по запуску.

**Содержимое:**
- Prerequisites: Node LTS, npm/pnpm.
- `npm install`.
- `npm run dev` — открывает `http://localhost:5173`. Требует поднятый бэк на `:8080`.
- `npm run typecheck`, `npm run lint`, `npm run test`, `npm run build`.
- Структура папок (краткий обзор + ссылка на `prompts/Frontend Architecture Layers.txt`).

### Пустые директории (с `.gitkeep`)

- `web/src/shared/api/.gitkeep` (заполнится в Phase 2)
- `web/src/shared/ui/.gitkeep` (Phase 3)
- `web/src/features/auth/.gitkeep` (Phase 4)
- `web/src/features/rooms/.gitkeep` (Phase 6)
- `web/src/features/channels/.gitkeep` (Phase 7)
- `web/src/features/chat/.gitkeep` (Phase 8)
- `web/src/pages/.gitkeep` (Phase 5)

## Файлы для модификации

В этой фазе только создание новых файлов. Никаких корневых файлов проекта не трогаем (ни `docker-compose.yml`, ни `Makefile` — это Phase 10).

## Ключевые решения

- **D-01** react-router-dom v6 — установлен в `package.json` (используется в Phase 5).
- **D-02** react-hook-form + zod — установлены (используются с Phase 4).
- **D-03** CSS Modules — встроены в Vite, отдельная конфигурация не нужна.
- **D-13** Feature-based — структура папок создана сразу.
- **D-21..D-25** — i18n-файл, theme.css, базовая инфраструктура для toaster/avatar/palette в shared (но сами компоненты — Phase 3).

## Verification

- [ ] `npm install` отрабатывает чисто.
- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок (на пустом коде).
- [ ] `npm --prefix web run build` — собирается, dist/ создаётся.
- [ ] `npm --prefix web run dev` поднимается, на `http://localhost:5173` видна заглушка «Rupor — coming soon».
- [ ] `npm --prefix web run test -- --run` — 0 тестов (но команда отрабатывает без ошибок).
- [ ] Структура папок соответствует `prompts/Frontend Architecture Layers.txt`.
- [ ] Никаких `any` в коде, strict-флаги включены.
- [ ] ESLint-правила `no-explicit-any`, `react/no-danger`, `no-console` (с allow warn/error) — настроены.
- [ ] CSS-переменные тёмной темы определены в `theme.css`.
