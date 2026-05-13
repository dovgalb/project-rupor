// Zod-схема валидации LoginForm.
// Тексты ошибок — из i18n/ru.ts (D-23).

import { z } from "zod";

import { ru } from "@/shared/lib/i18n/ru";

export const loginSchema = z.object({
  email: z.string().email(ru.auth.invalidEmail),
  password: z
    .string()
    .min(8, ru.auth.passwordMin)
    .max(72, ru.auth.passwordMax),
});

export type LoginValues = z.infer<typeof loginSchema>;
