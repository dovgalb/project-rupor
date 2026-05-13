// MessageItem: один элемент чата.
// Источник: docs/3_5_frontend/07-ui-contract.md (<MessageItem />).
//
// - Avatar по authorId.
// - authorPlaceholder: имя для своих сообщений (если есть) или UUID-short для чужих (D-11).
// - Время через formatTime.
// - Текст: linkify → text-node | <a target="_blank" rel="noopener noreferrer">.
// - Статус (только isAuthor): pending/committed/failed + кнопка retry для failed.
// - Без эмодзи (CLAUDE.md): используем текстовые маркеры.

import { useChatStore } from "@/features/chat/store";
import { Avatar, Button } from "@/shared/ui";
import { cn } from "@/shared/lib/cn";
import { formatTime } from "@/shared/lib/formatDate";
import { linkify } from "@/shared/lib/linkify";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./MessageItem.module.css";

import type { Message } from "@/features/chat/types";

type MessageItemProps = {
  message: Message;
  isAuthor: boolean;
  authorPlaceholder: string;
};

export function MessageItem({
  message,
  isAuthor,
  authorPlaceholder,
}: MessageItemProps): JSX.Element {
  const parts = linkify(message.text);
  const showPending = isAuthor && message.status === "pending";
  const showFailed = isAuthor && message.status === "failed";
  const wsStatus = useChatStore((s) => s.wsStatus);

  function handleRetry(): void {
    if (message.tempId === undefined) {
      return;
    }
    void useChatStore
      .getState()
      .retryMessage(message.channelId, message.tempId);
  }

  return (
    <li
      className={cn(
        styles.item,
        showPending && styles.pending,
        showFailed && styles.failed,
      )}
      data-status={message.status}
    >
      <Avatar userId={message.authorId} size="sm" />
      <div className={styles.body}>
        <div className={styles.header}>
          <span className={styles.author}>{authorPlaceholder}</span>
          <time className={styles.time} dateTime={message.createdAt}>
            {formatTime(message.createdAt)}
          </time>
          {showPending ? (
            <span className={styles.statusHint} aria-label={ru.chat.pending}>
              {ru.chat.pending}
            </span>
          ) : null}
        </div>
        <div className={styles.text}>
          {parts.map((part, i) =>
            part.type === "link" ? (
              <a
                key={`p-${i.toString()}`}
                href={part.href}
                target="_blank"
                rel="noopener noreferrer"
              >
                {part.value}
              </a>
            ) : (
              <span key={`p-${i.toString()}`}>{part.value}</span>
            ),
          )}
        </div>
        {showFailed ? (
          <div className={styles.failedRow} role="alert">
            <span>{ru.chat.failed}</span>
            <Button
              variant="ghost"
              size="sm"
              type="button"
              onClick={handleRetry}
              disabled={wsStatus !== "open"}
            >
              {ru.chat.retry}
            </Button>
          </div>
        ) : null}
      </div>
    </li>
  );
}
