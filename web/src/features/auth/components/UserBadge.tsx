// UserBadge: аватар + username + кнопка «Выйти».
// Используется в TopBar (см. 07-ui-contract.md / AppShell).

import { Avatar } from "@/shared/ui/Avatar";
import { Button } from "@/shared/ui/Button";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./UserBadge.module.css";

import type { CurrentUser } from "@/features/auth/types";

type UserBadgeProps = {
  user: CurrentUser;
  onLogout: () => void;
};

export function UserBadge({ user, onLogout }: UserBadgeProps): JSX.Element {
  return (
    <div className={styles.badge}>
      <Avatar userId={user.id} size="sm" />
      <span className={styles.username} title={user.email}>
        {user.username}
      </span>
      <Button
        variant="ghost"
        size="sm"
        onClick={onLogout}
        aria-label={ru.auth.logoutBtn}
      >
        {ru.auth.logoutBtn}
      </Button>
    </div>
  );
}
