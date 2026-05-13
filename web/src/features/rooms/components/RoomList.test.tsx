// Тесты <RoomList />.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { RoomList } from "@/features/rooms/components/RoomList";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";

import type { RoomWithRole } from "@/features/rooms/types";

const room1: RoomWithRole = {
  id: "r1",
  ownerId: "u1",
  name: "general",
  createdAt: "2025",
  role: "owner",
};

const room2: RoomWithRole = {
  id: "r2",
  ownerId: "u2",
  name: "random",
  createdAt: "2025",
  role: "member",
};

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

function renderAt(path: string): void {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/rooms" element={<RoomList />} />
        <Route path="/rooms/:roomId" element={<RoomList />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("<RoomList />", () => {
  beforeEach(() => {
    resetStore();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetStore();
  });

  it("idle → вызывает loadRooms и показывает spinner", async () => {
    const loadRoomsMock = vi.fn().mockResolvedValue(undefined);
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      loadRooms: loadRoomsMock,
    });

    renderAt("/rooms");

    expect(screen.getByRole("progressbar")).toBeInTheDocument();
    await waitFor(() => {
      expect(loadRoomsMock).toHaveBeenCalledTimes(1);
    });
  });

  it("ready + пустой список → empty state с кнопками Create / Join", async () => {
    useRoomsStore.setState({
      rooms: [],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms");

    expect(
      screen.getByText(ru.rooms.emptyDescription),
    ).toBeInTheDocument();
    // Кнопки могут быть в двух местах (header + empty CTA) — используем AllBy.
    expect(
      screen.getAllByRole("button", { name: ru.rooms.createRoom }).length,
    ).toBeGreaterThanOrEqual(1);
    expect(
      screen.getAllByRole("button", { name: ru.rooms.joinByCode }).length,
    ).toBeGreaterThanOrEqual(1);
  });

  it("ready + есть комнаты → рендерит список с RoomListItem", () => {
    useRoomsStore.setState({
      rooms: [room1, room2],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms");

    expect(
      screen.getByRole("link", { name: /general/ }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /random/ })).toBeInTheDocument();
  });

  it("активная комната подсвечена aria-current=page", () => {
    useRoomsStore.setState({
      rooms: [room1, room2],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });

    renderAt("/rooms/r2");

    const activeLink = screen.getByRole("link", { name: /random/ });
    expect(activeLink).toHaveAttribute("aria-current", "page");
    const inactiveLink = screen.getByRole("link", { name: /general/ });
    expect(inactiveLink).not.toHaveAttribute("aria-current");
  });

  it("error → показывает сообщение и кнопку Повторить", async () => {
    const user = userEvent.setup();
    useRoomsStore.setState({
      rooms: [],
      status: "error",
      error: "boom",
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });
    const loadRoomsMock = vi.fn().mockResolvedValue(undefined);
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      loadRooms: loadRoomsMock,
    });

    renderAt("/rooms");

    expect(screen.getByText(ru.rooms.loadError)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: ru.rooms.retry }));
    expect(loadRoomsMock).toHaveBeenCalledTimes(1);
  });
});
