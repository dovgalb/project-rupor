// MembersList: список участников активной комнаты в Sidebar.
// Грузит данные через useEffect при смене roomId.
// EMPTY_MEMBERS — модульная константа для стабильного селектора (см. 05-state-model.md).

import { useEffect } from "react";

import { MemberListItem } from "@/features/rooms/components/MemberListItem";
import { useRoomsStore } from "@/features/rooms/store";
import { useAuthStore } from "@/features/auth";
import { ru } from "@/shared/lib/i18n/ru";
import { Spinner } from "@/shared/ui/Spinner";

import styles from "./MembersList.module.css";

import type { Member } from "@/features/rooms/types";

type MembersListProps = {
  roomId: string;
};

const EMPTY_MEMBERS: Member[] = [];

export function MembersList({ roomId }: MembersListProps): JSX.Element {
  const members = useRoomsStore(
    (s) => s.membersByRoom[roomId] ?? EMPTY_MEMBERS,
  );
  const loading = useRoomsStore(
    (s) => s.membersLoadingByRoom[roomId] ?? false,
  );
  const currentUserId = useAuthStore((s) => s.currentUser?.id);
  const currentUsername = useAuthStore((s) => s.currentUser?.username);

  useEffect(() => {
    void useRoomsStore.getState().loadMembers(roomId);
  }, [roomId]);

  if (loading && members.length === 0) {
    return (
      <section className={styles.section}>
        <h3 className={styles.title}>{ru.rooms.members}</h3>
        <div className={styles.center}>
          <Spinner size="sm" />
        </div>
      </section>
    );
  }

  return (
    <section className={styles.section}>
      <h3 className={styles.title}>
        {ru.rooms.members} ({members.length})
      </h3>
      <ul className={styles.list}>
        {members.map((m) => (
          <MemberListItem
            key={m.userId}
            member={m}
            isMe={m.userId === currentUserId}
            {...(currentUsername !== undefined
              ? { meUsername: currentUsername }
              : {})}
          />
        ))}
      </ul>
    </section>
  );
}
