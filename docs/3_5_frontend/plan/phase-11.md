---
phase: 11
name: E2E Playwright
layer: test
depends_on: [phase-10]
plan: ./README.md
---

# Phase 11: E2E Playwright

## Цель

Настроить Playwright и написать 15 e2e-сценариев из `../04-testing.md`. После фазы — `npm --prefix web run e2e` прогоняет полный happy-path и ключевые edge cases против реального бэка (`make run`) и фронта (Vite dev или docker `web` сервис).

## Контекст

Phase 10 дала рабочий docker-сервис. Сценарии перечислены в `../04-testing.md` секция «E2E (Playwright)». Стандарты — `prompts/Tests Style (Web).txt`. ADR D-14 (Playwright).

## Файлы для создания

### `web/playwright.config.ts`

```ts
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? "github" : "html",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:5173",
    trace: "on-first-retry",
    video: "retain-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    // Firefox/WebKit — добавить если нужно после MVP
  ],
  // Опционально: автозапуск Vite (для CI без docker)
  webServer: process.env.CI ? {
    command: "npm run dev",
    url: "http://localhost:5173",
    reuseExistingServer: false,
    timeout: 60_000,
  } : undefined,
});
```

### `web/e2e/helpers.ts`

```ts
import { Page, expect } from "@playwright/test";

export function uniqueEmail(prefix = "user"): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@e2e.test`;
}

export function uniqueUsername(prefix = "u"): string {
  return `${prefix}${Date.now()}${Math.floor(Math.random() * 1000)}`;
}

export const TEST_PASSWORD = "super-secret-123";

export async function registerUser(page: Page, email = uniqueEmail(), username = uniqueUsername()) {
  await page.goto("/register");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel(/имя пользователя/i).fill(username);
  await page.getByLabel("Пароль").fill(TEST_PASSWORD);
  await page.getByRole("button", { name: /создать аккаунт/i }).click();
  await expect(page).toHaveURL(/\/rooms/);
  return { email, username };
}

export async function loginAs(page: Page, email: string, password = TEST_PASSWORD) {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Пароль").fill(password);
  await page.getByRole("button", { name: /войти/i }).click();
  await expect(page).toHaveURL(/\/rooms/);
}

export async function createRoom(page: Page, name: string): Promise<string> {
  await page.getByRole("button", { name: /\+/ }).click();
  await page.getByRole("menuitem", { name: /создать комнату/i }).click();
  await page.getByLabel("Название").fill(name);
  await page.getByRole("button", { name: /создать/i }).click();
  // навигация на /rooms/:id
  await expect(page).toHaveURL(/\/rooms\/[a-f0-9-]{36}/);
  const match = page.url().match(/\/rooms\/([a-f0-9-]{36})/);
  return match![1];
}
```

### Тестовые файлы

#### `web/e2e/auth.spec.ts`

```ts
import { test, expect } from "@playwright/test";
import { registerUser, loginAs, uniqueEmail, TEST_PASSWORD } from "./helpers";

test("E2E-1: register → main", async ({ page }) => {
  await registerUser(page);
  await expect(page).toHaveURL("/rooms");
  await expect(page.getByText(/мои комнаты/i)).toBeVisible();
});

test("E2E-2: login existing", async ({ page }) => {
  const { email } = await registerUser(page);
  // logout
  await page.getByRole("button", { name: /выйти/i }).click();
  await expect(page).toHaveURL("/login");
  await loginAs(page, email);
});

test("E2E-3: login with wrong password", async ({ page }) => {
  const { email } = await registerUser(page);
  await page.getByRole("button", { name: /выйти/i }).click();
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Пароль").fill("wrong-password-99");
  await page.getByRole("button", { name: /войти/i }).click();
  await expect(page.getByRole("alert")).toContainText(/неверный/i);
});

test("E2E-10: logout clears local storage", async ({ page }) => {
  await registerUser(page);
  await page.getByRole("button", { name: /выйти/i }).click();
  await expect(page).toHaveURL("/login");
  const access = await page.evaluate(() => localStorage.getItem("rupor.access"));
  expect(access).toBeNull();
});

