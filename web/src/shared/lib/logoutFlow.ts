// Каскадная очистка локального состояния при logout.
// Решение D-06: эндпоинта logout нет, операция чисто локальная.
//
// Registration pattern: features регистрируют свои clear()-листенеры при
// инициализации соответствующих сторов. Это позволяет shared/lib/logoutFlow
// не импортировать features/* и соблюсти правило 5 из 01-architecture.md
// («shared/ ни от кого внутри src/ не зависит»).

import { tokenStorage } from "@/shared/api/token-storage";
import { wsClient } from "@/shared/api/wsClient.singleton";
import { logger } from "@/shared/lib/logger";

type LogoutListener = () => void;

const listeners = new Set<LogoutListener>();

export function registerLogoutListener(fn: LogoutListener): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

// Только для тестов: сбросить всех зарегистрированных листенеров.
export function _resetLogoutListenersForTests(): void {
  listeners.clear();
}

export function logoutFlow(): void {
  tokenStorage.clear();
  // Закрываем WS с кодом 1000 — без последующего реконнекта.
  wsClient.disconnect();
  for (const fn of listeners) {
    try {
      fn();
    } catch (err) {
      logger.error("logoutFlow listener failed:", err);
    }
  }
  logger.info("logout: storage cleared, WS disconnected, listeners notified");
}
