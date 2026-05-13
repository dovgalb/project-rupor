// Zod-схема валидации MessageComposer.
// 1..4000 рун после trim — соответствует контракту бэка (см. CHAT-001).

import { z } from "zod";

import { ru } from "@/shared/lib/i18n/ru";

export const messageSchema = z.object({
  text: z.string().trim().min(1, ru.chat.errors.invalidText).max(4000, ru.chat.errors.invalidText),
});

export type MessageValues = z.infer<typeof messageSchema>;
