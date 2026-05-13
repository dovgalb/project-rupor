/* eslint-env node */
module.exports = {
  root: true,
  parser: "@typescript-eslint/parser",
  parserOptions: {
    ecmaVersion: 2022,
    sourceType: "module",
    ecmaFeatures: { jsx: true },
    project: "./tsconfig.json",
    tsconfigRootDir: __dirname,
  },
  env: {
    browser: true,
    es2022: true,
    node: true,
  },
  plugins: [
    "@typescript-eslint",
    "react",
    "react-hooks",
    "jsx-a11y",
    "import",
  ],
  extends: [
    "eslint:recommended",
    "plugin:@typescript-eslint/recommended",
    "plugin:react/recommended",
    "plugin:react/jsx-runtime",
    "plugin:react-hooks/recommended",
    "plugin:jsx-a11y/recommended",
    "plugin:import/recommended",
    "plugin:import/typescript",
  ],
  rules: {
    "@typescript-eslint/no-explicit-any": "error",
    "@typescript-eslint/consistent-type-imports": ["error", { prefer: "type-imports" }],
    "no-console": ["warn", { allow: ["warn", "error"] }],
    "react/no-danger": "error",
    "react/prop-types": "off",
    "import/no-default-export": "error",
    "import/order": ["warn", {
      "groups": ["builtin", "external", "internal", "parent", "sibling", "index", "type"],
      "newlines-between": "always",
    }],
  },
  settings: {
    react: { version: "detect" },
    "import/resolver": {
      typescript: { project: "./tsconfig.json" },
      node: true,
    },
  },
  overrides: [
    {
      // Vite/Playwright/Vitest config'и используют default-export по соглашению.
      files: ["vite.config.ts", "vitest.config.ts", "playwright.config.ts"],
      rules: {
        "import/no-default-export": "off",
      },
    },
    {
      // index.html и тесты-сетап без strict project linkage.
      files: ["src/test-setup.ts"],
      rules: {
        "import/no-default-export": "off",
      },
    },
  ],
  ignorePatterns: ["dist", "node_modules", "coverage", "playwright-report", "test-results"],
};
