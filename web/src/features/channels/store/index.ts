// useChannelsStore: Zustand-store домена channels.
// Источник: docs/3_5_frontend/05-state-model.md (useChannelsStore).
// Решения: D-18 (один store на фичу), D-19 (без persist).
//
// Phase 7 — чистый REST-only слайс. selectChannel НЕ подписывается на WS
// сам — это будет делать useChatStore.openChannel в Phase 8/9.

import { create } from "zustand";

import * as channelsApi from "@/features/channels/api/http";
import { mapChannelErrorToMessage } from "@/features/channels/store/mapErrors";
import { useRoomsStore } from "@/features/rooms";
import { registerLogoutListener } from "@/shared/lib/logoutFlow";
import { toast } from "@/shared/ui/useToast";

import type {
  Channel,
  ChannelId,
  CreateChannelRequest,
} from "@/features/channels/types";

type RoomId = string;

type ChannelsState = {
  channelsByRoom: Record<RoomId, Channel[]>;
  loadingByRoom: Record<RoomId, boolean>;
  errorByRoom: Record<RoomId, string | null>;
  activeChannelId: ChannelId | null;
};

type ChannelsActions = {
  loadChannels: (roomId: RoomId) => Promise<void>;
  createChannel: (
    roomId: RoomId,
    req: CreateChannelRequest,
  ) => Promise<Channel | null>;
  deleteChannel: (roomId: RoomId, channelId: ChannelId) => Promise<boolean>;
  selectChannel: (channelId: ChannelId | null) => void;
  clear: () => void;
};

const initialState: ChannelsState = {
  channelsByRoom: {},
  loadingByRoom: {},
  errorByRoom: {},
  activeChannelId: null,
};

// Каскадное удаление room из useRoomsStore при 403 not-member.
// Это исключение к правилу изоляции features (09-standards.md):
// channel-операция выявила, что мы больше не member комнаты.
// Импорт делается лениво через getState — нет циклических ссылок на уровне модулей.
function dropRoomFromRooms(roomId: RoomId): void {
  const rooms = useRoomsStore.getState().rooms;
  if (rooms.some((r) => r.id === roomId)) {
    useRoomsStore.setState({ rooms: rooms.filter((r) => r.id !== roomId) });
  }
}

export const useChannelsStore = create<ChannelsState & ChannelsActions>(
  (set, get) => ({
    ...initialState,

    async loadChannels(roomId) {
      // Идемпотентность: повторный вызов во время загрузки игнорируется.
      if (get().loadingByRoom[roomId] === true) {
        return;
      }
      set((s) => ({
        loadingByRoom: { ...s.loadingByRoom, [roomId]: true },
        errorByRoom: { ...s.errorByRoom, [roomId]: null },
      }));

      const r = await channelsApi.listChannels(roomId);

      if (r.ok) {
        set((s) => ({
          channelsByRoom: { ...s.channelsByRoom, [roomId]: r.data.items },
          loadingByRoom: { ...s.loadingByRoom, [roomId]: false },
          errorByRoom: { ...s.errorByRoom, [roomId]: null },
        }));
        return;
      }

      // Каскад «нас выкинули из комнаты»: чистим room из useRoomsStore.
      if (r.error.code === "CHANNEL-006" || r.error.code === "ROOM-003") {
        dropRoomFromRooms(roomId);
      }
      set((s) => ({
        loadingByRoom: { ...s.loadingByRoom, [roomId]: false },
        errorByRoom: {
          ...s.errorByRoom,
          [roomId]: mapChannelErrorToMessage(r.error.code),
        },
      }));
    },

    async createChannel(roomId, req) {
      const r = await channelsApi.createChannel(roomId, req);
      if (!r.ok) {
        toast.error(mapChannelErrorToMessage(r.error.code));
        return null;
      }
      set((s) => {
        const existing = s.channelsByRoom[roomId] ?? [];
        return {
          channelsByRoom: {
            ...s.channelsByRoom,
            [roomId]: [...existing, r.data],
          },
        };
      });
      return r.data;
    },

    async deleteChannel(roomId, channelId) {
      const r = await channelsApi.deleteChannel(roomId, channelId);
      if (!r.ok) {
        toast.error(mapChannelErrorToMessage(r.error.code));
        return false;
      }
      set((s) => {
        const existing = s.channelsByRoom[roomId] ?? [];
        return {
          channelsByRoom: {
            ...s.channelsByRoom,
            [roomId]: existing.filter((c) => c.id !== channelId),
          },
        };
      });
      return true;
    },

    selectChannel(channelId) {
      set({ activeChannelId: channelId });
    },

    clear() {
      set({ ...initialState });
    },
  }),
);

// Регистрируем clear() как listener для каскадного logoutFlow.
// См. shared/lib/logoutFlow.ts и features/auth/store/index.ts.
registerLogoutListener(() => {
  useChannelsStore.getState().clear();
});
