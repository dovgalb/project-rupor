// Чистая функция-маппер кода доменной ошибки channels в human-readable сообщение.
// Тексты — из i18n/ru.ts. Покрывает CHANNEL-001..007 + INTERNAL/NETWORK/default.
//
// Маппинг:
//   CHANNEL-001 → invalidName
//   CHANNEL-002 → invalidKind (bug-баннер-кейс, но текст оставляем)
//   CHANNEL-003 → notFound
//   CHANNEL-004 → nameTaken
//   CHANNEL-005 → invalidBody (bug-баннер-кейс)
//   CHANNEL-006 → notMember
//   CHANNEL-007 → adminOrOwnerRequired

import { ru } from "@/shared/lib/i18n/ru";

export function mapChannelErrorToMessage(code: string): string {
  switch (code) {
    case "CHANNEL-001":
      return ru.channels.errors.invalidName;
    case "CHANNEL-002":
      return ru.channels.errors.invalidKind;
    case "CHANNEL-003":
      return ru.channels.errors.notFound;
    case "CHANNEL-004":
      return ru.channels.errors.nameTaken;
    case "CHANNEL-005":
      return ru.channels.errors.invalidBody;
    case "CHANNEL-006":
      return ru.channels.errors.notMember;
    case "CHANNEL-007":
      return ru.channels.errors.adminOrOwnerRequired;
    case "INTERNAL":
      return ru.common.error;
    case "NETWORK":
      return ru.channels.errors.networkError;
    default:
      return ru.channels.errors.unknown;
  }
}
