// Phase 11: channels-сценарии E2E-5, 14, 15.

import { expect, test } from "@playwright/test";

import { createRoom, createTextChannel, registerUser } from "./helpers";

test("E2E-5: создать text-канал → редирект на /channels/:cid, виден # в Sidebar", async ({
  page,
}) => {
  await registerUser(page);
  await createRoom(page, `r-${Date.now().toString()}`);
  const channelId = await createTextChannel(page, "general");
  expect(channelId).toMatch(/^[0-9a-f-]{36}$/);
  // Канал виден в Sidebar в секции «Текстовые каналы».
  await expect(
    page.locator("nav[aria-label='Главная навигация']").getByText(/#\s*general/),
  ).toBeVisible();
});

// TODO: подключить DeleteChannelConfirm в ChannelListItem (или меню канала),
// после этого снять skip — сейчас в UI нет кнопки удаления канала.
test.skip("E2E-14: удалить text-канал → исчезает из Sidebar", async ({
  page,
}) => {
  await registerUser(page);
  await createRoom(page, `r-del-${Date.now().toString()}`);
  await createTextChannel(page, "to-delete");
  // ... сценарий подключим, когда добавим кнопку удаления канала в UI.
});

test("E2E-15: voice-канал создаётся, но в списке disabled (D-12)", async ({
  page,
}) => {
  await registerUser(page);
  await createRoom(page, `r-voice-${Date.now().toString()}`);
  // Открываем модалку создания канала.
  await page.getByRole("button", { name: "Создать канал" }).click();
  await page.getByLabel("Название канала").fill("ops-voice");
  await page.getByLabel("Голосовой").check();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Создать" })
    .click();
  // Voice-канал виден в секции «Голосовые каналы». aria-disabled у самой кнопки
  // (см. ChannelListItem.tsx: <button aria-disabled={...}>).
  const voiceButton = page
    .locator("nav[aria-label='Главная навигация']")
    .getByRole("button", { name: /ops-voice/ });
  await expect(voiceButton).toBeVisible();
  await expect(voiceButton).toHaveAttribute("aria-disabled", "true");
});
