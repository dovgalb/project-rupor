// MemberListItem: один участник комнаты в Sidebar.
// D-11: UUID-placeholder для никнеймов (бэк не отдаёт username в DTO членов).
// Для текущего пользователя — реальный username.

import { Avatar } from "@/shared/ui/Avatar";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./MemberListItem.module.css";

import type { Member, Role } from "@/features/rooms/types";

type MemberListItemProps = {
  member: Member;
  isMe: boolean;
  meUsername?: string;
};

// Берёт UUID, убирает дефисы, делает uppercase и берёт первые 6 символов.
function uuidPlaceholder(userId: string): string {
  return userId.replace(/-/g, "").slice(0, 6).toUpperCase();
}

function roleLabel(role: Role): string | null {
  if (role === "owner") return ru.rooms.roleOwner;
  if (role === "admin") return ru.rooms.roleAdmin;
  return null;
}

export function MemberListItem({
  member,
  isMe,
  meUsername,
}: MemberListItemProps): JSX.Element {
  const displayName =
    isMe && meUsername !== undefined && meUsername !== ""
      ? meUsername
      : uuidPlaceholder(member.userId);

  const badge = roleLabel(member.role);

  return (
    <li className={styles.item}>
      <Avatar userId={member.userId} size="sm" />
      <span className={styles.name}>{displayName}</span>
      {badge !== null ? (
        <span className={styles.badge} aria-label={`роль: ${badge}`}>
          {badge}
        </span>
      ) : null}
    </li>
  );
}
