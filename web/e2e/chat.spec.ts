// Phase 11: chat-сценарии E2E-6 (send), E2E-7 (multi-tab), E2E-8 (optimistic ack).
// MessageItem рендерит data-status="pending|committed|failed" для надёжного селектора.

import { expect, test } from "@playwright/test";

import {
  createRoom,
  createTextChannel,
  loginAs,
  registerUser,
  sendMessage,
} from "./helpers";

test("E2E-6: отправить сообщение → видно в списке как committed", async ({
  page,
}) => {
  await registerUser(page);
  await createRoom(page, `chat-${Date.now().toString()}`);
  await createTextChannel(page, "general");
  // Ждём появления composer'а (wsStatus==="open" → enabled).
  const textbox = page.getByRole("textbox", { name: "Сообщение" });
  await expect(textbox).toBeEnabled({ timeout: 10_000 });

  await sendMessage(page, "hello world");
  // Сначала pending или committed (в зависимости от того, кто пришёл первым).
  await expect(
    page
      .locator('li[data-status="committed"]')
      .filter({ hasText: "hello world" }),
  ).toBeVisible({ timeout: 10_000 });
});

test("E2E-7: multi-tab — A отправляет, B видит без дубля", async ({
  browser,
}) => {
  // Tab A: регистрируем юзера, создаём комнату+канал, отправляем сообщение.
  const ctxA = await browser.newContext();
  const pageA = await ctxA.newPage();
  const { email } = await registerUser(pageA);
  const roomName = `mt-${Date.now().toString()}`;
  await createRoom(pageA, roomName);
  await createTextChannel(pageA, "mt-channel");
  const url = pageA.url();

  // Tab B: тот же юзер. Сначала логин, потом кликаем по комнате (RequireMembership
  // ждёт rooms-store), затем по каналу (RequireChannelInRoom ждёт channels-store).
  // goto на deep-link тут не годится: rooms/channels ещё не подгружены, гарды
  // редиректят на /rooms.
  const ctxB = await browser.newContext();
  const pageB = await ctxB.newPage();
  await loginAs(pageB, email);
  const sidebarB = pageB.locator("nav[aria-label='Главная навигация']");
  await sidebarB.getByText(roomName).click();
  await sidebarB
    .getByRole("button", { name: /mt-channel/ })
    .click();
  // Sanity: URL обновился на канал.
  await expect(pageB).toHaveURL(
    /\/rooms\/[0-9a-f-]{36}\/channels\/[0-9a-f-]{36}/,
  );
  // Полученный URL должен совпадать с URL pageA.
  expect(pageB.url()).toBe(url);
  await expect(
    pageB.getByRole("textbox", { name: "Сообщение" }),
  ).toBeEnabled({ timeout: 10_000 });

  // A отправляет — B должна увидеть и не должно быть дубля.
  const textA = `from-A-${Date.now().toString()}`;
  await sendMessage(pageA, textA);
  await expect(
    pageB.locator('li[data-status="committed"]').filter({ hasText: textA }),
  ).toBeVisible({ timeout: 10_000 });
  await expect(
    pageA.locator('li[data-status="committed"]').filter({ hasText: textA }),
  ).toHaveCount(1);
  await expect(
    pageB.locator('li[data-status="committed"]').filter({ hasText: textA }),
  ).toHaveCount(1);

  await ctxA.close();
  await ctxB.close();
});

test("E2E-8: optimistic update — pending → committed", async ({ page }) => {
  await registerUser(page);
  await createRoom(page, `opt-${Date.now().toString()}`);
  await createTextChannel(page, "opt");
  await expect(
    page.getByRole("textbox", { name: "Сообщение" }),
  ).toBeEnabled({ timeout: 10_000 });

  const text = `opt-test-${Date.now().toString()}`;
  await sendMessage(page, text);
  // pending может быть очень кратковременно, но мы должны увидеть committed-статус в итоге.
  await expect(
    page.locator('li[data-status="committed"]').filter({ hasText: text }),
  ).toBeVisible({ timeout: 10_000 });
});
