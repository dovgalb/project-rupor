---
date: 2026-05-13
feature: 3_5-frontend
design: ../README.md
status: draft
---

# План кода: PR-3.5 Веб-клиент Rupor

## Overview

Реализация MVP веб-клиента согласно дизайну в `docs/3_5_frontend/`. Папка `web/` создаётся с нуля, поэтому план собирается **Foundation-first**: сначала фундамент (Vite, TS, shared/api, shared/ui), потом фичи (auth, rooms, channels, chat), потом WS-интеграция, в конце — docker и e2e.

Все принятые решения зафиксированы в `../03-decisions.md` (25 ADR). Все TS-типы DTO/WS — `../06-api-integration.md`. Компоненты — `../07-ui-contract.md`. Тесты — `../04-testing.md`.

## Phase Strategy: Foundation-first

| Аргумент | Объяснение |
|---|---|
| `web/` пуст | Каркас и shared-инфраструктура нужны раньше всех фич |
| Все фичи зависят от `shared/api/fetch.ts` (auto-refresh) | Не имеет смысла делать auth раньше fetch-обёртки |
| WS-клиент нужен только после chat REST-слоя | WS-фаза 9 после фаз 8 |
| Routes/Guards собираются после auth | Иначе нечего guard'ить |
| Docker — финальная упаковка | Должен работать собранный фронт |
| E2E — после всего | Полный happy-path |

**Не используем Vertical Slice** (auth-фича end-to-end → rooms-фича end-to-end) потому что shared/api и shared/ui нужны обеим, и каждый «вертикальный срез» спотыкался бы на shared.

## Phases

| # | Фаза | Слой | Зависимости | Status |
|---|------|------|-------------|--------|
| 1 | Bootstrap & Scaffolding | bootstrap | — | ☐ |
| 2 | Shared API foundation | shared | 1 | ☐ |
| 3 | Shared UI primitives + utils | shared | 1 | ☐ |
| 4 | Auth feature | feature | 2, 3 | ☐ |
| 5 | Routes + AppShell + Guards | route + page | 4 | ☐ |
| 6 | Rooms feature | feature | 5 | ☐ |
| 7 | Channels feature | feature | 6 | ☐ |
| 8 | Chat REST | feature | 7 | ☐ |
| 9 | Shared WS Client + integration | shared + feature | 8 | ☐ |
| 10 | Docker integration | bootstrap | 9 | ☐ |
| 11 | E2E Playwright | test | 10 | ☐ |

Phase 2 и 3 — потенциально могут идти параллельно (оба зависят только от 1).

## File Map

### New Files (по фазам)

**Phase 1 — Bootstrap:**
- `web/package.json`
- `web/vite.config.ts`
- `web/tsconfig.json` + `web/tsconfig.node.json`
- `web/.eslintrc.cjs`
- `web/.prettierrc`
- `web/.gitignore`
- `web/index.html`
- `web/vitest.config.ts`
- `web/src/main.tsx` — entry point
- `web/src/app/App.tsx` — корень с ErrorBoundary, RouterProvider, Toaster (минимально, заполняется в Phase 5)
- `web/src/app/styles/theme.css` — CSS-переменные (D-22)
- `web/src/app/styles/reset.css` — нормализация
- `web/src/routes.tsx` — пустой роутер с `/` → плейсхолдер (заполняется в Phase 5)
- `web/src/shared/lib/i18n/ru.ts` — словарь строк (D-23)
- `web/src/shared/lib/logger.ts` — единый logger
- `web/README.md` — кратко: как запустить, dev/build/test scripts

**Phase 2 — Shared API:**
- `web/src/shared/api/errors.ts` — `ApiResult`, `ErrorEnvelope`, IncomingFrame/OutgoingFrame TS-типы (snake_case payload)
- `web/src/shared/api/token-storage.ts` — localStorage обёртка
- `web/src/shared/api/refresh.ts` — single-flight `refreshAccessOnce()`
- `web/src/shared/api/fetch.ts` — `apiFetch<T>` с auto-refresh
- `web/src/shared/lib/logoutFlow.ts` — каскадная очистка (вызывается из fetch.ts при AUTH-007/008/009/010)
- `web/src/shared/api/*.test.ts` — тесты (см. Phase 2)

