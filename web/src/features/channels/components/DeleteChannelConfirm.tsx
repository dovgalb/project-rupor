// DeleteChannelConfirm: модалка подтверждения удаления канала.
// onClick «Удалить» → useChannelsStore.deleteChannel → on success onDeleted().
// На fail — toast уже показан в store; модалка остаётся открытой.

import { useState } from "react";

import { useChannelsStore } from "@/features/channels/store";
import { ru } from "@/shared/lib/i18n/ru";
import { Button } from "@/shared/ui/Button";
import { Modal } from "@/shared/ui/Modal";

type DeleteChannelConfirmProps = {
  open: boolean;
  roomId: string;
  channelId: string;
  channelName: string;
  onCancel: () => void;
  onDeleted: () => void;
};

export function DeleteChannelConfirm({
  open,
  roomId,
  channelId,
  channelName,
  onCancel,
  onDeleted,
}: DeleteChannelConfirmProps): JSX.Element | null {
  const [isDeleting, setIsDeleting] = useState(false);

  if (!open) {
    return null;
  }

  async function handleDelete(): Promise<void> {
    setIsDeleting(true);
    try {
      const ok = await useChannelsStore
        .getState()
        .deleteChannel(roomId, channelId);
      if (ok) {
        onDeleted();
      }
    } finally {
      setIsDeleting(false);
    }
  }

  return (
    <Modal open={open} onClose={onCancel} title={ru.channels.deleteTitle}>
      <p style={{ margin: 0 }}>{ru.channels.deleteWarning}</p>
      <div
        style={{
          display: "flex",
          justifyContent: "flex-end",
          gap: 8,
          marginTop: 16,
        }}
      >
        <Button variant="secondary" onClick={onCancel} disabled={isDeleting}>
          {ru.channels.cancel}
        </Button>
        <Button
          variant="danger"
          onClick={handleDelete}
          isLoading={isDeleting}
          aria-label={`${ru.channels.delete} ${channelName}`}
        >
          {ru.channels.delete}
        </Button>
      </div>
    </Modal>
  );
}
