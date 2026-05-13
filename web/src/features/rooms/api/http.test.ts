// Тесты HTTP-обёрток домена rooms.
// Проверяем URL (с префиксом /api/v1), method, body, Content-Type.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  createRoom,
  deleteRoom,
  getRoom,
  joinByCode,
  listMembers,
  listRooms,
  regenerateInvite,
} from "@/features/rooms/api/http";
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

describe("roomsApi", () => {
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

  it("listRooms: GET /api/v1/rooms", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ items: [] }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await listRooms();

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms");
    expect((init as RequestInit).method).toBe("GET");
    expect((init as RequestInit).body).toBeUndefined();
    expect(result.ok).toBe(true);
  });

  it("createRoom: POST /api/v1/rooms с JSON body и Content-Type", async () => {
    const room = {
      id: "r1",
      ownerId: "u1",
      name: "general",
      createdAt: "2030",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(room, 201));
    vi.stubGlobal("fetch", fetchMock);

    const result = await createRoom({ name: "general" });

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms");
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBe(
      JSON.stringify({ name: "general" }),
    );
    const headers = new Headers((init as RequestInit).headers);
    expect(headers.get("Content-Type")).toBe("application/json");
    expect(result.ok).toBe(true);
  });

  it("getRoom: GET /api/v1/rooms/{id}", async () => {
    const room = {
      id: "r1",
      ownerId: "u1",
      name: "general",
      createdAt: "2030",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(room));
    vi.stubGlobal("fetch", fetchMock);

    await getRoom("r1");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/r1");
    expect((init as RequestInit).method).toBe("GET");
  });

  it("deleteRoom: DELETE /api/v1/rooms/{id}", async () => {
    const fetchMock = vi.fn().mockResolvedValue(emptyResponse(204));
    vi.stubGlobal("fetch", fetchMock);

    const result = await deleteRoom("r1");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/r1");
    expect((init as RequestInit).method).toBe("DELETE");
    expect((init as RequestInit).body).toBeUndefined();
    expect(result.ok).toBe(true);
  });

  it("listMembers: GET /api/v1/rooms/{id}/members", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ items: [] }));
    vi.stubGlobal("fetch", fetchMock);

    await listMembers("r1");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/r1/members");
    expect((init as RequestInit).method).toBe("GET");
  });

  it("regenerateInvite: POST /api/v1/rooms/{id}/invite без body", async () => {
    const invite = {
      code: "ABCD1234",
      createdBy: "u1",
      createdAt: "2030",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(invite));
    vi.stubGlobal("fetch", fetchMock);

    const result = await regenerateInvite("r1");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/r1/invite");
    expect((init as RequestInit).method).toBe("POST");
    expect((init as RequestInit).body).toBeUndefined();
    expect(result.ok).toBe(true);
  });

  it("joinByCode: POST /api/v1/rooms/join/{code}", async () => {
    const room = {
      id: "r1",
      ownerId: "u1",
      name: "general",
      createdAt: "2030",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(room));
    vi.stubGlobal("fetch", fetchMock);

    await joinByCode("ABCD1234");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/join/ABCD1234");
    expect((init as RequestInit).method).toBe("POST");
  });

  it("roomId с символами URL-кодируется (encodeURIComponent)", async () => {
    const fetchMock = vi.fn().mockResolvedValue(emptyResponse(204));
    vi.stubGlobal("fetch", fetchMock);

    await deleteRoom("a/b");

    const [url] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("/api/v1/rooms/a%2Fb");
  });
});
