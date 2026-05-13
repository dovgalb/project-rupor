// WS-клиент: factory createWsClient.
// Источник: docs/3_5_frontend/06-api-integration.md (WebSocket-протокол) +
// docs/3_5_frontend/plan/phase-09.md.
//
// Контракт:
// - Outbound: плоский JSON, snake_case (SubscribeFrame, MessageSendFrame).
// - Inbound: { type, data }, snake_case в data (parseFrame).
// - Reconnect: backoff [1000, 2000, 5000, 15000] (последний — бесконечный потолок).
// - Закрытие 1000 — нормальное, реконнект не делаем.
// - WS-токен передаём в query ?token=<jwt>; в логах/исключениях не должен утекать.

import { logger } from "@/shared/lib/logger";

import type {
  IncomingFrame,
  OutgoingFrame,
} from "@/shared/api/errors";

// === Публичные типы ===

export type WsStatus =
  | "idle"
  | "connecting"
  | "open"
  | "reconnecting"
  | "closed";

export type FrameHandler = (frame: IncomingFrame) => void;
export type StatusHandler = (status: WsStatus) => void;

export type WsClient = {
  connect: () => void;
  disconnect: () => void;
  reconnect: () => void;
  send: (frame: OutgoingFrame) => void;
  getStatus: () => WsStatus;
  onFrame: (handler: FrameHandler) => () => void;
  onStatus: (handler: StatusHandler) => () => void;
};

export type CreateWsClientDeps = {
  getToken: () => string | null;
  // Если null/undefined — derive из window.location (текущий host, ws/wss по protocol).
  baseUrl?: string;
  // Перегружаемая фабрика WebSocket — для тестов.
  socketFactory?: (url: string) => WebSocket;
};

// Backoff-шаги при reconnect. Финальный шаг (15s) применяется бесконечно.
const BACKOFF_MS = [1000, 2000, 5000, 15000];

function pickBackoff(attempt: number): number {
  const last = BACKOFF_MS[BACKOFF_MS.length - 1] ?? 15000;
  return BACKOFF_MS[Math.min(attempt, BACKOFF_MS.length - 1)] ?? last;
}

// === parseFrame ===

// Проверяет, что значение — non-null object.
function isObj(v: unknown): v is Record<string, unknown> {
  return typeof v === "object" && v !== null;
}

function isString(v: unknown): v is string {
  return typeof v === "string";
}

// Парсер входящих фреймов. На любую невалидную форму — null + warn.
// Сама функция чистая: одинаковый ввод → одинаковый вывод.
export function parseFrame(raw: string): IncomingFrame | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    logger.warn("ws: failed to parse incoming frame as JSON");
    return null;
  }
  if (!isObj(parsed)) {
    logger.warn("ws: incoming frame is not an object");
    return null;
  }
  const type = parsed["type"];
  const data = parsed["data"];
  if (!isString(type) || !isObj(data)) {
    logger.warn("ws: incoming frame missing type/data");
    return null;
  }
  switch (type) {
    case "subscribed": {
      if (!isString(data["channel_id"])) {
        logger.warn("ws: subscribed frame missing channel_id");
        return null;
      }
      return { type: "subscribed", data: { channel_id: data["channel_id"] } };
    }
    case "message.new": {
      if (
        !isString(data["id"]) ||
        !isString(data["channel_id"]) ||
        !isString(data["author_id"]) ||
        !isString(data["text"]) ||
        !isString(data["created_at"])
      ) {
        logger.warn("ws: message.new frame is malformed");
        return null;
      }
      return {
        type: "message.new",
        data: {
          id: data["id"],
          channel_id: data["channel_id"],
          author_id: data["author_id"],
          text: data["text"],
          created_at: data["created_at"],
        },
      };
    }
    case "message.sent": {
      if (
        !isString(data["id"]) ||
        !isString(data["channel_id"]) ||
        !isString(data["created_at"])
      ) {
        logger.warn("ws: message.sent frame is malformed");
        return null;
      }
      return {
        type: "message.sent",
        data: {
          id: data["id"],
          channel_id: data["channel_id"],
          created_at: data["created_at"],
        },
      };
    }
    case "member.joined": {
      if (
        !isString(data["room_id"]) ||
        !isString(data["user_id"]) ||
        !isString(data["joined_at"])
      ) {
        logger.warn("ws: member.joined frame is malformed");
        return null;
      }
      return {
        type: "member.joined",
        data: {
          room_id: data["room_id"],
          user_id: data["user_id"],
          joined_at: data["joined_at"],
        },
      };
    }
    case "error": {
      if (!isString(data["code"]) || !isString(data["message"])) {
        logger.warn("ws: error frame is malformed");
        return null;
      }
      return {
        type: "error",
        data: { code: data["code"], message: data["message"] },
      };
    }
    default:
      logger.warn("ws: unsupported frame type");
      return null;
  }
}

// === buildWsUrl ===

// Сборка WS-URL. Токен ИДЁТ в query — иначе WebSocket API не поддерживает headers.
export function buildWsUrl(token: string, baseUrl?: string): string {
  const base = baseUrl ?? defaultBaseUrl();
  return `${base}/api/v1/ws?token=${encodeURIComponent(token)}`;
}

function defaultBaseUrl(): string {
  if (typeof window === "undefined") {
    return "ws://localhost";
  }
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${proto}//${window.location.host}`;
}

// === createWsClient ===

