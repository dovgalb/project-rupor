// ConnectionStatusBanner: баннер при connecting/reconnecting WS.
// Источник: docs/3_5_frontend/07-ui-contract.md (<ConnectionStatusBanner />).
//
// В Phase 8 wsStatus всегда "idle" — баннер не виден. Логика готова для Phase 9.

import { useChatStore } from "@/features/chat/store";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./ConnectionStatusBanner.module.css";

export function ConnectionStatusBanner(): JSX.Element | null {
  const status = useChatStore((s) => s.wsStatus);

  if (status === "open" || status === "idle" || status === "closed") {
    return null;
  }

  const message =
    status === "connecting" ? ru.chat.connecting : ru.chat.reconnecting;

  return (
    <div className={styles.banner} role="status" aria-live="polite">
      {message}
    </div>
  );
}
