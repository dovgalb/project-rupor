// Тесты маппера channel-кодов в human-readable сообщения.

import { describe, expect, it } from "vitest";

import { mapChannelErrorToMessage } from "@/features/channels/store/mapErrors";
import { ru } from "@/shared/lib/i18n/ru";

describe("mapChannelErrorToMessage", () => {
  it("CHANNEL-001 → invalidName", () => {
    expect(mapChannelErrorToMessage("CHANNEL-001")).toBe(
      ru.channels.errors.invalidName,
    );
  });

  it("CHANNEL-002 → invalidKind", () => {
    expect(mapChannelErrorToMessage("CHANNEL-002")).toBe(
      ru.channels.errors.invalidKind,
    );
  });

  it("CHANNEL-003 → notFound", () => {
    expect(mapChannelErrorToMessage("CHANNEL-003")).toBe(
      ru.channels.errors.notFound,
    );
  });

  it("CHANNEL-004 → nameTaken", () => {
    expect(mapChannelErrorToMessage("CHANNEL-004")).toBe(
      ru.channels.errors.nameTaken,
    );
  });

  it("CHANNEL-006 → notMember", () => {
    expect(mapChannelErrorToMessage("CHANNEL-006")).toBe(
      ru.channels.errors.notMember,
    );
  });

  it("CHANNEL-007 → adminOrOwnerRequired", () => {
    expect(mapChannelErrorToMessage("CHANNEL-007")).toBe(
      ru.channels.errors.adminOrOwnerRequired,
    );
  });

  it("INTERNAL → common.error", () => {
    expect(mapChannelErrorToMessage("INTERNAL")).toBe(ru.common.error);
  });

  it("NETWORK → networkError", () => {
    expect(mapChannelErrorToMessage("NETWORK")).toBe(
      ru.channels.errors.networkError,
    );
  });

  it("неизвестный код → unknown", () => {
    expect(mapChannelErrorToMessage("CHANNEL-999")).toBe(
      ru.channels.errors.unknown,
    );
  });
});
