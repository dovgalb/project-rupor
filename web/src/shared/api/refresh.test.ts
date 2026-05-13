// Тесты single-flight refresh.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { _resetRefreshForTests, refreshAccessOnce } from "@/shared/api/refresh";
import { tokenStorage } from "@/shared/api/token-storage";

import type { TokensResponse } from "@/shared/api/errors";

const tokensFixture: TokensResponse = {
  accessToken: "new-access",
  refreshToken: "new-refresh",
  accessExpiresAt: "2030-01-01T00:00:00Z",
  refreshExpiresAt: "2030-01-08T00:00:00Z",
};

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("refreshAccessOnce", () => {
  beforeEach(() => {
    localStorage.clear();
    _resetRefreshForTests();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    localStorage.clear();
    _resetRefreshForTests();
  });

  it("без refresh-токена в storage возвращает AUTH-007 без вызова fetch", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    const result = await refreshAccessOnce();

    expect(fetchMock).not.toHaveBeenCalled();
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.status).toBe(401);
      expect(result.error.code).toBe("AUTH-007");
    }
  });

  it("первый вызов делает POST /api/v1/auth/refresh и сохраняет токены", async () => {
    tokenStorage.set({
      access: "old-access",
      refresh: "old-refresh",
      accessExpiresAt: "2020-01-01T00:00:00Z",
      refreshExpiresAt: "2020-01-08T00:00:00Z",
    });

    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(tokensFixture, 200));
    vi.stubGlobal("fetch", fetchMock);

    const result = await refreshAccessOnce();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/auth/refresh");
    expect(init).toMatchObject({ method: "POST" });
    expect(JSON.parse((init as RequestInit).body as string)).toEqual({
      refreshToken: "old-refresh",
    });

    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data).toEqual(tokensFixture);
    }
    expect(tokenStorage.getAccess()).toBe("new-access");
    expect(tokenStorage.getRefresh()).toBe("new-refresh");
  });

  it("параллельные вызовы используют один и тот же fetch (single-flight)", async () => {
    tokenStorage.set({
      access: "old-access",
      refresh: "old-refresh",
      accessExpiresAt: "2020-01-01T00:00:00Z",
      refreshExpiresAt: "2020-01-08T00:00:00Z",
    });

    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(tokensFixture, 200));
    vi.stubGlobal("fetch", fetchMock);

    const [a, b, c] = await Promise.all([
      refreshAccessOnce(),
      refreshAccessOnce(),
      refreshAccessOnce(),
    ]);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(a.ok && b.ok && c.ok).toBe(true);
  });

  it("после resolve currentRefresh сбрасывается — следующий вызов делает новый fetch", async () => {
    tokenStorage.set({
      access: "old-access",
      refresh: "old-refresh",
      accessExpiresAt: "2020-01-01T00:00:00Z",
      refreshExpiresAt: "2020-01-08T00:00:00Z",
    });

    // Каждый вызов должен вернуть свежий Response — тело одного нельзя прочитать дважды.
    const fetchMock = vi.fn(async () => jsonResponse(tokensFixture, 200));
    vi.stubGlobal("fetch", fetchMock);

    await refreshAccessOnce();
    await refreshAccessOnce();

    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
