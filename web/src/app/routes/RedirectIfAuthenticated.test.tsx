// Тесты RedirectIfAuthenticated — гард для /login и /register.

import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";

import { useAuthStore } from "@/features/auth";

import { RedirectIfAuthenticated } from "./RedirectIfAuthenticated";

import type { CurrentUser } from "@/features/auth";

const userFixture: CurrentUser = {
  id: "user-1",
  email: "user@example.com",
  username: "user1",
  createdAt: "2025-01-01T00:00:00Z",
};

function renderAt(path: string): void {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/rooms" element={<div>rooms index</div>} />
        <Route
          path="/login"
          element={
            <RedirectIfAuthenticated>
              <div>login page</div>
            </RedirectIfAuthenticated>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("<RedirectIfAuthenticated />", () => {
  beforeEach(() => {
    useAuthStore.setState({
      status: "idle",
      currentUser: null,
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

  it("authenticated → редирект на /rooms", () => {
    useAuthStore.setState({ status: "authenticated", currentUser: userFixture });
    renderAt("/login");
    expect(screen.getByText("rooms index")).toBeInTheDocument();
    expect(screen.queryByText("login page")).not.toBeInTheDocument();
  });

  it("idle → рендерит children", () => {
    useAuthStore.setState({ status: "idle", currentUser: null });
    renderAt("/login");
    expect(screen.getByText("login page")).toBeInTheDocument();
    expect(screen.queryByText("rooms index")).not.toBeInTheDocument();
  });
});