test("E2E-12: session restore after reload", async ({ page }) => {
  await registerUser(page);
  await page.reload();
  await expect(page).toHaveURL("/rooms");
  await expect(page.getByText(/мои комнаты/i)).toBeVisible();
});
```

#### `web/e2e/rooms.spec.ts`

```ts
test("E2E-4: create room", async ({ page }) => {
  await registerUser(page);
  const roomId = await createRoom(page, "test-room-" + Date.now());
  await expect(page.getByText(`test-room-`)).toBeVisible();  // в sidebar
});
```

#### `web/e2e/channels.spec.ts`

```ts
test("E2E-5: create text channel", async ({ page }) => {
  await registerUser(page);
  await createRoom(page, "ch-room-" + Date.now());
  await page.getByRole("button", { name: /\+/i, exact: false }).nth(1).click();  // или specific aria
  await page.getByLabel("Название").fill("general");
  await page.getByLabel("Тип").locator("input[value=text]").check();
  await page.getByRole("button", { name: /создать/i }).click();
  await expect(page).toHaveURL(/channels\/[a-f0-9-]{36}/);
  await expect(page.getByText("#general")).toBeVisible();
});

test("E2E-14: delete channel", async ({ page }) => { /* ... */ });

test("E2E-15: voice channel is disabled", async ({ page }) => {
  await registerUser(page);
  await createRoom(page, "v-room-" + Date.now());
  // Open create modal, select voice, create
  // ...
  // Voice channel item should have aria-disabled
  await expect(page.getByRole("listitem", { name: /voice/i })).toHaveAttribute("aria-disabled", "true");
});
```

#### `web/e2e/chat.spec.ts`

```ts
test("E2E-6: send message", async ({ page }) => {
  await registerUser(page);
  await createRoom(page, "chat-room");
  // create text channel & open
  // ...
  await page.getByPlaceholder(/напиши сообщение/i).fill("hello world");
  await page.keyboard.press("Enter");
  await expect(page.getByText("hello world")).toBeVisible();
  // wait for committed status (not just pending)
  await expect(page.locator(`[data-status="committed"]`).filter({ hasText: "hello world" })).toBeVisible();
});

test("E2E-7: multi-tab receive", async ({ browser }) => {
  // Tab A
  const ctxA = await browser.newContext();
  const pageA = await ctxA.newPage();
  const { email } = await registerUser(pageA);
  const roomId = await createRoom(pageA, "mt-room");
  // ... create channel, open
  // Tab B same user
  const ctxB = await browser.newContext();
  const pageB = await ctxB.newPage();
  await loginAs(pageB, email);
  await pageB.goto(`/rooms/${roomId}/channels/...`);  // тот же channel
  // Tab A sends
  await pageA.getByPlaceholder(/напиши сообщение/i).fill("hi from A");
  await pageA.keyboard.press("Enter");
  // Tab B sees
  await expect(pageB.getByText("hi from A")).toBeVisible();
});

