// Sidebar: левая навигационная панель.
// Phase 6 — добавлены <RoomList /> сверху и <RoomScopedSidebar /> снизу.

import { RoomList } from "@/features/rooms";

import { RoomScopedSidebar } from "./RoomScopedSidebar";
import styles from "./Sidebar.module.css";

export function Sidebar(): JSX.Element {
  return (
    <nav className={styles.sidebar} aria-label="Главная навигация">
      <RoomList />
      <RoomScopedSidebar />
    </nav>
  );
}
