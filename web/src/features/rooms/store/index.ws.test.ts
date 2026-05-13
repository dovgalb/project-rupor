// Тесты Phase-9 actions useRoomsStore: appendMember (WS member.joined),
// wsClient.reconnect() как side-effect joinByCode.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as roomsApi from "@/features/rooms/api/http";
import { useRoomsStore } from "@/features/rooms/store";
import { createFakeWsClient } from "@/shared/api/__tests__/ws.fake";

import type { MemberDto, RoomDto } from "@/features/rooms/types";
import type { FakeWsClient } from "@/shared/api/__tests__/ws.fake";
import type { ApiResult } from "@/shared/api/errors";

let fake: FakeWsClient;
vi.mock("@/shared/api/wsClient.singleton", () => ({
  get wsClient(): FakeWsClient {
    return fake;
  },
}));

const room: RoomDto = {
  id: "r1",
  ownerId: "u-owner",
  name: "Room",
  createdAt: "2030-01-01T00:00:00Z",
};

const member1: MemberDto = {
  userId: "u1",
  role: "owner",
  joinedAt: "2030-01-01T00:00:00Z",
};

function ok<T>(data: T): ApiResult<T> {
  return { ok: true, data };
}

function resetStore(): void {
  useRoomsStore.setState({
    rooms: [],
    status: "idle",
    error: null,
    membersByRoom: {},
    membersLoadingByRoom: {},
    activeInviteByRoom: {},
  });
}

beforeEach(() => {
  fake = createFakeWsClient();
  resetStore();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useRoomsStore.appendMember", () => {
  it("добавляет нового участника в существующий список", () => {
    useRoomsStore.setState({
      rooms: [],
      status: "ready",
      error: null,
      membersByRoom: { r1: [member1] },
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    useRoomsStore.getState().appendMember({
      room_id: "r1",
      user_id: "u2",
      joined_at: "2030-01-02T00:00:00Z",
    });

    const list = useRoomsStore.getState().membersByRoom["r1"] ?? [];
    expect(list.length).toBe(2);
    expect(list[1]).toEqual({
      userId: "u2",
      role: "member",
      joinedAt: "2030-01-02T00:00:00Z",
    });
  });

  it("дедуплицирует по userId", () => {
    useRoomsStore.setState({
      rooms: [],
      status: "ready",
      error: null,
      membersByRoom: { r1: [member1] },
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    useRoomsStore.getState().appendMember({
      room_id: "r1",
      user_id: "u1",
      joined_at: "2030-01-02T00:00:00Z",
    });

    expect(useRoomsStore.getState().membersByRoom["r1"]?.length).toBe(1);
  });

  it("игнорирует комнаты, чей список members не загружен", () => {
    useRoomsStore.getState().appendMember({
      room_id: "r-unknown",
      user_id: "u2",
      joined_at: "2030-01-02T00:00:00Z",
    });
    expect(useRoomsStore.getState().membersByRoom["r-unknown"]).toBeUndefined();
  });
});

describe("useRoomsStore.joinByCode", () => {
  it("после ok вызывает wsClient.reconnect() (D-10)", async () => {
    vi.spyOn(roomsApi, "joinByCode").mockResolvedValue(ok(room));

    const result = await useRoomsStore.getState().joinByCode("CODE1234");

    expect(result.ok).toBe(true);
    expect(fake.reconnect).toHaveBeenCalledTimes(1);
  });

  it("при ошибке reconnect не вызывается", async () => {
    vi.spyOn(roomsApi, "joinByCode").mockResolvedValue({
      ok: false,
      status: 404,
      error: { code: "ROOM-007", message: "not found" },
    });

    const result = await useRoomsStore.getState().joinByCode("CODE1234");

    expect(result.ok).toBe(false);
    expect(fake.reconnect).not.toHaveBeenCalled();
  });
});
