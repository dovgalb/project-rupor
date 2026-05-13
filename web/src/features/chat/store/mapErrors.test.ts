// Тесты маппера chat-кодов в human-readable сообщения.

import { describe, expect, it } from "vitest";

import { mapChatErrorToMessage } from "@/features/chat/store/mapErrors";
import { ru } from "@/shared/lib/i18n/ru";

describe("mapChatErrorToMessage", () => {
  it("CHAT-001 → invalidText", () => {
    expect(mapChatErrorToMessage("CHAT-001")).toBe(ru.chat.errors.invalidText);
  });

  it("CHAT-002 → channelNotFound", () => {
    expect(mapChatErrorToMessage("CHAT-002")).toBe(
      ru.chat.errors.channelNotFound,
    );
  });

  it("CHAT-003 → notTextChannel", () => {
    expect(mapChatErrorToMessage("CHAT-003")).toBe(
      ru.chat.errors.notTextChannel,
    );
  });

  it("CHAT-004 → notMember", () => {
    expect(mapChatErrorToMessage("CHAT-004")).toBe(ru.chat.errors.notMember);
  });

  it("CHAT-005 → invalidUuid", () => {
    expect(mapChatErrorToMessage("CHAT-005")).toBe(ru.chat.errors.invalidUuid);
  });

  it("CHAT-006 → invalidLimit", () => {
    expect(mapChatErrorToMessage("CHAT-006")).toBe(ru.chat.errors.invalidLimit);
  });

  it("CHAT-007 → unsupportedType", () => {
    expect(mapChatErrorToMessage("CHAT-007")).toBe(
      ru.chat.errors.unsupportedType,
    );
  });

  it("INTERNAL → internalError", () => {
    expect(mapChatErrorToMessage("INTERNAL")).toBe(
      ru.chat.errors.internalError,
    );
  });

  it("NETWORK → networkError", () => {
    expect(mapChatErrorToMessage("NETWORK")).toBe(ru.chat.errors.networkError);
  });

  it("неизвестный код → unknown", () => {
    expect(mapChatErrorToMessage("CHAT-999")).toBe(ru.chat.errors.unknown);
  });
});
