// AppShell: трёхпанельный layout всех авторизованных экранов.
// Состав: TopBar сверху, Sidebar слева, Outlet (вложенный роут) — основная область.
// Источник: docs/3_5_frontend/07-ui-contract.md (Каркас приложения).

import { Outlet } from "react-router-dom";

import { Sidebar } from "./Sidebar";
import { TopBar } from "./TopBar";
import styles from "./AppShell.module.css";

export function AppShell(): JSX.Element {
  return (
    <div className={styles.shell}>
      <TopBar />
      <aside className={styles.sidebar}>
        <Sidebar />
      </aside>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  );
}
