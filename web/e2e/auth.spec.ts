// Phase 11: auth-сценарии E2E-1, 2, 3, 10, 12 (см. docs/3_5_frontend/04-testing.md).

import { expect, test } from "@playwright/test";

import {
  loginAs,
  logout,
  registerUser,
  TEST_PASSWORD,
  uniqueEmail,
} from "./helpers";

test("E2E-1: register → редирект на /rooms, виден заголовок Мои комнаты", async ({
  page,
}) => {
  await registerUser(page);
  await expect(page).toHaveURL(/\/rooms(?:\/|$)/);
  await expect(
    page.getByRole("heading", { name: "Мои комнаты" }),
  ).toBeVisible();
});

test("E2E-2: login существующего пользователя через logout → login", async ({
  page,
}) => {
  const { email } = await registerUser(page);
  await logout(page);
  await loginAs(page, email);
  await expect(
    page.getByRole("heading", { name: "Мои комнаты" }),
  ).toBeVisible();
});

test("E2E-3: login с неверным паролем → inline error", async ({ page }) => {
  const { email } = await registerUser(page);
  await logout(page);
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Пароль").fill("wrong-password-99");
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText(/неверный/i);
});

test("E2E-3b: login с неизвестным email → inline error", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill(uniqueEmail("nobody"));
  await page.getByLabel("Пароль").fill(TEST_PASSWORD);
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText(/неверный/i);
});

test("E2E-10: logout очищает localStorage (нет access/refresh)", async ({
  page,
}) => {
  await registerUser(page);
  // После login токены лежат в localStorage.
  const beforeLogout = await page.evaluate(() =>
    localStorage.getItem("rupor.access"),
  );
  expect(beforeLogout).not.toBeNull();
  await logout(page);
  const access = await page.evaluate(() =>
    localStorage.getItem("rupor.access"),
  );
  const refresh = await page.evaluate(() =>
    localStorage.getItem("rupor.refresh"),
  );
  expect(access).toBeNull();
  expect(refresh).toBeNull();
});

test("E2E-12: восстановление сессии после reload (persist + UC-16)", async ({
  page,
}) => {
  await registerUser(page);
  await page.reload();
  await expect(page).toHaveURL(/\/rooms(?:\/|$)/);
  await expect(
    page.getByRole("heading", { name: "Мои комнаты" }),
  ).toBeVisible();
});
