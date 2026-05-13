---
parent: ./README.md
view: quality
feature: 3_5-frontend
---

# 3.5 Frontend — Testing Strategy

Стек: **vitest** + **React Testing Library** (component) + **Playwright** (e2e). См. `prompts/Tests Style (Web).txt`.

## Coverage Mapping (ошибки бэка → тест)

Каждый код ошибки бэка, который UI визуализирует, имеет тест:

| Use Case | Backend Code | Тип теста | Имя теста |
|---|---|---|---|
| Register | `AUTH-001` invalid email | component | `<RegisterForm /> подсвечивает поле email при AUTH-001` |
| Register | `AUTH-002` invalid username | component | `<RegisterForm /> подсвечивает поле username при AUTH-002` |
| Register | `AUTH-003` invalid password | component | `<RegisterForm /> подсвечивает поле password при AUTH-003` |
| Register | `AUTH-004` email taken | component | `<RegisterForm /> подсвечивает email при AUTH-004` |
| Register | `AUTH-005` username taken | component | `<RegisterForm /> подсвечивает username при AUTH-005` |
| Login | `AUTH-006` invalid creds | component | `<LoginForm /> показывает form-level error при AUTH-006` |
| Login/Refresh | `AUTH-007/008/009` | unit (fetch.ts) | `apiFetch вызывает logoutFlow при AUTH-007/008/009 на /auth/refresh` |
| Любой | `AUTH-010` access invalid | unit (fetch.ts) | `apiFetch триггерит logout при AUTH-010` |
| Любой | `AUTH-011` access expired | unit (refresh.ts) | `single-flight refresh: параллельные 401 ждут один промис` |
| Любой | `AUTH-011` retry loop | unit (fetch.ts) | `apiFetch триггерит logout если повторный 401 AUTH-011 после refresh` |
| Create room | `ROOM-001` invalid name | component | `<CreateRoomModal /> подсвечивает name при ROOM-001` |
| Get room | `ROOM-002` not found | store | `useRoomsStore удаляет комнату из state при 404` |
| Membership | `ROOM-003` not a member | store | `useRoomsStore удаляет комнату и редиректит при ROOM-003` |
| Invite | `ROOM-004` admin/owner required | component | `<InviteCodeModal /> показывает toast при ROOM-004` |
| Delete room | `ROOM-005` only owner | component | `<DeleteRoomConfirm /> показывает toast при ROOM-005` |
| Join | `ROOM-006` already member | component | `<JoinByCodeModal /> показывает toast и navigate при ROOM-006` |
| Join | `ROOM-007` invite not found | component | `<JoinByCodeModal /> inline error при ROOM-007` |
| Join | `ROOM-008` invalid code | component | `<JoinByCodeModal /> inline error при ROOM-008` |
| Create channel | `CHANNEL-001` invalid name | component | `<CreateChannelModal /> inline error при CHANNEL-001` |
| Create channel | `CHANNEL-004` name taken | component | `<CreateChannelModal /> inline error при CHANNEL-004` |
| Create channel | `CHANNEL-007` admin/owner req | component | `<CreateChannelModal /> toast при CHANNEL-007` |
| List messages | `CHAT-004` not member | store | `useChatStore чистит state при CHAT-004` |
| List messages | `CHAT-002` channel not found | store | `useChatStore чистит state при CHAT-002` |
| WS error | `CHAT-001` invalid text | store | `useChatStore помечает pending как failed при WS CHAT-001` |
| WS error | `CHAT-002/003/004` | store | `useChatStore: toast + close channel при WS CHAT-002` |
| WS close 1006 | reconnect | unit (ws.ts) | `wsClient переходит в reconnecting и делает backoff` |
| Join + WS | reconnect после join | store | `useRoomsStore.joinByCode вызывает wsClient.reconnect()` |
| Multi-tab | `message.new` дубль | store | `useChatStore.appendMessage дедуплицирует по id` |
| Optimistic | `message.sent` ack | store | `useChatStore коммитит pending temp-id в server-id` |

---

## Unit tests — `web/src/shared/api/`

### `shared/api/fetch.ts` (~10 тестов)

| Тест | Что проверяет |
|---|---|
| подставляет `Authorization: Bearer` из tokenStorage | header выставлен |
| без access-token не подставляет header | header отсутствует |
| на 200 OK возвращает `{ ok: true, data }` | парсинг |
| на 4xx с ErrorEnvelope возвращает `{ ok: false, status, error }` | парсинг |
| на network error возвращает `{ ok: false, status: 0, error: { code: "NETWORK" } }` | fallback |
| на 401 AUTH-011 запускает refresh и retry-ит запрос с новым токеном | auto-refresh path |
| на 401 AUTH-007 НЕ запускает refresh, вызывает logoutFlow | logout path |
| на 401 AUTH-008/009/010 НЕ запускает refresh, вызывает logoutFlow | logout path |
| если оригинальный запрос — это сам /auth/refresh, не запускает refresh повторно | loop protection |
| если retry после refresh снова дал 401 AUTH-011, вызывает logoutFlow | loop protection |

