// Тесты useChatStore: REST-only (Phase 8) — loadHistory, loadMoreHistory, clear,
// каскад CHAT-002/004, идемпотентность, интеграция с logoutFlow.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as chatApi from "@/features/chat/api/http";
import { useChatStore } from "@/features/chat/store";
import { useChannelsStore } from "@/features/channels/store";
import { logoutFlow } from "@/shared/lib/logoutFlow";
import { useToastsStore } from "@/shared/ui/useToast";

import type {
  ListMessagesResponse,
  MessageDto,
} from "@/features/chat/types";
import type { ChannelDto } from "@/features/channels/types";
import type { ApiResult } from "@/shared/api/errors";

// === Фикстуры ===

const channelFixture: ChannelDto = {
  id: "c1",
  roomId: "r1",
  name: "general",
  kind: "text",
  createdAt: "2030-01-01T00:00:00Z",
};

const msg1: MessageDto = {
  id: "m1",
  channelId: "c1",
  authorId: "u1",
  text: "hello",
  createdAt: "2030-01-01T00:00:01Z",
};

const msg2: MessageDto = {
  id: "m2",
  channelId: "c1",
  authorId: "u2",
  text: "world",
  createdAt: "2030-01-01T00:00:02Z",
};

const olderMsg: MessageDto = {
  id: "m0",
  channelId: "c1",
  authorId: "u1",
  text: "older",
  createdAt: "2030-01-01T00:00:00.500Z",
};

// === Хелперы ===

function ok<T>(data: T): ApiResult<T> {
  return { ok: true, data };
}

function err(code: string, status = 400): ApiResult<never> {
  return { ok: false, status, error: { code, message: code } };
}

function resetChatStore(): void {
  useChatStore.setState({
    messagesByChannel: {},
    nextBeforeByChannel: {},
    loadingHistoryByChannel: {},
    loadingMoreHistoryByChannel: {},
    subscribedChannelId: null,
    wsStatus: "idle",
  });
}

function resetChannelsStore(): void {
  useChannelsStore.setState({
    channelsByRoom: {},
    loadingByRoom: {},
    errorByRoom: {},
    activeChannelId: null,
  });
}

function resetToasts(): void {
  useToastsStore.setState({ items: [] });
}

