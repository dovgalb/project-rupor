// ChatPanel: корневой компонент центральной чат-панели.
// Источник: docs/3_5_frontend/07-ui-contract.md (<ChatPanel />).

import { ConnectionStatusBanner } from "@/features/chat/components/ConnectionStatusBanner";
import { MessageComposer } from "@/features/chat/components/MessageComposer";
import { MessageList } from "@/features/chat/components/MessageList";

import styles from "./ChatPanel.module.css";

type ChatPanelProps = {
  channelId: string;
};

export function ChatPanel({ channelId }: ChatPanelProps): JSX.Element {
  return (
    <div className={styles.panel}>
      <ConnectionStatusBanner />
      <MessageList channelId={channelId} />
      <MessageComposer channelId={channelId} />
    </div>
  );
}
