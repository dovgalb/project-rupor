// Тесты <MessageComposer />: render, disabled state, counter, keyDown.

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { MessageComposer } from "@/features/chat/components/MessageComposer";
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

describe("<MessageComposer />", () => {
  beforeEach(() => {
    resetStore();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    resetStore();
  });

  it("рендерит textarea и кнопку 'Отправить'", () => {
    render(<MessageComposer channelId="c1" />);
    expect(screen.getByLabelText(ru.chat.messageLabel)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: ru.chat.sendBtn }),
    ).toBeInTheDocument();
  });

  it("wsStatus=idle → textarea и кнопка disabled, placeholder = 'Соединение недоступно'", () => {
    render(<MessageComposer channelId="c1" />);

    const textarea = screen.getByLabelText(ru.chat.messageLabel);
    expect(textarea).toBeDisabled();
    expect(textarea).toHaveAttribute("placeholder", ru.chat.composerDisabledHint);
    const sendBtn = screen.getByRole("button", { name: ru.chat.sendBtn });
    expect(sendBtn).toBeDisabled();
  });

  it("wsStatus=open → элементы не disabled и placeholder обычный", () => {
    useChatStore.setState({ wsStatus: "open" });
    render(<MessageComposer channelId="c1" />);

    const textarea = screen.getByLabelText(ru.chat.messageLabel);
    expect(textarea).not.toBeDisabled();
    expect(textarea).toHaveAttribute("placeholder", ru.chat.composerPlaceholder);
    expect(
      screen.getByRole("button", { name: ru.chat.sendBtn }),
    ).not.toBeDisabled();
  });

  it("counter появляется когда текст > 3000 символов", async () => {
    useChatStore.setState({ wsStatus: "open" });
    const user = userEvent.setup();
    render(<MessageComposer channelId="c1" />);

    const textarea = screen.getByLabelText(ru.chat.messageLabel);
    // user.type() с длинным текстом медленный; используем paste.
    const longText = "x".repeat(3500);
    await user.click(textarea);
    await user.paste(longText);

    expect(screen.getByText(`${longText.length}/4000`)).toBeInTheDocument();
  });

  it("Shift+Enter не сабмитит форму (новая строка)", async () => {
    useChatStore.setState({ wsStatus: "open" });
    const user = userEvent.setup();
    render(<MessageComposer channelId="c1" />);

    const textarea = screen.getByLabelText(ru.chat.messageLabel);
    await user.click(textarea);
    await user.keyboard("a{Shift>}{Enter}{/Shift}b");

    // Текст содержит "a\nb" — Enter c Shift не привёл к reset.
    expect((textarea as HTMLTextAreaElement).value).toContain("a");
    expect((textarea as HTMLTextAreaElement).value).toContain("b");
  });
});
