// Zod-схема валидации RegisterForm.
// Username: ASCII alnum/_/- 3..32 (см. 06-api-integration.md).

import { z } from "zod";

import { ru } from "@/shared/lib/i18n/ru";

export const registerSchema = z.object({
  email: z.string().email(ru.auth.invalidEmail),
  username: z
    .string()
    .min(3, ru.auth.usernameMin)
    .max(32, ru.auth.usernameMax)
    .regex(/^[A-Za-z0-9_-]+$/, ru.auth.usernameFormat),
  password: z
    .string()
    .min(8, ru.auth.passwordMin)
    .max(72, ru.auth.passwordMax),
});

export type RegisterValues = z.infer<typeof registerSchema>;
