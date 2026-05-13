
import { cn } from "@/shared/lib/cn";

import { Spinner } from "./Spinner";
import styles from "./Button.module.css";

import type { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "danger" | "ghost";
type ButtonSize = "sm" | "md";

type ButtonProps = {
  variant?: ButtonVariant;
  size?: ButtonSize;
  isLoading?: boolean;
  children?: ReactNode;
} & Omit<ButtonHTMLAttributes<HTMLButtonElement>, "children">;

function variantClass(v: ButtonVariant): string | undefined {
  switch (v) {
    case "primary":
      return styles.primary;
    case "secondary":
      return styles.secondary;
    case "danger":
      return styles.danger;
    case "ghost":
      return styles.ghost;
  }
}

function sizeClass(s: ButtonSize): string | undefined {
  switch (s) {
    case "sm":
      return styles.sizeSm;
    case "md":
      return styles.sizeMd;
  }
}

// Базовая кнопка. При isLoading → disabled + Spinner внутри.
// type по умолчанию "button" (страхует от случайного submit вне формы).
export function Button({
  variant = "primary",
  size = "md",
  isLoading = false,
  type = "button",
  disabled,
  className,
  children,
  ...rest
}: ButtonProps): JSX.Element {
  const isDisabled = disabled === true || isLoading;

  return (
    <button
      // eslint-disable-next-line react/button-has-type -- type приходит из props c default "button"
      type={type}
      className={cn(styles.button, variantClass(variant), sizeClass(size), className)}
      disabled={isDisabled}
      {...rest}
    >
      {isLoading ? <Spinner size="sm" /> : null}
      {children}
    </button>
  );
}
