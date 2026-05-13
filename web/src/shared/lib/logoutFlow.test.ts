// Тесты registration pattern для logoutFlow.
// Проверяют: очистку tokenStorage, вызов listeners, отписку, изоляцию падений.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { tokenStorage } from "@/shared/api/token-storage";
import {
  _resetLogoutListenersForTests,
  logoutFlow,
  registerLogoutListener,
} from "@/shared/lib/logoutFlow";

describe("logoutFlow", () => {
  beforeEach(() => {
    _resetLogoutListenersForTests();
    localStorage.clear();
    tokenStorage.set({
      access: "a",
      refresh: "r",
      accessExpiresAt: "2030-01-01T00:00:00Z",
      refreshExpiresAt: "2030-01-08T00:00:00Z",
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    _resetLogoutListenersForTests();
    localStorage.clear();
  });

  it("logoutFlow(): очищает tokenStorage", () => {
    expect(tokenStorage.getAccess()).toBe("a");
    logoutFlow();
    expect(tokenStorage.getAccess()).toBeNull();
    expect(tokenStorage.getRefresh()).toBeNull();
    expect(tokenStorage.getAccessExpiresAt()).toBeNull();
    expect(tokenStorage.getRefreshExpiresAt()).toBeNull();
  });

  it("logoutFlow(): вызывает всех зарегистрированных listeners", () => {
    const a = vi.fn();
    const b = vi.fn();
    registerLogoutListener(a);
    registerLogoutListener(b);

    logoutFlow();

    expect(a).toHaveBeenCalledTimes(1);
    expect(b).toHaveBeenCalledTimes(1);
  });

  it("registerLogoutListener: возвращает функцию отписки — после unregister listener не вызывается", () => {
    const listener = vi.fn();
    const unregister = registerLogoutListener(listener);

    unregister();
    logoutFlow();

    expect(listener).not.toHaveBeenCalled();
  });

  it("если один listener бросает — остальные всё равно вызываются", () => {
    // Глушим console.error от logger, чтобы не шуметь в выводе тестов.
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    const bad = vi.fn(() => {
      throw new Error("boom");
    });
    const good1 = vi.fn();
    const good2 = vi.fn();
    registerLogoutListener(good1);
    registerLogoutListener(bad);
    registerLogoutListener(good2);

    expect(() => logoutFlow()).not.toThrow();

    expect(good1).toHaveBeenCalledTimes(1);
    expect(bad).toHaveBeenCalledTimes(1);
    expect(good2).toHaveBeenCalledTimes(1);
    expect(errSpy).toHaveBeenCalled();
  });
});
