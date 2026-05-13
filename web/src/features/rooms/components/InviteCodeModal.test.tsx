// Тесты <InviteCodeModal />.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { InviteCodeModal } from "@/features/rooms/components/InviteCodeModal";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";
import { useToastsStore } from "@/shared/ui/useToast";

import type { InviteCodeDto } from "@/features/rooms/types";

const inviteFixture: InviteCodeDto = {
  code: "ABCD1234",
  createdBy: "u1",
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

describe("<InviteCodeModal />", () => {
  beforeEach(() => {
    resetStore();
    useToastsStore.setState({ items: [] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetStore();
    useToastsStore.setState({ items: [] });
  });

  it("нет кода → показывает placeholder и кнопку Сгенерировать", () => {
    render(<InviteCodeModal open={true} roomId="r1" onClose={vi.fn()} />);
    expect(screen.getByText(ru.rooms.inviteEmpty)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: ru.rooms.generate }),
    ).toBeInTheDocument();
  });

  it("click «Сгенерировать» → regenerateInvite вызван", async () => {
    const user = userEvent.setup();
    const regenerateMock = vi.fn().mockResolvedValue(inviteFixture);
    vi.spyOn(useRoomsStore, "getState").mockReturnValue({
      ...useRoomsStore.getState(),
      regenerateInvite: regenerateMock,
    });

    render(<InviteCodeModal open={true} roomId="r1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: ru.rooms.generate }));

    await waitFor(() => {
      expect(regenerateMock).toHaveBeenCalledWith("r1");
    });
  });

  it("есть код → отображается + кнопка Copy + Generate New", () => {
    useRoomsStore.setState({
      rooms: [],
      status: "idle",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: { r1: inviteFixture },
    });

    render(<InviteCodeModal open={true} roomId="r1" onClose={vi.fn()} />);

    expect(screen.getByText("ABCD1234")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: ru.rooms.copyCode }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: ru.rooms.generateNew }),
    ).toBeInTheDocument();
  });

  it("click «Скопировать» → clipboard.writeText вызван + toast.success", async () => {
    const user = userEvent.setup();
    useRoomsStore.setState({
      rooms: [],
      status: "idle",
      error: null,
      membersByRoom: {},
      membersLoadingByRoom: {},
      activeInviteByRoom: { r1: inviteFixture },
    });

    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });

    render(<InviteCodeModal open={true} roomId="r1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: ru.rooms.copyCode }));

    await waitFor(() => {
      expect(writeText).toHaveBeenCalledWith("ABCD1234");
      const toasts = useToastsStore.getState().items;
      const hasSuccess = toasts.some((t) => t.variant === "success");
      expect(hasSuccess).toBe(true);
    });
  });
});
