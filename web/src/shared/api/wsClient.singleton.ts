// Глобальный singleton WS-клиента.
// Импортируется сторами (chat/rooms) и App.tsx — единая точка получения instance.
// В тестах подменяется через `vi.mock("@/shared/api/wsClient.singleton", ...)`.

import { createWsClient } from "@/shared/api/ws";
import { tokenStorage } from "@/shared/api/token-storage";

export const wsClient = createWsClient({
  getToken: () => tokenStorage.getAccess(),
});
