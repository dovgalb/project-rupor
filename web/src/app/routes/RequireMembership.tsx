// RequireMembership: гард доступа к комнате.
// Источник: docs/3_5_frontend/08-routes.md (Guards).
// Проверяет, что :roomId присутствует в useRoomsStore.rooms.
// status=idle/loading → spinner; нет комнаты → toast + redirect /rooms.

import { useEffect, useRef } from "react";
import { Navigate, useParams } from "react-router-dom";

import { useRoomsStore } from "@/features/rooms";
import { ru } from "@/shared/lib/i18n/ru";
import { FullScreenSpinner } from "@/shared/ui";
import { toast } from "@/shared/ui/useToast";

import type { ReactNode } from "react";

type RequireMembershipProps = {
  children: ReactNode;
};

export function RequireMembership({
  children,
}: RequireMembershipProps): JSX.Element {
  const { roomId } = useParams<{ roomId: string }>();
  const status = useRoomsStore((s) => s.status);
  const isMember = useRoomsStore((s) =>
    s.rooms.some((r) => r.id === roomId),
  );
  const toastShownRef = useRef(false);

  // Toast о «not member» только когда уже загрузились и решили редиректить.
  // useEffect нужен, чтобы избежать побочного эффекта в фазе render
  // (StrictMode иначе повторно покажет тост).
  useEffect(() => {
    if (status === "ready" && !isMember && !toastShownRef.current) {
      toastShownRef.current = true;
      toast.error(ru.rooms.notMember);
    }
  }, [status, isMember]);

  if (status === "idle" || status === "loading") {
    return <FullScreenSpinner />;
  }
  if (!isMember) {
    return <Navigate to="/rooms" replace />;
  }
  return <>{children}</>;
}
