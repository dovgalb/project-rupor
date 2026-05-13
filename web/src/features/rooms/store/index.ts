// useRoomsStore: Zustand-store домена rooms.
// Источник: docs/3_5_frontend/05-state-model.md (useRoomsStore).
// Решения: D-18 (один store на фичу), D-19 (без persist),
// D-10 (WS reconnect после joinByCode для auto-subscribe на новый room).

import { create } from "zustand";

import * as roomsApi from "@/features/rooms/api/http";
import { mapRoomErrorToMessage } from "@/features/rooms/store/mapErrors";
import { wsClient } from "@/shared/api/wsClient.singleton";
import { registerLogoutListener } from "@/shared/lib/logoutFlow";
import { toast } from "@/shared/ui/useToast";

import type { MemberJoinedFrame } from "@/shared/api/errors";
import type {
  InviteCode,
  Member,
  Room,
  RoomId,
  RoomWithRole,
} from "@/features/rooms/types";

export type RoomsStatus = "idle" | "loading" | "ready" | "error";

type RoomsState = {
  rooms: RoomWithRole[];
  status: RoomsStatus;
  error: string | null;
  membersByRoom: Record<string, Member[]>;
  membersLoadingByRoom: Record<string, boolean>;
  activeInviteByRoom: Record<string, InviteCode>;
};

// Результат joinByCode: форма JoinByCodeModal должна различать ROOM-006/007/008
// и обрабатывать их inline / toast-навигацией.
export type JoinByCodeResult =
  | { ok: true; room: Room }
  | { ok: false; code: string };

type RoomsActions = {
  loadRooms: () => Promise<void>;
  createRoom: (name: string) => Promise<RoomWithRole | null>;
  deleteRoom: (roomId: RoomId) => Promise<boolean>;
  loadMembers: (roomId: RoomId) => Promise<void>;
  regenerateInvite: (roomId: RoomId) => Promise<InviteCode | null>;
  joinByCode: (code: string) => Promise<JoinByCodeResult>;
  appendMember: (data: MemberJoinedFrame["data"]) => void;
  clear: () => void;
};

const initialState: RoomsState = {
  rooms: [],
  status: "idle",
  error: null,
  membersByRoom: {},
  membersLoadingByRoom: {},
  activeInviteByRoom: {},
};

export const useRoomsStore = create<RoomsState & RoomsActions>((set) => ({
  ...initialState,

  async loadRooms() {
    set({ status: "loading", error: null });
    const r = await roomsApi.listRooms();
    if (r.ok) {
      set({ rooms: r.data.items, status: "ready", error: null });
      return;
    }
    set({
      status: "error",
      error: mapRoomErrorToMessage(r.error.code),
    });
  },

  async createRoom(name) {
    const r = await roomsApi.createRoom({ name });
    if (!r.ok) {
      toast.error(mapRoomErrorToMessage(r.error.code));
      return null;
    }
    // Создатель сразу получает роль owner (фиксируется доменом бэка).
    const newRoom: RoomWithRole = { ...r.data, role: "owner" };
    set((s) => ({ rooms: [...s.rooms, newRoom] }));
    return newRoom;
  },

  async deleteRoom(roomId) {
    const r = await roomsApi.deleteRoom(roomId);
    if (!r.ok) {
      toast.error(mapRoomErrorToMessage(r.error.code));
      return false;
    }
    set((s) => {
      // Каскадная очистка локальных кэшей по этой комнате.
      const nextMembers = { ...s.membersByRoom };
      delete nextMembers[roomId];
      const nextMembersLoading = { ...s.membersLoadingByRoom };
      delete nextMembersLoading[roomId];
      const nextInvites = { ...s.activeInviteByRoom };
      delete nextInvites[roomId];
      return {
        rooms: s.rooms.filter((rm) => rm.id !== roomId),
        membersByRoom: nextMembers,
        membersLoadingByRoom: nextMembersLoading,
        activeInviteByRoom: nextInvites,
      };
    });
    return true;
  },

  async loadMembers(roomId) {
    set((s) => ({
      membersLoadingByRoom: { ...s.membersLoadingByRoom, [roomId]: true },
    }));
    const r = await roomsApi.listMembers(roomId);
    if (r.ok) {
      set((s) => ({
        membersByRoom: { ...s.membersByRoom, [roomId]: r.data.items },
        membersLoadingByRoom: {
          ...s.membersLoadingByRoom,
          [roomId]: false,
        },
      }));
      return;
    }
    set((s) => ({
      membersLoadingByRoom: { ...s.membersLoadingByRoom, [roomId]: false },
    }));
    toast.error(mapRoomErrorToMessage(r.error.code));
  },

  async regenerateInvite(roomId) {
    const r = await roomsApi.regenerateInvite(roomId);
    if (!r.ok) {
      toast.error(mapRoomErrorToMessage(r.error.code));
      return null;
    }
    set((s) => ({
      activeInviteByRoom: { ...s.activeInviteByRoom, [roomId]: r.data },
    }));
    return r.data;
  },

  async joinByCode(code) {
    const r = await roomsApi.joinByCode(code);
    if (!r.ok) {
      // Ошибку обрабатывает форма JoinByCodeModal через код.
      return { ok: false, code: r.error.code };
    }
    const newRoom: RoomWithRole = { ...r.data, role: "member" };
    set((s) => {
      // Дедуп: если такая комната уже есть в списке — не добавляем.
      const exists = s.rooms.some((rm) => rm.id === newRoom.id);
      return { rooms: exists ? s.rooms : [...s.rooms, newRoom] };
    });
    // D-10: переподключаемся, чтобы бэк сделал auto-subscribe на новую комнату.
    wsClient.reconnect();
    return { ok: true, room: r.data };
  },

  appendMember(data) {
    set((s) => {
      const existing = s.membersByRoom[data.room_id];
      // Игнорируем member.joined для комнат, чьи members мы не открывали.
      if (existing === undefined) {
        return s;
      }
      // Дедуп по userId.
      if (existing.some((m) => m.userId === data.user_id)) {
        return s;
      }
      const newMember: Member = {
        userId: data.user_id,
        role: "member",
        joinedAt: data.joined_at,
      };
      return {
        membersByRoom: {
          ...s.membersByRoom,
          [data.room_id]: [...existing, newMember],
        },
      };
    });
  },

  clear() {
    set({ ...initialState });
  },
}));

// Регистрируем clear() как listener для каскадного logoutFlow.
// См. shared/lib/logoutFlow.ts и features/auth/store/index.ts.
registerLogoutListener(() => {
  useRoomsStore.getState().clear();
});