export function createWsClient(deps: CreateWsClientDeps): WsClient {
  // === Внутреннее состояние клиента ===
  let socket: WebSocket | null = null;
  let status: WsStatus = "idle";
  // Признак того, что закрытие инициировано клиентом (через disconnect).
  // В этом случае реконнект не делаем.
  let intentionalClose = false;
  // Номер текущей попытки реконнекта (для выбора задержки backoff).
  let attempt = 0;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  const frameHandlers = new Set<FrameHandler>();
  const statusHandlers = new Set<StatusHandler>();

  const factory =
    deps.socketFactory ?? ((url: string): WebSocket => new WebSocket(url));

  function setStatus(next: WsStatus): void {
    if (status === next) {
      return;
    }
    status = next;
    for (const h of statusHandlers) {
      try {
        h(next);
      } catch (err) {
        logger.error("ws: status handler failed", err);
      }
    }
  }

  function emitFrame(frame: IncomingFrame): void {
    for (const h of frameHandlers) {
      try {
        h(frame);
      } catch (err) {
        logger.error("ws: frame handler failed", err);
      }
    }
  }

  function clearReconnectTimer(): void {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  }

  function scheduleReconnect(): void {
    clearReconnectTimer();
    const delay = pickBackoff(attempt);
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      attempt += 1;
      connect();
    }, delay);
  }

  function connect(): void {
    if (status === "open" || status === "connecting") {
      return;
    }
    // На обычный connect — снимаем флаг intentionalClose (если был выставлен).
    intentionalClose = false;

    const token = deps.getToken();
    if (token === null) {
      // Без токена — подключаться не пытаемся, статус остаётся как был / станет idle.
      // Не реконнектим: пусть auth-слой сам инициирует connect когда login пройдёт.
      setStatus("idle");
      return;
    }

    setStatus("connecting");
    let ws: WebSocket;
    try {
      ws = factory(buildWsUrl(token, deps.baseUrl));
    } catch (err) {
      logger.error("ws: failed to construct WebSocket", err);
      // Считаем это эквивалентом разрыва — переходим в reconnecting с backoff.
      setStatus("reconnecting");
      scheduleReconnect();
      return;
    }
    socket = ws;

    ws.onopen = () => {
      attempt = 0;
      setStatus("open");
    };

    ws.onmessage = (event: MessageEvent) => {
      if (typeof event.data !== "string") {
        // Бинарные фреймы не используем.
        return;
      }
      const frame = parseFrame(event.data);
      if (frame !== null) {
        emitFrame(frame);
      }
    };

    ws.onerror = () => {
      // Логируем без деталей: подробности есть в onclose, плюс URL содержит токен.
      logger.warn("ws: socket error");
    };

    ws.onclose = (event: CloseEvent) => {
      // После close события socket нам больше не нужен.
      socket = null;
      if (intentionalClose) {
        setStatus("closed");
        return;
      }
      if (event.code === 1000) {
        // Нормальное закрытие со стороны сервера — реконнект не нужен.
        setStatus("closed");
        return;
      }
      setStatus("reconnecting");
      scheduleReconnect();
    };
  }

  function disconnect(): void {
    intentionalClose = true;
    clearReconnectTimer();
    attempt = 0;
    if (socket !== null) {
      try {
        socket.close(1000, "client logout");
      } catch (err) {
        logger.warn("ws: close threw", err);
      }
      socket = null;
    }
    setStatus("closed");
  }

  function reconnect(): void {
    // Принудительный переподключение без backoff (D-10: после joinByCode).
    clearReconnectTimer();
    attempt = 0;
    if (socket !== null) {
      // Помечаем как намеренное закрытие, чтобы onclose не запустил backoff,
      // и сразу же запускаем connect (он сбросит intentionalClose обратно).
      intentionalClose = true;
      try {
        socket.close(1000, "client reconnect");
      } catch (err) {
        logger.warn("ws: close threw", err);
      }
      socket = null;
    }
    // Между сетами/слушателями onclose может ещё не успеть отстреляться,
    // но статус мы обновим через connect → setStatus("connecting").
    intentionalClose = false;
    // Если status был "closed" — connect его подымет; если был "reconnecting" —
    // тоже норм, connect-guard разрешит этот переход.
    if (status !== "open" && status !== "connecting") {
      connect();
    } else {
      // Случай гонки: socket.close ещё не дошёл, но статус ещё "open".
      // Принудительно опускаем в reconnecting и connect через scheduleReconnect(0).
      setStatus("reconnecting");
      // Не используем scheduleReconnect (он берёт backoff из attempt) — делаем 0 delay.
      reconnectTimer = setTimeout(() => {
        reconnectTimer = null;
        connect();
      }, 0);
    }
  }

  function send(frame: OutgoingFrame): void {
    if (status !== "open" || socket === null) {
      logger.warn("ws: send called while not open, frame dropped", { type: frame.type });
      return;
    }
    try {
      socket.send(JSON.stringify(frame));
    } catch (err) {
      logger.warn("ws: send threw", err);
    }
  }

  function onFrame(handler: FrameHandler): () => void {
    frameHandlers.add(handler);
    return () => {
      frameHandlers.delete(handler);
    };
  }

  function onStatus(handler: StatusHandler): () => void {
    statusHandlers.add(handler);
    return () => {
      statusHandlers.delete(handler);
    };
  }

  function getStatus(): WsStatus {
    return status;
  }

  return {
    connect,
    disconnect,
    reconnect,
    send,
    getStatus,
    onFrame,
    onStatus,
  };
}
