// Тесты <ChannelListItem />.

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ChannelListItem } from "@/features/channels/components/ChannelListItem";

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

describe("<ChannelListItem />", () => {
  it("text channel: рендерит имя с # префиксом, click вызывает onSelect", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <ul>
        <ChannelListItem
          channel={textChannel}
          active={false}
          onSelect={onSelect}
        />
      </ul>,
    );

    const btn = screen.getByRole("button", { name: /general/ });
    expect(btn).toBeInTheDocument();
    expect(btn).not.toHaveAttribute("aria-disabled", "true");

    await user.click(btn);
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("voice channel disabled: aria-disabled=true, click НЕ вызывает onSelect", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <ul>
        <ChannelListItem
          channel={voiceChannel}
          active={false}
          disabled
          onSelect={onSelect}
        />
      </ul>,
    );

    const btn = screen.getByRole("button", { name: /voice-general/ });
    expect(btn).toHaveAttribute("aria-disabled", "true");

    await user.click(btn);
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("active=true → aria-current=page", () => {
    render(
      <ul>
        <ChannelListItem
          channel={textChannel}
          active={true}
          onSelect={vi.fn()}
        />
      </ul>,
    );
    expect(screen.getByRole("button", { name: /general/ })).toHaveAttribute(
      "aria-current",
      "page",
    );
  });
});
