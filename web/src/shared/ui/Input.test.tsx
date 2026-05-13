import { render, screen } from "@testing-library/react";
import { createRef } from "react";
import { describe, expect, it } from "vitest";

import { Input } from "./Input";

describe("<Input />", () => {
  it("рендерит native input и принимает value", () => {
    render(<Input defaultValue="hello" aria-label="text" />);
    const input = screen.getByLabelText("text") as HTMLInputElement;
    expect(input.value).toBe("hello");
  });

  it("forwardRef прокидывает ref на HTMLInputElement", () => {
    const ref = createRef<HTMLInputElement>();
    render(<Input ref={ref} aria-label="r" />);
    expect(ref.current).toBeInstanceOf(HTMLInputElement);
  });

  it("при invalid=true проставляет aria-invalid='true'", () => {
    render(<Input invalid aria-label="i" />);
    expect(screen.getByLabelText("i")).toHaveAttribute("aria-invalid", "true");
  });
});
