// ChatRoutePage: страница /rooms/:roomId/channels/:channelId.
// Phase 8 — рендерит <ChatPanel /> для выбранного канала.

import { useParams } from "react-router-dom";

import { ChatPanel } from "@/features/chat";

export function ChatRoutePage(): JSX.Element {
  const { channelId = "" } = useParams<{ channelId: string }>();
  return <ChatPanel channelId={channelId} />;
}