**Phase 3 — Shared UI/utils:**
- `web/src/shared/ui/Button.tsx` + `.module.css` + `.test.tsx`
- `web/src/shared/ui/Input.tsx` + `.module.css` + `.test.tsx`
- `web/src/shared/ui/Textarea.tsx` + `.module.css` + `.test.tsx`
- `web/src/shared/ui/FormField.tsx` + `.test.tsx`
- `web/src/shared/ui/Modal.tsx` + `.module.css` + `.test.tsx` (focus-trap)
- `web/src/shared/ui/Spinner.tsx` + `.module.css`
- `web/src/shared/ui/Toast.tsx` + `Toaster.tsx` + `useToast.ts` + `.module.css` + `.test.tsx`
- `web/src/shared/ui/Avatar.tsx` + `.module.css` + `.test.tsx`
- `web/src/shared/ui/EmptyState.tsx` + `.module.css`
- `web/src/shared/ui/ErrorBoundary.tsx` + `.test.tsx`
- `web/src/shared/ui/FullScreenSpinner.tsx`
- `web/src/shared/ui/index.ts` — barrel re-exports
- `web/src/shared/lib/uuidToColor.ts` (+ test) — палитра D-25
- `web/src/shared/lib/linkify.ts` (+ test) — валидация протоколов
- `web/src/shared/lib/formatDate.ts` (+ test)

**Phase 4 — Auth:**
- `web/src/features/auth/types.ts`
- `web/src/features/auth/api/http.ts` (+ test)
- `web/src/features/auth/store/index.ts` (+ test) — persist через middleware, partialize `currentUser`
- `web/src/features/auth/store/mapErrors.ts` — `AuthErrorCode → message`
- `web/src/features/auth/components/LoginForm.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/auth/components/LoginForm.schema.ts`
- `web/src/features/auth/components/RegisterForm.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/auth/components/RegisterForm.schema.ts`
- `web/src/features/auth/components/UserBadge.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/auth/index.ts`
- `web/src/pages/LoginPage.tsx` + `.module.css`
- `web/src/pages/RegisterPage.tsx` + `.module.css`

**Phase 5 — Routes + AppShell:**
- `web/src/app/routes/RequireAuth.tsx` + `.test.tsx`
- `web/src/app/routes/RedirectIfAuthenticated.tsx` + `.test.tsx`
- `web/src/app/routes/RequireMembership.tsx` + `.test.tsx`
- `web/src/app/routes/RequireChannelInRoom.tsx` + `.test.tsx`
- `web/src/app/AppShell.tsx` + `.module.css` + `.test.tsx`
- `web/src/app/TopBar.tsx` + `.module.css` + `.test.tsx`
- `web/src/app/Sidebar.tsx` + `.module.css`
- `web/src/app/RoomScopedSidebar.tsx` (рендерит ChannelList + MembersList — заполнится в фазах 6, 7)
- `web/src/pages/RoomsIndexPage.tsx` — empty state с CTA (или redirect на первую комнату — в фазе 6)
- `web/src/pages/RoomPage.tsx` — «Выбери канал»
- `web/src/pages/ChatRoutePage.tsx` — обёртка над `<ChatPanel />` (заполнится в фазе 8)
- `web/src/pages/NotFoundPage.tsx`
- Modified: `web/src/routes.tsx` — полная карта роутов
- Modified: `web/src/app/App.tsx` — RouterProvider + Toaster + ErrorBoundary

**Phase 6 — Rooms:**
- `web/src/features/rooms/types.ts`
- `web/src/features/rooms/api/http.ts` (+ test)
- `web/src/features/rooms/store/index.ts` (+ test) — REST-only, без WS handlers (member.joined — в Phase 9)
- `web/src/features/rooms/store/mapErrors.ts`
- `web/src/features/rooms/components/RoomList.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/rooms/components/RoomListItem.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/rooms/components/CreateRoomModal.tsx` + `.schema.ts` + `.test.tsx`
- `web/src/features/rooms/components/DeleteRoomConfirm.tsx` + `.test.tsx`
- `web/src/features/rooms/components/InviteCodeModal.tsx` + `.test.tsx`
- `web/src/features/rooms/components/JoinByCodeModal.tsx` + `.schema.ts` + `.test.tsx`
- `web/src/features/rooms/components/MembersList.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/rooms/components/MemberListItem.tsx` + `.test.tsx`
- `web/src/features/rooms/index.ts`
- Modified: `web/src/pages/RoomsIndexPage.tsx` — встраиваем CreateRoom/JoinByCode CTA
- Modified: `web/src/app/RoomScopedSidebar.tsx` — встраиваем MembersList

**Phase 7 — Channels:**
- `web/src/features/channels/types.ts`
- `web/src/features/channels/api/http.ts` (+ test)
- `web/src/features/channels/store/index.ts` (+ test)
- `web/src/features/channels/store/mapErrors.ts`
- `web/src/features/channels/components/ChannelList.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/channels/components/ChannelListItem.tsx` + `.test.tsx`
- `web/src/features/channels/components/CreateChannelModal.tsx` + `.schema.ts` + `.test.tsx`
- `web/src/features/channels/components/DeleteChannelConfirm.tsx` + `.test.tsx`
- `web/src/features/channels/index.ts`
- Modified: `web/src/app/RoomScopedSidebar.tsx` — добавить ChannelList

