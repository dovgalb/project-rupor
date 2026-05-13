// Тесты useRoomsStore: основные actions + интеграция с logoutFlow.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as roomsApi from "@/features/rooms/api/http";
import { useRoomsStore } from "@/features/rooms/store";
import { logoutFlow } from "@/shared/lib/logoutFlow";
import { useToastsStore } from "@/shared/ui/useToast";

import type {
  InviteCodeDto,
  MemberDto,
  RoomDto,
  RoomWithRoleDto,
} from "@/features/rooms/types";
import type { ApiResult } from "@/shared/api/errors";

const roomFixture: RoomDto = {
  id: "room-1",
  ownerId: "user-1",
  name: "General",
  createdAt: "2025-01-01T00:00:00Z",
};

const roomWithRoleFixture: RoomWithRoleDto = {
  ...roomFixture,
  role: "member",
};

const memberFixture: MemberDto = {
  userId: "user-2",
  role: "member",
  joinedAt: "2025-01-02T00:00:00Z",
};

const inviteFixture: InviteCodeDto = {
  code: "ABCD1234",
  createdBy: "user-1",
  createdAt: "2025-01-03T00:00:00Z",
};

function ok<T>(data: T): ApiResult<T> {
  return { ok: true, data };
}

function err(code: string, status = 400): ApiResult<never> {
  return { ok: false, status, error: { code, message: code } };
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

function resetToasts(): void {
  useToastsStore.setState({ items: [] });
}

describe("useRoomsStore", () => {
  beforeEach(() => {
    resetStore();
    resetToasts();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetStore();
    resetToasts();
  });

  it("loadRooms success → status=ready, rooms заполнены", async () => {
    vi.spyOn(roomsApi, "listRooms").mockResolvedValue(
      ok({ items: [roomWithRoleFixture] }),
    );

    await useRoomsStore.getState().loadRooms();

    const state = useRoomsStore.getState();
    expect(state.status).toBe("ready");
    expect(state.rooms).toEqual([roomWithRoleFixture]);
    expect(state.error).toBeNull();
  });

  it("loadRooms fail → status=error, error содержит сообщение", async () => {
    vi.spyOn(roomsApi, "listRooms").mockResolvedValue(err("ROOM-001", 400));

    await useRoomsStore.getState().loadRooms();

    const state = useRoomsStore.getState();
    expect(state.status).toBe("error");
    expect(state.error).not.toBeNull();
  });

  it("createRoom success → новая комната с role=owner добавлена в список", async () => {
    vi.spyOn(roomsApi, "createRoom").mockResolvedValue(ok(roomFixture));

    const result = await useRoomsStore.getState().createRoom("General");

    expect(result).not.toBeNull();
    expect(result?.role).toBe("owner");
    expect(useRoomsStore.getState().rooms).toEqual([
      { ...roomFixture, role: "owner" },
    ]);
  });

  it("createRoom fail → toast.error, null, список не изменился", async () => {
    vi.spyOn(roomsApi, "createRoom").mockResolvedValue(err("ROOM-001", 400));

    const result = await useRoomsStore.getState().createRoom("");

    expect(result).toBeNull();
    expect(useRoomsStore.getState().rooms).toEqual([]);
    expect(useToastsStore.getState().items.length).toBe(1);
    expect(useToastsStore.getState().items[0]?.variant).toBe("error");
  });

  it("deleteRoom success → удалена из rooms, membersByRoom очищен, activeInviteByRoom очищен", async () => {
    useRoomsStore.setState({
      rooms: [roomWithRoleFixture],
      status: "ready",
      error: null,
      membersByRoom: { [roomFixture.id]: [memberFixture] },
      membersLoadingByRoom: {},
      activeInviteByRoom: { [roomFixture.id]: inviteFixture },
    });
    vi.spyOn(roomsApi, "deleteRoom").mockResolvedValue(ok(null));

    const result = await useRoomsStore.getState().deleteRoom(roomFixture.id);

    expect(result).toBe(true);
    const state = useRoomsStore.getState();
    expect(state.rooms).toEqual([]);
    expect(state.membersByRoom[roomFixture.id]).toBeUndefined();
    expect(state.activeInviteByRoom[roomFixture.id]).toBeUndefined();
  });

  it("deleteRoom fail (ROOM-005) → toast.error, false, list не изменился", async () => {
    useRoomsStore.setState({
      rooms: [roomWithRoleFixture],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });
    vi.spyOn(roomsApi, "deleteRoom").mockResolvedValue(err("ROOM-005", 403));

    const result = await useRoomsStore.getState().deleteRoom(roomFixture.id);

    expect(result).toBe(false);
    expect(useRoomsStore.getState().rooms).toEqual([roomWithRoleFixture]);
    expect(useToastsStore.getState().items.length).toBe(1);
  });

  it("loadMembers success → membersByRoom[roomId] заполнен, loading сброшен", async () => {
    vi.spyOn(roomsApi, "listMembers").mockResolvedValue(
      ok({ items: [memberFixture] }),
    );

    await useRoomsStore.getState().loadMembers(roomFixture.id);

    const state = useRoomsStore.getState();
    expect(state.membersByRoom[roomFixture.id]).toEqual([memberFixture]);
    expect(state.membersLoadingByRoom[roomFixture.id]).toBe(false);
  });

  it("regenerateInvite success → activeInviteByRoom[roomId] заполнен", async () => {
    vi.spyOn(roomsApi, "regenerateInvite").mockResolvedValue(ok(inviteFixture));

    const result = await useRoomsStore
      .getState()
      .regenerateInvite(roomFixture.id);

    expect(result).toEqual(inviteFixture);
    expect(useRoomsStore.getState().activeInviteByRoom[roomFixture.id]).toEqual(
      inviteFixture,
    );
  });

  it("joinByCode success → новая комната с role=member добавлена в список", async () => {
    vi.spyOn(roomsApi, "joinByCode").mockResolvedValue(ok(roomFixture));

    const result = await useRoomsStore.getState().joinByCode("ABCD1234");

    expect(result.ok).toBe(true);
    const state = useRoomsStore.getState();
    expect(state.rooms.length).toBe(1);
    expect(state.rooms[0]?.role).toBe("member");
  });

  it("joinByCode на уже существующую комнату → не дублирует", async () => {
    useRoomsStore.setState({
      rooms: [roomWithRoleFixture],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });
    vi.spyOn(roomsApi, "joinByCode").mockResolvedValue(ok(roomFixture));

    await useRoomsStore.getState().joinByCode("ABCD1234");

    expect(useRoomsStore.getState().rooms).toEqual([roomWithRoleFixture]);
  });

  it("joinByCode fail → возвращает { ok: false, code } без toast", async () => {
    vi.spyOn(roomsApi, "joinByCode").mockResolvedValue(err("ROOM-007", 404));

    const result = await useRoomsStore.getState().joinByCode("BADCODE0");

    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("ROOM-007");
    }
    expect(useToastsStore.getState().items.length).toBe(0);
  });

  it("clear → сброс ко initial", () => {
    useRoomsStore.setState({
      rooms: [roomWithRoleFixture],
      status: "ready",
      error: "boom",
      membersByRoom: { [roomFixture.id]: [memberFixture] },
      membersLoadingByRoom: { [roomFixture.id]: false },
      activeInviteByRoom: { [roomFixture.id]: inviteFixture },
    });

    useRoomsStore.getState().clear();

    const state = useRoomsStore.getState();
    expect(state.rooms).toEqual([]);
    expect(state.status).toBe("idle");
    expect(state.error).toBeNull();
    expect(state.membersByRoom).toEqual({});
    expect(state.activeInviteByRoom).toEqual({});
  });

  it("logoutFlow интеграция → очищает useRoomsStore", () => {
    useRoomsStore.setState({
      rooms: [roomWithRoleFixture],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    logoutFlow();

    expect(useRoomsStore.getState().rooms).toEqual([]);
    expect(useRoomsStore.getState().status).toBe("idle");
  });
});
