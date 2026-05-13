// Phase 11: WS reconnect-сценарий E2E-13.
//
// SKIPPED: Playwright не предоставляет надёжного способа закрыть живой WebSocket
// в Chromium из теста: context.setOffline(true) блокирует новые HTTP/WS upgrade,
// но НЕ обрывает уже установленные WS-соединения. Логика reconnect полностью
// покрыта unit-тестами (см. web/src/shared/api/ws.test.ts: «close 1006 →
// reconnecting + backoff»). Manual verification — см. phase-09.md / manual_qa.

import { test } from "@playwright/test";

test.skip("E2E-13: разрыв WS → banner reconnecting → восстановление", () => {
  // Логика проверена в shared/api/ws.test.ts (backoff и resubscribe).
  // Ручной сценарий: kill бэка → banner появляется; restart → banner исчезает.
});
