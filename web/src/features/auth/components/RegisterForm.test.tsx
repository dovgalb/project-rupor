// Тесты RegisterForm: render + маппинг кодов AUTH-001..005 на поля.

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { RegisterForm } from "@/features/auth/components/RegisterForm";
import { useAuthStore } from "@/features/auth/store";
import { ru } from "@/shared/lib/i18n/ru";

function renderForm(onSuccess: () => void = vi.fn()): void {
  render(
    <MemoryRouter>
      <RegisterForm onSuccess={onSuccess} />
    </MemoryRouter>,
  );
}

async function fillValid(user: ReturnType<typeof userEvent.setup>): Promise<void> {
  await user.type(screen.getByLabelText(ru.auth.email), "user@example.com");
  await user.type(screen.getByLabelText(ru.auth.username), "user_1");
  await user.type(screen.getByLabelText(ru.auth.password), "secret123");
}

function mockRegister(result: boolean, code: string | null = null): void {
  const baseState = useAuthStore.getState();
  vi.spyOn(useAuthStore, "getState").mockReturnValue({
    ...baseState,
    register: vi.fn().mockResolvedValue(result),
    lastErrorCode: code,
  });
}

describe("<RegisterForm />", () => {
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

  it("рендерит email, username, password и submit", () => {
    renderForm();
    expect(screen.getByLabelText(ru.auth.email)).toBeInTheDocument();
    expect(screen.getByLabelText(ru.auth.username)).toBeInTheDocument();
    expect(screen.getByLabelText(ru.auth.password)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: ru.auth.registerBtn })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: ru.auth.toLogin })).toHaveAttribute(
      "href",
      "/login",
    );
  });

  it("success → onSuccess вызван", async () => {
    const user = userEvent.setup();
    const onSuccess = vi.fn();
    mockRegister(true);
    renderForm(onSuccess);

    await fillValid(user);
    await user.click(screen.getByRole("button", { name: ru.auth.registerBtn }));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledTimes(1);
    });
  });

  it("AUTH-001 → ошибка на поле email", async () => {
    const user = userEvent.setup();
    mockRegister(false, "AUTH-001");
    renderForm();

    await fillValid(user);
    await user.click(screen.getByRole("button", { name: ru.auth.registerBtn }));

    await waitFor(() => {
      const alerts = screen.getAllByRole("alert");
      expect(alerts.some((el) => el.textContent === ru.auth.invalidEmail)).toBe(true);
    });
  });

  it("AUTH-004 → ошибка emailTaken на поле email", async () => {
    const user = userEvent.setup();
    mockRegister(false, "AUTH-004");
    renderForm();

    await fillValid(user);
    await user.click(screen.getByRole("button", { name: ru.auth.registerBtn }));

    await waitFor(() => {
      const alerts = screen.getAllByRole("alert");
      expect(alerts.some((el) => el.textContent === ru.auth.emailTaken)).toBe(true);
    });
  });

  it("AUTH-002 → ошибка usernameFormat на поле username", async () => {
    const user = userEvent.setup();
    mockRegister(false, "AUTH-002");
    renderForm();

    await fillValid(user);
    await user.click(screen.getByRole("button", { name: ru.auth.registerBtn }));

    await waitFor(() => {
      const alerts = screen.getAllByRole("alert");
      expect(alerts.some((el) => el.textContent === ru.auth.usernameFormat)).toBe(true);
    });
  });

  it("AUTH-005 → ошибка usernameTaken на поле username", async () => {
    const user = userEvent.setup();
    mockRegister(false, "AUTH-005");
    renderForm();

    await fillValid(user);
    await user.click(screen.getByRole("button", { name: ru.auth.registerBtn }));

    await waitFor(() => {
      const alerts = screen.getAllByRole("alert");
      expect(alerts.some((el) => el.textContent === ru.auth.usernameTaken)).toBe(true);
    });
  });

  it("AUTH-003 → ошибка passwordMin на поле password", async () => {
    const user = userEvent.setup();
    mockRegister(false, "AUTH-003");
    renderForm();

    await fillValid(user);
    await user.click(screen.getByRole("button", { name: ru.auth.registerBtn }));

    await waitFor(() => {
      const alerts = screen.getAllByRole("alert");
      expect(alerts.some((el) => el.textContent === ru.auth.passwordMin)).toBe(true);
    });
  });

  it("неизвестный код → form-level error через genericError", async () => {
    const user = userEvent.setup();
    mockRegister(false, "AUTH-099");
    renderForm();

    await fillValid(user);
    await user.click(screen.getByRole("button", { name: ru.auth.registerBtn }));

    await waitFor(() => {
      const alerts = screen.getAllByRole("alert");
      expect(alerts.some((el) => el.textContent === ru.auth.genericError)).toBe(true);
    });
  });
});
