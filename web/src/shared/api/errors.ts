// Общие TS-типы для API (REST + WebSocket). Без runtime-кода.
// Источник: docs/3_5_frontend/06-api-integration.md.

// === ErrorEnvelope и ApiResult ===

export type ErrorEnvelope = {
  error: { code: string; message: string };
};

export type ApiOk<T> = { ok: true; data: T };
export type ApiErr = { ok: false; status: number; error: ErrorEnvelope["error"] };
export type ApiResult<T> = ApiOk<T> | ApiErr;

// === Литералы доменных кодов ===

export type AuthErrorCode =
  | "AUTH-001"
  | "AUTH-002"
  | "AUTH-003"
  | "AUTH-004"
  | "AUTH-005"
  | "AUTH-006"
  | "AUTH-007"
  | "AUTH-008"
  | "AUTH-009"
  | "AUTH-010"
  | "AUTH-011"
  | "AUTH-012";

export type RoomErrorCode =
  | "ROOM-001"
  | "ROOM-002"
  | "ROOM-003"
  | "ROOM-004"
  | "ROOM-005"
  | "ROOM-006"
  | "ROOM-007"
  | "ROOM-008"
  | "ROOM-009";

export type ChannelErrorCode =
  | "CHANNEL-001"
  | "CHANNEL-002"
  | "CHANNEL-003"
  | "CHANNEL-004"
  | "CHANNEL-005"
  | "CHANNEL-006"
  | "CHANNEL-007";

export type ChatErrorCode =
  | "CHAT-001"
  | "CHAT-002"
  | "CHAT-003"
  | "CHAT-004"
  | "CHAT-005"
  | "CHAT-006"
  | "CHAT-007";

// NETWORK/CLIENT/INTERNAL — псевдо-коды, которые ставит fetch-обёртка
// при сетевой ошибке / невалидном JSON / внутренней ошибке клиента.
export type ClientErrorCode = "NETWORK" | "CLIENT" | "INTERNAL";

// === REST DTO, нужные внутри shared/api ===

// TokensResponse — единственный REST-DTO, который нужен в shared/api
// (используется в refresh.ts). Остальные DTO живут в features/*/types.ts.
export type TokensResponse = {
  accessToken: string;
  refreshToken: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
};

// === WebSocket outbound frames (snake_case в payload) ===

export type SubscribeFrame = {
  type: "subscribe";
  channel_id: string;
};

export type MessageSendFrame = {
  type: "message.send";
  channel_id: string;
  text: string;
};

export type OutgoingFrame = SubscribeFrame | MessageSendFrame;

// === WebSocket inbound frames (envelope { type, data }, snake_case в data) ===

export type SubscribedFrame = {
  type: "subscribed";
  data: { channel_id: string };
};

export type MessageNewFrame = {
  type: "message.new";
  data: {
    id: string;
    channel_id: string;
    author_id: string;
    text: string;
    created_at: string;
  };
};

export type MessageSentFrame = {
  type: "message.sent";
  data: {
    id: string;
    channel_id: string;
    created_at: string;
  };
};

export type MemberJoinedFrame = {
  type: "member.joined";
  data: {
    room_id: string;
    user_id: string;
    joined_at: string;
  };
};

export type ErrorFrame = {
  type: "error";
  data: { code: string; message: string };
};

export type IncomingFrame =
  | SubscribedFrame
  | MessageNewFrame
  | MessageSentFrame
  | MemberJoinedFrame
  | ErrorFrame;
