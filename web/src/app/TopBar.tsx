// TopBar: верхняя панель shell с UserBadge.
// Источник: docs/3_5_frontend/07-ui-contract.md (TopBar, UserBadge).

import { useNavigate } from "react-router-dom";

import { UserBadge, useAuthStore } from "@/features/auth";

import styles from "./TopBar.module.css";

export function TopBar(): JSX.Element | null {
  const currentUser = useAuthStore((s) => s.currentUser);
  const logout = useAuthStore((s) => s.logout);
  const navigate = useNavigate();

  if (!currentUser) {
    return null;
  }

  const handleLogout = (): void => {
    logout();
    navigate("/login", { replace: true });
  };

  return (
    <header className={styles.topbar}>
      <UserBadge user={currentUser} onLogout={handleLogout} />
    </header>
  );
}
