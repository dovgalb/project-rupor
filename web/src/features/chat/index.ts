// Публичный API feature-слайса chat.
// Внешний код должен импортировать только отсюда.
export { useChatStore } from "./store";
export { ChatPanel } from "./components/ChatPanel";
export type { Message, MessageStatus, WsStatus } from "./types";
