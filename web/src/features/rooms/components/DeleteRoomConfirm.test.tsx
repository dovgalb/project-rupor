// Тесты <DeleteRoomConfirm />.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DeleteRoomConfirm } from "@/features/rooms/components/DeleteRoomConfirm";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";

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

describe("<DeleteRoomConfirm />", () => {
  beforeEach(() => {
    resetStore();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetStore();
  });

  it("рендерит warning и кнопку с aria-label содержащей имя комнаты", () => {
    render(
      <DeleteRoomConfirm
        open={true}
        roomId="r1"
        roomName="General"
        onCancel={vi.fn()}
        onDeleted={vi.fn()}
      />,
    );
    expect(screen.getByText(ru.rooms.deleteWarning)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: `${ru.rooms.delete} General` }),
    ).toBeInTheDocument();
  });

  it("click «Отмена» → onCancel вызван", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(
      <DeleteRoomConfirm
        open={true}
        roomId="r1"
        roomName="General"
        onCancel={onCancel}
        onDeleted={vi.fn()}
      />,
    );
    await user.click(screen.getByRole("button", { name: ru.rooms.cancel }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("click «Удалить» → deleteRoom вызван + onDeleted на success", async () => {
    const user = userEvent.setup();
    const onDeleted = vi.fn();
    const deleteRoomMock = vi.fn().mockResolvedValue(true);
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      deleteRoom: deleteRoomMock,
    });

    render(
      <DeleteRoomConfirm
        open={true}
        roomId="r1"
        roomName="General"
        onCancel={vi.fn()}
        onDeleted={onDeleted}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: `${ru.rooms.delete} General` }),
    );

    await waitFor(() => {
      expect(deleteRoomMock).toHaveBeenCalledWith("r1");
      expect(onDeleted).toHaveBeenCalledTimes(1);
    });
  });
});
