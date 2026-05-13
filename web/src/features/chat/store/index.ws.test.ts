// Тесты Phase-9 actions useChatStore: openChannel, sendMessage, retryMessage,
// onSubscribed/onMessageNew/onMessageSent/onWsError/setWsStatus.
// Используем подменный wsClient (vi.mock singleton): vi.mock хоистится наверх файла.

import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
  type Mock,
} from "vitest";

import * as chatApi from "@/features/chat/api/http";
import { useAuthStore } from "@/features/auth/store";
import { useChannelsStore } from "@/features/channels/store";
import { useChatStore } from "@/features/chat/store";
import { createFakeWsClient } from "@/shared/api/__tests__/ws.fake";
import { useToastsStore } from "@/shared/ui/useToast";

import type { CurrentUser } from "@/features/auth/types";
import type { ChannelDto } from "@/features/channels/types";
import type { Message } from "@/features/chat/types";
import type { FakeWsClient } from "@/shared/api/__tests__/ws.fake";

// Mock singleton: возвращает текущий fake через геттер, чтобы пересоздаваться в beforeEach.
let fake: FakeWsClient;
vi.mock("@/shared/api/wsClient.singleton", () => ({
  get wsClient(): FakeWsClient {
    return fake;
  },
}));

// === Фикстуры ===

const me: CurrentUser = {
  id: "u-me",
  email: "me@example.com",
  username: "me",
  createdAt: "2030-01-01T00:00:00Z",
};

const channelFixture: ChannelDto = {
  id: "c1",
  roomId: "r1",
  name: "general",
  kind: "text",
  createdAt: "2030-01-01T00:00:00Z",
};

function resetAll(): void {
  useChatStore.setState({
    messagesByChannel: {},
    nextBeforeByChannel: {},
    loadingHistoryByChannel: {},
    loadingMoreHistoryByChannel: {},
    subscribedChannelId: null,
    wsStatus: "idle",
  });
  useChannelsStore.setState({
    channelsByRoom: {},
    loadingByRoom: {},
    errorByRoom: {},
    activeChannelId: null,
  });
  useAuthStore.setState({
    status: "authenticated",
    currentUser: me,
    error: null,
    lastErrorCode: null,
  });
  useToastsStore.setState({ items: [] });
}

beforeEach(() => {
  fake = createFakeWsClient();
  resetAll();
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useChatStore.openChannel", () => {
  it("устанавливает subscribedChannelId и грузит историю через REST", async () => {
    const spy = vi.spyOn(chatApi, "listMessages").mockResolvedValue({
      ok: true,
      data: { items: [], nextBefore: null },
    });

    await useChatStore.getState().openChannel("c1");

    expect(useChatStore.getState().subscribedChannelId).toBe("c1");
    expect(spy).toHaveBeenCalledWith("c1", { limit: 50 });
  });

  it("если wsStatus=open — сразу шлёт subscribe фрейм", async () => {
    vi.spyOn(chatApi, "listMessages").mockResolvedValue({
      ok: true,
      data: { items: [], nextBefore: null },
    });
    useChatStore.setState({ wsStatus: "open" });

    await useChatStore.getState().openChannel("c1");

    expect(fake.send).toHaveBeenCalledWith({
      type: "subscribe",
      channel_id: "c1",
    });
  });

  it("если wsStatus !== open — subscribe не шлёт (отправится позже в setWsStatus)", async () => {
    vi.spyOn(chatApi, "listMessages").mockResolvedValue({
      ok: true,
      data: { items: [], nextBefore: null },
    });
    useChatStore.setState({ wsStatus: "connecting" });

    await useChatStore.getState().openChannel("c1");

    expect(fake.send).not.toHaveBeenCalled();
  });
});

