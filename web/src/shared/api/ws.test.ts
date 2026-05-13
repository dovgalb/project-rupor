// Тесты shared/api/ws.ts:
// - URL содержит ?token из tokenStorage;
// - onopen → status="open" + сброс backoff;
// - close 1000 → status="closed", без reconnect;
// - close 1006 → status="reconnecting", через 1s — новый connect;
// - disconnect() → close(1000), reconnect-таймер отменяется;
// - send() пишет в socket; при !open — drop;
// - parseFrame: валидные/невалидные формы.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { buildWsUrl, createWsClient, parseFrame } from "@/shared/api/ws";

import type { CreateWsClientDeps, WsClient } from "@/shared/api/ws";

// === Helpers: мокаемый WebSocket ===

type Listeners = {
  onopen: ((ev?: Event) => void) | null;
  onmessage: ((ev: MessageEvent) => void) | null;
  onerror: ((ev?: Event) => void) | null;
  onclose: ((ev: CloseEvent) => void) | null;
};

class FakeSocket implements Listeners {
  static instances: FakeSocket[] = [];
  // Очередь "отправленных" фреймов.
  sent: string[] = [];
  close = vi.fn();
  onopen: ((ev?: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  onerror: ((ev?: Event) => void) | null = null;
  onclose: ((ev: CloseEvent) => void) | null = null;
  url: string;

  constructor(url: string) {
    this.url = url;
    FakeSocket.instances.push(this);
  }

  send = vi.fn((data: string) => {
    this.sent.push(data);
  });

  emitOpen(): void {
    this.onopen?.();
  }
  emitClose(code: number, reason = ""): void {
    this.onclose?.({ code, reason, wasClean: code === 1000 } as CloseEvent);
  }
  emitMessage(raw: string): void {
    this.onmessage?.({ data: raw } as MessageEvent);
  }
}

function makeFactory(): (url: string) => WebSocket {
  return (url: string) => new FakeSocket(url) as unknown as WebSocket;
}

function buildClient(overrides: Partial<CreateWsClientDeps> = {}): WsClient {
  return createWsClient({
    getToken: () => "tkn-123",
    baseUrl: "ws://test.local",
    socketFactory: makeFactory(),
    ...overrides,
  });
}

beforeEach(() => {
  FakeSocket.instances.length = 0;
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("buildWsUrl", () => {
  it("подставляет baseUrl и URL-encoded токен в query", () => {
    const url = buildWsUrl("a b/c", "ws://example.com");
    expect(url).toBe("ws://example.com/api/v1/ws?token=a%20b%2Fc");
  });
});

describe("parseFrame", () => {
  it("разбирает валидный message.new", () => {
    const raw = JSON.stringify({
      type: "message.new",
      data: {
        id: "m1",
        channel_id: "c1",
        author_id: "u1",
        text: "hi",
        created_at: "2030-01-01T00:00:00Z",
      },
    });
    expect(parseFrame(raw)).toEqual({
      type: "message.new",
      data: {
        id: "m1",
        channel_id: "c1",
        author_id: "u1",
        text: "hi",
        created_at: "2030-01-01T00:00:00Z",
      },
    });
  });

  it("возвращает null на неизвестный type", () => {
    const raw = JSON.stringify({ type: "weird", data: {} });
    expect(parseFrame(raw)).toBeNull();
  });

  it("возвращает null на невалидный JSON", () => {
    expect(parseFrame("not-json")).toBeNull();
  });

  it("возвращает null если у message.new отсутствует поле", () => {
    const raw = JSON.stringify({
      type: "message.new",
      data: { id: "m1", channel_id: "c1" }, // нет author_id/text/created_at
    });
    expect(parseFrame(raw)).toBeNull();
  });
});

describe("createWsClient.connect", () => {
  it("на connect ставит status=connecting и собирает URL с ?token=", () => {
    const statuses: string[] = [];
    const client = buildClient();
    client.onStatus((s) => statuses.push(s));
    client.connect();
    expect(statuses).toContain("connecting");
    const inst = FakeSocket.instances[0];
    expect(inst).toBeDefined();
    expect(inst?.url).toBe("ws://test.local/api/v1/ws?token=tkn-123");
  });

  it("на onopen ставит status=open", () => {
    const statuses: string[] = [];
    const client = buildClient();
    client.onStatus((s) => statuses.push(s));
    client.connect();
    FakeSocket.instances[0]?.emitOpen();
    expect(client.getStatus()).toBe("open");
    expect(statuses).toContain("open");
  });

  it("при getToken=null остаётся idle и не создаёт сокет", () => {
    const client = buildClient({ getToken: () => null });
    client.connect();
    expect(FakeSocket.instances.length).toBe(0);
    expect(client.getStatus()).toBe("idle");
  });
});

describe("createWsClient close-coded", () => {
  it("close 1000 → status=closed и реконнекта не происходит", () => {
    const client = buildClient();
    client.connect();
    const sock = FakeSocket.instances[0];
    sock?.emitOpen();
    sock?.emitClose(1000);
    expect(client.getStatus()).toBe("closed");
    vi.advanceTimersByTime(60_000);
    expect(FakeSocket.instances.length).toBe(1);
  });

  it("close 1006 → status=reconnecting, через 1s — новый connect", () => {
    const client = buildClient();
    client.connect();
    FakeSocket.instances[0]?.emitOpen();
    FakeSocket.instances[0]?.emitClose(1006);
    expect(client.getStatus()).toBe("reconnecting");
    expect(FakeSocket.instances.length).toBe(1);
    vi.advanceTimersByTime(1000);
    expect(FakeSocket.instances.length).toBe(2);
    expect(client.getStatus()).toBe("connecting");
  });

  it("второй разрыв даёт backoff 2s; пятый — 15s (потолок)", () => {
    const client = buildClient();
    client.connect();
    // attempt 0 → 1000ms
    FakeSocket.instances[0]?.emitOpen();
    FakeSocket.instances[0]?.emitClose(1006);
    vi.advanceTimersByTime(1000);
    // attempt 1 → 2000ms
    FakeSocket.instances[1]?.emitClose(1006);
    vi.advanceTimersByTime(1999);
    expect(FakeSocket.instances.length).toBe(2);
    vi.advanceTimersByTime(1);
    expect(FakeSocket.instances.length).toBe(3);
    // attempt 2 → 5000ms
    FakeSocket.instances[2]?.emitClose(1006);
    vi.advanceTimersByTime(5000);
    expect(FakeSocket.instances.length).toBe(4);
    // attempt 3 → 15000ms
    FakeSocket.instances[3]?.emitClose(1006);
    vi.advanceTimersByTime(15000);
    expect(FakeSocket.instances.length).toBe(5);
    // attempt 4+ → потолок 15s
    FakeSocket.instances[4]?.emitClose(1006);
    vi.advanceTimersByTime(15000);
    expect(FakeSocket.instances.length).toBe(6);
  });
});

describe("createWsClient.disconnect", () => {
  it("закрывает сокет с кодом 1000 и отменяет reconnect", () => {
    const client = buildClient();
    client.connect();
    FakeSocket.instances[0]?.emitOpen();
    client.disconnect();
    // close вызван с 1000
    expect(FakeSocket.instances[0]?.close).toHaveBeenCalledWith(
      1000,
      "client logout",
    );
    expect(client.getStatus()).toBe("closed");
    // Эмулируем onclose который мог прийти после disconnect — реконнекта быть не должно.
    FakeSocket.instances[0]?.emitClose(1006);
    vi.advanceTimersByTime(60_000);
    expect(FakeSocket.instances.length).toBe(1);
  });
});

describe("createWsClient.send", () => {
  it("после open отправляет JSON-фрейм в сокет", () => {
    const client = buildClient();
    client.connect();
    FakeSocket.instances[0]?.emitOpen();
    client.send({ type: "subscribe", channel_id: "c1" });
    expect(FakeSocket.instances[0]?.sent).toEqual([
      JSON.stringify({ type: "subscribe", channel_id: "c1" }),
    ]);
  });

  it("если статус != open — фрейм дропается без исключения", () => {
    const client = buildClient();
    client.connect();
    // НЕ эмулируем onopen.
    client.send({ type: "subscribe", channel_id: "c1" });
    expect(FakeSocket.instances[0]?.sent.length).toBe(0);
  });
});

describe("createWsClient.onFrame", () => {
  it("после onmessage эмитит распарсенный фрейм всем подписчикам", () => {
    const client = buildClient();
    const seen: string[] = [];
    client.onFrame((f) => seen.push(f.type));
    client.connect();
    FakeSocket.instances[0]?.emitOpen();
    FakeSocket.instances[0]?.emitMessage(
      JSON.stringify({ type: "subscribed", data: { channel_id: "c1" } }),
    );
    expect(seen).toEqual(["subscribed"]);
  });

  it("невалидный фрейм не попадает в onFrame", () => {
    const client = buildClient();
    const seen: string[] = [];
    client.onFrame((f) => seen.push(f.type));
    client.connect();
    FakeSocket.instances[0]?.emitOpen();
    FakeSocket.instances[0]?.emitMessage("garbage");
    expect(seen).toEqual([]);
  });
});

describe("createWsClient.reconnect", () => {
  it("после reconnect статус сначала reconnecting, потом connecting на тик", () => {
    const client = buildClient();
    client.connect();
    FakeSocket.instances[0]?.emitOpen();
    expect(client.getStatus()).toBe("open");
    client.reconnect();
    // socket.close был вызван, статус reconnecting (через гонку-ветку)
    expect(FakeSocket.instances[0]?.close).toHaveBeenCalled();
    // На тик 0ms запускается новый connect.
    vi.advanceTimersByTime(0);
    expect(FakeSocket.instances.length).toBe(2);
    expect(client.getStatus()).toBe("connecting");
  });
});
