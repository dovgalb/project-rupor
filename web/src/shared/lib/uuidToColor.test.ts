import { describe, expect, it } from "vitest";

import { AVATAR_PALETTE, uuidToColor } from "./uuidToColor";

describe("uuidToColor", () => {
  it("возвращает детерминированный цвет для одного uuid", () => {
    const uuid = "550e8400-e29b-41d4-a716-446655440000";
    expect(uuidToColor(uuid)).toBe(uuidToColor(uuid));
  });

  it("разные uuid могут давать разные цвета", () => {
    const colors = new Set([
      uuidToColor("550e8400-e29b-41d4-a716-446655440000"),
      uuidToColor("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
      uuidToColor("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
      uuidToColor("00000000-0000-0000-0000-000000000001"),
    ]);
    expect(colors.size).toBeGreaterThan(1);
  });

  it("индекс всегда внутри палитры (цвет принадлежит палитре)", () => {
    const samples = [
      "550e8400-e29b-41d4-a716-446655440000",
      "00000000-0000-0000-0000-000000000000",
      "ffffffff-ffff-ffff-ffff-ffffffffffff",
      "12345678-1234-1234-1234-123456789012",
      "abcdefab-cdef-abcd-efab-cdefabcdefab",
    ];
    for (const uuid of samples) {
      expect(AVATAR_PALETTE).toContain(uuidToColor(uuid));
    }
  });
});