### `shared/api/refresh.ts` (~3 теста)

| Тест | Что проверяет |
|---|---|
| первый вызов делает POST /auth/refresh | сеть |
| параллельные вызовы получают один и тот же promise (single-flight) | concurrency |
| после resolve currentRefresh сбрасывается в null | cleanup |

### `shared/api/ws.ts` (~6 тестов)

| Тест | Что проверяет |
|---|---|
| connect подставляет `?token=` из tokenStorage | URL |
| on open dispatches setWsStatus("open") | handler |
| на закрытие 1000 (нормально) НЕ запускает reconnect | reconnect logic |
| на закрытие 1006 запускает reconnect с backoff 1s | reconnect logic |
| при reconnect восстанавливает subscribe на текущий открытый канал | resubscribe |
| disconnect() закрывает с 1000 и не реконнектит | cleanup |

### `shared/api/token-storage.ts` (~3 теста)

| Тест | Что проверяет |
|---|---|
| set записывает 4 ключа в localStorage | persist |
| getAccess возвращает то, что положили | round-trip |
| clear удаляет все 4 ключа | cleanup |

---

## Unit tests — Stores

### `useAuthStore` (~10 тестов)

| Тест | Что проверяет |
|---|---|
| login: сохраняет токены и грузит /me | happy path |
| login: status=error при AUTH-006 | error mapping |
| register: при ok вызывает auto-login | composition |
| register: status=error при AUTH-004 | error mapping |
| loadMe: status=authenticated при ok | mount-flow |
| loadMe: clear при 401 (после fail refresh) | logout-flow |
| logout: вызывает clear на всех 4 сторах | cascade |
| logout: clear на tokenStorage | cleanup |
| persist: currentUser восстанавливается из localStorage | hydration |
| selectIsAuthenticated: возвращает true только при status=authenticated | selector |

### `useRoomsStore` (~12 тестов)

| Тест | Что проверяет |
|---|---|
| loadRooms: записывает items, status=ready | happy |
| loadRooms: status=error при failure | error |
| createRoom: добавляет в state, role=owner | happy |
| createRoom: показывает inline error при ROOM-001 | mapping |
| deleteRoom: удаляет из rooms, чистит channelsByRoom | cascade |
| deleteRoom: при ROOM-005 не удаляет из state | mapping |
| loadMembers: записывает в membersByRoom | happy |
| regenerateInvite: записывает в activeInviteByRoom | happy |
| joinByCode: вызывает wsClient.reconnect() | side-effect |
| joinByCode: ROOM-006 → не добавляет дубль | mapping |
| appendMember: дедуплицирует по userId | dedup |
| clear: сбрасывает state до initial | cleanup |

### `useChannelsStore` (~8 тестов)

| Тест | Что проверяет |
|---|---|
| loadChannels: записывает в channelsByRoom[rid] | happy |
| createChannel: добавляет в channelsByRoom[rid] | happy |
| createChannel: inline error при CHANNEL-004 | mapping |
| deleteChannel: удаляет из channelsByRoom[rid] | happy |
| selectChannel(id): обновляет activeChannelId | happy |
| на 404 CHANNEL-003 удаляет из state | mapping |
| voice-каналы попадают в channelsByRoom как обычные | API parity |
| clear: сбрасывает state | cleanup |

### `useChatStore` (~14 тестов)

| Тест | Что проверяет |
|---|---|
| openChannel: вызывает listMessages и записывает в messagesByChannel | happy |
| openChannel: вызывает wsClient.send subscribe | happy |
| openChannel: НЕ грузит историю если уже есть в state | cache |
| loadMoreHistory: использует nextBefore как cursor | pagination |
| loadMoreHistory: останавливается при nextBefore=null | end |
| sendMessage: добавляет pending с tempId | optimistic |
| sendMessage: ws.send message.send с правильным payload | wire |
| onMessageSent: коммитит pending → committed с server id | reconciliation |
| onMessageNew: добавляет committed если нет дубля | append |
| onMessageNew: игнорирует дубль по id | dedup |
| onWsError CHAT-001: помечает последний pending как failed | error |
| onWsError CHAT-004: toast + close channel | error |
| setWsStatus(reconnecting): обновляет state | status |
| clear: сбрасывает state | cleanup |

