// Тесты маппера room-кодов в human-readable сообщения.

import { describe, expect, it } from "vitest";

import { mapRoomErrorToMessage } from "@/features/rooms/store/mapErrors";
import { ru } from "@/shared/lib/i18n/ru";

describe("mapRoomErrorToMessage", () => {
  it("ROOM-001 → invalidName", () => {
    expect(mapRoomErrorToMessage("ROOM-001")).toBe(
      ru.rooms.errors.invalidName,
    );
  });

  it("ROOM-005 → onlyOwnerCanDelete", () => {
    expect(mapRoomErrorToMessage("ROOM-005")).toBe(
      ru.rooms.errors.onlyOwnerCanDelete,
    );
  });

  it("ROOM-008 → invalidInviteCode", () => {
    expect(mapRoomErrorToMessage("ROOM-008")).toBe(
      ru.rooms.errors.invalidInviteCode,
    );
  });

  it("INTERNAL → common.error", () => {
    expect(mapRoomErrorToMessage("INTERNAL")).toBe(ru.common.error);
  });

  it("NETWORK → networkError", () => {
    expect(mapRoomErrorToMessage("NETWORK")).toBe(
      ru.rooms.errors.networkError,
    );
  });

  it("неизвестный код → unknown", () => {
    expect(mapRoomErrorToMessage("ROOM-999")).toBe(ru.rooms.errors.unknown);
  });
});
