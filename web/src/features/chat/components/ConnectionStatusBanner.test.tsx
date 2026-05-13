// Тесты <ConnectionStatusBanner />.

import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { ConnectionStatusBanner } from "@/features/chat/components/ConnectionStatusBanner";
import { useChatStore } from "@/features/chat/store";
import { ru } from "@/shared/lib/i18n/ru";

function resetStore(): void {
  useChatStore.setState({
    messagesByChannel: {},
    nextBeforeByChannel: {},
    loadingHistoryByChannel: {},
    loadingMoreHistoryByChannel: {},
    subscribedChannelId: null,
    wsStatus: "idle",
  });
}

describe("<ConnectionStatusBanner />", () => {
  beforeEach(() => {
    resetStore();
  });

  afterEach(() => {
    resetStore();
  });

  it("wsStatus=idle/open/closed → банер не рендерится", () => {
    const { rerender, container } = render(<ConnectionStatusBanner />);
    expect(container.firstChild).toBeNull();

    useChatStore.setState({ wsStatus: "open" });
    rerender(<ConnectionStatusBanner />);
    expect(container.firstChild).toBeNull();

    useChatStore.setState({ wsStatus: "closed" });
    rerender(<ConnectionStatusBanner />);
    expect(container.firstChild).toBeNull();
  });

  it("wsStatus=connecting → виден баннер с текстом и role=status", () => {
    useChatStore.setState({ wsStatus: "connecting" });
    render(<ConnectionStatusBanner />);

    const banner = screen.getByRole("status");
    expect(banner).toHaveTextContent(ru.chat.connecting);
    expect(banner).toHaveAttribute("aria-live", "polite");
  });

  it("wsStatus=reconnecting → виден баннер с текстом reconnecting", () => {
    useChatStore.setState({ wsStatus: "reconnecting" });
    render(<ConnectionStatusBanner />);

    expect(screen.getByRole("status")).toHaveTextContent(ru.chat.reconnecting);
  });
});
