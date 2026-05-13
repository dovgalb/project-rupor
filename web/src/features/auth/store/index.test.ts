// Тесты useAuthStore: основные actions + persist.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as authApi from "@/features/auth/api/http";
import { useAuthStore } from "@/features/auth/store";
import { tokenStorage } from "@/shared/api/token-storage";

import type { CurrentUser, TokensResponse } from "@/features/auth/types";
import type { ApiResult } from "@/shared/api/errors";

const tokensFixture: TokensResponse = {
  accessToken: "access-1",
  refreshToken: "refresh-1",
  accessExpiresAt: "2030-01-01T00:00:00Z",
  refreshExpiresAt: "2030-01-08T00:00:00Z",
};

const userFixture: CurrentUser = {
  id: "user-1",
  email: "user@example.com",
  username: "user1",
  createdAt: "2025-01-01T00:00:00Z",
};

function ok<T>(data: T): ApiResult<T> {
  return { ok: true, data };
}

function err(code: string, status = 400): ApiResult<never> {
  return { ok: false, status, error: { code, message: code } };
}

describe("useAuthStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      status: "idle",
      currentUser: null,
      error: null,
      lastErrorCode: null,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("login success: статус authenticated, currentUser задан, токены в storage", async () => {
    vi.spyOn(authApi, "login").mockResolvedValue(ok(tokensFixture));
    vi.spyOn(authApi, "me").mockResolvedValue(ok(userFixture));

    const result = await useAuthStore
      .getState()
      .login({ email: "user@example.com", password: "secret12" });

    expect(result).toBe(true);
    const state = useAuthStore.getState();
    expect(state.status).toBe("authenticated");
    expect(state.currentUser).toEqual(userFixture);
    expect(state.error).toBeNull();
    expect(tokenStorage.getAccess()).toBe(tokensFixture.accessToken);
    expect(tokenStorage.getRefresh()).toBe(tokensFixture.refreshToken);
  });

  it("login fail AUTH-006: status=error, lastErrorCode=AUTH-006, error содержит invalidCredentials", async () => {
    vi.spyOn(authApi, "login").mockResolvedValue(err("AUTH-006", 401));

    const result = await useAuthStore
      .getState()
      .login({ email: "user@example.com", password: "secret12" });

    expect(result).toBe(false);
    const state = useAuthStore.getState();
    expect(state.status).toBe("error");
    expect(state.lastErrorCode).toBe("AUTH-006");
    expect(state.error).toContain("Неверный");
    expect(tokenStorage.getAccess()).toBeNull();
  });

  it("register success: auto-login → authenticated", async () => {
    vi.spyOn(authApi, "register").mockResolvedValue(ok(userFixture));
    vi.spyOn(authApi, "login").mockResolvedValue(ok(tokensFixture));
    vi.spyOn(authApi, "me").mockResolvedValue(ok(userFixture));

    const result = await useAuthStore.getState().register({
      email: "user@example.com",
      username: "user1",
      password: "secret12",
    });

    expect(result).toBe(true);
    expect(useAuthStore.getState().status).toBe("authenticated");
    expect(useAuthStore.getState().currentUser).toEqual(userFixture);
  });

  it("register fail AUTH-004: status=error, lastErrorCode=AUTH-004; login НЕ вызван", async () => {
    vi.spyOn(authApi, "register").mockResolvedValue(err("AUTH-004", 409));
    const loginSpy = vi.spyOn(authApi, "login");

    const result = await useAuthStore.getState().register({
      email: "taken@example.com",
      username: "user1",
      password: "secret12",
    });

    expect(result).toBe(false);
    expect(loginSpy).not.toHaveBeenCalled();
    expect(useAuthStore.getState().status).toBe("error");
    expect(useAuthStore.getState().lastErrorCode).toBe("AUTH-004");
  });

  it("loadMe success: authenticated с currentUser", async () => {
    vi.spyOn(authApi, "me").mockResolvedValue(ok(userFixture));

    const result = await useAuthStore.getState().loadMe();

    expect(result).toBe(true);
    expect(useAuthStore.getState().status).toBe("authenticated");
    expect(useAuthStore.getState().currentUser).toEqual(userFixture);
  });

  it("loadMe fail: status=idle, currentUser=null", async () => {
    vi.spyOn(authApi, "me").mockResolvedValue(err("AUTH-010", 401));

    const result = await useAuthStore.getState().loadMe();

    expect(result).toBe(false);
    expect(useAuthStore.getState().status).toBe("idle");
    expect(useAuthStore.getState().currentUser).toBeNull();
  });

  it("logout: tokenStorage очищен, currentUser=null, status=idle", () => {
    tokenStorage.set({
      access: "a",
      refresh: "r",
      accessExpiresAt: "2030",
      refreshExpiresAt: "2030",
    });
    useAuthStore.setState({
      status: "authenticated",
      currentUser: userFixture,
      error: null,
      lastErrorCode: null,
    });

    useAuthStore.getState().logout();

    expect(tokenStorage.getAccess()).toBeNull();
    expect(useAuthStore.getState().currentUser).toBeNull();
    expect(useAuthStore.getState().status).toBe("idle");
  });

  it("clear: сброс state до initial", () => {
    useAuthStore.setState({
      status: "error",
      currentUser: userFixture,
      error: "boom",
      lastErrorCode: "AUTH-006",
    });

    useAuthStore.getState().clear();

    const state = useAuthStore.getState();
    expect(state.status).toBe("idle");
    expect(state.currentUser).toBeNull();
    expect(state.error).toBeNull();
    expect(state.lastErrorCode).toBeNull();
  });

  it("persist: после login currentUser сериализуется в localStorage rupor.auth", async () => {
    vi.spyOn(authApi, "login").mockResolvedValue(ok(tokensFixture));
    vi.spyOn(authApi, "me").mockResolvedValue(ok(userFixture));

    await useAuthStore.getState().login({ email: "user@example.com", password: "secret12" });

    const raw = localStorage.getItem("rupor.auth");
    expect(raw).not.toBeNull();
    const parsed = JSON.parse(raw ?? "{}") as { state?: { currentUser?: CurrentUser } };
    expect(parsed.state?.currentUser).toEqual(userFixture);
  });

  it("persist: rehydrate восстанавливает currentUser из localStorage", async () => {
    // Кладём предварительно «сохранённое» состояние в localStorage.
    localStorage.setItem(
      "rupor.auth",
      JSON.stringify({ state: { currentUser: userFixture }, version: 1 }),
    );

    // Принудительная rehydration: persist API доступен через useAuthStore.persist.
    await useAuthStore.persist.rehydrate();

    expect(useAuthStore.getState().currentUser).toEqual(userFixture);
  });

  it("partialize: status/error/lastErrorCode НЕ сериализуются", async () => {
    useAuthStore.setState({
      status: "authenticated",
      currentUser: userFixture,
      error: "should-not-persist",
      lastErrorCode: "AUTH-006",
    });

    // Триггерим запись в storage.
    await new Promise((r) => setTimeout(r, 0));

    const raw = localStorage.getItem("rupor.auth");
    const parsed = JSON.parse(raw ?? "{}") as { state?: Record<string, unknown> };
    expect(parsed.state?.currentUser).toEqual(userFixture);
    expect(parsed.state).not.toHaveProperty("status");
    expect(parsed.state).not.toHaveProperty("error");
    expect(parsed.state).not.toHaveProperty("lastErrorCode");
  });
});
