// App: корневой компонент. ErrorBoundary + RouterProvider + Toaster.
//
// Жизненный цикл WS:
//   - На mount: если есть access-токен — loadMe (UC-16).
//   - При переходе auth.status в "authenticated" — wsClient.connect().
//   - При переходе из "authenticated" в любой другой статус — wsClient.disconnect().
//   - Все WS-фреймы прокидываются в соответствующие сторы.

import { useEffect } from "react";
import { RouterProvider } from "react-router-dom";

import { useAuthStore } from "@/features/auth";
import { useChatStore } from "@/features/chat/store";
import { useRoomsStore } from "@/features/rooms/store";
import { router } from "@/routes";
import { tokenStorage } from "@/shared/api/token-storage";
import { wsClient } from "@/shared/api/wsClient.singleton";
import { ErrorBoundary, Toaster } from "@/shared/ui";

export function App(): JSX.Element {
  useEffect(() => {
    // UC-16: восстановление сессии после reload.
    if (tokenStorage.getAccess()) {
      void useAuthStore.getState().loadMe();
    }
  }, []);

  // WS connect/disconnect по auth-статусу.
  useEffect(() => {
    let prevStatus = useAuthStore.getState().status;
    // Стартовый случай: если уже authenticated на момент монтирования (persist
    // восстановил currentUser, плюс прошёл loadMe выше) — connect сразу.
    if (prevStatus === "authenticated") {
      wsClient.connect();
    }
    const unsub = useAuthStore.subscribe((state) => {
      const next = state.status;
      if (next === prevStatus) {
        return;
      }
      if (next === "authenticated" && prevStatus !== "authenticated") {
        wsClient.connect();
      } else if (prevStatus === "authenticated" && next !== "authenticated") {
        wsClient.disconnect();
      }
      prevStatus = next;
    });
    return unsub;
  }, []);

  // Подписка на входящие WS-фреймы → рассылка по сторам.
  useEffect(() => {
    const unsubFrame = wsClient.onFrame((frame) => {
      switch (frame.type) {
        case "subscribed":
          useChatStore.getState().onSubscribed(frame.data);
          break;
        case "message.new":
          useChatStore.getState().onMessageNew(frame.data);
          break;
        case "message.sent":
          useChatStore.getState().onMessageSent(frame.data);
          break;
        case "error":
          useChatStore.getState().onWsError(frame.data);
          break;
        case "member.joined":
          useRoomsStore.getState().appendMember(frame.data);
          break;
      }
    });
    const unsubStatus = wsClient.onStatus((status) => {
      useChatStore.getState().setWsStatus(status);
    });
    return () => {
      unsubFrame();
      unsubStatus();
    };
  }, []);

  return (
    <ErrorBoundary>
      <RouterProvider router={router} />
      <Toaster />
    </ErrorBoundary>
  );
}
