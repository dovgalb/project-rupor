// Тесты маппера auth-кодов в human-readable сообщения.

import { describe, expect, it } from "vitest";

import { mapAuthErrorToMessage } from "@/features/auth/store/mapErrors";
import { ru } from "@/shared/lib/i18n/ru";

describe("mapAuthErrorToMessage", () => {
  it("AUTH-006 → invalidCredentials", () => {
    expect(mapAuthErrorToMessage("AUTH-006")).toBe(ru.auth.invalidCredentials);
  });

  it("AUTH-004 → emailTaken", () => {
    expect(mapAuthErrorToMessage("AUTH-004")).toBe(ru.auth.emailTaken);
  });

  it("AUTH-001 → invalidEmail", () => {
    expect(mapAuthErrorToMessage("AUTH-001")).toBe(ru.auth.invalidEmail);
  });

  it("INTERNAL → internalError", () => {
    expect(mapAuthErrorToMessage("INTERNAL")).toBe(ru.auth.internalError);
  });

  it("NETWORK → networkError", () => {
    expect(mapAuthErrorToMessage("NETWORK")).toBe(ru.auth.networkError);
  });

  it("неизвестный код → genericError", () => {
    expect(mapAuthErrorToMessage("UNKNOWN-999")).toBe(ru.auth.genericError);
  });
});
