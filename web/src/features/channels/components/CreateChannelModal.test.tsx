// Тесты <CreateChannelModal />.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CreateChannelModal } from "@/features/channels/components/CreateChannelModal";
import { useChannelsStore } from "@/features/channels/store";
import { ru } from "@/shared/lib/i18n/ru";
import { useToastsStore } from "@/shared/ui/useToast";

import type { Channel } from "@/features/channels/types";

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

function resetChannelsStore(): void {
  useChannelsStore.setState({
    channelsByRoom: {},
    loadingByRoom: {},
    errorByRoom: {},
    activeChannelId: null,
  });
}

function resetToasts(): void {
  useToastsStore.setState({ items: [] });
}

describe("<CreateChannelModal />", () => {
  beforeEach(() => {
    resetChannelsStore();
    resetToasts();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetChannelsStore();
    resetToasts();
  });

  it("open=false → не рендерится", () => {
    render(
      <CreateChannelModal
        open={false}
        roomId="r1"
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("open=true → рендерит форму с input, radio и кнопками", () => {
    render(
      <CreateChannelModal
        open={true}
        roomId="r1"
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByLabelText(ru.channels.nameLabel)).toBeInTheDocument();
    expect(screen.getByLabelText(ru.channels.kindText)).toBeInTheDocument();
    expect(screen.getByLabelText(ru.channels.kindVoice)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: ru.channels.create }),
    ).toBeInTheDocument();
  });

  it("zod-валидация: пустое имя → inline error", async () => {
    const user = userEvent.setup();
    render(
      <CreateChannelModal
        open={true}
        roomId="r1"
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: ru.channels.create }));

    await waitFor(() => {
      expect(
        screen.getByText(ru.channels.errors.nameRequired),
      ).toBeInTheDocument();
    });
  });

  it("успех с kind=text → onCreated и onClose вызваны, voice toast НЕ показан", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    const onClose = vi.fn();
    vi.spyOn(useChannelsStore, "getState").mockReturnValue({
      ...useChannelsStore.getState(),
      createChannel: vi.fn().mockResolvedValue(textChannel),
    });

    render(
      <CreateChannelModal
        open={true}
        roomId="r1"
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    await user.type(
      screen.getByLabelText(ru.channels.nameLabel),
      "general",
    );
    await user.click(screen.getByRole("button", { name: ru.channels.create }));

    await waitFor(() => {
      expect(onCreated).toHaveBeenCalledWith(textChannel);
      expect(onClose).toHaveBeenCalledTimes(1);
    });
    // voice-toast не должен быть показан для text-канала.
    expect(useToastsStore.getState().items.length).toBe(0);
  });

  it("успех с kind=voice → toast.info со специальной подсказкой", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    const onClose = vi.fn();
    vi.spyOn(useChannelsStore, "getState").mockReturnValue({
      ...useChannelsStore.getState(),
      createChannel: vi.fn().mockResolvedValue(voiceChannel),
    });

    render(
      <CreateChannelModal
        open={true}
        roomId="r1"
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    await user.type(
      screen.getByLabelText(ru.channels.nameLabel),
      "voice-general",
    );
    await user.click(screen.getByLabelText(ru.channels.kindVoice));
    await user.click(screen.getByRole("button", { name: ru.channels.create }));

    await waitFor(() => {
      expect(onCreated).toHaveBeenCalledWith(voiceChannel);
      expect(onClose).toHaveBeenCalledTimes(1);
    });
    // Должен быть один info-toast.
    const items = useToastsStore.getState().items;
    expect(items.length).toBe(1);
    expect(items[0]?.variant).toBe("info");
    expect(items[0]?.message).toBe(ru.channels.voiceCreatedHint);
  });

  it("fail → onCreated НЕ вызван, форма не закрывается", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    const onClose = vi.fn();
    vi.spyOn(useChannelsStore, "getState").mockReturnValue({
      ...useChannelsStore.getState(),
      createChannel: vi.fn().mockResolvedValue(null),
    });

    render(
      <CreateChannelModal
        open={true}
        roomId="r1"
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    await user.type(
      screen.getByLabelText(ru.channels.nameLabel),
      "general",
    );
    await user.click(screen.getByRole("button", { name: ru.channels.create }));

    await waitFor(() => {
      expect(onCreated).not.toHaveBeenCalled();
      expect(onClose).not.toHaveBeenCalled();
    });
  });
});
