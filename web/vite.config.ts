import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "node:path";

// Конфиг Vite: alias @ → ./src, dev-прокси /api/v1 на бэк :8080 (REST и WS).
export default defineConfig(({ mode }) => ({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    host: true,
    proxy: {
      "/api/v1": {
        target: "http://localhost:8080",
        changeOrigin: true,
        ws: true,
      },
    },
  },
  build: {
    sourcemap: mode !== "production",
  },
  css: {
    modules: {
      localsConvention: "camelCaseOnly",
    },
  },
}));
