// Обёртка над localStorage для access/refresh токенов и их сроков жизни.
// Решение D-04: localStorage. Запрет R-09: НЕ логировать значения.

const KEY_ACCESS = "rupor.access";
const KEY_REFRESH = "rupor.refresh";
const KEY_ACCESS_EXP = "rupor.accessExpiresAt";
const KEY_REFRESH_EXP = "rupor.refreshExpiresAt";

export type StoredTokens = {
  access: string;
  refresh: string;
  accessExpiresAt: string;
  refreshExpiresAt: string;
};

export const tokenStorage = {
  getAccess(): string | null {
    return localStorage.getItem(KEY_ACCESS);
  },
  getRefresh(): string | null {
    return localStorage.getItem(KEY_REFRESH);
  },
  getAccessExpiresAt(): string | null {
    return localStorage.getItem(KEY_ACCESS_EXP);
  },
  getRefreshExpiresAt(): string | null {
    return localStorage.getItem(KEY_REFRESH_EXP);
  },
  set(tokens: StoredTokens): void {
    localStorage.setItem(KEY_ACCESS, tokens.access);
    localStorage.setItem(KEY_REFRESH, tokens.refresh);
    localStorage.setItem(KEY_ACCESS_EXP, tokens.accessExpiresAt);
    localStorage.setItem(KEY_REFRESH_EXP, tokens.refreshExpiresAt);
  },
  clear(): void {
    localStorage.removeItem(KEY_ACCESS);
    localStorage.removeItem(KEY_REFRESH);
    localStorage.removeItem(KEY_ACCESS_EXP);
    localStorage.removeItem(KEY_REFRESH_EXP);
  },
};