---

## Component tests — RTL

Каждый компонент имеет тесты на ВСЕ применимые состояния.

### `<LoginForm />` (~6 тестов)

| Тест | Что проверяет |
|---|---|
| рендерит поля email и password (a11y by label) | render |
| Tab переходит email → password → submit | keyboard |
| Enter в любом поле — submit формы | keyboard |
| при isSubmitting кнопка disabled и со спиннером | loading state |
| при form-level error отображает `role="alert"` | error state |
| невалидный email — zod-ошибка inline | client validation |

### `<RegisterForm />` (~7 тестов)
Аналогично + поле username и mapping `AUTH-001/002/003/004/005` на конкретные поля.

### `<RoomList />` (~5 тестов)

| Тест | Что проверяет |
|---|---|
| рендерит skeletons при loading | loading state |
| рендерит EmptyState при rooms=[] | empty state |
| рендерит список при ready | success state |
| activeRoomId подсвечивается (aria-current=page) | a11y |
| click по item вызывает onClick | interaction |

### `<CreateRoomModal />` (~6 тестов)

| Тест | Что проверяет |
|---|---|
| открыт/закрыт через props | lifecycle |
| Esc закрывает модалку | keyboard |
| focus trap внутри modal | a11y |
| submit вызывает store action и onCreated | happy |
| при error ROOM-001 показывает inline | mapping |
| длинное имя > 64 рун — zod inline | validation |

### `<JoinByCodeModal />` (~5 тестов)
Те же паттерны + специфическая validation кода.

### `<InviteCodeModal />` (~4 теста)
Состояния: no-code / generating / has-code / error.

### `<DeleteRoomConfirm />` (~3 теста)
Открыт/закрыт, confirm/cancel, danger-aria.

### `<ChannelList />` (~6 тестов)
Loading / empty / ready (text + voice) / canManage=true показывает «+» / voice disabled / click text вызывает selectChannel.

### `<ChannelListItem />` (~3 теста)
Active highlight / disabled для voice / click handler.

### `<CreateChannelModal />` (~5 тестов)
Поля + kind radio, validation, mapping ошибок.

### `<MembersList />` (~4 теста)
Loading / ready / placeholder UUID-имя для других / реальный username для me.

### `<Avatar />` (~3 теста)
Цвет детерминирован по userId / aria-hidden / два символа из id.

### `<MessageList />` (~8 тестов)

| Тест | Что проверяет |
|---|---|
| рендерит loading при loadingHistory | loading |
| рендерит `<EmptyChat />` при messages=[] | empty |
| рендерит список при messages>0 | ready |
| scroll к низу при новом message.new (если уже у низа) | scroll behavior |
| показывает pill «↓ новые (N)» если scroll не у низа | scroll behavior |
| scroll к верху триггерит loadMoreHistory | infinite scroll |
| при nextBefore=null рендерит «Начало канала» | end state |
| role="log", aria-live="polite" | a11y |

### `<MessageItem />` (~6 тестов)
Layout / status pending / committed / failed / linkify валидных URL / НЕ рендерит HTML.

### `<MessageComposer />` (~6 тестов)
Enter submit / Shift+Enter newline / counter показан при >3000 / disabled при wsStatus !== open / submit вызывает store / валидация min/max.

### `<ConnectionStatusBanner />` (~3 теста)
Скрыт при wsStatus=open / показан при reconnecting / role=status aria-live=polite.

### `<UserBadge />` (~3 теста)
Рендерит username / click «Выйти» вызывает onLogout / tooltip с email.

### Guards (~4 теста)

| Гард | Тест |
|---|---|
| `<RequireAuth>` | редиректит на /login если idle, рендерит children если authenticated |
| `<RedirectIfAuthenticated>` | редиректит на /rooms если authenticated |
| `<RequireMembership>` | редиректит на /rooms если room нет в стора |
| `<RequireChannelInRoom>` | редиректит на /rooms/:rid если channel нет; не пускает на voice |

---

## E2E (Playwright)

### Окружение

- Бэк: `make dc-up` + `make migrate-up` + `make run`.
- Фронт: `npm --prefix web run dev`.
- Конфиг `web/playwright.config.ts`:
  - `baseURL: "http://localhost:5173"`.
  - `webServer` — опционально автозапуск Vite (для CI).

### Сценарии

