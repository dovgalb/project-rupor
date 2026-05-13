// Тесты <CreateRoomModal />.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CreateRoomModal } from "@/features/rooms/components/CreateRoomModal";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";
import { useToastsStore } from "@/shared/ui/useToast";

import type { RoomWithRole } from "@/features/rooms/types";

const createdRoom: RoomWithRole = {
  id: "r1",
  ownerId: "u1",
  name: "general",
  createdAt: "2030",
  role: "owner",
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

describe("<CreateRoomModal />", () => {
  beforeEach(() => {
    resetStore();
    useToastsStore.setState({ items: [] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetStore();
    useToastsStore.setState({ items: [] });
  });

  it("open=false → не рендерится в DOM", () => {
    render(
      <CreateRoomModal open={false} onClose={vi.fn()} onCreated={vi.fn()} />,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("open=true → рендерит форму с input и кнопкой Создать", () => {
    render(
      <CreateRoomModal open={true} onClose={vi.fn()} onCreated={vi.fn()} />,
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByLabelText(ru.rooms.nameLabel)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: ru.rooms.create }),
    ).toBeInTheDocument();
  });

  it("zod-валидация: пустое имя → inline error", async () => {
    const user = userEvent.setup();
    render(
      <CreateRoomModal open={true} onClose={vi.fn()} onCreated={vi.fn()} />,
    );

    await user.click(screen.getByRole("button", { name: ru.rooms.create }));

    await waitFor(() => {
      expect(screen.getByText(ru.rooms.nameMin)).toBeInTheDocument();
    });
  });

  it("успех → onCreated и onClose вызваны", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    const onClose = vi.fn();
    vi.spyOn(useRoomsStore.getState(), "createRoom").mockResolvedValue(
      createdRoom,
    );
    // Подменяем getState, чтобы форма получала свежий мок.
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      createRoom: vi.fn().mockResolvedValue(createdRoom),
    });

    render(
      <CreateRoomModal
        open={true}
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    await user.type(screen.getByLabelText(ru.rooms.nameLabel), "general");
    await user.click(screen.getByRole("button", { name: ru.rooms.create }));

    await waitFor(() => {
      expect(onCreated).toHaveBeenCalledWith(createdRoom);
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it("fail → onCreated НЕ вызван, форма не закрывается", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    const onClose = vi.fn();
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      createRoom: vi.fn().mockResolvedValue(null),
    });

    render(
      <CreateRoomModal
        open={true}
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    await user.type(screen.getByLabelText(ru.rooms.nameLabel), "general");
    await user.click(screen.getByRole("button", { name: ru.rooms.create }));

    await waitFor(() => {
      // createRoom вызвался, но onCreated/onClose — нет.
      expect(onCreated).not.toHaveBeenCalled();
      expect(onClose).not.toHaveBeenCalled();
    });
  });
});
