import styles from "./EmptyState.module.css";

import type { ReactNode } from "react";


type EmptyStateProps = {
  title: string;
  description?: string;
  action?: ReactNode;
};

// Плейсхолдер для пустых состояний (список комнат, чат, etc).
export function EmptyState({
  title,
  description,
  action,
}: EmptyStateProps): JSX.Element {
  return (
    <div className={styles.root}>
      <h3 className={styles.title}>{title}</h3>
      {description ? <p className={styles.description}>{description}</p> : null}
      {action ? <div className={styles.action}>{action}</div> : null}
    </div>
  );
}
