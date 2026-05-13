// Тесты AppShell — каркас layout-а.

import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { useAuthStore } from "@/features/auth";

import { AppShell } from "./AppShell";

import type { CurrentUser } from "@/features/auth";

const userFixture: CurrentUser = {
  id: "user-1",
  email: "user@example.com",
  username: "user1",
  createdAt: "2025-01-01T00:00:00Z",
};

function renderShellAt(path: string): void {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/rooms" element={<AppShell />}>
          <Route index element={<div>rooms index outlet</div>} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );
}

describe("<AppShell />", () => {
  beforeEach(() => {
    useAuthStore.setState({
      status: "authenticated",
      currentUser: userFixture,
      error: null,
      lastErrorCode: null,
    });
  });

  afterEach(() => {
    useAuthStore.setState({
      status: "idle",
      currentUser: null,
      error: null,
      lastErrorCode: null,
    });
  });

  it("рендерит вложенный роут через Outlet", () => {
    renderShellAt("/rooms");
    expect(screen.getByText("rooms index outlet")).toBeInTheDocument();
  });

  it("рендерит TopBar (header) и main", () => {
    renderShellAt("/rooms");
    expect(screen.getByRole("banner")).toBeInTheDocument();
    expect(screen.getByRole("main")).toBeInTheDocument();
  });
});
