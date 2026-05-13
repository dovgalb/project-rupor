// Тесты RequireAuth — гард защищённых роутов.

import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";

import { useAuthStore } from "@/features/auth";

import { RequireAuth } from "./RequireAuth";

import type { CurrentUser } from "@/features/auth";

const userFixture: CurrentUser = {
  id: "user-1",
  email: "user@example.com",
  username: "user1",
  createdAt: "2025-01-01T00:00:00Z",
};

// Маленький компонент-spy для проверки state.from при редиректе.
function LoginSpy(): JSX.Element {
  const location = useLocation();
  const state = location.state as { from?: { pathname?: string } } | null;
  return (
    <div>
      <span>login page</span>
      <span data-testid="from-pathname">{state?.from?.pathname ?? ""}</span>
    </div>
  );
}

function renderAt(path: string): void {
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/login" element={<LoginSpy />} />
        <Route
          path="/protected"
          element={
            <RequireAuth>
              <div>secret content</div>
            </RequireAuth>
          }
        />
      </Routes>
    </MemoryRouter>,
  );
}

describe("<RequireAuth />", () => {
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

  it("authenticated → рендерит children", () => {
    useAuthStore.setState({ status: "authenticated", currentUser: userFixture });
    renderAt("/protected");
    expect(screen.getByText("secret content")).toBeInTheDocument();
    expect(screen.queryByText("login page")).not.toBeInTheDocument();
  });

  it("idle → редиректит на /login", () => {
    useAuthStore.setState({ status: "idle", currentUser: null });
    renderAt("/protected");
    expect(screen.getByText("login page")).toBeInTheDocument();
    expect(screen.queryByText("secret content")).not.toBeInTheDocument();
  });

  it("loading → показывает FullScreenSpinner", () => {
    useAuthStore.setState({ status: "loading", currentUser: null });
    renderAt("/protected");
    // Spinner — не login page и не secret content.
    expect(screen.queryByText("login page")).not.toBeInTheDocument();
    expect(screen.queryByText("secret content")).not.toBeInTheDocument();
    // Spinner имеет role="progressbar" (см. shared/ui/Spinner).
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
  });

  it("idle → передаёт from-state с pathname исходного URL", () => {
    useAuthStore.setState({ status: "idle", currentUser: null });
    renderAt("/protected");
    expect(screen.getByTestId("from-pathname")).toHaveTextContent("/protected");
  });
});
