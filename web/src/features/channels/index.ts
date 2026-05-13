// Публичный API feature-слайса channels.
// Внешний код должен импортировать только отсюда.
export { useChannelsStore } from "./store";
export { ChannelList } from "./components/ChannelList";
export { ChannelListItem } from "./components/ChannelListItem";
export { CreateChannelModal } from "./components/CreateChannelModal";
export { DeleteChannelConfirm } from "./components/DeleteChannelConfirm";
export type {
  Channel,
  ChannelKind,
  ChannelId,
  CreateChannelRequest,
} from "./types";
