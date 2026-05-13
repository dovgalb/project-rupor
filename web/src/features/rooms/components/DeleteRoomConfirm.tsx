// DeleteRoomConfirm: модалка подтверждения удаления комнаты.
// onClick «Удалить» → useRoomsStore.deleteRoom → on success onDeleted().
// На fail — toast уже показан в store; модалка остаётся открытой.

import { useState } from "react";

import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";
import { Button } from "@/shared/ui/Button";
import { Modal } from "@/shared/ui/Modal";

type DeleteRoomConfirmProps = {
  open: boolean;
  roomId: string;
  roomName: string;
  onCancel: () => void;
  onDeleted: () => void;
};

export function DeleteRoomConfirm({
  open,
  roomId,
  roomName,
  onCancel,
  onDeleted,
}: DeleteRoomConfirmProps): JSX.Element | null {
  const [isDeleting, setIsDeleting] = useState(false);

  if (!open) {
    return null;
  }

  async function handleDelete(): Promise<void> {
    setIsDeleting(true);
    try {
      const ok = await useRoomsStore.getState().deleteRoom(roomId);
      if (ok) {
        onDeleted();
      }
    } finally {
      setIsDeleting(false);
    }
  }

  return (
    <Modal open={open} onClose={onCancel} title={ru.rooms.deleteTitle}>
      <p style={{ margin: 0 }}>{ru.rooms.deleteWarning}</p>
      <div
        style={{
          display: "flex",
          justifyContent: "flex-end",
          gap: 8,
          marginTop: 16,
        }}
      >
        <Button variant="secondary" onClick={onCancel} disabled={isDeleting}>
          {ru.rooms.cancel}
        </Button>
        <Button
          variant="danger"
          onClick={handleDelete}
          isLoading={isDeleting}
          aria-label={`${ru.rooms.delete} ${roomName}`}
        >
          {ru.rooms.delete}
        </Button>
      </div>
    </Modal>
  );
}
