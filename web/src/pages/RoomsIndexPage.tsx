// RoomsIndexPage: стартовая страница /rooms — список комнат.
// Phase 6:
// - грузит rooms при mount (если idle).
// - loading → FullScreenSpinner;
// - rooms.length > 0 → redirect на первую комнату;
// - rooms.length == 0 → EmptyState.

import { useEffect } from "react";
import { Navigate } from "react-router-dom";

import { useRoomsStore } from "@/features/rooms";
import { ru } from "@/shared/lib/i18n/ru";
import { EmptyState, FullScreenSpinner } from "@/shared/ui";

export function RoomsIndexPage(): JSX.Element {
  const rooms = useRoomsStore((s) => s.rooms);
  const status = useRoomsStore((s) => s.status);

  useEffect(() => {
    if (status === "idle") {
      void useRoomsStore.getState().loadRooms();
    }
  }, [status]);

  if (status === "idle" || status === "loading") {
    return <FullScreenSpinner />;
  }

  if (rooms.length > 0) {
    const first = rooms[0];
    if (first !== undefined) {
      return <Navigate to={`/rooms/${first.id}`} replace />;
    }
  }

  return (
    <div style={{ padding: 24 }}>
      <EmptyState
        title={ru.rooms.emptyTitle}
        description={ru.rooms.emptyDescription}
      />
    </div>
  );
}
