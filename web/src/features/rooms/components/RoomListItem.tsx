// RoomListItem: одна комната в Sidebar.
// Источник: docs/3_5_frontend/07-ui-contract.md (<RoomListItem />).

import { Link } from "react-router-dom";

import { cn } from "@/shared/lib/cn";
import { Avatar } from "@/shared/ui/Avatar";

import styles from "./RoomListItem.module.css";

import type { RoomWithRole } from "@/features/rooms/types";

type RoomListItemProps = {
  room: RoomWithRole;
  active: boolean;
};

export function RoomListItem({ room, active }: RoomListItemProps): JSX.Element {
  return (
    <li className={styles.item}>
      <Link
        to={`/rooms/${room.id}`}
        className={cn(styles.link, active && styles.active)}
        aria-current={active ? "page" : undefined}
      >
        <Avatar userId={room.id} size="sm" />
        <span className={styles.name}>{room.name}</span>
      </Link>
    </li>
  );
}
