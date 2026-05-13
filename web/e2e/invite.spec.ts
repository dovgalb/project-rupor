// Phase 11: invite/join сценарий E2E-9. Два пользователя в двух browser context'ах.
// User A создаёт комнату, генерирует код. User B вступает по коду.

import { expect, test } from "@playwright/test";

import {
  createRoom,
  createTextChannel,
  registerUser,
  sendMessage,
} from "./helpers";

test("E2E-9: invite + join — B видит комнату A, обмен сообщениями работает", async ({
  browser,
}) => {
  // ===== User A =====
  const ctxA = await browser.newContext();
  const pageA = await ctxA.newPage();
  await registerUser(pageA);
  const roomName = `inv-${Date.now().toString()}`;
  await createRoom(pageA, roomName);
  // Сначала генерируем код приглашения — кнопка «Пригласить» живёт только на
  // /rooms/:rid (RoomPage), а после createTextChannel мы уйдём в ChatRoutePage.
  await pageA.getByRole("button", { name: "Пригласить", exact: true }).click();
  await pageA
    .getByRole("dialog")
    .getByRole("button", { name: /^сгенерировать$/i })
    .click();
  const codeLocator = pageA.getByRole("dialog").getByLabel("Код приглашения");
  await expect(codeLocator).toBeVisible();
  const code = (await codeLocator.innerText()).trim();
  expect(code).toMatch(/^[A-Z0-9]{8}$/);
  // Закрываем модалку (Modal не подписан на Escape — закрываем через "×").
  await pageA
    .getByRole("dialog")
    .getByRole("button", { name: "Закрыть" })
    .click();
  await expect(pageA.getByRole("dialog")).toHaveCount(0);
  await createTextChannel(pageA, "lobby");

  // ===== User B =====
  const ctxB = await browser.newContext();
  const pageB = await ctxB.newPage();
  await registerUser(pageB);
  // На empty-state Sidebar показывает 2 кнопки «Войти по коду» (header + CTA),
  // берём первую — она в header (RoomList aria-label).
  await pageB
    .getByRole("button", { name: "Войти по коду" })
    .first()
    .click();
  await pageB.getByLabel("Код приглашения").fill(code);
  await pageB.getByRole("dialog").getByRole("button", { name: "Войти" }).click();

  // После успешного join фронт сам делает navigate на /rooms/:rid.
  await expect(pageB).toHaveURL(/\/rooms\/[0-9a-f-]{36}/, { timeout: 10_000 });
  // B видит комнату в своём Sidebar.
  await expect(
    pageB
      .locator("nav[aria-label='Главная навигация']")
      .getByText(roomName),
  ).toBeVisible({ timeout: 10_000 });

  // B заходит в канал lobby (loadChannels подтянет его).
  await pageB
    .locator("nav[aria-label='Главная навигация']")
    .getByRole("button", { name: /lobby/ })
    .click();
  await expect(
    pageB.getByRole("textbox", { name: "Сообщение" }),
  ).toBeEnabled({ timeout: 10_000 });

  // B пишет → A видит (realtime).
  const text = `hi-from-B-${Date.now().toString()}`;
  await sendMessage(pageB, text);
  // 1. Сначала ждём ack у самого B (optimistic → committed) — гарантирует, что
  //    backend получил, обработал и отбродкастил.
  await expect(
    pageB.locator('li[data-status="committed"]').filter({ hasText: text }),
  ).toBeVisible({ timeout: 15_000 });
  // 2. Теперь A через broadcast в channel:lobby (A был подписан с момента
  //    собственного openChannel).
  await expect(
    pageA.locator('li[data-status="committed"]').filter({ hasText: text }),
  ).toBeVisible({ timeout: 15_000 });

  await ctxA.close();
  await ctxB.close();
});
