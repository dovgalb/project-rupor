import { forwardRef } from "react";


import { cn } from "@/shared/lib/cn";

import styles from "./Textarea.module.css";

import type { TextareaHTMLAttributes } from "react";

type TextareaProps = {
  invalid?: boolean;
} & TextareaHTMLAttributes<HTMLTextAreaElement>;

// Auto-size через CSS `field-sizing: content` (см. .module.css).
// forwardRef — для react-hook-form.
export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  function Textarea({ invalid = false, className, ...rest }, ref) {
    return (
      <textarea
        ref={ref}
        className={cn(styles.textarea, invalid && styles.invalid, className)}
        aria-invalid={invalid || undefined}
        {...rest}
      />
    );
  },
);
