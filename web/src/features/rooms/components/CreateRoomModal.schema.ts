// Zod-схема валидации CreateRoomModal.
// Тексты ошибок — из i18n/ru.ts (D-23).

import { z } from "zod";

import { ru } from "@/shared/lib/i18n/ru";

export const createRoomSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, ru.rooms.nameMin)
    .max(64, ru.rooms.nameMax),
});

export type CreateRoomValues = z.infer<typeof createRoomSchema>;
