// Тесты useChannelsStore: основные actions + интеграция с logoutFlow.
// Каскад 403 (CHANNEL-006/ROOM-003) → удаление room из useRoomsStore.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as channelsApi from "@/features/channels/api/http";
import { useChannelsStore } from "@/features/channels/store";
import { useRoomsStore } from "@/features/rooms/store";
import { logoutFlow } from "@/shared/lib/logoutFlow";
import { useToastsStore } from "@/shared/ui/useToast";

import type { ChannelDto } from "@/features/channels/types";
import type { RoomWithRoleDto } from "@/features/rooms/types";
import type { ApiResult } from "@/shared/api/errors";

const channelFixture: ChannelDto = {
  id: "c1",
  roomId: "r1",
  name: "general",
  kind: "text",
  createdAt: "2030",
};

const voiceChannelFixture: ChannelDto = {
  id: "c2",
  roomId: "r1",
  name: "voice-general",
  kind: "voice",
  createdAt: "2030",
};

const roomFixture: RoomWithRoleDto = {
  id: "r1",
  ownerId: "u1",
  name: "General",
  createdAt: "2030",
  role: "member",
};

function ok<T>(data: T): ApiResult<T> {
  return { ok: true, data };
}

function err(code: string, status = 400): ApiResult<never> {
  return { ok: false, status, error: { code, message: code } };
}

function resetChannelsStore(): void {
  useChannelsStore.setState({
    channelsByRoom: {},
    loadingByRoom: {},
    errorByRoom: {},
    activeChannelId: null,
  });
}

function resetRoomsStore(): void {
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

describe("useChannelsStore", () => {
  beforeEach(() => {
    resetChannelsStore();
    resetRoomsStore();
    resetToasts();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetChannelsStore();
    resetRoomsStore();
    resetToasts();
  });

  it("loadChannels success → channelsByRoom[roomId] заполнен, loading=false, error=null", async () => {
    vi.spyOn(channelsApi, "listChannels").mockResolvedValue(
      ok({ items: [channelFixture, voiceChannelFixture] }),
    );

    await useChannelsStore.getState().loadChannels("r1");

    const state = useChannelsStore.getState();
    expect(state.channelsByRoom["r1"]).toEqual([
      channelFixture,
      voiceChannelFixture,
    ]);
    expect(state.loadingByRoom["r1"]).toBe(false);
    expect(state.errorByRoom["r1"]).toBeNull();
  });

  it("loadChannels fail CHANNEL-006 → каскадное удаление room из useRoomsStore", async () => {
    useRoomsStore.setState({
      rooms: [roomFixture],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });
    vi.spyOn(channelsApi, "listChannels").mockResolvedValue(
      err("CHANNEL-006", 403),
    );

    await useChannelsStore.getState().loadChannels("r1");

    expect(useRoomsStore.getState().rooms).toEqual([]);
    const state = useChannelsStore.getState();
    expect(state.loadingByRoom["r1"]).toBe(false);
    expect(state.errorByRoom["r1"]).not.toBeNull();
  });

  it("loadChannels идемпотентный: второй вызов во время loading не делает второй fetch", async () => {
    let resolveFirst!: (value: ApiResult<{ items: ChannelDto[] }>) => void;
    const pendingPromise = new Promise<ApiResult<{ items: ChannelDto[] }>>(
      (resolve) => {
        resolveFirst = resolve;
      },
    );
    const spy = vi
      .spyOn(channelsApi, "listChannels")
      .mockReturnValue(pendingPromise);

    // Первый вызов: ставит loading=true, ждёт ответа.
    const p1 = useChannelsStore.getState().loadChannels("r1");
    // Второй вызов во время loading — должен быть проигнорирован.
    const p2 = useChannelsStore.getState().loadChannels("r1");

    expect(spy).toHaveBeenCalledTimes(1);
    expect(useChannelsStore.getState().loadingByRoom["r1"]).toBe(true);

    // Завершаем первый, дожидаемся обоих.
    resolveFirst(ok({ items: [channelFixture] }));
    await Promise.all([p1, p2]);

    expect(spy).toHaveBeenCalledTimes(1);
    expect(useChannelsStore.getState().channelsByRoom["r1"]).toEqual([
      channelFixture,
    ]);
  });

  it("createChannel success → добавляется в channelsByRoom[roomId]", async () => {
    vi.spyOn(channelsApi, "createChannel").mockResolvedValue(ok(channelFixture));

    const result = await useChannelsStore
      .getState()
      .createChannel("r1", { name: "general", kind: "text" });

    expect(result).toEqual(channelFixture);
    expect(useChannelsStore.getState().channelsByRoom["r1"]).toEqual([
      channelFixture,
    ]);
  });

  it("createChannel fail CHANNEL-007 → toast.error, null, список не изменился", async () => {
    vi.spyOn(channelsApi, "createChannel").mockResolvedValue(
      err("CHANNEL-007", 403),
    );

    const result = await useChannelsStore
      .getState()
      .createChannel("r1", { name: "general", kind: "text" });

    expect(result).toBeNull();
    expect(useChannelsStore.getState().channelsByRoom["r1"]).toBeUndefined();
    expect(useToastsStore.getState().items.length).toBe(1);
    expect(useToastsStore.getState().items[0]?.variant).toBe("error");
  });

  it("deleteChannel success → удаляется из channelsByRoom[roomId]", async () => {
    useChannelsStore.setState({
      channelsByRoom: { r1: [channelFixture, voiceChannelFixture] },
      loadingByRoom: {},
      errorByRoom: {},
      activeChannelId: null,
    });
    vi.spyOn(channelsApi, "deleteChannel").mockResolvedValue(ok(null));

    const result = await useChannelsStore
      .getState()
      .deleteChannel("r1", "c1");

    expect(result).toBe(true);
    expect(useChannelsStore.getState().channelsByRoom["r1"]).toEqual([
      voiceChannelFixture,
    ]);
  });

  it("deleteChannel fail → toast.error, false, список не изменился", async () => {
    useChannelsStore.setState({
      channelsByRoom: { r1: [channelFixture] },
      loadingByRoom: {},
      errorByRoom: {},
      activeChannelId: null,
    });
    vi.spyOn(channelsApi, "deleteChannel").mockResolvedValue(
      err("CHANNEL-007", 403),
    );

    const result = await useChannelsStore
      .getState()
      .deleteChannel("r1", "c1");

    expect(result).toBe(false);
    expect(useChannelsStore.getState().channelsByRoom["r1"]).toEqual([
      channelFixture,
    ]);
    expect(useToastsStore.getState().items.length).toBe(1);
  });

  it("selectChannel + clear: activeChannelId меняется и сбрасывается", () => {
    useChannelsStore.getState().selectChannel("c1");
    expect(useChannelsStore.getState().activeChannelId).toBe("c1");

    useChannelsStore.getState().clear();
    const state = useChannelsStore.getState();
    expect(state.activeChannelId).toBeNull();
    expect(state.channelsByRoom).toEqual({});
    expect(state.loadingByRoom).toEqual({});
    expect(state.errorByRoom).toEqual({});
  });

  it("logoutFlow интеграция → очищает useChannelsStore", () => {
    useChannelsStore.setState({
      channelsByRoom: { r1: [channelFixture] },
      loadingByRoom: { r1: false },
      errorByRoom: { r1: null },
      activeChannelId: "c1",
    });

    logoutFlow();

    const state = useChannelsStore.getState();
    expect(state.channelsByRoom).toEqual({});
    expect(state.activeChannelId).toBeNull();
  });
});
