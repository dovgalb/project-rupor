// Публичный API feature-слайса rooms.
// Внешний код должен импортировать только отсюда.
export { useRoomsStore } from "./store";
export { RoomList } from "./components/RoomList";
export { RoomListItem } from "./components/RoomListItem";
export { MembersList } from "./components/MembersList";
export { MemberListItem } from "./components/MemberListItem";
export { CreateRoomModal } from "./components/CreateRoomModal";
export { DeleteRoomConfirm } from "./components/DeleteRoomConfirm";
export { InviteCodeModal } from "./components/InviteCodeModal";
export { JoinByCodeModal } from "./components/JoinByCodeModal";
export type {
  Room,
  RoomWithRole,
  Member,
  InviteCode,
  Role,
  RoomId,
} from "./types";
