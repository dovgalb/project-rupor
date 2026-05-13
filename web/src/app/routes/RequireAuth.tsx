// RequireAuth: гард для защищённых роутов.
// Пускает только authenticated; loading → spinner; иначе redirect /login с from-state.
// Источник: docs/3_5_frontend/08-routes.md (Guards).

import { Navigate, useLocation } from "react-router-dom";

import { useAuthStore } from "@/features/auth";
import { FullScreenSpinner } from "@/shared/ui";

import type { ReactNode } from "react";

type RequireAuthProps = {
  children: ReactNode;
};

export function RequireAuth({ children }: RequireAuthProps): JSX.Element {
  const status = useAuthStore((s) => s.status);
  const location = useLocation();

  if (status === "loading") {
    return <FullScreenSpinner />;
  }
  if (status !== "authenticated") {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <>{children}</>;
}
