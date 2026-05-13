// DTO и ViewModel для домена chat.
// Источник: docs/3_5_frontend/06-api-integration.md (REST: Chat history).
// Без runtime-кода: только TS-типы.

// === REST DTO (camelCase, из бэка) ===

export type MessageDto = {
  id: string;
  channelId: string;
  authorId: string;
  text: string;
  // RFC3339Nano UTC.
  createdAt: string;
};

export type ListMessagesQuery = {
  // UUID последнего сообщения предыдущей страницы (курсор пагинации).
  before?: string;
  // 1..100, default 50.
  limit?: number;
};

export type ListMessagesResponse = {
  items: MessageDto[];
  nextBefore: string | null;
};

// === ViewModel ===

// Статус сообщения в локальном state:
// - pending — optimistic, ack от сервера ещё не пришёл (Phase 9);
// - committed — подтверждено сервером (через message.sent или message.new);
// - failed — отправка не удалась (Phase 9).
export type MessageStatus = "pending" | "committed" | "failed";

// ViewModel расширяет DTO статусом и опциональным tempId (для pending в Phase 9).
export type Message = MessageDto & {
  status: MessageStatus;
  // Локальный uuid v4, присутствует только пока сообщение pending.
  tempId?: string;
};

// State machine WS-соединения (используется ConnectionStatusBanner и useChatStore).
// Канонический тип живёт в shared/api/ws.ts — здесь re-export для ergonomic-импорта
// из feature-кода.
export type { WsStatus } from "@/shared/api/ws";
