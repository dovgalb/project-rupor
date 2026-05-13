// Тесты <MembersList />.

import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { MembersList } from "@/features/rooms/components/MembersList";
import { useAuthStore } from "@/features/auth";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";

import type { CurrentUser } from "@/features/auth";
import type { Member } from "@/features/rooms/types";

const currentUser: CurrentUser = {
  id: "user-me",
  email: "me@example.com",
  username: "me",
  createdAt: "2025",
};

const memberMe: Member = {
  userId: "user-me",
  role: "owner",
  joinedAt: "2025",
};

const memberOther: Member = {
  userId: "abcdef12-3456-7890-1234-567890abcdef",
  role: "member",
  joinedAt: "2025",
};

function resetRooms(): void {
  useRoomsStore.setState({
    rooms: [],
    status: "idle",
    error: null,
    membersByRoom: {},
    membersLoadingByRoom: {},
    activeInviteByRoom: {},
  });
}

function resetAuth(): void {
  useAuthStore.setState({
    status: "authenticated",
    currentUser,
    error: null,
    lastErrorCode: null,
  });
}

describe("<MembersList />", () => {
  beforeEach(() => {
    resetRooms();
    resetAuth();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetRooms();
  });

  it("loading + пустой список → Spinner", () => {
    useRoomsStore.setState({
      rooms: [],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: { r1: true },
      activeInviteByRoom: {},
    });
    // Подменяем loadMembers, чтобы не лез в сеть.
    vi.spyOn(useRoomsStore.getState(), "loadMembers").mockResolvedValue(
      undefined,
    );

    render(<MembersList roomId="r1" />);

    expect(screen.getByRole("progressbar")).toBeInTheDocument();
  });

  it("ready + есть members → рендерится заголовок и список", async () => {
    useRoomsStore.setState({
      rooms: [],
      status: "ready",
      error: null,
      membersByRoom: { r1: [memberMe, memberOther] },
      membersLoadingByRoom: { r1: false },
      activeInviteByRoom: {},
    });
    vi.spyOn(useRoomsStore.getState(), "loadMembers").mockResolvedValue(
      undefined,
    );

    render(<MembersList roomId="r1" />);

    expect(screen.getByText(/Участники \(2\)/)).toBeInTheDocument();
    expect(screen.getByText("me")).toBeInTheDocument();
    expect(screen.getByText("ABCDEF")).toBeInTheDocument();
  });

  it("при mount вызывает loadMembers(roomId)", async () => {
    const loadMembersMock = vi.fn().mockResolvedValue(undefined);
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      loadMembers: loadMembersMock,
    });

    render(<MembersList roomId="r1" />);

    await waitFor(() => {
      expect(loadMembersMock).toHaveBeenCalledWith("r1");
    });
  });

  it("empty members → defensive: рендерится заголовок (0)", () => {
    useRoomsStore.setState({
      rooms: [],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: { r1: false },
      activeInviteByRoom: {},
    });
    vi.spyOn(useRoomsStore.getState(), "loadMembers").mockResolvedValue(
      undefined,
    );

    render(<MembersList roomId="r1" />);

    expect(screen.getByText(`${ru.rooms.members} (0)`)).toBeInTheDocument();
  });
});
