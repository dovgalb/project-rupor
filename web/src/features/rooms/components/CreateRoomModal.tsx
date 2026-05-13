// CreateRoomModal: модалка создания комнаты.
// react-hook-form + zod. Server-error → form.setError или toast (через store).

import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";

import { createRoomSchema } from "@/features/rooms/components/CreateRoomModal.schema";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";
import { Button } from "@/shared/ui/Button";
import { FormField } from "@/shared/ui/FormField";
import { Input } from "@/shared/ui/Input";
import { Modal } from "@/shared/ui/Modal";

import type { CreateRoomValues } from "@/features/rooms/components/CreateRoomModal.schema";
import type { RoomWithRole } from "@/features/rooms/types";

type CreateRoomModalProps = {
  open: boolean;
  onClose: () => void;
  onCreated: (room: RoomWithRole) => void;
};

export function CreateRoomModal({
  open,
  onClose,
  onCreated,
}: CreateRoomModalProps): JSX.Element | null {
  const form = useForm<CreateRoomValues>({
    resolver: zodResolver(createRoomSchema),
    defaultValues: { name: "" },
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = form;

  // При закрытии модалки очищаем форму (готовим к следующему открытию).
  useEffect(() => {
    if (!open) {
      reset({ name: "" });
    }
  }, [open, reset]);

  if (!open) {
    return null;
  }

  async function onSubmit(values: CreateRoomValues): Promise<void> {
    const room = await useRoomsStore.getState().createRoom(values.name);
    if (room) {
      onCreated(room);
      onClose();
    }
    // На fail — toast уже показан в store. Форма остаётся открытой.
  }

  return (
    <Modal open={open} onClose={onClose} title={ru.rooms.createTitle}>
      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        style={{ display: "flex", flexDirection: "column", gap: 12 }}
      >
        <FormField
          label={ru.rooms.nameLabel}
          htmlFor="create-room-name"
          {...(errors.name?.message !== undefined
            ? { error: errors.name.message }
            : {})}
        >
          <Input
            id="create-room-name"
            type="text"
            autoComplete="off"
            invalid={errors.name !== undefined}
            {...register("name")}
          />
        </FormField>

        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            {ru.rooms.cancel}
          </Button>
          <Button type="submit" isLoading={isSubmitting}>
            {ru.rooms.create}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