test("E2E-8: optimistic ack", async ({ page }) => {
  // ... отправить → видим pending → видим committed
  await page.getByPlaceholder(/напиши сообщение/i).fill("opt test");
  await page.keyboard.press("Enter");
  // Сначала pending
  await expect(page.locator('[data-status="pending"]').filter({ hasText: "opt test" })).toBeVisible();
  // Через некоторое время — committed
  await expect(page.locator('[data-status="committed"]').filter({ hasText: "opt test" })).toBeVisible({ timeout: 5000 });
});
```

**Note:** для надёжных селекторов компонентам нужно добавить `data-status="..."` атрибуты — это явное API для тестов. Добавляется в `<MessageItem>` (можно сделать в Phase 9 или внести `data-testid` в Phase 11).

#### `web/e2e/invite.spec.ts`

```ts
test("E2E-9: invite + join (two users)", async ({ browser }) => {
  // User A: register, create room, generate invite
  const ctxA = await browser.newContext();
  const pageA = await ctxA.newPage();
  await registerUser(pageA);
  await createRoom(pageA, "invite-room");
  await pageA.getByRole("button", { name: /пригласить/i }).click();
  await pageA.getByRole("button", { name: /сгенерировать/i }).click();
  const code = await pageA.locator(`[data-test="invite-code"]`).innerText();

  // User B: register, join by code
  const ctxB = await browser.newContext();
  const pageB = await ctxB.newPage();
  await registerUser(pageB);
  await pageB.getByRole("button", { name: /\+/i }).click();
  await pageB.getByRole("menuitem", { name: /войти по коду/i }).click();
  await pageB.getByLabel("Код").fill(code);
  await pageB.getByRole("button", { name: /войти/i }).click();
  // B видит комнату
  await expect(pageB.getByText("invite-room")).toBeVisible();

  // B пишет → A видит (realtime)
  // ...
});
```

#### `web/e2e/deeplink.spec.ts`

```ts
test("E2E-11: deep-link redirects to login and back", async ({ page }) => {
  const targetUrl = "/rooms/00000000-0000-0000-0000-000000000000";
  await page.goto(targetUrl);
  await expect(page).toHaveURL(/\/login/);
  await registerUser(page);
  // После register получаем редирект на /rooms (по умолчанию), не на deeplink
  // (т.к. register был сам по себе, а не из ловушки /login state.from)
  // Чтобы тест проверял именно deeplink: register, logout, открыть deeplink, login → должны попасть на deeplink
  // ...
});
```

#### `web/e2e/reconnect.spec.ts`

```ts
test("E2E-13: WS reconnect banner", async ({ page }) => {
  await registerUser(page);
  await createRoom(page, "reconnect-room");
  // ... open channel
  // Прервать WS — через page.route или CDP
  await page.context().route("**/api/v1/ws*", (route) => route.abort());
  // Подождать banner
  await expect(page.getByRole("status").filter({ hasText: /переподключ/i })).toBeVisible({ timeout: 10_000 });
  // Снять блокировку
  await page.context().unroute("**/api/v1/ws*");
  // Banner исчезает
  await expect(page.getByRole("status").filter({ hasText: /переподключ/i })).not.toBeVisible({ timeout: 30_000 });
});
```

## Файлы для модификации

- `web/package.json` — добавить script `"e2e": "playwright test"`, `"e2e:install": "playwright install"`.
- `Makefile` — `web-e2e` target (добавлен в Phase 10, проверить).
- Опционально: некоторым компонентам (`<MessageItem />`, invite code) добавить `data-status` / `data-test` атрибуты для надёжных селекторов.

## Ключевые решения

- **D-14** Playwright — без альтернатив, выбран в дизайне.
- Каждый сценарий создаёт уникального пользователя через `uniqueEmail()` — изоляция.
- Никаких `page.waitForTimeout()` — только `expect(...).toBeVisible()` и `waitForResponse`.
- Селекторы: `getByRole` / `getByLabel` приоритетно. `data-test` — только когда `role/label` не подходит.
- E2E-13 (reconnect) использует `page.context().route()` для блокировки WS — не убиваем бэк.

## Verification

- [ ] `npm --prefix web run e2e:install` ставит Playwright-браузеры.
- [ ] `make dc-up && make migrate-up && make run` (бэк + Postgres локально).
- [ ] `npm --prefix web run dev` (фронт на 5173).
- [ ] `npm --prefix web run e2e` — все 15 сценариев зелёные.
- [ ] При CI `playwright.config.ts.webServer` автозапускает Vite.
- [ ] `playwright-report/` показывает зелёные сценарии при openе.
- [ ] Multi-tab сценарий (E2E-7) использует два разных browser context, реально проверяет broadcast.
- [ ] Reconnect (E2E-13) проверяет появление и исчезновение banner.
- [ ] Никаких `waitForTimeout` в коде тестов.
- [ ] Бьётся ли test isolation? Каждый test создаёт уникальный email → новый user, БД не пересекается между тестами.

## Cross-phase Final Review

После Phase 11 — финальный cross-phase review (см. `/implement_frontend` Phase 3). Проверки:

- [ ] Нет дубликатов api-функций.
- [ ] Все TS-типы DTO собраны в `features/<feature>/types.ts`.
- [ ] Нет cross-feature импортов.
- [ ] Нет осиротевших компонентов.
- [ ] Нет TODO/FIXME без issue-ссылок (особенно — нет ни одного `// TODO(phase-NN)` в финале).
- [ ] Все Zustand-сторы имеют `clear()` и подключены к `logoutFlow`.
- [ ] Auto-refresh + WS reconnect реализованы в `shared/api/`, не дублируются в фичах.
- [ ] Все роуты в `web/src/routes.tsx`.
- [ ] bundle size < разумного порога (~500KB gz JS).
- [ ] manual_qa/frontend/3_5/test-flow.md написан и пройден.
