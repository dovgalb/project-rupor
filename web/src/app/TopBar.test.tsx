// Тесты TopBar — верхней панели с UserBadge.

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";

import { useAuthStore } from "@/features/auth";

import { TopBar } from "./TopBar";

import type { CurrentUser } from "@/features/auth";
import type * as ReactRouterDom from "react-router-dom";

const userFixture: CurrentUser = {
  id: "user-1",
  email: "user@example.com",
  username: "user1",
  createdAt: "2025-01-01T00:00:00Z",
};

// Подменяем useNavigate, чтобы проверить что вызывается с "/login".
const navigateMock = vi.fn();
vi.mock("react-router-dom", async (importActual) => {
  const actual = (await importActual()) as typeof ReactRouterDom;
  return {
    ...actual,
    useNavigate: () => navigateMock,
  };
});

describe("<TopBar />", () => {
  beforeEach(() => {
    navigateMock.mockReset();
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

  it("рендерит UserBadge с currentUser", () => {
    render(
      <MemoryRouter>
        <TopBar />
      </MemoryRouter>,
    );
    expect(screen.getByText("user1")).toBeInTheDocument();
    expect(screen.getByRole("banner")).toBeInTheDocument();
  });

  it("возвращает null если currentUser отсутствует", () => {
    useAuthStore.setState({ status: "idle", currentUser: null });
    const { container } = render(
      <MemoryRouter>
        <TopBar />
      </MemoryRouter>,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("click «Выйти» → logout + navigate('/login', { replace: true })", async () => {
    const logoutSpy = vi.fn();
    useAuthStore.setState({
      status: "authenticated",
      currentUser: userFixture,
      // Подменяем logout-функцию через прямой setState, action остаётся вызываемым.
    });
    // Подменяем logout через setState нельзя (он создан в фабрике),
    // используем spyOn на самом store.
    const realLogout = useAuthStore.getState().logout;
    const logoutWrapper = (): void => {
      logoutSpy();
      realLogout();
    };
    useAuthStore.setState({ logout: logoutWrapper });

    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <TopBar />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: /выйти/i }));

    expect(logoutSpy).toHaveBeenCalledOnce();
    expect(navigateMock).toHaveBeenCalledWith("/login", { replace: true });
  });
});
