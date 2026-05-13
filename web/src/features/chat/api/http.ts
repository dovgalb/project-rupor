// HTTP-обёртки домена chat.
// Префикс /api/v1 подставляется внутри apiFetch.
// Источник: docs/3_5_frontend/06-api-integration.md (REST: Chat history).

import { apiFetch } from "@/shared/api/fetch";

import type {
  ListMessagesQuery,
  ListMessagesResponse,
} from "@/features/chat/types";
import type { ApiResult } from "@/shared/api/errors";

// GET /channels/{channelId}/messages?before=...&limit=...
// before — UUID курсор; limit — 1..100 (default 50 на бэке).
export function listMessages(
  channelId: string,
  query?: ListMessagesQuery,
): Promise<ApiResult<ListMessagesResponse>> {
  const params = new URLSearchParams();
  if (query?.before !== undefined) {
    params.set("before", query.before);
  }
  if (query?.limit !== undefined) {
    params.set("limit", String(query.limit));
  }
  const qs = params.toString();
  const path = `/channels/${encodeURIComponent(channelId)}/messages${
    qs.length > 0 ? `?${qs}` : ""
  }`;
  return apiFetch<ListMessagesResponse>(path, { method: "GET" });
}
