// Phase 11: rooms-сценарий E2E-4.

import { expect, test } from "@playwright/test";

import { createRoom, registerUser } from "./helpers";

test("E2E-4: создать комнату → она появляется в Sidebar и URL ведёт на /rooms/:id", async ({
  page,
}) => {
  await registerUser(page);
  const roomName = `room-${Date.now().toString()}`;
  const roomId = await createRoom(page, roomName);
  // В Sidebar появилась запись с этим именем.
  await expect(
    page.locator("nav[aria-label='Главная навигация']").getByText(roomName),
  ).toBeVisible();
  expect(roomId).toMatch(/^[0-9a-f-]{36}$/);
});
