// In-memory fake WsClient для unit-тестов сторов.
// Используется через vi.mock("@/shared/api/wsClient.singleton", ...).
//
// API совпадает с WsClient + два test-hook'а:
//   _emit(frame)    — эмулировать входящий фрейм
//   _setStatus(s)   — эмулировать смену WS-статуса
// Дополнительно: send — vi.fn(), мы проверяем вызовы.

import { vi, type Mock } from "vitest";

import type {
  IncomingFrame,
  OutgoingFrame,
} from "@/shared/api/errors";
import type {
  FrameHandler,
  StatusHandler,
  WsClient,
  WsStatus,
} from "@/shared/api/ws";

export type FakeWsClient = WsClient & {
  send: Mock<(frame: OutgoingFrame) => void>;
  _emit: (frame: IncomingFrame) => void;
  _setStatus: (status: WsStatus) => void;
  _frameHandlers: Set<FrameHandler>;
  _statusHandlers: Set<StatusHandler>;
};

export function createFakeWsClient(): FakeWsClient {
  let status: WsStatus = "idle";
  const frameHandlers = new Set<FrameHandler>();
  const statusHandlers = new Set<StatusHandler>();

  const fake: FakeWsClient = {
    connect: vi.fn(),
    disconnect: vi.fn(),
    reconnect: vi.fn(),
    send: vi.fn(),
    getStatus: (): WsStatus => status,
    onFrame: (h: FrameHandler): (() => void) => {
      frameHandlers.add(h);
      return () => {
        frameHandlers.delete(h);
      };
    },
    onStatus: (h: StatusHandler): (() => void) => {
      statusHandlers.add(h);
      return () => {
        statusHandlers.delete(h);
      };
    },
    _emit: (frame: IncomingFrame): void => {
      for (const h of frameHandlers) {
        h(frame);
      }
    },
    _setStatus: (s: WsStatus): void => {
      status = s;
      for (const h of statusHandlers) {
        h(s);
      }
    },
    _frameHandlers: frameHandlers,
    _statusHandlers: statusHandlers,
  };
  return fake;
}
