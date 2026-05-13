// useChatStore: Zustand-store домена chat.
// Источник: docs/3_5_frontend/05-state-model.md (useChatStore).
// Решения: D-18 (один store на фичу), D-19 (без persist), D-07 (optimistic),
// D-08 (dedup по id), D-09 (resubscribe при reconnect).
//
// Phase 9: добавлены openChannel, sendMessage, retryMessage, WS-handlers.

import { create } from "zustand";

import * as chatApi from "@/features/chat/api/http";
import { mapChatErrorToMessage } from "@/features/chat/store/mapErrors";
import { useAuthStore } from "@/features/auth/store";
import { useChannelsStore } from "@/features/channels";
import { wsClient } from "@/shared/api/wsClient.singleton";
import { registerLogoutListener } from "@/shared/lib/logoutFlow";
import { toast } from "@/shared/ui/useToast";

import type {
  ErrorFrame,
  MessageNewFrame,
  MessageSentFrame,
  SubscribedFrame,
} from "@/shared/api/errors";
import type { WsStatus } from "@/shared/api/ws";
import type { Message } from "@/features/chat/types";

type ChannelId = string;

// Размер страницы загрузки истории по контракту: 1..100, default 50.
const HISTORY_LIMIT = 50;
// Таймаут ожидания message.sent ack от сервера. По истечении — pending → failed.
const SEND_ACK_TIMEOUT_MS = 10_000;

type ChatState = {
  // Сообщения канала — всегда отсортированы createdAt ASC.
  messagesByChannel: Record<ChannelId, Message[]>;
  // Курсор следующей страницы (UUID). null — пользователь достиг начала истории.
  nextBeforeByChannel: Record<ChannelId, string | null>;
  loadingHistoryByChannel: Record<ChannelId, boolean>;
  loadingMoreHistoryByChannel: Record<ChannelId, boolean>;
  subscribedChannelId: ChannelId | null;
  wsStatus: WsStatus;
};

type ChatActions = {
  loadHistory: (channelId: ChannelId) => Promise<void>;
  loadMoreHistory: (channelId: ChannelId) => Promise<void>;
  openChannel: (channelId: ChannelId) => Promise<void>;
  sendMessage: (channelId: ChannelId, text: string) => Promise<void>;
  retryMessage: (channelId: ChannelId, tempId: string) => Promise<void>;
  onSubscribed: (data: SubscribedFrame["data"]) => void;
  onMessageNew: (data: MessageNewFrame["data"]) => void;
  onMessageSent: (data: MessageSentFrame["data"]) => void;
  onWsError: (data: ErrorFrame["data"]) => void;
  setWsStatus: (status: WsStatus) => void;
  clear: () => void;
};

const initialState: ChatState = {
  messagesByChannel: {},
  nextBeforeByChannel: {},
  loadingHistoryByChannel: {},
  loadingMoreHistoryByChannel: {},
  subscribedChannelId: null,
  wsStatus: "idle",
};

// Каскад на CHAT-002 / CHAT-004: канал недоступен или нас выкинули из комнаты.
// Перегружаем список каналов у соответствующей комнаты — useChannelsStore
// сам прореагирует на 403 каскадом удаления room (см. useChannelsStore.loadChannels).
function cascadeOnChannelLoss(channelId: ChannelId): void {
  const channelsByRoom = useChannelsStore.getState().channelsByRoom;
  for (const [roomId, channels] of Object.entries(channelsByRoom)) {
    if (channels.some((c) => c.id === channelId)) {
      void useChannelsStore.getState().loadChannels(roomId);
      return;
    }
  }
}

// Вставка committed-сообщения с сохранением сортировки ASC по createdAt.
function insertCommittedSorted(
  list: Message[],
  incoming: Message,
): Message[] {
  // Большинство случаев — incoming идёт в конец (новое сообщение).
  const last = list[list.length - 1];
  if (last === undefined || last.createdAt <= incoming.createdAt) {
    return [...list, incoming];
  }
  // Иначе — общая вставка по сравнению createdAt.
  const idx = list.findIndex((m) => m.createdAt > incoming.createdAt);
  if (idx === -1) {
    return [...list, incoming];
  }
  return [...list.slice(0, idx), incoming, ...list.slice(idx)];
}

