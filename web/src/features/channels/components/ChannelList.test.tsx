// Тесты <ChannelList />.

import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { ChannelList } from "@/features/channels/components/ChannelList";
import { useChannelsStore } from "@/features/channels/store";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";

import type { Channel } from "@/features/channels/types";
import type { RoomWithRole } from "@/features/rooms/types";

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

const memberRoom: RoomWithRole = {
  id: "r1",
  ownerId: "u1",
  name: "Room",
  createdAt: "2030",
  role: "member",
};

const ownerRoom: RoomWithRole = { ...memberRoom, role: "owner" };

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

function renderAt(path: string): void {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/rooms/:roomId" element={<ChannelList roomId="r1" />} />
        <Route
          path="/rooms/:roomId/channels/:channelId"
          element={<ChannelList roomId="r1" />}
        />
      </Routes>
    </MemoryRouter>,
  );
}

// Мок loadChannels — чтобы useEffect-вызов не делал реальный fetch
// и не перетирал state, заданный в каждом тесте.
function mockLoadChannels(): ReturnType<typeof vi.fn> {
  const loadChannelsMock = vi.fn().mockResolvedValue(undefined);
  vi.spyOn(useChannelsStore, "getState").mockReturnValue({
    ...useChannelsStore.getState(),
    loadChannels: loadChannelsMock,
  });
  return loadChannelsMock;
}

describe("<ChannelList />", () => {
  beforeEach(() => {
    resetChannelsStore();
    resetRoomsStore();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetChannelsStore();
    resetRoomsStore();
  });

  it("loading state → показывает spinner и вызывает loadChannels", async () => {
    const loadChannelsMock = mockLoadChannels();
    useChannelsStore.setState({
      channelsByRoom: {},
      loadingByRoom: { r1: true },
      errorByRoom: {},
      activeChannelId: null,
    });
    useRoomsStore.setState({
      rooms: [memberRoom],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms/r1");

    expect(screen.getByRole("progressbar")).toBeInTheDocument();
    await waitFor(() => {
      expect(loadChannelsMock).toHaveBeenCalledWith("r1");
    });
  });

  it("empty + non-manage (member) → нет CTA «+» и нет кнопки создать", () => {
    mockLoadChannels();
    useChannelsStore.setState({
      channelsByRoom: { r1: [] },
      loadingByRoom: { r1: false },
      errorByRoom: {},
      activeChannelId: null,
    });
    useRoomsStore.setState({
      rooms: [memberRoom],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms/r1");

    // Кнопки создания не должно быть для member.
    expect(
      screen.queryByRole("button", { name: ru.channels.createChannel }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: ru.channels.createBtn }),
    ).not.toBeInTheDocument();
    expect(screen.getByText(ru.channels.emptyDescription)).toBeInTheDocument();
  });

  it("empty + canManage (owner) → CTA «+» и empty-кнопка показаны", () => {
    mockLoadChannels();
    useChannelsStore.setState({
      channelsByRoom: { r1: [] },
      loadingByRoom: { r1: false },
      errorByRoom: {},
      activeChannelId: null,
    });
    useRoomsStore.setState({
      rooms: [ownerRoom],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms/r1");

    // Header "+" с aria-label "Создать канал" — кнопка для админ/owner.
    expect(
      screen.getByRole("button", { name: ru.channels.createChannel }),
    ).toBeInTheDocument();
    // Empty-state CTA с текстом "Создать первый канал".
    expect(
      screen.getByRole("button", { name: ru.channels.createBtn }),
    ).toBeInTheDocument();
  });

  it("ready: рендерит секцию text-каналов", () => {
    mockLoadChannels();
    useChannelsStore.setState({
      channelsByRoom: { r1: [textChannel] },
      loadingByRoom: { r1: false },
      errorByRoom: {},
      activeChannelId: null,
    });
    useRoomsStore.setState({
      rooms: [memberRoom],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms/r1");

    expect(screen.getByText(ru.channels.textChannels)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /general/ }),
    ).toBeInTheDocument();
  });

  it("voice-каналы рендерятся в отдельной секции как disabled", () => {
    mockLoadChannels();
    useChannelsStore.setState({
      channelsByRoom: { r1: [textChannel, voiceChannel] },
      loadingByRoom: { r1: false },
      errorByRoom: {},
      activeChannelId: null,
    });
    useRoomsStore.setState({
      rooms: [memberRoom],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms/r1");

    expect(screen.getByText(ru.channels.voiceChannels)).toBeInTheDocument();
    const voiceBtn = screen.getByRole("button", { name: /voice-general/ });
    expect(voiceBtn).toHaveAttribute("aria-disabled", "true");
  });

  it("активный канал получает aria-current=page", () => {
    mockLoadChannels();
    useChannelsStore.setState({
      channelsByRoom: { r1: [textChannel] },
      loadingByRoom: { r1: false },
      errorByRoom: {},
      activeChannelId: null,
    });
    useRoomsStore.setState({
      rooms: [memberRoom],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms/r1/channels/c1");

    const activeBtn = screen.getByRole("button", { name: /general/ });
    expect(activeBtn).toHaveAttribute("aria-current", "page");
  });
});
