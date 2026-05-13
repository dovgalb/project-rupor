// Чистая функция-маппер кода доменной ошибки в human-readable сообщение.
// Принимает просто string, потому что коды приходят из разных union-ов
// (AuthErrorCode + ClientErrorCode: NETWORK/CLIENT/INTERNAL).
// Тексты — из i18n/ru.ts.

import { ru } from "@/shared/lib/i18n/ru";

export function mapAuthErrorToMessage(code: string): string {
  switch (code) {
    case "AUTH-001":
      return ru.auth.invalidEmail;
    case "AUTH-002":
      return ru.auth.usernameFormat;
    case "AUTH-003":
      return ru.auth.passwordMin;
    case "AUTH-004":
      return ru.auth.emailTaken;
    case "AUTH-005":
      return ru.auth.usernameTaken;
    case "AUTH-006":
      return ru.auth.invalidCredentials;
    case "AUTH-012":
    case "CLIENT":
      return ru.auth.genericError;
    case "INTERNAL":
      return ru.auth.internalError;
    case "NETWORK":
      return ru.auth.networkError;
    default:
      return ru.auth.genericError;
  }
}
