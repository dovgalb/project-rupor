// Тесты <MessageList />: loading / empty / ready / beginning / placeholder автора.

import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as chatApi from "@/features/chat/api/http";
import { MessageList } from "@/features/chat/components/MessageList";
import { useChatStore } from "@/features/chat/store";
import { useAuthStore } from "@/features/auth/store";
import { ru } from "@/shared/lib/i18n/ru";

import type { Message } from "@/features/chat/types";
import type { ApiResult } from "@/shared/api/errors";

function ok<T>(data: T): ApiResult<T> {
  return { ok: true, data };
}

function resetChat(): void {
  useChatStore.setState({
    messagesByChannel: {},
    nextBeforeByChannel: {},
    loadingHistoryByChannel: {},
    loadingMoreHistoryByChannel: {},
    subscribedChannelId: null,
    wsStatus: "idle",
  });
}

function resetAuth(): void {
  useAuthStore.setState({
    status: "idle",
    currentUser: null,
    error: null,
    lastErrorCode: null,
  });
}

// IntersectionObserver отсутствует в jsdom — заглушка.
class IOMock {
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();
  takeRecords = vi.fn().mockReturnValue([]);
  // Нужный констуктору для совместимости с TS-сигнатурой.
  constructor(
    public callback: IntersectionObserverCallback,
    public options?: IntersectionObserverInit,
  ) {}
}

describe("<MessageList />", () => {
  beforeEach(() => {
    vi.stubGlobal("IntersectionObserver", IOMock);
    resetChat();
    resetAuth();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    resetChat();
    resetAuth();
  });

  it("loading state: пока loadingHistory=true и сообщений нет → спиннер", () => {
    // Не разрешаем промис → loadingHistory остаётся true.
    vi.spyOn(chatApi, "listMessages").mockReturnValue(
      new Promise(() => {
        /* never */
      }),
    );
    render(<MessageList channelId="c1" />);
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
  });

  it("empty state: ready и 0 сообщений → <EmptyChat />", async () => {
    vi.spyOn(chatApi, "listMessages").mockResolvedValue(
      ok({ items: [], nextBefore: null }),
    );
    render(<MessageList channelId="c1" />);

    await waitFor(() => {
      expect(screen.getByText(ru.chat.emptyTitle)).toBeInTheDocument();
    });
  });

  it("ready: рендерит число MessageItem по числу сообщений", async () => {
    const messages: Message[] = [
      {
        id: "m1",
        channelId: "c1",
        authorId: "11111111-2222-3333-4444-555555555555",
        text: "первое",
        createdAt: "2030-01-01T10:00:00Z",
        status: "committed",
      },
      {
        id: "m2",
        channelId: "c1",
        authorId: "66666666-7777-8888-9999-000000000000",
        text: "второе",
        createdAt: "2030-01-01T10:01:00Z",
        status: "committed",
      },
    ];
    useChatStore.setState({
      messagesByChannel: { c1: messages },
      nextBeforeByChannel: { c1: null },
      loadingHistoryByChannel: { c1: false },
      loadingMoreHistoryByChannel: {},
      subscribedChannelId: null,
      wsStatus: "idle",
    });
    vi.spyOn(chatApi, "listMessages").mockResolvedValue(
      ok({ items: messages.slice().reverse(), nextBefore: null }),
    );

    render(<MessageList channelId="c1" />);

    const log = screen.getByRole("log");
    expect(log).toBeInTheDocument();
    expect(log.querySelectorAll("li").length).toBe(2);
    expect(screen.getByText("первое")).toBeInTheDocument();
    expect(screen.getByText("второе")).toBeInTheDocument();
  });

  it("nextBefore=null + есть сообщения → подпись 'Это начало канала'", async () => {
    useChatStore.setState({
      messagesByChannel: {
        c1: [
          {
            id: "m1",
            channelId: "c1",
            authorId: "abc",
            text: "x",
            createdAt: "2030",
            status: "committed",
          },
        ],
      },
      nextBeforeByChannel: { c1: null },
      loadingHistoryByChannel: { c1: false },
      loadingMoreHistoryByChannel: {},
      subscribedChannelId: null,
      wsStatus: "idle",
    });
    vi.spyOn(chatApi, "listMessages").mockResolvedValue(
      ok({ items: [], nextBefore: null }),
    );

    render(<MessageList channelId="c1" />);

    expect(screen.getByText(ru.chat.beginningOfChannel)).toBeInTheDocument();
  });

  it("автор = текущий пользователь → отображается meUsername; иначе UUID-placeholder", async () => {
    useAuthStore.setState({
      status: "authenticated",
      currentUser: {
        id: "11111111-2222-3333-4444-555555555555",
        email: "me@example.com",
        username: "mihej",
        createdAt: "2030",
      },
      error: null,
      lastErrorCode: null,
    });
    useChatStore.setState({
      messagesByChannel: {
        c1: [
          {
            id: "m1",
            channelId: "c1",
            authorId: "11111111-2222-3333-4444-555555555555",
            text: "ave",
            createdAt: "2030",
            status: "committed",
          },
          {
            id: "m2",
            channelId: "c1",
            authorId: "abcdef12-3456-7890-abcd-ef1234567890",
            text: "salve",
            createdAt: "2030",
            status: "committed",
          },
        ],
      },
      nextBeforeByChannel: { c1: null },
      loadingHistoryByChannel: { c1: false },
      loadingMoreHistoryByChannel: {},
      subscribedChannelId: null,
      wsStatus: "idle",
    });
    vi.spyOn(chatApi, "listMessages").mockResolvedValue(
      ok({ items: [], nextBefore: null }),
    );

    render(<MessageList channelId="c1" />);

    expect(screen.getByText("mihej")).toBeInTheDocument();
    expect(screen.getByText("ABCDEF")).toBeInTheDocument();
  });
});
