// Тесты tokenStorage: round-trip и cleanup.

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { tokenStorage, type StoredTokens } from "@/shared/api/token-storage";

const sample: StoredTokens = {
  access: "access-jwt-value",
  refresh: "refresh-base64url-value",
  accessExpiresAt: "2030-01-01T00:00:00Z",
  refreshExpiresAt: "2030-01-08T00:00:00Z",
};

describe("tokenStorage", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  it("без предварительного set все getter возвращают null", () => {
    expect(tokenStorage.getAccess()).toBeNull();
    expect(tokenStorage.getRefresh()).toBeNull();
    expect(tokenStorage.getAccessExpiresAt()).toBeNull();
    expect(tokenStorage.getRefreshExpiresAt()).toBeNull();
  });

  it("round-trip: set записывает, getter'ы возвращают записанное", () => {
    tokenStorage.set(sample);

    expect(tokenStorage.getAccess()).toBe(sample.access);
    expect(tokenStorage.getRefresh()).toBe(sample.refresh);
    expect(tokenStorage.getAccessExpiresAt()).toBe(sample.accessExpiresAt);
    expect(tokenStorage.getRefreshExpiresAt()).toBe(sample.refreshExpiresAt);
  });

  it("set записывает значения в 4 ключа rupor.*", () => {
    tokenStorage.set(sample);

    expect(localStorage.getItem("rupor.access")).toBe(sample.access);
    expect(localStorage.getItem("rupor.refresh")).toBe(sample.refresh);
    expect(localStorage.getItem("rupor.accessExpiresAt")).toBe(sample.accessExpiresAt);
    expect(localStorage.getItem("rupor.refreshExpiresAt")).toBe(sample.refreshExpiresAt);
  });

  it("clear удаляет все 4 ключа", () => {
    tokenStorage.set(sample);
    tokenStorage.clear();

    expect(tokenStorage.getAccess()).toBeNull();
    expect(tokenStorage.getRefresh()).toBeNull();
    expect(tokenStorage.getAccessExpiresAt()).toBeNull();
    expect(tokenStorage.getRefreshExpiresAt()).toBeNull();
  });
});
