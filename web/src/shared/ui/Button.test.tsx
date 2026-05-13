import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Button } from "./Button";

describe("<Button />", () => {
  it("рендерит все variants без падения", () => {
    const variants = ["primary", "secondary", "danger", "ghost"] as const;
    for (const v of variants) {
      const { unmount } = render(<Button variant={v}>{v}</Button>);
      expect(screen.getByRole("button", { name: v })).toBeInTheDocument();
      unmount();
    }
  });

  it("при isLoading кнопка disabled и содержит спиннер", () => {
    render(<Button isLoading>Сохранить</Button>);
    const btn = screen.getByRole("button", { name: /Сохранить/ });
    expect(btn).toBeDisabled();
    expect(screen.getByRole("progressbar")).toBeInTheDocument();
  });

  it("onClick НЕ вызывается при disabled", async () => {
    const handler = vi.fn();
    const user = userEvent.setup();
    render(
      <Button disabled onClick={handler}>
        click
      </Button>,
    );
    await user.click(screen.getByRole("button"));
    expect(handler).not.toHaveBeenCalled();
  });

  it("по умолчанию type=button; явно type=submit прокидывается", () => {
    const { rerender } = render(<Button>def</Button>);
    expect(screen.getByRole("button")).toHaveAttribute("type", "button");

    rerender(<Button type="submit">sub</Button>);
    expect(screen.getByRole("button")).toHaveAttribute("type", "submit");
  });
});
