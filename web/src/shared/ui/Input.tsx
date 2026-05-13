import { forwardRef } from "react";


import { cn } from "@/shared/lib/cn";

import styles from "./Input.module.css";

import type { InputHTMLAttributes } from "react";

type InputProps = {
  invalid?: boolean;
} & InputHTMLAttributes<HTMLInputElement>;

// forwardRef нужен для интеграции с react-hook-form (register возвращает ref).
// invalid → aria-invalid и красная рамка через CSS-модуль.
export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { invalid = false, className, ...rest },
  ref,
) {
  return (
    <input
      ref={ref}
      className={cn(styles.input, invalid && styles.invalid, className)}
      aria-invalid={invalid || undefined}
      {...rest}
    />
  );
});
