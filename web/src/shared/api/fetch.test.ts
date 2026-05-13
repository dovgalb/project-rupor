// Тесты apiFetch: header, парсинг, auto-refresh, logout-flow.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { apiFetch } from "@/shared/api/fetch";
import { _resetRefreshForTests } from "@/shared/api/refresh";
import { tokenStorage } from "@/shared/api/token-storage";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function emptyResponse(status: number): Response {
  // У null-body статусов (204/205/304) Response не разрешает тело.
  return new Response(null, { status });
}

function errorResponse(code: string, status: number, message = "err"): Response {
  return jsonResponse({ error: { code, message } }, status);
}

// Достаём заголовки из аргументов fetch-мока.
function headersOfCall(args: unknown[]): Headers {
  const init = args[1] as RequestInit | undefined;
  return new Headers(init?.headers);
}

const refreshOk = {
  accessToken: "new-access",
  refreshToken: "new-refresh",
  accessExpiresAt: "2030-01-01T00:00:00Z",
  refreshExpiresAt: "2030-01-08T00:00:00Z",
};

describe("apiFetch", () => {
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

  it("подставляет Authorization: Bearer из tokenStorage", async () => {
    tokenStorage.set({
      access: "acc",
      refresh: "ref",
      accessExpiresAt: "2030",
      refreshExpiresAt: "2030",
    });
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: "u1" }));
    vi.stubGlobal("fetch", fetchMock);

    await apiFetch("/auth/me");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const headers = headersOfCall(fetchMock.mock.calls[0] ?? []);
    expect(headers.get("Authorization")).toBe("Bearer acc");
  });

  it("без токена не подставляет Authorization-header", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: "u1" }));
    vi.stubGlobal("fetch", fetchMock);

    await apiFetch("/auth/me");

    const headers = headersOfCall(fetchMock.mock.calls[0] ?? []);
    expect(headers.has("Authorization")).toBe(false);
  });

  it("подставляет префикс /api/v1 для относительных путей", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({}));
    vi.stubGlobal("fetch", fetchMock);

    await apiFetch("/auth/me");

    const [url] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/auth/me");
  });

  it("при body выставляет Content-Type: application/json, если не задан", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({}, 201));
    vi.stubGlobal("fetch", fetchMock);

    await apiFetch("/rooms", {
      method: "POST",
      body: JSON.stringify({ name: "r1" }),
    });

    const headers = headersOfCall(fetchMock.mock.calls[0] ?? []);
    expect(headers.get("Content-Type")).toBe("application/json");
  });

  it("204 No Content → { ok: true, data: null }", async () => {
    const fetchMock = vi.fn().mockResolvedValue(emptyResponse(204));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch<null>("/rooms/abc");

    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data).toBeNull();
    }
  });

  it("200 с JSON → { ok: true, data }", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: "u1", email: "e@x" }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch<{ id: string; email: string }>("/auth/me");

    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data).toEqual({ id: "u1", email: "e@x" });
    }
  });

  it("4xx с ErrorEnvelope → { ok: false, status, error }", async () => {
    const fetchMock = vi.fn().mockResolvedValue(errorResponse("ROOM-001", 400, "invalid name"));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch("/rooms", { method: "POST", body: "{}" });

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.status).toBe(400);
      expect(result.error).toEqual({ code: "ROOM-001", message: "invalid name" });
    }
  });

  it("network error (fetch throws) → { ok: false, status: 0, error.code: NETWORK }", async () => {
    const fetchMock = vi.fn().mockRejectedValue(new TypeError("net"));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch("/auth/me");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.status).toBe(0);
      expect(result.error.code).toBe("NETWORK");
    }
  });

  it("invalid JSON в теле → { ok: false, error.code: CLIENT }", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("not-a-json", { status: 500 }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch("/auth/me");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("CLIENT");
    }
  });

  it("401 AUTH-011 → refresh → retry с новым access → ok", async () => {
    tokenStorage.set({
      access: "old-acc",
      refresh: "old-ref",
      accessExpiresAt: "2020",
      refreshExpiresAt: "2030",
    });

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(errorResponse("AUTH-011", 401))
      .mockResolvedValueOnce(jsonResponse(refreshOk))
      .mockResolvedValueOnce(jsonResponse({ items: [] }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch<{ items: unknown[] }>("/rooms");

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1]?.[0]).toBe("/api/v1/auth/refresh");

    // Retry должен идти с обновлённым access.
    const retryHeaders = headersOfCall(fetchMock.mock.calls[2] ?? []);
    expect(retryHeaders.get("Authorization")).toBe("Bearer new-access");

    expect(result.ok).toBe(true);
  });

  it("401 AUTH-011 → refresh fail → logoutFlow + возвращает исходную 401", async () => {
    tokenStorage.set({
      access: "old-acc",
      refresh: "old-ref",
      accessExpiresAt: "2020",
      refreshExpiresAt: "2030",
    });

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(errorResponse("AUTH-011", 401))
      .mockResolvedValueOnce(errorResponse("AUTH-008", 401));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch("/rooms");

    // Был оригинальный запрос + refresh (fail). Retry не делается.
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error.code).toBe("AUTH-011");
    }
    // logoutFlow очищает tokenStorage.
    expect(tokenStorage.getAccess()).toBeNull();
    expect(tokenStorage.getRefresh()).toBeNull();
  });

  it("401 AUTH-011 → refresh ok → retry снова AUTH-011 → logoutFlow + возвращает retry", async () => {
    tokenStorage.set({
      access: "old-acc",
      refresh: "old-ref",
      accessExpiresAt: "2020",
      refreshExpiresAt: "2030",
    });

    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(errorResponse("AUTH-011", 401))
      .mockResolvedValueOnce(jsonResponse(refreshOk))
      .mockResolvedValueOnce(errorResponse("AUTH-011", 401));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch("/rooms");

    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(result.ok).toBe(false);
    expect(tokenStorage.getAccess()).toBeNull();
  });

  it("401 AUTH-007 → logoutFlow без refresh", async () => {
    tokenStorage.set({
      access: "acc",
      refresh: "ref",
      accessExpiresAt: "2030",
      refreshExpiresAt: "2030",
    });
    const fetchMock = vi.fn().mockResolvedValueOnce(errorResponse("AUTH-007", 401));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch("/rooms");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(result.ok).toBe(false);
    expect(tokenStorage.getAccess()).toBeNull();
  });

  it("401 AUTH-010 → logoutFlow без refresh", async () => {
    tokenStorage.set({
      access: "acc",
      refresh: "ref",
      accessExpiresAt: "2030",
      refreshExpiresAt: "2030",
    });
    const fetchMock = vi.fn().mockResolvedValueOnce(errorResponse("AUTH-010", 401));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch("/rooms");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(result.ok).toBe(false);
    expect(tokenStorage.getAccess()).toBeNull();
  });

  it("401 на /auth/refresh не запускает рекурсивный refresh; AUTH-008 → logoutFlow", async () => {
    tokenStorage.set({
      access: "acc",
      refresh: "ref",
      accessExpiresAt: "2030",
      refreshExpiresAt: "2030",
    });
    const fetchMock = vi.fn().mockResolvedValueOnce(errorResponse("AUTH-008", 401));
    vi.stubGlobal("fetch", fetchMock);

    const result = await apiFetch("/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refreshToken: "ref" }),
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(result.ok).toBe(false);
    expect(tokenStorage.getAccess()).toBeNull();
  });

  it("single-flight: два параллельных AUTH-011 → один refresh + два retry", async () => {
    tokenStorage.set({
      access: "old-acc",
      refresh: "old-ref",
      accessExpiresAt: "2020",
      refreshExpiresAt: "2030",
    });

    const fetchMock = vi.fn().mockImplementation(async (...args: unknown[]) => {
      const url = args[0] as string;
      const init = args[1] as RequestInit | undefined;
      if (url === "/api/v1/auth/refresh") {
        return jsonResponse(refreshOk);
      }
      const headers = new Headers(init?.headers);
      const auth = headers.get("Authorization");
      if (auth === "Bearer old-acc") {
        return errorResponse("AUTH-011", 401);
      }
      return jsonResponse({ url });
    });
    vi.stubGlobal("fetch", fetchMock);

    const [a, b] = await Promise.all([
      apiFetch<{ url: string }>("/rooms"),
      apiFetch<{ url: string }>("/channels"),
    ]);

    // 2 оригинальных + 1 refresh (single-flight) + 2 retry = 5 вызовов.
    expect(fetchMock).toHaveBeenCalledTimes(5);
    const refreshCalls = fetchMock.mock.calls.filter(
      (c) => c[0] === "/api/v1/auth/refresh",
    );
    expect(refreshCalls).toHaveLength(1);
    expect(a.ok && b.ok).toBe(true);
  });
});
