// HTTP-обёртки для домена auth.
// Каждая функция — одиночный вызов apiFetch.
// Префикс /api/v1 подставляется внутри apiFetch.
// Источник: docs/3_5_frontend/06-api-integration.md (REST: Auth).

import { apiFetch } from "@/shared/api/fetch";

import type {
  CurrentUser,
  LoginRequest,
  RefreshRequest,
  RegisterRequest,
  TokensResponse,
  UserResponse,
} from "@/features/auth/types";
import type { ApiResult } from "@/shared/api/errors";

export function register(req: RegisterRequest): Promise<ApiResult<UserResponse>> {
  return apiFetch<UserResponse>("/auth/register", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export function login(req: LoginRequest): Promise<ApiResult<TokensResponse>> {
  return apiFetch<TokensResponse>("/auth/login", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export function refresh(req: RefreshRequest): Promise<ApiResult<TokensResponse>> {
  return apiFetch<TokensResponse>("/auth/refresh", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export function me(): Promise<ApiResult<CurrentUser>> {
  return apiFetch<CurrentUser>("/auth/me", { method: "GET" });
}
