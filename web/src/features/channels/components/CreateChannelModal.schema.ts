// Zod-схема валидации CreateChannelModal.
// Тексты ошибок — из i18n/ru.ts (D-23).

import { z } from "zod";

import { ru } from "@/shared/lib/i18n/ru";

export const createChannelSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, ru.channels.errors.nameRequired)
    .max(64, ru.channels.errors.nameTooLong),
  kind: z.enum(["text", "voice"], {
    required_error: ru.channels.errors.kindRequired,
  }),
});

export type CreateChannelValues = z.infer<typeof createChannelSchema>;
