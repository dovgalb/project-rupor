import { cn } from "@/shared/lib/cn";

import styles from "./Spinner.module.css";

type SpinnerSize = "sm" | "md" | "lg";

type SpinnerProps = {
  size?: SpinnerSize;
};

function sizeClass(size: SpinnerSize): string | undefined {
  switch (size) {
    case "sm":
      return styles.sizeSm;
    case "md":
      return styles.sizeMd;
    case "lg":
      return styles.sizeLg;
  }
}

// Pure-presentational спиннер. a11y: progressbar с русским aria-label.
export function Spinner({ size = "md" }: SpinnerProps): JSX.Element {
  return (
    <span
      className={cn(styles.spinner, sizeClass(size))}
      role="progressbar"
      aria-label="Загрузка"
    />
  );
}
