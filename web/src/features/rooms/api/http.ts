// HTTP-обёртки для домена rooms.
// Каждая функция — одиночный вызов apiFetch.
// Префикс /api/v1 подставляется внутри apiFetch.
// Источник: docs/3_5_frontend/06-api-integration.md (REST: Rooms).

import { apiFetch } from "@/shared/api/fetch";

import type {
  CreateRoomRequest,
  InviteCodeDto,
  ListMembersResponse,
  ListRoomsResponse,
  RoomDto,
} from "@/features/rooms/types";
import type { ApiResult } from "@/shared/api/errors";

export function listRooms(): Promise<ApiResult<ListRoomsResponse>> {
  return apiFetch<ListRoomsResponse>("/rooms", { method: "GET" });
}

export function createRoom(
  req: CreateRoomRequest,
): Promise<ApiResult<RoomDto>> {
  return apiFetch<RoomDto>("/rooms", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export function getRoom(roomId: string): Promise<ApiResult<RoomDto>> {
  return apiFetch<RoomDto>(`/rooms/${encodeURIComponent(roomId)}`, {
    method: "GET",
  });
}

export function deleteRoom(roomId: string): Promise<ApiResult<null>> {
  return apiFetch<null>(`/rooms/${encodeURIComponent(roomId)}`, {
    method: "DELETE",
  });
}

export function listMembers(
  roomId: string,
): Promise<ApiResult<ListMembersResponse>> {
  return apiFetch<ListMembersResponse>(
    `/rooms/${encodeURIComponent(roomId)}/members`,
    { method: "GET" },
  );
}

export function regenerateInvite(
  roomId: string,
): Promise<ApiResult<InviteCodeDto>> {
  return apiFetch<InviteCodeDto>(
    `/rooms/${encodeURIComponent(roomId)}/invite`,
    { method: "POST" },
  );
}

export function joinByCode(code: string): Promise<ApiResult<RoomDto>> {
  return apiFetch<RoomDto>(`/rooms/join/${encodeURIComponent(code)}`, {
    method: "POST",
  });
}
