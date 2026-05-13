// Тесты RequireChannelInRoom — гард доступа к каналу внутри комнаты.

import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { useChannelsStore } from "@/features/channels";
import { useToastsStore } from "@/shared/ui/useToast";

import { RequireChannelInRoom } from "./RequireChannelInRoom";

import type { Channel } from "@/features/channels";

const textChannel: Channel = {
  id: "c1",
  roomId: "r1",
  name: "general",
  kind: "text",
  createdAt: "2030",
};

const voiceChannel: Channel = {
  id: "c2",
  roomId: "r1",
  name: "voice-general",
  kind: "voice",
  createdAt: "2030",
};

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

function renderAt(path: string): void {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/rooms/:roomId" element={<div>room page</div>} />
        <Route
          path="/rooms/:roomId/channels/:channelId"
          element={
            <RequireChannelInRoom>
              <div>channel content</div>
            </RequireChannelInRoom>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("<RequireChannelInRoom />", () => {
  beforeEach(() => {
    resetChannelsStore();
    resetToasts();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetChannelsStore();
    resetToasts();
  });

  it("loading=true → показывает Spinner, children скрыты", () => {
    useChannelsStore.setState({
      channelsByRoom: {},
      loadingByRoom: { r1: true },
      errorByRoom: {},
      activeChannelId: null,
    });
    renderAt("/rooms/r1/channels/c1");

    expect(screen.getByRole("progressbar")).toBeInTheDocument();
    expect(screen.queryByText("channel content")).not.toBeInTheDocument();
  });

  it("канал отсутствует → toast.error + redirect на /rooms/:roomId", () => {
    useChannelsStore.setState({
      channelsByRoom: { r1: [] },
      loadingByRoom: { r1: false },
      errorByRoom: {},
      activeChannelId: null,
    });

    renderAt("/rooms/r1/channels/missing");

    expect(screen.getByText("room page")).toBeInTheDocument();
    expect(screen.queryByText("channel content")).not.toBeInTheDocument();
    const items = useToastsStore.getState().items;
    expect(items.length).toBe(1);
    expect(items[0]?.variant).toBe("error");
  });

  it("канал voice → toast.info + redirect на /rooms/:roomId", () => {
    useChannelsStore.setState({
      channelsByRoom: { r1: [voiceChannel] },
      loadingByRoom: { r1: false },
      errorByRoom: {},
      activeChannelId: null,
    });

    renderAt("/rooms/r1/channels/c2");

    expect(screen.getByText("room page")).toBeInTheDocument();
    expect(screen.queryByText("channel content")).not.toBeInTheDocument();
    const items = useToastsStore.getState().items;
    expect(items.length).toBe(1);
    expect(items[0]?.variant).toBe("info");
  });

  it("канал text найден → рендерит children", () => {
    useChannelsStore.setState({
      channelsByRoom: { r1: [textChannel] },
      loadingByRoom: { r1: false },
      errorByRoom: {},
      activeChannelId: null,
    });

    renderAt("/rooms/r1/channels/c1");

    expect(screen.getByText("channel content")).toBeInTheDocument();
  });
});
