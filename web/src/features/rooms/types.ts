// DTO и ViewModel для домена rooms.
// Источник: docs/3_5_frontend/06-api-integration.md (REST: Rooms).
// Без runtime-кода: только TS-типы.

export type Role = "owner" | "admin" | "member";
export type RoomId = string;

// === REST DTO (camelCase, из бэка) ===

export type RoomDto = {
  id: string;
  ownerId: string;
  name: string;
  createdAt: string;
};

export type RoomWithRoleDto = RoomDto & { role: Role };

export type MemberDto = {
  userId: string;
  role: Role;
  joinedAt: string;
};

export type InviteCodeDto = {
  code: string;
  createdBy: string;
  createdAt: string;
};

export type CreateRoomRequest = { name: string };

export type ListRoomsResponse = { items: RoomWithRoleDto[] };

export type ListMembersResponse = { items: MemberDto[] };

// === ViewModel = DTO (без переименований, D-11/D-19) ===

export type Room = RoomDto;
export type RoomWithRole = RoomWithRoleDto;
export type Member = MemberDto;
export type InviteCode = InviteCodeDto;
