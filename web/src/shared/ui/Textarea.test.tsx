import { render, screen } from "@testing-library/react";
import { createRef } from "react";
import { describe, expect, it } from "vitest";

import { Textarea } from "./Textarea";

describe("<Textarea />", () => {
  it("рендерит native textarea и принимает value", () => {
    render(<Textarea defaultValue="hi" aria-label="t" />);
    const ta = screen.getByLabelText("t") as HTMLTextAreaElement;
    expect(ta.value).toBe("hi");
  });

  it("forwardRef прокидывает ref на HTMLTextAreaElement", () => {
    const ref = createRef<HTMLTextAreaElement>();
    render(<Textarea ref={ref} aria-label="r" />);
    expect(ref.current).toBeInstanceOf(HTMLTextAreaElement);
  });

  it("при invalid=true проставляет aria-invalid='true'", () => {
    render(<Textarea invalid aria-label="i" />);
    expect(screen.getByLabelText("i")).toHaveAttribute("aria-invalid", "true");
  });
});