export const useChatStore = create<ChatState & ChatActions>((set, get) => ({
  ...initialState,

  async loadHistory(channelId) {
    const state = get();
    if (state.loadingHistoryByChannel[channelId] === true) {
      return;
    }
    if (channelId in state.messagesByChannel) {
      return;
    }

    set((s) => ({
      loadingHistoryByChannel: {
        ...s.loadingHistoryByChannel,
        [channelId]: true,
      },
    }));

    const r = await chatApi.listMessages(channelId, { limit: HISTORY_LIMIT });

    if (r.ok) {
      const ascMessages: Message[] = r.data.items
        .map((m) => ({ ...m, status: "committed" as const }))
        .reverse();
      set((s) => ({
        messagesByChannel: {
          ...s.messagesByChannel,
          [channelId]: ascMessages,
        },
        nextBeforeByChannel: {
          ...s.nextBeforeByChannel,
          [channelId]: r.data.nextBefore,
        },
        loadingHistoryByChannel: {
          ...s.loadingHistoryByChannel,
          [channelId]: false,
        },
      }));
      return;
    }

    set((s) => ({
      loadingHistoryByChannel: {
        ...s.loadingHistoryByChannel,
        [channelId]: false,
      },
    }));
    if (r.error.code === "CHAT-002" || r.error.code === "CHAT-004") {
      cascadeOnChannelLoss(channelId);
    }
    toast.error(mapChatErrorToMessage(r.error.code));
  },

  async loadMoreHistory(channelId) {
    const state = get();
    if (state.loadingMoreHistoryByChannel[channelId] === true) {
      return;
    }
    const cursor = state.nextBeforeByChannel[channelId];
    if (cursor === null || cursor === undefined) {
      return;
    }

    set((s) => ({
      loadingMoreHistoryByChannel: {
        ...s.loadingMoreHistoryByChannel,
        [channelId]: true,
      },
    }));

    const r = await chatApi.listMessages(channelId, {
      before: cursor,
      limit: HISTORY_LIMIT,
    });

    if (r.ok) {
      const olderAsc: Message[] = r.data.items
        .map((m) => ({ ...m, status: "committed" as const }))
        .reverse();
      set((s) => {
        const existing = s.messagesByChannel[channelId] ?? [];
        return {
          messagesByChannel: {
            ...s.messagesByChannel,
            [channelId]: [...olderAsc, ...existing],
          },
          nextBeforeByChannel: {
            ...s.nextBeforeByChannel,
            [channelId]: r.data.nextBefore,
          },
          loadingMoreHistoryByChannel: {
            ...s.loadingMoreHistoryByChannel,
            [channelId]: false,
          },
        };
      });
      return;
    }

    set((s) => ({
      loadingMoreHistoryByChannel: {
        ...s.loadingMoreHistoryByChannel,
        [channelId]: false,
      },
    }));
    toast.error(mapChatErrorToMessage(r.error.code));
  },

  async openChannel(channelId) {
    set({ subscribedChannelId: channelId });
    // Грузим историю один раз (loadHistory сам идемпотентен).
    await get().loadHistory(channelId);
    if (get().wsStatus === "open") {
      wsClient.send({ type: "subscribe", channel_id: channelId });
    }
    // Если wsStatus !== "open" — subscribe отправит setWsStatus при переходе в "open".
  },

  async sendMessage(channelId, text) {
    const trimmed = text;
    if (trimmed.length === 0) {
      return;
    }
    const currentUser = useAuthStore.getState().currentUser;
    if (currentUser === null) {
      // Без авторизованного пользователя отправлять нельзя.
      return;
    }
    const tempId = makeTempId();
    const optimistic: Message = {
      id: tempId,
      channelId,
      authorId: currentUser.id,
      text: trimmed,
      createdAt: new Date().toISOString(),
      status: "pending",
      tempId,
    };
    set((s) => {
      const existing = s.messagesByChannel[channelId] ?? [];
      return {
        messagesByChannel: {
          ...s.messagesByChannel,
          [channelId]: [...existing, optimistic],
        },
      };
    });

    wsClient.send({ type: "message.send", channel_id: channelId, text: trimmed });

    // Таймаут ack: если через SEND_ACK_TIMEOUT_MS сообщение всё ещё pending — failed.
    setTimeout(() => {
      const list = get().messagesByChannel[channelId] ?? [];
      const target = list.find((m) => m.tempId === tempId);
      if (target !== undefined && target.status === "pending") {
        set((s) => ({
          messagesByChannel: {
            ...s.messagesByChannel,
            [channelId]: (s.messagesByChannel[channelId] ?? []).map((m) =>
              m.tempId === tempId ? { ...m, status: "failed" } : m,
            ),
          },
        }));
      }
    }, SEND_ACK_TIMEOUT_MS);
  },

  async retryMessage(channelId, tempId) {
    const list = get().messagesByChannel[channelId] ?? [];
    const target = list.find((m) => m.tempId === tempId);
    if (target === undefined || target.status !== "failed") {
      return;
    }
    set((s) => ({
      messagesByChannel: {
        ...s.messagesByChannel,
        [channelId]: (s.messagesByChannel[channelId] ?? []).map((m) =>
          m.tempId === tempId
            ? { ...m, status: "pending", createdAt: new Date().toISOString() }
            : m,
        ),
      },
    }));
    wsClient.send({
      type: "message.send",
      channel_id: channelId,
      text: target.text,
    });
    setTimeout(() => {
      const next = get().messagesByChannel[channelId] ?? [];
      const t = next.find((m) => m.tempId === tempId);
      if (t !== undefined && t.status === "pending") {
        set((s) => ({
          messagesByChannel: {
            ...s.messagesByChannel,
            [channelId]: (s.messagesByChannel[channelId] ?? []).map((m) =>
              m.tempId === tempId ? { ...m, status: "failed" } : m,
            ),
          },
        }));
      }
    }, SEND_ACK_TIMEOUT_MS);
  },

  onSubscribed() {
    // Никакого изменения state не требуется: фактическая подписка — на стороне сервера.
    // Если нужен флаг "ready" — добавим в будущей итерации.
  },

  onMessageNew(data) {
    const currentUserId = useAuthStore.getState().currentUser?.id;
    set((s) => {
      const existing = s.messagesByChannel[data.channel_id] ?? [];
      // D-08 dedup: если уже есть с таким id — игнор.
      if (existing.some((m) => m.id === data.id)) {
        return s;
      }
      // Бэк рассылает message.new ВСЕМ подписчикам канала, включая автора.
      // Если этот фрейм — broadcast моего own send (есть pending с тем же текстом),
      // коммитим этот pending, а не вставляем новый committed (иначе дубль).
      if (currentUserId !== undefined && data.author_id === currentUserId) {
        const pendingIdx = existing.findIndex(
          (m) => m.status === "pending" && m.text === data.text,
        );
        if (pendingIdx !== -1) {
          const next = existing.map((m, i) => {
            if (i !== pendingIdx) {
              return m;
            }
            // eslint-disable-next-line @typescript-eslint/no-unused-vars -- выкидываем tempId через rest
            const { tempId: _tempId, ...rest } = m;
            return {
              ...rest,
              id: data.id,
              createdAt: data.created_at,
              status: "committed" as const,
            };
          });
          next.sort((a, b) => (a.createdAt < b.createdAt ? -1 : 1));
          return {
            messagesByChannel: {
              ...s.messagesByChannel,
              [data.channel_id]: next,
            },
          };
        }
      }
      const incoming: Message = {
        id: data.id,
        channelId: data.channel_id,
        authorId: data.author_id,
        text: data.text,
        createdAt: data.created_at,
        status: "committed",
      };
      return {
        messagesByChannel: {
          ...s.messagesByChannel,
          [data.channel_id]: insertCommittedSorted(existing, incoming),
        },
      };
    });
  },

  onMessageSent(data) {
    set((s) => {
      const existing = s.messagesByChannel[data.channel_id] ?? [];
      // Берём последний pending (UC по дизайну: send → ack в порядке очереди).
      let lastPendingIdx = -1;
      for (let i = existing.length - 1; i >= 0; i -= 1) {
        const m = existing[i];
        if (m !== undefined && m.status === "pending") {
          lastPendingIdx = i;
          break;
        }
      }
      if (lastPendingIdx === -1) {
        return s;
      }
      const next: Message[] = existing.map((m, i) => {
        if (i !== lastPendingIdx) {
          return m;
        }
        // tempId опционален с exactOptionalPropertyTypes — выкидываем поле, а не сетим undefined.
        const {
          // eslint-disable-next-line @typescript-eslint/no-unused-vars -- удаляем tempId через rest
          tempId: _tempId,
          ...rest
        } = m;
        return {
          ...rest,
          id: data.id,
          createdAt: data.created_at,
          status: "committed" as const,
        };
      });
      // На случай, если created_at от сервера сместил порядок — пересортируем.
      next.sort((a, b) => (a.createdAt < b.createdAt ? -1 : 1));
      return {
        messagesByChannel: { ...s.messagesByChannel, [data.channel_id]: next },
      };
    });
  },

  onWsError(data) {
    const code = data.code;
    if (code === "CHAT-001") {
      // Невалидный текст: помечаем последний pending в текущем подписанном канале как failed.
      const channelId = get().subscribedChannelId;
      if (channelId !== null) {
        set((s) => {
          const existing = s.messagesByChannel[channelId] ?? [];
          let lastPendingIdx = -1;
          for (let i = existing.length - 1; i >= 0; i -= 1) {
            const m = existing[i];
            if (m !== undefined && m.status === "pending") {
              lastPendingIdx = i;
              break;
            }
          }
          if (lastPendingIdx === -1) {
            return s;
          }
          const next = existing.map((m, i) =>
            i === lastPendingIdx ? { ...m, status: "failed" as const } : m,
          );
          return {
            messagesByChannel: { ...s.messagesByChannel, [channelId]: next },
          };
        });
      }
      toast.error(mapChatErrorToMessage(code));
      return;
    }
    if (code === "CHAT-002" || code === "CHAT-004") {
      const channelId = get().subscribedChannelId;
      if (channelId !== null) {
        cascadeOnChannelLoss(channelId);
      }
      toast.error(mapChatErrorToMessage(code));
      return;
    }
    if (code === "CHAT-003" || code === "INTERNAL") {
      toast.error(mapChatErrorToMessage(code));
      return;
    }
    // Остальные коды (CHAT-005/007) — это фронт-баг: log и тихо.
    toast.error(mapChatErrorToMessage(code));
  },

  setWsStatus(status) {
    const prev = get().wsStatus;
    if (prev === status) {
      return;
    }
    set({ wsStatus: status });
    // При переходе в "open" с активной подпиской — (ре)подписываемся.
    if (prev !== "open" && status === "open") {
      const channelId = get().subscribedChannelId;
      if (channelId !== null) {
        wsClient.send({ type: "subscribe", channel_id: channelId });
      }
    }
  },

  clear() {
    set({ ...initialState });
  },
}));

// Регистрируем clear() как listener для каскадного logoutFlow.
registerLogoutListener(() => {
  useChatStore.getState().clear();
});

// crypto.randomUUID есть в Node 19+ / современных браузерах. Тип в lib.dom есть.
// Wrap'аем для возможности подмены в тестах, если потребуется.
function makeTempId(): string {
  return crypto.randomUUID();
}
