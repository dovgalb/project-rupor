// InviteCodeModal: показать инвайт-код комнаты + кнопка «Сгенерировать новый».
// На open НЕ генерируем автоматически — только по click.
// Copy через navigator.clipboard.writeText с fallback на document.execCommand.

import { useState } from "react";

import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";
import { Button } from "@/shared/ui/Button";
import { Modal } from "@/shared/ui/Modal";
import { toast } from "@/shared/ui/useToast";

import styles from "./InviteCodeModal.module.css";

type InviteCodeModalProps = {
  open: boolean;
  roomId: string;
  onClose: () => void;
};

// Копируем строку в clipboard: сначала пробуем современный API, потом legacy.
// Возвращаем true, если что-то получилось.
async function copyToClipboard(text: string): Promise<boolean> {
  if (
    typeof navigator !== "undefined" &&
    navigator.clipboard !== undefined &&
    typeof navigator.clipboard.writeText === "function"
  ) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Падаем в fallback ниже.
    }
  }
  if (typeof document !== "undefined") {
    try {
      const textarea = document.createElement("textarea");
      textarea.value = text;
      textarea.style.position = "fixed";
      textarea.style.left = "-9999px";
      document.body.appendChild(textarea);
      textarea.select();
      const ok = document.execCommand("copy");
      document.body.removeChild(textarea);
      return ok;
    } catch {
      return false;
    }
  }
  return false;
}

export function InviteCodeModal({
  open,
  roomId,
  onClose,
}: InviteCodeModalProps): JSX.Element | null {
  const invite = useRoomsStore((s) => s.activeInviteByRoom[roomId]);
  const [isGenerating, setIsGenerating] = useState(false);

  if (!open) {
    return null;
  }

  async function handleGenerate(): Promise<void> {
    setIsGenerating(true);
    try {
      await useRoomsStore.getState().regenerateInvite(roomId);
    } finally {
      setIsGenerating(false);
    }
  }

  async function handleCopy(): Promise<void> {
    if (invite === undefined) {
      return;
    }
    const ok = await copyToClipboard(invite.code);
    if (ok) {
      toast.success(ru.rooms.inviteCopiedToast);
    } else {
      toast.error(ru.rooms.copyFailed);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={ru.rooms.inviteTitle}>
      <div className={styles.body}>
        <p style={{ margin: 0 }}>{ru.rooms.inviteDescription}</p>

        {invite !== undefined ? (
          <>
            <div className={styles.codeRow}>
              <div className={styles.code} aria-label={ru.rooms.codeLabel}>
                {invite.code}
              </div>
              <Button variant="secondary" onClick={handleCopy}>
                {ru.rooms.copyCode}
              </Button>
            </div>
            <p className={styles.warning}>{ru.rooms.regenerateWarning}</p>
            <div className={styles.actions}>
              <Button onClick={handleGenerate} isLoading={isGenerating}>
                {ru.rooms.generateNew}
              </Button>
            </div>
          </>
        ) : (
          <>
            <div className={styles.placeholder}>{ru.rooms.inviteEmpty}</div>
            <div className={styles.actions}>
              <Button onClick={handleGenerate} isLoading={isGenerating}>
                {ru.rooms.generate}
              </Button>
            </div>
          </>
        )}
      </div>
    </Modal>
  );
}
