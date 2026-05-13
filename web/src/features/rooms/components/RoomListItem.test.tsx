// Тесты <RoomListItem />.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MemoryRouter } from "react-router-dom";

import { RoomListItem } from "@/features/rooms/components/RoomListItem";

import type { RoomWithRole } from "@/features/rooms/types";

const roomFixture: RoomWithRole = {
  id: "room-1",
  ownerId: "user-1",
  name: "General",
  createdAt: "2025-01-01T00:00:00Z",
  role: "owner",
};

function renderItem(active: boolean): void {
  render(
    <MemoryRouter>
      <ul>
        <RoomListItem room={roomFixture} active={active} />
      </ul>
    </MemoryRouter>,
  );
}

describe("<RoomListItem />", () => {
  it("рендерит ссылку на /rooms/:id с названием комнаты", () => {
    renderItem(false);
    const link = screen.getByRole("link", { name: /General/ });
    expect(link).toHaveAttribute("href", "/rooms/room-1");
  });

  it("active=true → aria-current=page", () => {
    renderItem(true);
    expect(screen.getByRole("link", { name: /General/ })).toHaveAttribute(
      "aria-current",
      "page",
    );
  });

  it("active=false → без aria-current", () => {
    renderItem(false);
    expect(screen.getByRole("link", { name: /General/ })).not.toHaveAttribute(
      "aria-current",
    );
  });
});
