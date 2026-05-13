// RoomPage: страница /rooms/:roomId без выбранного канала.
// Phase 6 — header с действиями (Delete для owner; Invite для admin/owner).
// Phase 7 расширит (выбор канала из sidebar).

import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  DeleteRoomConfirm,
  InviteCodeModal,
  useRoomsStore,
} from "@/features/rooms";
import { ru } from "@/shared/lib/i18n/ru";
import { Button, EmptyState } from "@/shared/ui";

export function RoomPage(): JSX.Element | null {
  const { roomId } = useParams<{ roomId: string }>();
  const navigate = useNavigate();
  const room = useRoomsStore((s) =>
    s.rooms.find((r) => r.id === roomId),
  );
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [inviteOpen, setInviteOpen] = useState(false);

  if (room === undefined || roomId === undefined) {
    return null;
  }

  const canManage = room.role === "owner" || room.role === "admin";
  const canDelete = room.role === "owner";

  return (
    <div style={{ padding: 24 }}>
      <header
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 24,
        }}
      >
        <h1 style={{ margin: 0 }}>{room.name}</h1>
        <div style={{ display: "flex", gap: 8 }}>
          {canManage ? (
            <Button variant="secondary" onClick={() => setInviteOpen(true)}>
              {ru.rooms.invite}
            </Button>
          ) : null}
          {canDelete ? (
            <Button variant="danger" onClick={() => setConfirmDelete(true)}>
              {ru.rooms.delete}
            </Button>
          ) : null}
        </div>
      </header>

      <EmptyState title={ru.rooms.selectChannel} />

      <DeleteRoomConfirm
        open={confirmDelete}
        roomId={roomId}
        roomName={room.name}
        onCancel={() => setConfirmDelete(false)}
        onDeleted={() => {
          setConfirmDelete(false);
          navigate("/rooms");
        }}
      />
      <InviteCodeModal
        open={inviteOpen}
        roomId={roomId}
        onClose={() => setInviteOpen(false)}
      />
    </div>
  );
}
