// Barrel-реэкспорт утилит shared/lib.
// НЕ включаем logoutFlow (это не утилита — отдельная точка композиции).
export { uuidToColor, AVATAR_PALETTE } from "./uuidToColor";
export { linkify } from "./linkify";
export type { LinkifyPart } from "./linkify";
export { formatTime, formatDateTime, formatRelative } from "./formatDate";
export { logger } from "./logger";
export { cn } from "./cn";
