// Чистая функция-маппер кода доменной ошибки rooms в human-readable сообщение.
// Тексты — из i18n/ru.ts. Покрывает ROOM-001..009 + INTERNAL/NETWORK/default.

import { ru } from "@/shared/lib/i18n/ru";

export function mapRoomErrorToMessage(code: string): string {
  switch (code) {
    case "ROOM-001":
      return ru.rooms.errors.invalidName;
    case "ROOM-002":
      return ru.rooms.errors.notFound;
    case "ROOM-003":
      return ru.rooms.errors.notMember;
    case "ROOM-004":
      return ru.rooms.errors.adminOrOwnerRequired;
    case "ROOM-005":
      return ru.rooms.errors.onlyOwnerCanDelete;
    case "ROOM-006":
      return ru.rooms.errors.alreadyMember;
    case "ROOM-007":
      return ru.rooms.errors.inviteNotFound;
    case "ROOM-008":
      return ru.rooms.errors.invalidInviteCode;
    case "ROOM-009":
      return ru.rooms.errors.invalidBody;
    case "INTERNAL":
      return ru.common.error;
    case "NETWORK":
      return ru.rooms.errors.networkError;
    default:
      return ru.rooms.errors.unknown;
  }
}
