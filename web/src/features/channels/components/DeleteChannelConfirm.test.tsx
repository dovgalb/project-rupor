// Тесты <DeleteChannelConfirm />.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DeleteChannelConfirm } from "@/features/channels/components/DeleteChannelConfirm";
import { useChannelsStore } from "@/features/channels/store";
import { ru } from "@/shared/lib/i18n/ru";

function resetStore(): void {
  useChannelsStore.setState({
    channelsByRoom: {},
    loadingByRoom: {},
    errorByRoom: {},
    activeChannelId: null,
  });
}

describe("<DeleteChannelConfirm />", () => {
  beforeEach(() => {
    resetStore();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetStore();
  });

  it("рендерит warning и кнопку с aria-label содержащей имя канала", () => {
    render(
      <DeleteChannelConfirm
        open={true}
        roomId="r1"
        channelId="c1"
        channelName="general"
        onCancel={vi.fn()}
        onDeleted={vi.fn()}
      />,
    );
    expect(screen.getByText(ru.channels.deleteWarning)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: `${ru.channels.delete} general` }),
    ).toBeInTheDocument();
  });

  it("click «Отмена» → onCancel вызван", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(
      <DeleteChannelConfirm
        open={true}
        roomId="r1"
        channelId="c1"
        channelName="general"
        onCancel={onCancel}
        onDeleted={vi.fn()}
      />,
    );
    await user.click(screen.getByRole("button", { name: ru.channels.cancel }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("click «Удалить» → deleteChannel вызван + onDeleted на success", async () => {
    const user = userEvent.setup();
    const onDeleted = vi.fn();
    const deleteChannelMock = vi.fn().mockResolvedValue(true);
    vi.spyOn(useChannelsStore, "getState").mockReturnValue({
      ...useChannelsStore.getState(),
      deleteChannel: deleteChannelMock,
    });

    render(
      <DeleteChannelConfirm
        open={true}
        roomId="r1"
        channelId="c1"
        channelName="general"
        onCancel={vi.fn()}
        onDeleted={onDeleted}
      />,
    );

    await user.click(
      screen.getByRole("button", { name: `${ru.channels.delete} general` }),
    );

    await waitFor(() => {
      expect(deleteChannelMock).toHaveBeenCalledWith("r1", "c1");
      expect(onDeleted).toHaveBeenCalledTimes(1);
    });
  });
});
