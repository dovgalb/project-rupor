// Phase 11: конфиг Playwright для e2e-сценариев.
// Источник: docs/3_5_frontend/04-testing.md секция «E2E (Playwright)» + plan/phase-11.md.
//
// Зависимости: должен быть запущен бэк (`make run` на :8080) и фронт (`npm run dev` на :5173),
// либо docker compose web (на :3000). E2E_BASE_URL переопределяет адрес.
// В CI секция webServer автоматически поднимает Vite.

import { defineConfig, devices } from "@playwright/test";

const isCI = process.env["CI"] !== undefined && process.env["CI"] !== "";

export default defineConfig({
  testDir: "./e2e",
  // Параллельность — для разных тестовых файлов; внутри файла — последовательно.
  fullyParallel: true,
  forbidOnly: isCI,
  retries: isCI ? 2 : 0,
  // В локалке оставляем дефолт (Playwright сам подберёт по cpu); в CI ограничиваем 1.
  ...(isCI ? { workers: 1 } : {}),
  reporter: isCI ? "github" : "html",
  use: {
    baseURL: process.env["E2E_BASE_URL"] ?? "http://localhost:5173",
    trace: "on-first-retry",
    video: "retain-on-failure",
    // Безопасный таймаут для UI-операций; реалтайм-чат может «думать» несколько секунд.
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
  },
  expect: {
    timeout: 7_000,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  // В локальной разработке полагаемся на уже запущенный `npm run dev`.
  // В CI Playwright сам поднимет Vite (см. ниже секцию webServer).
  ...(isCI
    ? {
        webServer: {
          command: "npm run dev",
          url: "http://localhost:5173",
          reuseExistingServer: false,
          timeout: 60_000,
        },
      }
    : {}),
});
