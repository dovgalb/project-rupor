// DTO и ViewModel для домена auth.
// Источник: docs/3_5_frontend/06-api-integration.md (REST: Auth).
// Без runtime-кода: только TS-типы.

// === REST DTO (camelCase, из бэка) ===

export type RegisterRequest = {
  email: string;
  username: string;
  password: string;
};

export type LoginRequest = {
  email: string;
  password: string;
};

export type RefreshRequest = {
  refreshToken: string;
};

export type UserResponse = {
  id: string;
  email: string;
  username: string;
  createdAt: string;
};

export type TokensResponse = {
  accessToken: string;
  refreshToken: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
};

// === ViewModel ===

// Текущий пользователь — DTO без переименований (см. D-11).
export type CurrentUser = UserResponse;
