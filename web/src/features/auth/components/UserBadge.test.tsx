// Тесты UserBadge: рендер username, click → onLogout, aria-label.

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { UserBadge } from "@/features/auth/components/UserBadge";
import { ru } from "@/shared/lib/i18n/ru";

import type { CurrentUser } from "@/features/auth/types";

const user: CurrentUser = {
  id: "11111111-2222-3333-4444-555555555555",
  email: "u@example.com",
  username: "user_1",
  createdAt: "2025-01-01T00:00:00Z",
};

describe("<UserBadge />", () => {
  it("рендерит username и email-tooltip через title", () => {
    render(<UserBadge user={user} onLogout={vi.fn()} />);
    const usernameEl = screen.getByText("user_1");
    expect(usernameEl).toBeInTheDocument();
    expect(usernameEl).toHaveAttribute("title", "u@example.com");
  });

  it("click на logout-кнопку вызывает onLogout", async () => {
    const onLogout = vi.fn();
    const u = userEvent.setup();
    render(<UserBadge user={user} onLogout={onLogout} />);

    await u.click(screen.getByRole("button", { name: ru.auth.logoutBtn }));
    expect(onLogout).toHaveBeenCalledTimes(1);
  });

  it("aria-label на кнопке logout", () => {
    render(<UserBadge user={user} onLogout={vi.fn()} />);
    const btn = screen.getByRole("button", { name: ru.auth.logoutBtn });
    expect(btn).toHaveAttribute("aria-label", ru.auth.logoutBtn);
  });
});
