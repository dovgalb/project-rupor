import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { FormField } from "./FormField";

describe("<FormField />", () => {
  it("рендерит label, input и не показывает error если его нет", () => {
    render(
      <FormField label="Email" htmlFor="email">
        <input id="email" />
      </FormField>,
    );

    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("показывает error через role=alert", () => {
    render(
      <FormField label="Email" htmlFor="email" error="Невалидный email">
        <input id="email" />
      </FormField>,
    );

    const alert = screen.getByRole("alert");
    expect(alert).toHaveTextContent("Невалидный email");
  });
});