**Phase 8 — Chat REST:**
- `web/src/features/chat/types.ts`
- `web/src/features/chat/api/http.ts` (+ test) — `listMessages`
- `web/src/features/chat/store/index.ts` (+ test, без WS) — `messagesByChannel`, `loadHistory`, `loadMoreHistory`, `wsStatus` (initial "idle")
- `web/src/features/chat/store/mapErrors.ts`
- `web/src/features/chat/components/ChatPanel.tsx` + `.module.css`
- `web/src/features/chat/components/MessageList.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/chat/components/MessageItem.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/chat/components/MessageComposer.tsx` + `.module.css` + `.schema.ts` + `.test.tsx`
- `web/src/features/chat/components/EmptyChat.tsx`
- `web/src/features/chat/components/ConnectionStatusBanner.tsx` + `.module.css` + `.test.tsx`
- `web/src/features/chat/index.ts`
- Modified: `web/src/pages/ChatRoutePage.tsx` — `<ChatPanel channelId={...} />`

**Phase 9 — WS Client + integration:**
- `web/src/shared/api/ws.ts` (+ test) — `createWsClient({getToken, onFrame, onStatus})`
- `web/src/shared/api/wsClient.singleton.ts` — единый экземпляр через фабрику; экспорт для использования из сторов
- Modified: `web/src/features/chat/store/index.ts` — добавить WS-handlers (`onSubscribed`, `onMessageNew`, `onMessageSent`, `onWsError`, `setWsStatus`), `openChannel` подписывает, `sendMessage` оптимистично + ws.send, retryMessage
- Modified: `web/src/features/rooms/store/index.ts` — `appendMember` (WS handler `member.joined`), `joinByCode` вызывает `wsClient.reconnect()`
- Modified: `web/src/shared/lib/logoutFlow.ts` — вызывает `wsClient.disconnect()`
- Modified: `web/src/app/App.tsx` — подключает WS connect при `status==="authenticated"`, disconnect при logout
- Mocking infrastructure: `web/src/shared/api/__tests__/ws.fake.ts` — fake WsClient для тестов

**Phase 10 — Docker:**
- `web/Dockerfile` — multi-stage: `node:lts-alpine` builder → `nginx:alpine` runtime
- `web/nginx.conf` — SPA fallback + proxy `/api/v1`
- `web/.dockerignore`
- Modified: `docker-compose.yml` — добавить сервис `web`, обновить сервис `postgres` для общей сети, добавить сервис `api` (Dockerfile для бэка тоже нужен — но он out of scope этой фазы; вариант: запускаем `api` через `make run` локально, `web` proxy к `host.docker.internal:8080`). См. ниже «Открытое решение по api в compose».
- Modified: `Makefile` — `web-dev`, `web-build`, `web-test`, `web-lint`, `web-e2e`, `dc-up` включает `web`

**Phase 11 — E2E:**
- `web/playwright.config.ts`
- `web/e2e/auth.spec.ts` — E2E-1, E2E-2, E2E-3, E2E-10, E2E-12
- `web/e2e/rooms.spec.ts` — E2E-4
- `web/e2e/channels.spec.ts` — E2E-5, E2E-14, E2E-15
- `web/e2e/chat.spec.ts` — E2E-6, E2E-7, E2E-8
- `web/e2e/invite.spec.ts` — E2E-9
- `web/e2e/deeplink.spec.ts` — E2E-11
- `web/e2e/reconnect.spec.ts` — E2E-13
- `web/e2e/helpers.ts` — `createUser`, `loginAs`, `seedRoom`, etc.

### Modified Files (corner cases)

Перечислены в каждой фазе с диапазоном строк. Финальный composition root для фронта — `web/src/app/App.tsx` после Phase 9.

## Зависимости от бэка

Все эндпоинты и WS-фреймы уже работают на бэке в `main` (commit `7d7da4e`, ветка `feature/PR-3_5_frontend`). Контракты — `../06-api-integration.md`. Подтверждение реализации — `internal/<domain>/transport/*` и `manual_qa/<phase>/`.

Бэк-фазы, на которые опирается PR-3.5:
- PR-1.3 Auth — `docs/1_3_auth_domen/` ✅
- PR-1.4 Auth middleware — `docs/1_4_auth_middleware/` ✅
- PR-2.1 Rooms & Channels — `docs/2_1_rooms_and_channels/` ✅
- PR-3.1 Realtime Chat — `docs/3_1_realtime_chat/` ✅

## Error Mapping (свод)

