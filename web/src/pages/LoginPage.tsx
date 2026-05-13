// Страница логина. Использует from-state, чтобы вернуться туда, откуда был redirect.
// Phase 5: добавлен getSafeFrom — защита от open-redirect (protocol-relative и absolute URL).

import { useLocation, useNavigate } from "react-router-dom";

import { LoginForm } from "@/features/auth";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./LoginPage.module.css";

import type { Location } from "react-router-dom";

// Возвращает безопасный path для возврата после логина.
// Принимаем только относительные пути ("/..."), но НЕ protocol-relative ("//...").
function getSafeFrom(location: Location): string {
  const state = location.state as { from?: { pathname?: unknown } } | null;
  const raw = state?.from?.pathname;
  if (typeof raw !== "string") {
    return "/rooms";
  }
  if (!raw.startsWith("/") || raw.startsWith("//")) {
    return "/rooms";
  }
  return raw;
}

export function LoginPage(): JSX.Element {
  const navigate = useNavigate();
  const location = useLocation();
  const from = getSafeFrom(location);

  return (
    <div className={styles.page}>
      <h1>{ru.auth.loginTitle}</h1>
      <LoginForm onSuccess={() => navigate(from, { replace: true })} />
    </div>
  );
}
