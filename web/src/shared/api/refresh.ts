// Single-flight refresh access-токена.
// Решение D-05: параллельные 401 объединяются в один вызов /auth/refresh.
// Внутри используется СЫРОЙ fetch (не apiFetch) — это защита от рекурсии,
// потому что apiFetch сам зовёт refreshAccessOnce при 401 AUTH-011.

import { tokenStorage } from "@/shared/api/token-storage";

import type { ApiResult, ErrorEnvelope, TokensResponse } from "@/shared/api/errors";

// Модульный single-flight promise. Сбрасывается в .finally() после resolve/reject.
let currentRefresh: Promise<ApiResult<TokensResponse>> | null = null;

const REFRESH_URL = "/api/v1/auth/refresh";

export function refreshAccessOnce(): Promise<ApiResult<TokensResponse>> {
  if (currentRefresh !== null) {
    return currentRefresh;
  }

  const refreshToken = tokenStorage.getRefresh();
  if (refreshToken === null) {
    // Имитируем ответ бэка: refresh отсутствует — для UI это AUTH-007.
    return Promise.resolve<ApiResult<TokensResponse>>({
      ok: false,
      status: 401,
      error: { code: "AUTH-007", message: "no refresh token" },
    });
  }

  currentRefresh = doRefresh(refreshToken)
    .then((result) => {
      if (result.ok) {
        tokenStorage.set({
          access: result.data.accessToken,
          refresh: result.data.refreshToken,
          accessExpiresAt: result.data.accessExpiresAt,
          refreshExpiresAt: result.data.refreshExpiresAt,
        });
      }
      return result;
    })
    .finally(() => {
      currentRefresh = null;
    });

  return currentRefresh;
}

// Сырой POST на /auth/refresh без apiFetch — для loop-protection.
// НЕ логируем тело запроса/ответа (содержит refreshToken).
async function doRefresh(refreshToken: string): Promise<ApiResult<TokensResponse>> {
  let res: Response;
  try {
    res = await fetch(REFRESH_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
    });
  } catch {
    return {
      ok: false,
      status: 0,
      error: { code: "NETWORK", message: "network error" },
    };
  }

  const text = await res.text();
  let parsed: unknown = null;
  if (text.length > 0) {
    try {
      parsed = JSON.parse(text);
    } catch {
      return {
        ok: false,
        status: res.status,
        error: { code: "CLIENT", message: "invalid json" },
      };
    }
  }

  if (res.ok) {
    return { ok: true, data: parsed as TokensResponse };
  }

  const envelope = parsed as Partial<ErrorEnvelope> | null;
  const error = envelope?.error ?? { code: "CLIENT", message: "unknown" };
  return { ok: false, status: res.status, error };
}

// Только для тестов: обнуляет single-flight promise между прогонами.
export function _resetRefreshForTests(): void {
  currentRefresh = null;
}
