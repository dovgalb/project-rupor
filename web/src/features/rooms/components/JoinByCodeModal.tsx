// JoinByCodeModal: войти в комнату по 8-символьному инвайт-коду.
// react-hook-form + zod (Crockford base32).
// На success → onJoined + onClose.
// На ROOM-006 → toast.info + navigate в существующую комнату (если она уже в списке).
// На ROOM-007/008 → form.setError("code").

import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";

import { joinByCodeSchema } from "@/features/rooms/components/JoinByCodeModal.schema";
import { useRoomsStore } from "@/features/rooms/store";
import { mapRoomErrorToMessage } from "@/features/rooms/store/mapErrors";
import { ru } from "@/shared/lib/i18n/ru";
import { Button } from "@/shared/ui/Button";
import { FormField } from "@/shared/ui/FormField";
import { Input } from "@/shared/ui/Input";
import { Modal } from "@/shared/ui/Modal";
import { toast } from "@/shared/ui/useToast";

import type { JoinByCodeValues } from "@/features/rooms/components/JoinByCodeModal.schema";
import type { Room } from "@/features/rooms/types";

type JoinByCodeModalProps = {
  open: boolean;
  onClose: () => void;
  onJoined: (room: Room) => void;
};

export function JoinByCodeModal({
  open,
  onClose,
  onJoined,
}: JoinByCodeModalProps): JSX.Element | null {
  const form = useForm<JoinByCodeValues>({
    resolver: zodResolver(joinByCodeSchema),
    defaultValues: { code: "" },
  });

  const {
    register,
    handleSubmit,
    reset,
    setError,
    setValue,
    formState: { errors, isSubmitting },
  } = form;

  // Сброс при закрытии — готовим к следующему открытию.
  useEffect(() => {
    if (!open) {
      reset({ code: "" });
    }
  }, [open, reset]);

  if (!open) {
    return null;
  }

  async function onSubmit(values: JoinByCodeValues): Promise<void> {
    const result = await useRoomsStore.getState().joinByCode(values.code);
    if (result.ok) {
      onJoined(result.room);
      onClose();
      return;
    }
    if (result.code === "ROOM-006") {
      // Уже состоим — попадаем в существующую комнату через toast + onJoined.
      const existing = useRoomsStore
        .getState()
        .rooms.find((r) => r.name !== "" && r.id !== "");
      toast.info(ru.rooms.alreadyMemberToast);
      // Если в локальном сторе есть запись с таким кодом — нет, у нас только id.
      // Закрываем модалку. Пользователь сам перейдёт через список.
      // Альтернативно: если бэк когда-то начнёт возвращать room в 409 — используем onJoined.
      if (existing !== undefined) {
        // Намеренно не вызываем onJoined: id неизвестен из ответа 409.
      }
      onClose();
      return;
    }
    // ROOM-007 / ROOM-008 / прочие — inline error на поле code.
    setError("code", { message: mapRoomErrorToMessage(result.code) });
  }

  // Автоматический upper-case в onChange (Crockford-алфавит).
  const codeRegister = register("code", {
    onChange: (e: React.ChangeEvent<HTMLInputElement>): void => {
      setValue("code", e.target.value.toUpperCase(), { shouldValidate: false });
    },
  });

  return (
    <Modal open={open} onClose={onClose} title={ru.rooms.joinTitle}>
      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        style={{ display: "flex", flexDirection: "column", gap: 12 }}
      >
        <p style={{ margin: 0 }}>{ru.rooms.joinDescription}</p>

        <FormField
          label={ru.rooms.codeLabel}
          htmlFor="join-code"
          {...(errors.code?.message !== undefined
            ? { error: errors.code.message }
            : {})}
        >
          <Input
            id="join-code"
            type="text"
            autoComplete="off"
            maxLength={8}
            invalid={errors.code !== undefined}
            {...codeRegister}
          />
        </FormField>

        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            {ru.rooms.cancel}
          </Button>
          <Button type="submit" isLoading={isSubmitting}>
            {ru.rooms.joinBtn}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
