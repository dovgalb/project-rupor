// Единая fetch-обёртка с auto-refresh для 401 AUTH-011.
// Источник: docs/3_5_frontend/06-api-integration.md, UC-2/UC-3.

import { refreshAccessOnce } from "@/shared/api/refresh";
import { tokenStorage } from "@/shared/api/token-storage";
import { logoutFlow } from "@/shared/lib/logoutFlow";

import type { ApiResult, ErrorEnvelope } from "@/shared/api/errors";

const API_PREFIX = "/api/v1";
const REFRESH_PATH = "/auth/refresh";

// Коды, на которых auto-refresh не помогает — сразу logout.
const FATAL_AUTH_CODES = new Set(["AUTH-007", "AUTH-008", "AUTH-009", "AUTH-010"]);

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<ApiResult<T>> {
  const isRefreshCall = path === REFRESH_PATH;
  const result = await doFetch<T>(path, init);

  if (result.ok) {
    return result;
  }

  // Не-401 — возвращаем как есть.
  if (result.status !== 401) {
    return result;
  }

  // 401 на самом /auth/refresh — рекурсивный refresh запрещён (loop protection).
  if (isRefreshCall) {
    if (FATAL_AUTH_CODES.has(result.error.code)) {
      logoutFlow();
    }
    return result;
  }

  // 401 AUTH-011 — пытаемся освежить access и повторить запрос.
  if (result.error.code === "AUTH-011") {
    const refreshed = await refreshAccessOnce();
    if (!refreshed.ok) {
      logoutFlow();
      return result;
    }
    const retry = await doFetch<T>(path, init);
    if (!retry.ok && retry.status === 401 && retry.error.code === "AUTH-011") {
      // Повторный AUTH-011 — что-то очень странно, выходим из цикла.
      logoutFlow();
      return retry;
    }
    return retry;
  }

  // 401 AUTH-007/008/009/010 — refresh не поможет, logout.
  if (FATAL_AUTH_CODES.has(result.error.code)) {
    logoutFlow();
    return result;
  }

  // Остальные 401-коды (например, AUTH-012) — возвращаем без спец-обработки.
  return result;
}

// === Внутренние помощники ===

async function doFetch<T>(path: string, init: RequestInit): Promise<ApiResult<T>> {
  const url = buildUrl(path);
  const headers = buildHeaders(init);

  let res: Response;
  try {
    res = await fetch(url, { ...init, headers });
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
    // Пустое тело (например, 204 No Content) → data=null.
    return { ok: true, data: (parsed ?? null) as T };
  }

  const envelope = parsed as Partial<ErrorEnvelope> | null;
  const error = envelope?.error ?? { code: "CLIENT", message: "unknown" };
  return { ok: false, status: res.status, error };
}

// Префикс /api/v1 подставляется только для относительных API-путей.
// Внешние URL и пути, уже начинающиеся с /api, остаются как есть.
function buildUrl(path: string): string {
  if (path.startsWith("/") && !path.startsWith("/api")) {
    return `${API_PREFIX}${path}`;
  }
  return path;
}

function buildHeaders(init: RequestInit): Headers {
  const headers = new Headers(init.headers);

  const access = tokenStorage.getAccess();
  if (access !== null && !headers.has("Authorization")) {
    // Запрет R-09: Authorization не логируем нигде.
    headers.set("Authorization", `Bearer ${access}`);
  }

  if (init.body !== undefined && init.body !== null && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  return headers;
}