describe("useChatStore", () => {
  beforeEach(() => {
    resetChatStore();
    resetChannelsStore();
    resetToasts();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetChatStore();
    resetChannelsStore();
    resetToasts();
  });

  it("loadHistory success → messages в ASC, nextBefore сохранён", async () => {
    // Бэк возвращает DESC: [msg2, msg1]; в state должно быть ASC: [msg1, msg2].
    const response: ListMessagesResponse = {
      items: [msg2, msg1],
      nextBefore: "cursor-1",
    };
    vi.spyOn(chatApi, "listMessages").mockResolvedValue(ok(response));

    await useChatStore.getState().loadHistory("c1");

    const state = useChatStore.getState();
    expect(state.messagesByChannel["c1"]).toEqual([
      { ...msg1, status: "committed" },
      { ...msg2, status: "committed" },
    ]);
    expect(state.nextBeforeByChannel["c1"]).toBe("cursor-1");
    expect(state.loadingHistoryByChannel["c1"]).toBe(false);
  });

  it("loadHistory идемпотентен: повторный вызов с уже загруженной историей не делает второй fetch", async () => {
    const response: ListMessagesResponse = {
      items: [msg1],
      nextBefore: null,
    };
    const spy = vi
      .spyOn(chatApi, "listMessages")
      .mockResolvedValue(ok(response));

    await useChatStore.getState().loadHistory("c1");
    await useChatStore.getState().loadHistory("c1");

    expect(spy).toHaveBeenCalledTimes(1);
  });

  it("loadMoreHistory success → старые сообщения префиксуются, nextBefore обновляется", async () => {
    // Стартовое состояние: уже есть [msg1, msg2] в ASC, курсор = "cursor-1".
    useChatStore.setState({
      messagesByChannel: {
        c1: [
          { ...msg1, status: "committed" },
          { ...msg2, status: "committed" },
        ],
      },
      nextBeforeByChannel: { c1: "cursor-1" },
      loadingHistoryByChannel: {},
      loadingMoreHistoryByChannel: {},
      subscribedChannelId: null,
      wsStatus: "idle",
    });

    const response: ListMessagesResponse = {
      items: [olderMsg],
      nextBefore: null,
    };
    vi.spyOn(chatApi, "listMessages").mockResolvedValue(ok(response));

    await useChatStore.getState().loadMoreHistory("c1");

    const state = useChatStore.getState();
    expect(state.messagesByChannel["c1"]).toEqual([
      { ...olderMsg, status: "committed" },
      { ...msg1, status: "committed" },
      { ...msg2, status: "committed" },
    ]);
    expect(state.nextBeforeByChannel["c1"]).toBeNull();
    expect(state.loadingMoreHistoryByChannel["c1"]).toBe(false);
  });

  it("loadMoreHistory с nextBefore=null → no-op (нет API-вызова)", async () => {
    useChatStore.setState({
      messagesByChannel: { c1: [{ ...msg1, status: "committed" }] },
      nextBeforeByChannel: { c1: null },
      loadingHistoryByChannel: {},
      loadingMoreHistoryByChannel: {},
      subscribedChannelId: null,
      wsStatus: "idle",
    });
    const spy = vi.spyOn(chatApi, "listMessages");

    await useChatStore.getState().loadMoreHistory("c1");

    expect(spy).not.toHaveBeenCalled();
  });

  it("loadHistory fail CHAT-002 → каскадно вызывается loadChannels для соответствующей комнаты", async () => {
    // Имитируем: channelsByRoom содержит c1 в r1.
    useChannelsStore.setState({
      channelsByRoom: { r1: [channelFixture] },
      loadingByRoom: {},
      errorByRoom: {},
      activeChannelId: null,
    });
    vi.spyOn(chatApi, "listMessages").mockResolvedValue(err("CHAT-002", 404));
    const loadChannelsSpy = vi
      .spyOn(useChannelsStore.getState(), "loadChannels")
      .mockResolvedValue(undefined);

    await useChatStore.getState().loadHistory("c1");

    expect(loadChannelsSpy).toHaveBeenCalledWith("r1");
    // Toast тоже показан.
    expect(useToastsStore.getState().items.length).toBe(1);
    expect(useToastsStore.getState().items[0]?.variant).toBe("error");
  });

  it("clear → state сбрасывается к initial", () => {
    useChatStore.setState({
      messagesByChannel: { c1: [{ ...msg1, status: "committed" }] },
      nextBeforeByChannel: { c1: "x" },
      loadingHistoryByChannel: { c1: true },
      loadingMoreHistoryByChannel: { c1: true },
      subscribedChannelId: "c1",
      wsStatus: "open",
    });

    useChatStore.getState().clear();

    const state = useChatStore.getState();
    expect(state.messagesByChannel).toEqual({});
    expect(state.nextBeforeByChannel).toEqual({});
    expect(state.loadingHistoryByChannel).toEqual({});
    expect(state.loadingMoreHistoryByChannel).toEqual({});
    expect(state.subscribedChannelId).toBeNull();
    expect(state.wsStatus).toBe("idle");
  });

  it("logoutFlow интеграция → очищает useChatStore", () => {
    useChatStore.setState({
      messagesByChannel: { c1: [{ ...msg1, status: "committed" }] },
      nextBeforeByChannel: { c1: "x" },
      loadingHistoryByChannel: {},
      loadingMoreHistoryByChannel: {},
      subscribedChannelId: "c1",
      wsStatus: "open",
    });

    logoutFlow();

    const state = useChatStore.getState();
    expect(state.messagesByChannel).toEqual({});
    expect(state.subscribedChannelId).toBeNull();
    expect(state.wsStatus).toBe("idle");
  });

  it("loadHistory fail NETWORK → toast.error показан", async () => {
    vi.spyOn(chatApi, "listMessages").mockResolvedValue(err("NETWORK", 0));

    await useChatStore.getState().loadHistory("c1");

    expect(useToastsStore.getState().items.length).toBe(1);
    expect(useToastsStore.getState().items[0]?.variant).toBe("error");
    expect(useChatStore.getState().loadingHistoryByChannel["c1"]).toBe(false);
  });
});
