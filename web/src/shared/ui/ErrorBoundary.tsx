import { Component } from "react";

import { logger } from "@/shared/lib/logger";

import type { ErrorInfo, ReactNode } from "react";


type ErrorBoundaryProps = {
  children: ReactNode;
  fallback?: ReactNode;
};

type ErrorBoundaryState = {
  hasError: boolean;
  error: Error | null;
};

// React error boundary. Ловит ошибки рендера и lifecycle ниже по дереву.
// componentDidCatch логирует через единый logger (R-09 — без токенов).
export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  override state: ErrorBoundaryState = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  override componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    logger.error("[ErrorBoundary]", error, errorInfo);
  }

  override render(): ReactNode {
    if (this.state.hasError) {
      if (this.props.fallback !== undefined) {
        return this.props.fallback;
      }
      return (
        <div
          role="alert"
          style={{
            padding: 24,
            color: "var(--color-text-primary)",
            backgroundColor: "var(--color-bg-primary)",
            fontFamily: "var(--font-base)",
          }}
        >
          Что-то пошло не так. Перезагрузите страницу.
        </div>
      );
    }
    return this.props.children;
  }
}
