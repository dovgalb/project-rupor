// Чистая функция-маппер кода доменной ошибки chat в human-readable сообщение.
// Тексты — из i18n/ru.ts. Покрывает CHAT-001..007 + INTERNAL/NETWORK/default.
//
// Маппинг:
//   CHAT-001 → invalidText
//   CHAT-002 → channelNotFound
//   CHAT-003 → notTextChannel
//   CHAT-004 → notMember
//   CHAT-005 → invalidUuid (bug-баннер-кейс)
//   CHAT-006 → invalidLimit
//   CHAT-007 → unsupportedType (bug-баннер-кейс)

import { ru } from "@/shared/lib/i18n/ru";

export function mapChatErrorToMessage(code: string): string {
  switch (code) {
    case "CHAT-001":
      return ru.chat.errors.invalidText;
    case "CHAT-002":
      return ru.chat.errors.channelNotFound;
    case "CHAT-003":
      return ru.chat.errors.notTextChannel;
    case "CHAT-004":
      return ru.chat.errors.notMember;
    case "CHAT-005":
      return ru.chat.errors.invalidUuid;
    case "CHAT-006":
      return ru.chat.errors.invalidLimit;
    case "CHAT-007":
      return ru.chat.errors.unsupportedType;
    case "INTERNAL":
      return ru.chat.errors.internalError;
    case "NETWORK":
      return ru.chat.errors.networkError;
    default:
      return ru.chat.errors.unknown;
  }
}
