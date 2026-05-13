// MessageList: список сообщений + infinite-scroll вверх + auto-scroll к низу.
// Источник: docs/3_5_frontend/07-ui-contract.md (<MessageList />), UC-13.
//
// Состояния:
//   loading первичный + пусто       → Spinner
//   ready + 0 сообщений             → <EmptyChat />
//   ready + есть сообщения          → <ul role="log" aria-live="polite">
//
// Infinite scroll: IntersectionObserver на sentinel в начале списка.
// При nextBefore=null показываем "Это начало канала".

import { useEffect, useRef } from "react";

import { EmptyChat } from "@/features/chat/components/EmptyChat";
import { MessageItem } from "@/features/chat/components/MessageItem";
import { useChatStore } from "@/features/chat/store";
import { useAuthStore } from "@/features/auth/store";
import { Spinner } from "@/shared/ui/Spinner";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./MessageList.module.css";

import type { Message } from "@/features/chat/types";

type MessageListProps = {
  channelId: string;
};

// Стабильная ссылка для пустого массива — селектор не вызывает ререндер.
const EMPTY_MESSAGES: Message[] = [];

// Короткий placeholder автора для D-11: первые 6 hex-символов uppercase.
function firstSixHex(userId: string): string {
  return userId.replace(/-/g, "").slice(0, 6).toUpperCase();
}

export function MessageList({ channelId }: MessageListProps): JSX.Element {
  const messages = useChatStore(
    (s) => s.messagesByChannel[channelId] ?? EMPTY_MESSAGES,
  );
  const loadingHistory = useChatStore(
    (s) => s.loadingHistoryByChannel[channelId] ?? false,
  );
  const loadingMore = useChatStore(
    (s) => s.loadingMoreHistoryByChannel[channelId] ?? false,
  );
  const nextBefore = useChatStore(
    (s) => s.nextBeforeByChannel[channelId] ?? null,
  );

  const currentUserId = useAuthStore((s) => s.currentUser?.id);
  const meUsername = useAuthStore((s) => s.currentUser?.username);

  const containerRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  // Чтобы автоскролл сработал ровно один раз после первичной загрузки.
  const hasScrolledRef = useRef(false);

  // При смене канала: грузим историю + подписываемся через WS.
  useEffect(() => {
    hasScrolledRef.current = false;
    void useChatStore.getState().openChannel(channelId);
  }, [channelId]);

  // Автоскролл к низу после первичной загрузки сообщений.
  useEffect(() => {
    if (
      !loadingHistory &&
      messages.length > 0 &&
      !hasScrolledRef.current &&
      containerRef.current !== null
    ) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
      hasScrolledRef.current = true;
    }
  }, [loadingHistory, messages.length]);

  // IntersectionObserver на sentinel — подгружаем старые сообщения при скролле вверх.
  useEffect(() => {
    const sentinel = sentinelRef.current;
    const root = containerRef.current;
    if (sentinel === null || root === null) {
      return;
    }
    if (typeof IntersectionObserver === "undefined") {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        const first = entries[0];
        if (first === undefined || !first.isIntersecting) {
          return;
        }
        if (nextBefore === null) {
          return;
        }
        if (useChatStore.getState().loadingMoreHistoryByChannel[channelId]) {
          return;
        }
        void useChatStore.getState().loadMoreHistory(channelId);
      },
      { root },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [channelId, nextBefore]);

  // === Рендер ===

  if (loadingHistory && messages.length === 0) {
    return (
      <div className={styles.center}>
        <Spinner size="md" />
      </div>
    );
  }

  if (!loadingHistory && messages.length === 0) {
    return (
      <div className={styles.center}>
        <EmptyChat />
      </div>
    );
  }

  return (
    <div ref={containerRef} className={styles.scroll}>
      <div ref={sentinelRef} className={styles.sentinel} aria-hidden="true" />
      {loadingMore ? (
        <div className={styles.loadingMore}>
          <Spinner size="sm" />
        </div>
      ) : null}
      {nextBefore === null && messages.length > 0 ? (
        <p className={styles.beginning}>{ru.chat.beginningOfChannel}</p>
      ) : null}
      <ul className={styles.list} role="log" aria-live="polite">
        {messages.map((m) => {
          const isAuthor = currentUserId !== undefined && m.authorId === currentUserId;
          const placeholder =
            isAuthor && meUsername !== undefined
              ? meUsername
              : firstSixHex(m.authorId);
          return (
            <MessageItem
              key={m.tempId ?? m.id}
              message={m}
              isAuthor={isAuthor}
              authorPlaceholder={placeholder}
            />
          );
        })}
      </ul>
    </div>
  );
}
