import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Toaster } from "./Toaster";
import { toast, useToastsStore } from "./useToast";

describe("<Toaster /> + toast API", () => {
  beforeEach(() => {
    // Сбрасываем стор перед каждым тестом.
    useToastsStore.setState({ items: [] });
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    useToastsStore.setState({ items: [] });
  });

  it("toast.info() пушит в стор и тост рендерится", () => {
    render(<Toaster />);
    act(() => {
      toast.info("Привет");
    });
    expect(screen.getByRole("status")).toHaveTextContent("Привет");
  });

  it("toast.error() рендерится с role=alert", () => {
    render(<Toaster />);
    act(() => {
      toast.error("Беда");
    });
    expect(screen.getByRole("alert")).toHaveTextContent("Беда");
  });

  it("toast.dismiss(id) убирает тост из стора", () => {
    render(<Toaster />);
    let id = "";
    act(() => {
      id = toast.info("temp");
    });
    expect(screen.getByRole("status")).toBeInTheDocument();
    act(() => {
      toast.dismiss(id);
    });
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("auto-dismiss через duration убирает тост (fake timers)", () => {
    render(<Toaster />);
    act(() => {
      toast.info("disappear", { duration: 4000 });
    });
    expect(screen.getByRole("status")).toBeInTheDocument();
    act(() => {
      vi.advanceTimersByTime(4000);
    });
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });
});
