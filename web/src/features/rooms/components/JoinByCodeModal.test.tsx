// Тесты <JoinByCodeModal />.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { JoinByCodeModal } from "@/features/rooms/components/JoinByCodeModal";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";
import { useToastsStore } from "@/shared/ui/useToast";

import type { Room } from "@/features/rooms/types";

const roomFixture: Room = {
  id: "r1",
  ownerId: "u1",
  name: "general",
  createdAt: "2030",
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

describe("<JoinByCodeModal />", () => {
  beforeEach(() => {
    resetStore();
    useToastsStore.setState({ items: [] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetStore();
    useToastsStore.setState({ items: [] });
  });

  it("zod-валидация: короткий код → inline error", async () => {
    const user = userEvent.setup();
    render(
      <JoinByCodeModal open={true} onClose={vi.fn()} onJoined={vi.fn()} />,
    );

    await user.type(screen.getByLabelText(ru.rooms.codeLabel), "ABCD");
    await user.click(screen.getByRole("button", { name: ru.rooms.joinBtn }));

    await waitFor(() => {
      expect(screen.getByText(ru.rooms.codeFormat)).toBeInTheDocument();
    });
  });

  it("zod-валидация: запрещённый символ I → inline error", async () => {
    const user = userEvent.setup();
    render(
      <JoinByCodeModal open={true} onClose={vi.fn()} onJoined={vi.fn()} />,
    );

    await user.type(screen.getByLabelText(ru.rooms.codeLabel), "ABCDEFGI");
    await user.click(screen.getByRole("button", { name: ru.rooms.joinBtn }));

    await waitFor(() => {
      expect(screen.getByText(ru.rooms.codeFormat)).toBeInTheDocument();
    });
  });

  it("успех → joinByCode вызван с upper-cased кодом, onJoined и onClose", async () => {
    const user = userEvent.setup();
    const onJoined = vi.fn();
    const onClose = vi.fn();
    const joinByCodeMock = vi
      .fn()
      .mockResolvedValue({ ok: true, room: roomFixture });
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      joinByCode: joinByCodeMock,
    });

    render(
      <JoinByCodeModal open={true} onClose={onClose} onJoined={onJoined} />,
    );

    // Печатаем в нижнем регистре — компонент сам делает upper-case в onChange.
    await user.type(screen.getByLabelText(ru.rooms.codeLabel), "abcd1234");
    await user.click(screen.getByRole("button", { name: ru.rooms.joinBtn }));

    await waitFor(() => {
      expect(joinByCodeMock).toHaveBeenCalledWith("ABCD1234");
      expect(onJoined).toHaveBeenCalledWith(roomFixture);
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it("ROOM-007 → form error на code", async () => {
    const user = userEvent.setup();
    const joinByCodeMock = vi
      .fn()
      .mockResolvedValue({ ok: false, code: "ROOM-007" });
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      joinByCode: joinByCodeMock,
    });

    render(
      <JoinByCodeModal open={true} onClose={vi.fn()} onJoined={vi.fn()} />,
    );

    await user.type(screen.getByLabelText(ru.rooms.codeLabel), "ABCD1234");
    await user.click(screen.getByRole("button", { name: ru.rooms.joinBtn }));

    await waitFor(() => {
      expect(
        screen.getByText(ru.rooms.errors.inviteNotFound),
      ).toBeInTheDocument();
    });
  });

  it("ROOM-006 → toast.info + onClose, onJoined НЕ вызван", async () => {
    const user = userEvent.setup();
    const onJoined = vi.fn();
    const onClose = vi.fn();
    const joinByCodeMock = vi
      .fn()
      .mockResolvedValue({ ok: false, code: "ROOM-006" });
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      joinByCode: joinByCodeMock,
    });

    render(
      <JoinByCodeModal open={true} onClose={onClose} onJoined={onJoined} />,
    );

    await user.type(screen.getByLabelText(ru.rooms.codeLabel), "ABCD1234");
    await user.click(screen.getByRole("button", { name: ru.rooms.joinBtn }));

    await waitFor(() => {
      const toasts = useToastsStore.getState().items;
      const hasInfo = toasts.some((t) => t.variant === "info");
      expect(hasInfo).toBe(true);
      expect(onClose).toHaveBeenCalledTimes(1);
      expect(onJoined).not.toHaveBeenCalled();
    });
  });
});
