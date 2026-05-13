import type { ReactNode } from "react";

type FormFieldProps = {
  label: string;
  htmlFor: string;
  error?: string;
  children: ReactNode;
};

// Обёртка: label + input + (optional error). Без CSS-модуля — минимальный layout.
// error отображается через role="alert" для assistive-tech.
export function FormField({
  label,
  htmlFor,
  error,
  children,
}: FormFieldProps): JSX.Element {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      <label
        htmlFor={htmlFor}
        style={{ fontSize: 13, color: "var(--color-text-muted)" }}
      >
        {label}
      </label>
      {children}
      {error ? (
        <span
          role="alert"
          style={{ fontSize: 12, color: "var(--color-danger)" }}
        >
          {error}
        </span>
      ) : null}
    </div>
  );
}
