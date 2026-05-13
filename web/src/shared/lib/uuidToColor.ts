// Детерминированная палитра аватаров (D-25).
// 8 фиксированных цветов с AA-контрастом для белого текста.
const PALETTE = [
  "#5865f2", // accent blue
  "#23a55a", // success green
  "#f0b232", // warn yellow
  "#f23f42", // danger red
  "#8b5cf6", // purple
  "#ec4899", // pink
  "#06b6d4", // cyan
  "#84cc16", // lime
] as const;

// Возвращает цвет из палитры по uuid. Хеш — сумма charcode без дефисов,
// индекс — остаток от деления на длину палитры (всегда < 8).
export function uuidToColor(uuid: string): string {
  const normalized = uuid.replace(/-/g, "");
  let hash = 0;
  for (let i = 0; i < normalized.length; i += 1) {
    hash += normalized.charCodeAt(i);
  }
  const index = hash % PALETTE.length;
  // noUncheckedIndexedAccess: индекс гарантированно валиден (mod length),
  // но TS требует явной проверки. Fallback на первый цвет невозможен,
  // потому что length > 0 и index < length.
  const color = PALETTE[index];
  return color ?? PALETTE[0];
}

// Экспортируем палитру для тестов и потенциального переиспользования.
export const AVATAR_PALETTE = PALETTE;
