// Тесты HTTP-обёрток домена channels.
// Проверяем URL (с префиксом /api/v1), method, body, Content-Type.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  createChannel,
  deleteChannel,
  listChannels,
} from "@/features/channels/api/http";
import { _resetRefreshForTests } from "@/shared/api/refresh";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function emptyResponse(status = 204): Response {
  return new Response(null, { status });
}

describe("channelsApi", () => {
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

  it("listChannels: GET /api/v1/rooms/{roomId}/channels", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ items: [] }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await listChannels("r1");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/r1/channels");
    expect((init as RequestInit).method).toBe("GET");
    expect((init as RequestInit).body).toBeUndefined();
    expect(result.ok).toBe(true);
  });

  it("createChannel: POST /api/v1/rooms/{roomId}/channels с JSON body и Content-Type", async () => {
    const channel = {
      id: "c1",
      roomId: "r1",
      name: "general",
      kind: "text",
      createdAt: "2030",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(channel, 201));
    vi.stubGlobal("fetch", fetchMock);

    const result = await createChannel("r1", {
      name: "general",
      kind: "text",
    });

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/r1/channels");
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ name: "general", kind: "text" }),
    );
    const headers = new Headers((init as RequestInit).headers);
    expect(headers.get("Content-Type")).toBe("application/json");
    expect(result.ok).toBe(true);
  });

  it("deleteChannel: DELETE /api/v1/rooms/{roomId}/channels/{channelId}", async () => {
    const fetchMock = vi.fn().mockResolvedValue(emptyResponse(204));
    vi.stubGlobal("fetch", fetchMock);

    const result = await deleteChannel("r1", "c1");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/r1/channels/c1");
    expect((init as RequestInit).method).toBe("DELETE");
    expect((init as RequestInit).body).toBeUndefined();
    expect(result.ok).toBe(true);
  });

  it("roomId и channelId с символами URL-кодируются (encodeURIComponent)", async () => {
    const fetchMock = vi.fn().mockResolvedValue(emptyResponse(204));
    vi.stubGlobal("fetch", fetchMock);

    await deleteChannel("a/b", "c d");

    const [url] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/a%2Fb/channels/c%20d");
  });
});
