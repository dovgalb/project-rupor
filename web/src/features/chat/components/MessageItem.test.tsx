// Тесты <MessageItem />: рендер, статусы, linkify, безопасность ссылок.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MessageItem } from "@/features/chat/components/MessageItem";
import { ru } from "@/shared/lib/i18n/ru";

import type { Message } from "@/features/chat/types";

const baseMessage: Message = {
  id: "m1",
  channelId: "c1",
  authorId: "11111111-2222-3333-4444-555555555555",
  text: "hello world",
  createdAt: "2030-01-01T10:30:00Z",
  status: "committed",
};

describe("<MessageItem />", () => {
  it("рендерит автора, текст и время", () => {
    render(
      <ul>
        <MessageItem
          message={baseMessage}
          isAuthor={false}
          authorPlaceholder="111122"
        />
      </ul>,
    );
    expect(screen.getByText("111122")).toBeInTheDocument();
    expect(screen.getByText("hello world")).toBeInTheDocument();
    // <time> с dateTime атрибутом.
    expect(screen.getByText(/[0-9]+:[0-9]+/)).toBeInTheDocument();
  });

  it("status=pending у автора → показывается hint 'отправляется'", () => {
    render(
      <ul>
        <MessageItem
          message={{ ...baseMessage, status: "pending" }}
          isAuthor={true}
          authorPlaceholder="me"
        />
      </ul>,
    );
    expect(screen.getByText(ru.chat.pending)).toBeInTheDocument();
  });

  it("status=failed у автора → показывается 'Не удалось отправить' и кнопка 'Повторить' (disabled в Phase 8)", () => {
    render(
      <ul>
        <MessageItem
          message={{ ...baseMessage, status: "failed" }}
          isAuthor={true}
          authorPlaceholder="me"
        />
      </ul>,
    );
    expect(screen.getByText(ru.chat.failed)).toBeInTheDocument();
    const retryBtn = screen.getByRole("button", { name: ru.chat.retry });
    expect(retryBtn).toBeDisabled();
  });

  it("status=pending у чужого сообщения → hint НЕ показывается", () => {
    render(
      <ul>
        <MessageItem
          message={{ ...baseMessage, status: "pending" }}
          isAuthor={false}
          authorPlaceholder="abc123"
        />
      </ul>,
    );
    expect(screen.queryByText(ru.chat.pending)).not.toBeInTheDocument();
  });

  it("linkify: http-URL рендерится как <a target=_blank rel=noopener noreferrer>", () => {
    render(
      <ul>
        <MessageItem
          message={{ ...baseMessage, text: "go to https://example.com now" }}
          isAuthor={false}
          authorPlaceholder="abc123"
        />
      </ul>,
    );
    const link = screen.getByRole("link", { name: "https://example.com" });
    expect(link).toHaveAttribute("href", "https://example.com");
    expect(link).toHaveAttribute("target", "_blank");
    expect(link).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("javascript:URL остаётся текстом (R-01 XSS)", () => {
    render(
      <ul>
        <MessageItem
          message={{ ...baseMessage, text: "click javascript:alert(1) here" }}
          isAuthor={false}
          authorPlaceholder="abc123"
        />
      </ul>,
    );
    // Никаких ссылок не должно быть.
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
