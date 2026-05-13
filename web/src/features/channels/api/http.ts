// HTTP-обёртки для домена channels.
// Каждая функция — одиночный вызов apiFetch.
// Префикс /api/v1 подставляется внутри apiFetch.
// Источник: docs/3_5_frontend/06-api-integration.md (REST: Channels).

import { apiFetch } from "@/shared/api/fetch";

import type {
  ChannelDto,
  CreateChannelRequest,
  ListChannelsResponse,
} from "@/features/channels/types";
import type { ApiResult } from "@/shared/api/errors";

export function listChannels(
  roomId: string,
): Promise<ApiResult<ListChannelsResponse>> {
  return apiFetch<ListChannelsResponse>(
    `/rooms/${encodeURIComponent(roomId)}/channels`,
    { method: "GET" },
  );
}

export function createChannel(
  roomId: string,
  req: CreateChannelRequest,
): Promise<ApiResult<ChannelDto>> {
  return apiFetch<ChannelDto>(
    `/rooms/${encodeURIComponent(roomId)}/channels`,
    {
      method: "POST",
      body: JSON.stringify(req),
    },
  );
}

export function deleteChannel(
  roomId: string,
  channelId: string,
): Promise<ApiResult<null>> {
  return apiFetch<null>(
    `/rooms/${encodeURIComponent(roomId)}/channels/${encodeURIComponent(channelId)}`,
    { method: "DELETE" },
  );
}
