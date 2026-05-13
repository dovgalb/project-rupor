// Тесты LoginForm: render, zod-валидация, AUTH-006 form-error, успешный submit.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { LoginForm } from "@/features/auth/components/LoginForm";
import { useAuthStore } from "@/features/auth/store";
import { ru } from "@/shared/lib/i18n/ru";

function renderForm(onSuccess: () => void = vi.fn()): void {
  render(
    <MemoryRouter>
      <LoginForm onSuccess={onSuccess} />
    </MemoryRouter>,
  );
}

describe("<LoginForm />", () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.setState({
      status: "idle",
      currentUser: null,
      error: null,
      lastErrorCode: null,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("рендерит поля email/password, кнопку submit, ссылку на /register", () => {
    renderForm();
    expect(screen.getByLabelText(ru.auth.email)).toBeInTheDocument();
    expect(screen.getByLabelText(ru.auth.password)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: ru.auth.loginBtn })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: ru.auth.toRegister })).toHaveAttribute(
      "href",
      "/register",
    );
  });

  it("zod-валидация показывает inline-ошибки при пустых полях", async () => {
    const user = userEvent.setup();
    renderForm();

    await user.click(screen.getByRole("button", { name: ru.auth.loginBtn }));

    await waitFor(() => {
      expect(screen.getByText(ru.auth.invalidEmail)).toBeInTheDocument();
    });
    expect(screen.getByText(ru.auth.passwordMin)).toBeInTheDocument();
  });

  it("успешный login вызывает onSuccess", async () => {
    const user = userEvent.setup();
    const onSuccess = vi.fn();
    vi.spyOn(useAuthStore, "getState").mockReturnValue({
      ...useAuthStore.getState(),
      login: vi.fn().mockResolvedValue(true),
    });
    renderForm(onSuccess);

    await user.type(screen.getByLabelText(ru.auth.email), "user@example.com");
    await user.type(screen.getByLabelText(ru.auth.password), "secret123");
    await user.click(screen.getByRole("button", { name: ru.auth.loginBtn }));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledTimes(1);
    });
  });

  it("AUTH-006 → form-level error с invalidCredentials", async () => {
    const user = userEvent.setup();
    const baseState = useAuthStore.getState();
    vi.spyOn(useAuthStore, "getState").mockReturnValue({
      ...baseState,
      login: vi.fn().mockResolvedValue(false),
      lastErrorCode: "AUTH-006",
    });
    renderForm();

    await user.type(screen.getByLabelText(ru.auth.email), "user@example.com");
    await user.type(screen.getByLabelText(ru.auth.password), "wrongpassword");
    await user.click(screen.getByRole("button", { name: ru.auth.loginBtn }));

    await waitFor(() => {
      // serverError <div role="alert"> с invalidCredentials
      const alerts = screen.getAllByRole("alert");
      const hasInvalidMsg = alerts.some((el) =>
        el.textContent?.includes(ru.auth.invalidCredentials),
      );
      expect(hasInvalidMsg).toBe(true);
    });
  });

  it("Tab navigation проходит email → password → submit → link", async () => {
    const user = userEvent.setup();
    renderForm();

    const emailInput = screen.getByLabelText(ru.auth.email);
    const passwordInput = screen.getByLabelText(ru.auth.password);
    const submitBtn = screen.getByRole("button", { name: ru.auth.loginBtn });
    const link = screen.getByRole("link", { name: ru.auth.toRegister });

    emailInput.focus();
    expect(emailInput).toHaveFocus();
    await user.tab();
    expect(passwordInput).toHaveFocus();
    await user.tab();
    expect(submitBtn).toHaveFocus();
    await user.tab();
    expect(link).toHaveFocus();
  });

  it("Enter в поле password сабмитит форму", async () => {
    const user = userEvent.setup();
    const onSuccess = vi.fn();
    vi.spyOn(useAuthStore, "getState").mockReturnValue({
      ...useAuthStore.getState(),
      login: vi.fn().mockResolvedValue(true),
    });
    renderForm(onSuccess);

    await user.type(screen.getByLabelText(ru.auth.email), "user@example.com");
    const passwordInput = screen.getByLabelText(ru.auth.password);
    await user.type(passwordInput, "secret123{Enter}");

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledTimes(1);
    });
  });
});
