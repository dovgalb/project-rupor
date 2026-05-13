// Тесты HTTP-обёрток домена chat.
// Проверяем URL (с префиксом /api/v1), method, query-string.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { listMessages } from "@/features/chat/api/http";
import { _resetRefreshForTests } from "@/shared/api/refresh";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("chatApi.listMessages", () => {
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

  it("без параметров: URL без query-string", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ items: [], nextBefore: null }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await listMessages("c1");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/channels/c1/messages");
    expect((init as RequestInit).method).toBe("GET");
    expect(result.ok).toBe(true);
  });

  it("с before: URL содержит ?before=<uuid>", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ items: [], nextBefore: null }));
    vi.stubGlobal("fetch", fetchMock);

    await listMessages("c1", { before: "00000000-0000-0000-0000-000000000001" });

    const [url] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe(
      "/api/v1/channels/c1/messages?before=00000000-0000-0000-0000-000000000001",
    );
  });

  it("с limit: URL содержит ?limit=N", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ items: [], nextBefore: null }));
    vi.stubGlobal("fetch", fetchMock);

    await listMessages("c1", { limit: 50 });

    const [url] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/channels/c1/messages?limit=50");
  });

  it("с before и limit: оба параметра в URL", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ items: [], nextBefore: null }));
    vi.stubGlobal("fetch", fetchMock);

    await listMessages("c1", { before: "abc", limit: 25 });

    const [url] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/channels/c1/messages?before=abc&limit=25");
  });

  it("channelId с спецсимволами URL-кодируется", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ items: [], nextBefore: null }));
    vi.stubGlobal("fetch", fetchMock);

    await listMessages("a/b c");

    const [url] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/channels/a%2Fb%20c/messages");
  });
});
