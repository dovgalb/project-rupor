// RedirectIfAuthenticated: гард для /login и /register.
// Если пользователь уже залогинен — редиректит на /rooms.
// Источник: docs/3_5_frontend/08-routes.md (Guards).

import { Navigate } from "react-router-dom";

import { useAuthStore } from "@/features/auth";

import type { ReactNode } from "react";

type RedirectIfAuthenticatedProps = {
  children: ReactNode;
};

export function RedirectIfAuthenticated({
  children,
}: RedirectIfAuthenticatedProps): JSX.Element {
  const status = useAuthStore((s) => s.status);
  if (status === "authenticated") {
    return <Navigate to="/rooms" replace />;
  }
  return <>{children}</>;
}
