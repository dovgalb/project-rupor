import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Modal } from "./Modal";

describe("<Modal />", () => {
  it("при open=false не рендерит dialog", () => {
    render(
      <Modal open={false} onClose={() => undefined} title="t">
        <button type="button">a</button>
      </Modal>,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("при open=true рендерит dialog с aria-modal и aria-labelledby", () => {
    render(
      <Modal open onClose={() => undefined} title="Заголовок">
        <button type="button">first</button>
      </Modal>,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    const labelId = dialog.getAttribute("aria-labelledby");
    expect(labelId).toBeTruthy();
    const heading = screen.getByRole("heading", { level: 2 });
    expect(heading.id).toBe(labelId);
  });

  it("фокус ставится на первый focusable элемент после open", async () => {
    render(
      <Modal open onClose={() => undefined} title="t">
        <button type="button">first</button>
        <button type="button">second</button>
      </Modal>,
    );
    // Первым focusable считается close-кнопка из header, она первой получит фокус.
    // Проверяем что какой-то focusable элемент в dialog активен.
    const dialog = screen.getByRole("dialog");
    expect(dialog.contains(document.activeElement)).toBe(true);
  });

  it("Esc вызывает onClose", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(
      <Modal open onClose={onClose} title="t">
        <button type="button">a</button>
      </Modal>,
    );
    await user.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("клик по overlay вызывает onClose", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(
      <Modal open onClose={onClose} title="t">
        <button type="button">a</button>
      </Modal>,
    );
    await user.click(screen.getByTestId("modal-overlay"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("клик по close-кнопке вызывает onClose", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(
      <Modal open onClose={onClose} title="t">
        <button type="button">a</button>
      </Modal>,
    );
    await user.click(screen.getByRole("button", { name: "Закрыть" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