describe("useChatStore.sendMessage", () => {
  it("добавляет pending optimistic с tempId и шлёт message.send", async () => {
    useChatStore.setState({ wsStatus: "open", subscribedChannelId: "c1" });

    await useChatStore.getState().sendMessage("c1", "hello");

    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect(list.length).toBe(1);
    const first = list[0] as Message;
    expect(first.status).toBe("pending");
    expect(first.text).toBe("hello");
    expect(first.authorId).toBe("u-me");
    expect(first.tempId).toBeDefined();
    expect(first.id).toBe(first.tempId);

    expect(fake.send).toHaveBeenCalledWith({
      type: "message.send",
      channel_id: "c1",
      text: "hello",
    });
  });

  it("пустой текст — no-op", async () => {
    await useChatStore.getState().sendMessage("c1", "");
    expect(useChatStore.getState().messagesByChannel["c1"]).toBeUndefined();
    expect(fake.send).not.toHaveBeenCalled();
  });

  it("при currentUser=null — no-op", async () => {
    useAuthStore.setState({ currentUser: null, status: "idle" });
    await useChatStore.getState().sendMessage("c1", "hi");
    expect(fake.send).not.toHaveBeenCalled();
  });

  it("если ack не пришёл за 10s — pending становится failed", async () => {
    vi.useFakeTimers();
    try {
      useChatStore.setState({ wsStatus: "open", subscribedChannelId: "c1" });
      await useChatStore.getState().sendMessage("c1", "hello");
      const before = useChatStore.getState().messagesByChannel["c1"] ?? [];
      expect((before[0] as Message).status).toBe("pending");

      vi.advanceTimersByTime(10_001);

      const after = useChatStore.getState().messagesByChannel["c1"] ?? [];
      expect((after[0] as Message).status).toBe("failed");
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("useChatStore.onMessageSent", () => {
  it("коммитит последний pending в committed с серверным id и created_at", async () => {
    useChatStore.setState({ wsStatus: "open", subscribedChannelId: "c1" });
    await useChatStore.getState().sendMessage("c1", "hi");
    const pendingMsg = (
      useChatStore.getState().messagesByChannel["c1"] ?? []
    )[0] as Message;
    const tempId = pendingMsg.tempId as string;

    useChatStore.getState().onMessageSent({
      id: "server-id",
      channel_id: "c1",
      created_at: "2030-01-01T00:00:05Z",
    });

    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect(list.length).toBe(1);
    const m = list[0] as Message;
    expect(m.id).toBe("server-id");
    expect(m.createdAt).toBe("2030-01-01T00:00:05Z");
    expect(m.status).toBe("committed");
    expect(m.tempId).toBeUndefined();
    expect(tempId).not.toBe("server-id");
  });

  it("игнорирует если нет pending", () => {
    useChatStore.getState().onMessageSent({
      id: "x",
      channel_id: "c1",
      created_at: "2030-01-01T00:00:00Z",
    });
    expect(useChatStore.getState().messagesByChannel["c1"]).toBeUndefined();
  });
});

describe("useChatStore.onMessageNew", () => {
  it("вставляет committed-сообщение в конец списка", () => {
    useChatStore.getState().onMessageNew({
      id: "m1",
      channel_id: "c1",
      author_id: "u2",
      text: "hi",
      created_at: "2030-01-01T00:00:00Z",
    });
    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect(list.length).toBe(1);
    expect((list[0] as Message).id).toBe("m1");
  });

  it("дедуплицирует по id (D-08 multi-tab)", () => {
    const data = {
      id: "m1",
      channel_id: "c1",
      author_id: "u2",
      text: "hi",
      created_at: "2030-01-01T00:00:00Z",
    };
    useChatStore.getState().onMessageNew(data);
    useChatStore.getState().onMessageNew(data);
    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect(list.length).toBe(1);
  });

  it("message.new для own send (есть pending с тем же текстом) коммитит pending, а не вставляет дубль", async () => {
    useChatStore.setState({ wsStatus: "open", subscribedChannelId: "c1" });
    await useChatStore.getState().sendMessage("c1", "we");
    expect((useChatStore.getState().messagesByChannel["c1"] ?? []).length).toBe(1);

    // Broadcast от бэка прилетает первым (типичный порядок: broadcast → ack).
    useChatStore.getState().onMessageNew({
      id: "server-id",
      channel_id: "c1",
      author_id: "u-me",
      text: "we",
      created_at: "2030-01-01T00:00:05Z",
    });

    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect(list.length).toBe(1);
    const m = list[0] as Message;
    expect(m.id).toBe("server-id");
    expect(m.status).toBe("committed");
    expect(m.tempId).toBeUndefined();
  });

  it("message.new для own send без pending (multi-tab вкладка B) — обычная вставка committed", () => {
    // Та же вкладка получает свой own broadcast, но pending здесь нет (отправлено в другой вкладке).
    useChatStore.getState().onMessageNew({
      id: "server-id",
      channel_id: "c1",
      author_id: "u-me",
      text: "from other tab",
      created_at: "2030-01-01T00:00:05Z",
    });
    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect(list.length).toBe(1);
    expect((list[0] as Message).status).toBe("committed");
    expect((list[0] as Message).id).toBe("server-id");
  });

  it("вставка с более ранним createdAt сохраняет ASC-сортировку", () => {
    useChatStore.setState({
      messagesByChannel: {
        c1: [
          {
            id: "m2",
            channelId: "c1",
            authorId: "u2",
            text: "later",
            createdAt: "2030-01-01T00:00:10Z",
            status: "committed",
          },
        ],
      },
    });
    useChatStore.getState().onMessageNew({
      id: "m1",
      channel_id: "c1",
      author_id: "u1",
      text: "earlier",
      created_at: "2030-01-01T00:00:05Z",
    });
    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect(list.map((m) => m.id)).toEqual(["m1", "m2"]);
  });
});

describe("useChatStore.onWsError", () => {
  it("CHAT-001 → помечает последний pending в подписанном канале как failed + toast", async () => {
    useChatStore.setState({ wsStatus: "open", subscribedChannelId: "c1" });
    await useChatStore.getState().sendMessage("c1", "hi");

    useChatStore
      .getState()
      .onWsError({ code: "CHAT-001", message: "invalid text" });

    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect((list[0] as Message).status).toBe("failed");
    expect(useToastsStore.getState().items.length).toBe(1);
  });

  it("CHAT-004 → каскадно перегружает channels и показывает toast", () => {
    useChannelsStore.setState({
      channelsByRoom: { r1: [channelFixture] },
      loadingByRoom: {},
      errorByRoom: {},
      activeChannelId: null,
    });
    useChatStore.setState({ subscribedChannelId: "c1" });
    const loadChannelsSpy = vi
      .spyOn(useChannelsStore.getState(), "loadChannels")
      .mockResolvedValue(undefined);

    useChatStore
      .getState()
      .onWsError({ code: "CHAT-004", message: "not member" });

    expect(loadChannelsSpy).toHaveBeenCalledWith("r1");
    expect(useToastsStore.getState().items.length).toBe(1);
  });
});

describe("useChatStore.setWsStatus", () => {
  it("переход в open с подписанным каналом → шлёт subscribe", () => {
    useChatStore.setState({ subscribedChannelId: "c1", wsStatus: "connecting" });
    useChatStore.getState().setWsStatus("open");
    expect(fake.send).toHaveBeenCalledWith({
      type: "subscribe",
      channel_id: "c1",
    });
  });

  it("переход reconnecting → open тоже триггерит subscribe", () => {
    useChatStore.setState({
      subscribedChannelId: "c1",
      wsStatus: "reconnecting",
    });
    useChatStore.getState().setWsStatus("open");
    expect(fake.send).toHaveBeenCalledWith({
      type: "subscribe",
      channel_id: "c1",
    });
  });

  it("переход в open без subscribedChannelId — subscribe не шлёт", () => {
    useChatStore.setState({ subscribedChannelId: null, wsStatus: "connecting" });
    useChatStore.getState().setWsStatus("open");
    expect(fake.send).not.toHaveBeenCalled();
  });

  it("повторный setWsStatus(open) без перехода не дублирует subscribe", () => {
    useChatStore.setState({ subscribedChannelId: "c1", wsStatus: "open" });
    useChatStore.getState().setWsStatus("open");
    expect(fake.send).not.toHaveBeenCalled();
  });
});

describe("useChatStore.retryMessage", () => {
  it("failed → pending + повторный send", async () => {
    useChatStore.setState({
      wsStatus: "open",
      subscribedChannelId: "c1",
      messagesByChannel: {
        c1: [
          {
            id: "tmp-1",
            tempId: "tmp-1",
            channelId: "c1",
            authorId: "u-me",
            text: "hi",
            createdAt: "2030-01-01T00:00:00Z",
            status: "failed",
          },
        ],
      },
    });
    (fake.send as Mock).mockClear();

    await useChatStore.getState().retryMessage("c1", "tmp-1");

    const list = useChatStore.getState().messagesByChannel["c1"] ?? [];
    expect((list[0] as Message).status).toBe("pending");
    expect(fake.send).toHaveBeenCalledWith({
      type: "message.send",
      channel_id: "c1",
      text: "hi",
    });
  });

  it("если сообщение не failed — no-op", async () => {
    useChatStore.setState({
      messagesByChannel: {
        c1: [
          {
            id: "tmp-1",
            tempId: "tmp-1",
            channelId: "c1",
            authorId: "u-me",
            text: "hi",
            createdAt: "2030-01-01T00:00:00Z",
            status: "pending",
          },
        ],
      },
    });
    (fake.send as Mock).mockClear();

    await useChatStore.getState().retryMessage("c1", "tmp-1");

    expect(fake.send).not.toHaveBeenCalled();
  });
});
