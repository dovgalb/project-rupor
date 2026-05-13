// CreateChannelModal: модалка создания канала.
// react-hook-form + zod. На voice — после создания toast.info с подсказкой.
// На fail — toast уже показан в store; форма остаётся открытой.

import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";

import { createChannelSchema } from "@/features/channels/components/CreateChannelModal.schema";
import { useChannelsStore } from "@/features/channels/store";
import { ru } from "@/shared/lib/i18n/ru";
import { Button } from "@/shared/ui/Button";
import { FormField } from "@/shared/ui/FormField";
import { Input } from "@/shared/ui/Input";
import { Modal } from "@/shared/ui/Modal";
import { toast } from "@/shared/ui/useToast";

import type { CreateChannelValues } from "@/features/channels/components/CreateChannelModal.schema";
import type { Channel } from "@/features/channels/types";

type CreateChannelModalProps = {
  open: boolean;
  roomId: string;
  onClose: () => void;
  onCreated: (channel: Channel) => void;
};

const DEFAULT_VALUES: CreateChannelValues = { name: "", kind: "text" };

export function CreateChannelModal({
  open,
  roomId,
  onClose,
  onCreated,
}: CreateChannelModalProps): JSX.Element | null {
  const form = useForm<CreateChannelValues>({
    resolver: zodResolver(createChannelSchema),
    defaultValues: DEFAULT_VALUES,
  });

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = form;

  // При закрытии модалки сбрасываем форму к дефолтам.
  useEffect(() => {
    if (!open) {
      reset(DEFAULT_VALUES);
    }
  }, [open, reset]);

  if (!open) {
    return null;
  }

  async function onSubmit(values: CreateChannelValues): Promise<void> {
    const channel = await useChannelsStore
      .getState()
      .createChannel(roomId, values);
    if (channel === null) {
      // Toast уже показан в store. Форма остаётся открытой.
      return;
    }
    if (channel.kind === "voice") {
      toast.info(ru.channels.voiceCreatedHint);
    }
    onCreated(channel);
    onClose();
  }

  return (
    <Modal open={open} onClose={onClose} title={ru.channels.createModalTitle}>
      <form
        onSubmit={handleSubmit(onSubmit)}
        noValidate
        style={{ display: "flex", flexDirection: "column", gap: 12 }}
      >
        <FormField
          label={ru.channels.nameLabel}
          htmlFor="create-channel-name"
          {...(errors.name?.message !== undefined
            ? { error: errors.name.message }
            : {})}
        >
          <Input
            id="create-channel-name"
            type="text"
            autoComplete="off"
            invalid={errors.name !== undefined}
            {...register("name")}
          />
        </FormField>

        <fieldset
          style={{
            border: "none",
            padding: 0,
            margin: 0,
            display: "flex",
            flexDirection: "column",
            gap: 6,
          }}
        >
          <legend
            style={{ fontSize: 12, color: "var(--color-text-muted)" }}
          >
            {ru.channels.kindLabel}
          </legend>
          <label
            style={{ display: "flex", gap: 6, alignItems: "center" }}
          >
            <input type="radio" value="text" {...register("kind")} />
            {ru.channels.kindText}
          </label>
          <label
            style={{ display: "flex", gap: 6, alignItems: "center" }}
          >
            <input type="radio" value="voice" {...register("kind")} />
            {ru.channels.kindVoice}
          </label>
          {errors.kind?.message !== undefined ? (
            <span
              role="alert"
              style={{ color: "var(--color-danger)", fontSize: 12 }}
            >
              {errors.kind.message}
            </span>
          ) : null}
        </fieldset>

        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            {ru.channels.cancel}
          </Button>
          <Button type="submit" isLoading={isSubmitting}>
            {ru.channels.create}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