| # | Сценарий | Шаги |
|---|----------|------|
| E2E-1 | Happy path register → main | register → должны попасть на /rooms |
| E2E-2 | Login существующего | login → /rooms |
| E2E-3 | Login с неверным паролем | login → form-level error |
| E2E-4 | Создать комнату | login → click `+` → create → должна появиться в sidebar и навигироваться |
| E2E-5 | Создать text-канал | login → choose room → click `+` channel → create → list пополняется |
| E2E-6 | Отправить сообщение | login → choose channel → composer Enter → сообщение появляется committed |
| E2E-7 | Получить сообщение (multi-tab) | tab A открывает → tab B отправляет в той же комнате → tab A видит message.new |
| E2E-8 | Optimistic ack | tab A отправляет → видит pending → видит commit (status «committed») |
| E2E-9 | Invite + join (двое пользователей) | A создаёт комнату, генерирует invite, B регистрируется и joining по коду → B видит комнату → может писать |
| E2E-10 | Logout | logout → /login + localStorage очищен |
| E2E-11 | Deep-link без авторизации | открываем /rooms/:rid → редирект /login → после логина возвращаемся на /rooms/:rid |
| E2E-12 | Воссоздание сессии после reload | login → reload → остаёмся в /rooms (loadMe восстанавливает state) |
| E2E-13 | WS reconnect | open chat → kill bg бэка → видим banner «Переподключение…» → restart бэка → banner исчезает |
| E2E-14 | Delete channel | admin создаёт канал → удаляет → канал исчезает из списка |
| E2E-15 | Voice-канал disabled | пробуем click на voice-канал → ничего не происходит, tooltip «coming soon» |

### Правила e2e

- Каждый сценарий создаёт уникального пользователя (`user-${Date.now()}@e2e.test`).
- Selectors — `getByRole`, `getByLabel`. Никаких CSS-классов.
- Никаких `page.waitForTimeout()` — только `waitForResponse`, `waitFor`, `toBeVisible`.
- E2E-13 (reconnect) использует `route.abort()` для эмуляции потери сети.

---

## Stubs / Mocks

### `shared/api/fetch.ts`

В unit-тестах — мок через `vi.stubGlobal("fetch", vi.fn())`:

```ts
vi.stubGlobal("fetch", vi.fn().mockResolvedValueOnce(
  new Response(JSON.stringify(data), { status: 200, headers: {...} })
));
```

В тестах сторов — мокаем конкретные api-функции через `vi.spyOn(authApi, "login")`.

### `shared/api/ws.ts`

`createWsClient` — фабрика. В тестах подменяем на in-memory implementation:

```ts
function createFakeWsClient(): WsClient {
  const handlers = { onFrame: noop, onStatus: noop };
  return {
    connect: () => handlers.onStatus("open"),
    disconnect: () => handlers.onStatus("closed"),
    send: vi.fn(),
    status: "idle",
    onFrame: (h) => { handlers.onFrame = h; return noop; },
    onStatus: (h) => { handlers.onStatus = h; return noop; },
    /** test-only: эмулировать входящий фрейм */
    _emit: (frame: IncomingFrame) => handlers.onFrame(frame),
  };
}
```

### react-router

`<MemoryRouter initialEntries={["/rooms"]}>` оборачивает рендер компонентов в RTL-тестах.

### Toast

Глобальный toaster в тестах — мокаем через `vi.spyOn(toast, "error")`.

---

## Test Count Summary

| Слой | Кол-во |
|------|--------|
| shared/api/fetch.ts | ~10 |
| shared/api/refresh.ts | ~3 |
| shared/api/ws.ts | ~6 |
| shared/api/token-storage.ts | ~3 |
| useAuthStore | ~10 |
| useRoomsStore | ~12 |
| useChannelsStore | ~8 |
| useChatStore | ~14 |
| Component tests (форма + sidebar + chat + guards) | ~70 |
| E2E Playwright | 15 |
| **Total** | **~151** |

---

## Что НЕ тестируем

- Внутренности react-router-dom, zustand, react-hook-form, zod.
- Стили / визуальные регрессии (нет дизайн-системы и visual-regression-набора в MVP).
- Performance — после MVP, если потребуется.

---

## Чек-лист перед merge PR-3.5

- [ ] `npm --prefix web run typecheck` — 0 ошибок
- [ ] `npm --prefix web run lint` — 0 ошибок
- [ ] `npm --prefix web run test -- --run` — все ~151 unit/component тестов зелёные
- [ ] `npm --prefix web run build` — собирается чисто, bundle-size в пределах разумного
- [ ] `npm --prefix web run e2e` — все 15 сценариев зелёные (на CI и локально)
- [ ] Все коды ошибок из coverage mapping покрыты тестами
- [ ] Все компоненты со `state` имеют тест на каждое состояние
- [ ] Никаких `it.only` / `describe.only`
- [ ] Никаких snapshot-тестов
