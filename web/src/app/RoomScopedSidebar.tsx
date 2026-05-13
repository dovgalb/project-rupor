// RoomScopedSidebar: вложенный sidebar, виден только когда выбрана комната (:roomId).
// Phase 7 — добавлен <ChannelList /> ПЕРЕД <MembersList />.

import { useParams } from "react-router-dom";

import { ChannelList } from "@/features/channels";
import { MembersList } from "@/features/rooms";

export function RoomScopedSidebar(): JSX.Element | null {
  const { roomId } = useParams<{ roomId: string }>();
  if (roomId === undefined) {
    return null;
  }
  return (
    <div data-testid="room-scoped-sidebar">
      <ChannelList roomId={roomId} />
      <MembersList roomId={roomId} />
    </div>
  );
}
