import { useEffect } from "react";

import { cn } from "@/shared/lib/cn";

import { useToastsStore } from "./useToast";
import styles from "./Toaster.module.css";

import type { ToastItem } from "./useToast";

type ToastProps = {
  item: ToastItem;
};

function variantClass(v: ToastItem["variant"]): string | undefined {
  switch (v) {
    case "info":
      return styles.info;
    case "success":
      return styles.success;
    case "warn":
      return styles.warn;
    case "error":
      return styles.error;
  }
}

// Один тост. На mount запускает таймер автодиссмиса. Чистит таймер на unmount.
// role зависит от variant: error → alert (срочное), остальное → status.
export function Toast({ item }: ToastProps): JSX.Element {
  const dismiss = useToastsStore((s) => s.dismiss);

  useEffect(() => {
    const id = item.id;
    const duration = item.duration;
    const timer = window.setTimeout(() => {
      dismiss(id);
    }, duration);
    return () => {
      window.clearTimeout(timer);
    };
  }, [item.id, item.duration, dismiss]);

  const role = item.variant === "error" ? "alert" : "status";

  return (
    <div
      className={cn(styles.toast, variantClass(item.variant))}
      role={role}
      aria-live="polite"
    >
      {item.message}
    </div>
  );
}
