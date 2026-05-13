// Zod-схема валидации JoinByCodeModal.
// Crockford base32 без I, L, O, U; ровно 8 uppercase-символов.

import { z } from "zod";

import { ru } from "@/shared/lib/i18n/ru";

export const joinByCodeSchema = z.object({
  code: z
    .string()
    .trim()
    .regex(/^[0-9A-HJKMNP-TV-Z]{8}$/, ru.rooms.codeFormat),
});

export type JoinByCodeValues = z.infer<typeof joinByCodeSchema>;
