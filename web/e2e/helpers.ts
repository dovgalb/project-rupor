// Phase 11: общие e2e-хелперы (registerUser, loginAs, createRoom, createTextChannel).
// Селекторы выбраны под фактические aria-labels компонентов (см. Sidebar/RoomList/ChannelList/RegisterForm).
//
// Контракт:
// - Все имена пользователей и emails — уникальные, чтобы тесты были изолированы (нет cleanup БД).
// - Никаких waitForTimeout: только expect().toBeVisible() / toHaveURL().

import { expect, type Page } from "@playwright/test";

export const TEST_PASSWORD = "super-secret-123";

export function uniqueEmail(prefix = "u"): string {
  const t = Date.now();
  const r = Math.random().toString(36).slice(2, 8);
  return `${prefix}-${t.toString()}-${r}@e2e.test`;
}

// Username: 3..32 latin alnum/_/-. Не используем дефис в начале/конце.
export function uniqueUsername(prefix = "u"): string {
  const r = Math.random().toString(36).slice(2, 8);
  // 3..32 — гарантированно влезает.
  return `${prefix}_${(Date.now() % 1_000_000).toString()}_${r}`;
}

export type RegisteredUser = { email: string; username: string };

// Зарегистрировать нового пользователя. После регистрации фронт сам делает
// auto-login и редиректит на /rooms.
export async function registerUser(
  page: Page,
  overrides: Partial<RegisteredUser> = {},
): Promise<RegisteredUser> {
  const email = overrides.email ?? uniqueEmail();
  const username = overrides.username ?? uniqueUsername();
  await page.goto("/register");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Имя пользователя").fill(username);
  await page.getByLabel("Пароль").fill(TEST_PASSWORD);
  await page.getByRole("button", { name: "Создать аккаунт" }).click();
  await expect(page).toHaveURL(/\/rooms(?:\/|$)/);
  return { email, username };
}

// Логин уже зарегистрированного пользователя.
export async function loginAs(
  page: Page,
  email: string,
  password = TEST_PASSWORD,
): Promise<void> {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Пароль").fill(password);
  // Кнопка submit называется "Войти". В sidebar есть кнопка "Войти по коду" —
  // её не будет на /login, так что exact-match не обязателен.
  await page.getByRole("button", { name: "Войти", exact: true }).click();
  await expect(page).toHaveURL(/\/rooms(?:\/|$)/);
}

// Logout: кнопка "Выйти" в TopBar.
export async function logout(page: Page): Promise<void> {
  await page.getByRole("button", { name: "Выйти" }).click();
  await expect(page).toHaveURL(/\/login/);
}

// Создать комнату через CreateRoomModal. Возвращает roomId из URL после редиректа.
export async function createRoom(page: Page, name: string): Promise<string> {
  // В RoomList есть кнопка "+" с aria-label "Создать комнату".
  await page.getByRole("button", { name: "Создать комнату" }).first().click();
  await page.getByLabel("Название комнаты").fill(name);
  // Внутри модалки кнопка submit — "Создать" (диалог + одна кнопка).
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Создать" })
    .click();
  await expect(page).toHaveURL(/\/rooms\/[0-9a-f-]{36}/);
  const match = page.url().match(/\/rooms\/([0-9a-f-]{36})/);
  if (match === null || match[1] === undefined) {
    throw new Error("Не удалось извлечь roomId из URL после createRoom");
  }
  return match[1];
}

// Создать текстовый канал в активной комнате. Возвращает channelId.
export async function createTextChannel(
  page: Page,
  name: string,
): Promise<string> {
  await page.getByRole("button", { name: "Создать канал" }).click();
  await page.getByLabel("Название канала").fill(name);
  // Radio "Текстовый" — выбран по умолчанию (DEFAULT_VALUES.kind = "text"),
  // но сделаем явный check для надёжности.
  await page.getByLabel("Текстовый").check();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Создать" })
    .click();
  await expect(page).toHaveURL(/\/rooms\/[0-9a-f-]{36}\/channels\/[0-9a-f-]{36}/);
  const match = page.url().match(/\/channels\/([0-9a-f-]{36})/);
  if (match === null || match[1] === undefined) {
    throw new Error("Не удалось извлечь channelId из URL после createTextChannel");
  }
  return match[1];
}

// Отправить сообщение в активный канал через composer.
export async function sendMessage(page: Page, text: string): Promise<void> {
  const textarea = page.getByRole("textbox", { name: "Сообщение" });
  await textarea.fill(text);
  await textarea.press("Enter");
}
