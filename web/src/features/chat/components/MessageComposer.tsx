// MessageComposer: textarea + кнопка отправки.
// Источник: docs/3_5_frontend/07-ui-contract.md (<MessageComposer />).
//
// Phase 9: composer enabled при wsStatus === "open", onSubmit → useChatStore.sendMessage.

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { messageSchema } from "@/features/chat/components/MessageComposer.schema";
import { useChatStore } from "@/features/chat/store";
import { Button } from "@/shared/ui/Button";
import { Textarea } from "@/shared/ui/Textarea";
import { ru } from "@/shared/lib/i18n/ru";

import styles from "./MessageComposer.module.css";

import type { MessageValues } from "@/features/chat/components/MessageComposer.schema";
import type { KeyboardEvent } from "react";

type MessageComposerProps = {
  channelId: string;
};

// Счётчик символов появляется ближе к лимиту.
const COUNTER_THRESHOLD = 3000;
const MAX_LENGTH = 4000;

export function MessageComposer({
  channelId,
}: MessageComposerProps): JSX.Element {
  const wsStatus = useChatStore((s) => s.wsStatus);
  const disabled = wsStatus !== "open";

  const form = useForm<MessageValues>({
    resolver: zodResolver(messageSchema),
    defaultValues: { text: "" },
  });

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors, isSubmitting },
  } = form;

  const textValue = watch("text");
  const length = textValue.length;
  const showCounter = length > COUNTER_THRESHOLD;

  async function onSubmit(values: MessageValues): Promise<void> {
    // Дополнительный guard: если за время заполнения связь упала — отправлять не пытаемся.
    if (useChatStore.getState().wsStatus !== "open") {
      return;
    }
    await useChatStore.getState().sendMessage(channelId, values.text);
    reset({ text: "" });
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>): void {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void handleSubmit(onSubmit)();
    }
  }

  const placeholder = disabled
    ? ru.chat.composerDisabledHint
    : ru.chat.composerPlaceholder;

  return (
    <form
      className={styles.composer}
      onSubmit={handleSubmit(onSubmit)}
      noValidate
    >
      <Textarea
        rows={1}
        placeholder={placeholder}
        aria-label={ru.chat.messageLabel}
        disabled={disabled}
        maxLength={MAX_LENGTH}
        invalid={errors.text !== undefined}
        onKeyDown={handleKeyDown}
        {...register("text")}
      />
      <div className={styles.bottomRow}>
        {showCounter ? (
          <span className={styles.counter} aria-live="polite">
            {length}/{MAX_LENGTH}
          </span>
        ) : (
          <span />
        )}
        <Button
          type="submit"
          size="sm"
          aria-label={ru.chat.sendBtn}
          disabled={disabled}
          isLoading={isSubmitting}
        >
          {ru.chat.sendBtn}
        </Button>
      </div>
    </form>
  );
}
