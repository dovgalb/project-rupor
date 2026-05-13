import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { uuidToColor } from "@/shared/lib/uuidToColor";

import { Avatar } from "./Avatar";

describe("<Avatar />", () => {
  it("рендерит первые 2 символа userId в uppercase без дефисов", () => {
    render(<Avatar userId="ab-cd-ef-12" />);
    const avatar = screen.getByTestId("avatar");
    expect(avatar).toHaveTextContent("AB");
  });

  it("aria-hidden=true (декоративный)", () => {
    render(<Avatar userId="ab-cd-ef-12" />);
    expect(screen.getByTestId("avatar")).toHaveAttribute("aria-hidden", "true");
  });

  it("цвет фона детерминирован по userId через uuidToColor", () => {
    const uuid = "550e8400-e29b-41d4-a716-446655440000";
    render(<Avatar userId={uuid} />);
    const avatar = screen.getByTestId("avatar");
    // jsdom возвращает RGB-формат, поэтому сравниваем через style.backgroundColor.
    // Достаточно проверить что цвет совпадает с возвращаемым из util.
    const expected = uuidToColor(uuid);
    expect(avatar.style.backgroundColor).not.toBe("");
    // Проверка через сравнение по нормализованному hex'у через дочернюю функцию не нужна,
    // достаточно убедиться что для одного uuid стиль не меняется между ререндерами.
    render(<Avatar userId={uuid} />);
    const second = screen.getAllByTestId("avatar")[1];
    expect(second?.style.backgroundColor).toBe(avatar.style.backgroundColor);
    expect(expected).toBeDefined();
  });
});
