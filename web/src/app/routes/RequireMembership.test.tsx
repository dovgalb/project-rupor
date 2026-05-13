// Тесты RequireMembership — гард доступа к комнате.

import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { useRoomsStore } from "@/features/rooms";

import { RequireMembership } from "./RequireMembership";

import type { RoomWithRole } from "@/features/rooms";

const roomFixture: RoomWithRole = {
  id: "r1",
  ownerId: "u1",
  name: "general",
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
        <Route path="/rooms" element={<div>rooms index</div>} />
        <Route
          path="/rooms/:roomId"
          element={
            <RequireMembership>
              <div>room content</div>
            </RequireMembership>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("<RequireMembership />", () => {
  beforeEach(() => {
    resetStore();
  });

  afterEach(() => {
    resetStore();
  });

  it("idle → показывает Spinner", () => {
    useRoomsStore.setState({
      rooms: [],
      status: "idle",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });
    renderAt("/rooms/r1");
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
    expect(screen.queryByText("room content")).not.toBeInTheDocument();
  });

  it("loading → показывает Spinner", () => {
    useRoomsStore.setState({
      rooms: [],
      status: "loading",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });
    renderAt("/rooms/r1");
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
  });

  it("ready + не member → redirect на /rooms", () => {
    useRoomsStore.setState({
      rooms: [],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });
    renderAt("/rooms/r1");
    expect(screen.getByText("rooms index")).toBeInTheDocument();
    expect(screen.queryByText("room content")).not.toBeInTheDocument();
  });

  it("ready + member → рендерит children", () => {
    useRoomsStore.setState({
      rooms: [roomFixture],
      status: "ready",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: {},
    });
    renderAt("/rooms/r1");
    expect(screen.getByText("room content")).toBeInTheDocument();
  });
});
