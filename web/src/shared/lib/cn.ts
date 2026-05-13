// Сборка className из набора возможно-undefined строк.
// Нужен для безопасной работы с CSS-модулями под noUncheckedIndexedAccess
// (vite типизирует CSSModuleClasses через index-signature, что даёт string|undefined).
export function cn(...classes: Array<string | false | null | undefined>): string {
  return classes.filter((c): c is string => Boolean(c)).join(" ");
}