| Backend code | HTTP / WS | Frontend reaction |
|---|---|---|
| `AUTH-001/002/003` | 400 | inline field error в RegisterForm |
| `AUTH-004` email taken | 409 | inline field error на `email` |
| `AUTH-005` username taken | 409 | inline field error на `username` |
| `AUTH-006` invalid creds | 401 (login) | form-level error |
| `AUTH-007/008/009` | 401 (refresh) | logoutFlow |
| `AUTH-010` access invalid | 401 (любой) | logoutFlow |
| `AUTH-011` access expired | 401 (любой) | single-flight refresh + retry |
| `AUTH-012` malformed body | 400 | bug-баннер |
| `ROOM-001` invalid name | 400 | inline |
| `ROOM-002` not found | 404 | удалить из стора + redirect /rooms |
| `ROOM-003` not member | 403 | удалить из стора + redirect /rooms |
| `ROOM-004` admin/owner req | 403 | toast |
| `ROOM-005` only owner | 403 | toast |
| `ROOM-006` already member | 409 | toast + navigate |
| `ROOM-007` invite not found | 404 | inline |
| `ROOM-008` invalid code | 400 | inline |
| `ROOM-009` invalid body | 400 | bug-баннер |
| `CHANNEL-001` invalid name | 400 | inline |
| `CHANNEL-002` invalid kind | 400 | bug-баннер |
| `CHANNEL-003` not found | 404 | удалить из стора + redirect |
| `CHANNEL-004` name taken | 409 | inline |
| `CHANNEL-005` invalid body | 400 | bug-баннер |
| `CHANNEL-006` not member | 403 | удалить из стора + redirect |
| `CHANNEL-007` admin/owner req | 403 | toast |
| `CHAT-001` invalid text | WS error | inline + pending → failed |
| `CHAT-002` channel not found | 404 / WS error | toast + close channel |
| `CHAT-003` not text | WS error | toast |
| `CHAT-004` not member | 403 / WS error | toast + reload rooms |
| `CHAT-005` invalid uuid | 400 / WS error | bug-баннер |
| `CHAT-006` invalid limit | 400 | bug-баннер |
| `CHAT-007` unsupported type | WS error | log warn (не показываем) |
| `INTERNAL` | 500 | toast |
| `NETWORK` | n/a (fetch fallback) | toast «нет соединения» |
| `CLIENT` | n/a | bug-баннер |

## Success Criteria

- [ ] Все 11 фаз завершены, чекбоксы проставлены.
- [ ] `npm --prefix web run typecheck` — 0 ошибок.
- [ ] `npm --prefix web run lint` — 0 ошибок.
- [ ] `npm --prefix web run test -- --run` — все ~151 unit/component тестов зелёные.
- [ ] `npm --prefix web run build` — собирается чисто, bundle-size в пределах разумного (~500KB gz).
- [ ] `npm --prefix web run e2e` — все 15 e2e сценариев зелёные.
- [ ] `docker compose up -d` поднимает `postgres + (api) + web`, пользователь может зайти на `http://localhost:8080/` (или прокси-порт) и пройти полный happy-path.
- [ ] Все критерии приёмки из `../README.md` (18 пунктов) выполнены.
- [ ] Все коды ошибок из `Error Mapping` имеют unit/component тест (coverage mapping в `../04-testing.md`).
- [ ] Никаких TODO/FIXME без issue-ссылок.
- [ ] Все 25 ADR из `../03-decisions.md` соблюдены.
- [ ] Никаких `any` без обоснования.
- [ ] Все Zustand-сторы экспортируют узкие селекторы и не нарушают изоляцию.
- [ ] `manual_qa/frontend/3_5/test-flow.md` написан и пройден.

## Открытое решение по `api` в docker-compose

Сейчас в `docker-compose.yml` есть только Postgres. Бэк-сервер не контейнеризован — запускается через `make run`. Phase 10 предлагает два варианта:

**A. Полный compose** — добавить сервис `api` с Dockerfile в корне репо. Минус: out of scope PR-3.5 (это бэк-инфраструктура).

**B. Hybrid** — `web` контейнер проксирует на `host.docker.internal:8080`, бэк всё ещё через `make run`. Плюс: минимальная инвазия. Минус: prod-deploy всё равно потребует Dockerfile для api позже.

**Рекомендация (фиксируется в Phase 10):** B — Hybrid, с TODO для будущей фазы «Dockerize api».

## Замечания

- Каждый phase-файл — **самодостаточный**, читается без других фаз. Имеет ссылки на design-документы (`../01-architecture.md`, `../02-behavior.md`, `../06-api-integration.md`, `../07-ui-contract.md`).
- Implementer'у через `/implement_frontend docs/3_5_frontend/plan/phase-NN.md` подаётся ОДИН файл фазы.
- Прогресс трекается чекбоксами в таблице «Phases» выше.
- Final review (cross-phase) и smoke test — после Phase 11.
