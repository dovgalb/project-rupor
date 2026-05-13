import path from "node:path";

import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  test: {
    environment: "jsdom",
    // jsdom без url создаёт opaque origin — localStorage в нём недоступен.
    environmentOptions: {
      jsdom: { url: "http://localhost/" },
    },
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
    css: true,
    // Unit-тесты живут только в src/. e2e/ — это Playwright (см. playwright.config.ts).
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    exclude: ["node_modules", "dist", "e2e"],
  },
});
