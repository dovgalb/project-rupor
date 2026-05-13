import { describe, expect, it } from "vitest";

import { formatDateTime, formatRelative, formatTime } from "./formatDate";

describe("formatTime", () => {
  it("возвращает HH:MM в 24-часовом формате", () => {
    // 2026-05-13T14:32:00Z даст разное время в зависимости от TZ.
    // Используем локальное создание Date, чтобы тест был детерминистичен.
    const local = new Date(2026, 4, 13, 14, 32, 0);
    expect(formatTime(local.toISOString())).toBe("14:32");
  });

  it("возвращает пустую строку для невалидной даты", () => {
    expect(formatTime("not-a-date")).toBe("");
  });
});

describe("formatDateTime", () => {
  it("возвращает dd.MM.yyyy, HH:MM", () => {
    const local = new Date(2026, 4, 13, 14, 32, 0);
    // Intl-ru вставляет запятую между датой и временем.
    expect(formatDateTime(local.toISOString())).toBe("13.05.2026, 14:32");
  });
});

describe("formatRelative", () => {
  const now = new Date("2026-05-13T14:00:00.000Z");

  it("< 60 секунд → 'только что'", () => {
    const ts = new Date(now.getTime() - 30 * 1000).toISOString();
    expect(formatRelative(ts, now)).toBe("только что");
  });

  it("1..59 минут → 'N мин назад'", () => {
    const ts = new Date(now.getTime() - 5 * 60 * 1000).toISOString();
    expect(formatRelative(ts, now)).toBe("5 мин назад");
  });

  it("1..23 часов → 'N ч назад'", () => {
    const ts = new Date(now.getTime() - 3 * 60 * 60 * 1000).toISOString();
    expect(formatRelative(ts, now)).toBe("3 ч назад");
  });

  it("1..6 дней → 'N дн назад'", () => {
    const ts = new Date(now.getTime() - 2 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelative(ts, now)).toBe("2 дн назад");
  });

  it("7+ дней → формат полной даты", () => {
    const past = new Date(2026, 4, 1, 9, 15, 0);
    const result = formatRelative(past.toISOString(), now);
    expect(result).toMatch(/^01\.05\.2026, \d{2}:\d{2}$/);
  });
});
