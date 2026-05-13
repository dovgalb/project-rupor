// Тесты HTTP-обёрток домена auth.
// Проверяем URL (с префиксом /api/v1), method, body, Content-Type.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { login, me, refresh, register } from "@/features/auth/api/http";
import { _resetRefreshForTests } from "@/shared/api/refresh";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function headersOfCall(args: unknown[]): Headers {
  const init = args[1] as RequestInit | undefined;
  return new Headers(init?.headers);
}

describe("authApi", () => {
  beforeEach(() => {
    localStorage.clear();
    _resetRefreshForTests();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    localStorage.clear();
    _resetRefreshForTests();
  });

  it("register: POST /api/v1/auth/register с JSON body и Content-Type", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ id: "u1", email: "a@b", username: "u", createdAt: "2030" }, 201));
    vi.stubGlobal("fetch", fetchMock);

    const result = await register({ email: "a@b", username: "u", password: "secret12" });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/auth/register");
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(JSON.stringify({ email: "a@b", username: "u", password: "secret12" }));
    const headers = headersOfCall(fetchMock.mock.calls[0] ?? []);
    expect(headers.get("Content-Type")).toBe("application/json");
    expect(result.ok).toBe(true);
  });

  it("login: POST /api/v1/auth/login с JSON body", async () => {
    const tokens = {
      accessToken: "a",
      refreshToken: "r",
      accessExpiresAt: "2030",
      refreshExpiresAt: "2030",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(tokens));
    vi.stubGlobal("fetch", fetchMock);

    const result = await login({ email: "a@b", password: "secret12" });

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/auth/login");
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(JSON.stringify({ email: "a@b", password: "secret12" }));
    expect(result.ok).toBe(true);
  });

  it("refresh: POST /api/v1/auth/refresh с refreshToken в body", async () => {
    const tokens = {
      accessToken: "a",
      refreshToken: "r2",
      accessExpiresAt: "2030",
      refreshExpiresAt: "2030",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(tokens));
    vi.stubGlobal("fetch", fetchMock);

    await refresh({ refreshToken: "old" });

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/auth/refresh");
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(JSON.stringify({ refreshToken: "old" }));
  });

  it("me: GET /api/v1/auth/me, без body", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ id: "u1", email: "a@b", username: "u", createdAt: "2030" }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await me();

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/auth/me");
    expect((init as RequestInit).method).toBe("GET");
    expect((init as RequestInit).body).toBeUndefined();
    expect(result.ok).toBe(true);
  });
});
