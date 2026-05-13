import { create } from "zustand";

export type ToastVariant = "info" | "success" | "warn" | "error";

export type ToastItem = {
  id: string;
  variant: ToastVariant;
  message: string;
  duration: number;
};

// Default-длительности по типу (D-24).
const DEFAULT_DURATION: Record<ToastVariant, number> = {
  info: 4000,
  success: 4000,
  warn: 4000,
  error: 7000,
};

type ToastsState = {
  items: ToastItem[];
  push: (item: ToastItem) => void;
  dismiss: (id: string) => void;
};

// Стор тостов. Это единственный store, живущий в shared/ui — оправдано,
// потому что toaster — глобальная UI-инфраструктура (D-24).
export const useToastsStore = create<ToastsState>((set) => ({
  items: [],
  push: (item) =>
    set((state) => ({
      items: [...state.items, item],
    })),
  dismiss: (id) =>
    set((state) => ({
      items: state.items.filter((t) => t.id !== id),
    })),
}));

// Безопасная генерация ID. crypto.randomUUID есть в современных браузерах
// и jsdom 25+ (см. package.json). Фолбэк — простой счётчик.
let fallbackCounter = 0;
function generateId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  fallbackCounter += 1;
  return `toast-${Date.now()}-${fallbackCounter}`;
}

type ToastOptions = {
  duration?: number;
};

function pushToast(
  variant: ToastVariant,
  message: string,
  opts?: ToastOptions,
): string {
  const id = generateId();
  const duration = opts?.duration ?? DEFAULT_DURATION[variant];
  useToastsStore.getState().push({ id, variant, message, duration });
  return id;
}

// Публичный API. Импортируется как `import { toast } from "@/shared/ui/useToast"`.
export const toast = {
  info: (message: string, opts?: ToastOptions): string =>
    pushToast("info", message, opts),
  success: (message: string, opts?: ToastOptions): string =>
    pushToast("success", message, opts),
  warn: (message: string, opts?: ToastOptions): string =>
    pushToast("warn", message, opts),
  error: (message: string, opts?: ToastOptions): string =>
    pushToast("error", message, opts),
  dismiss: (id: string): void => useToastsStore.getState().dismiss(id),
};
