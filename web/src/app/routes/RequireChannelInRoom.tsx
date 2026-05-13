// RequireChannelInRoom: гард доступа к каналу внутри комнаты.
// Источник: docs/3_5_frontend/08-routes.md (Guards).
// Если канала нет — toast + redirect /rooms/:roomId.
// Если канал voice (D-12) — toast.info + redirect /rooms/:roomId.
// useEffect-toast паттерн (см. RequireMembership): избегаем побочного эффекта
// в фазе render под StrictMode.

import { useEffect, useRef } from "react";
import { Navigate, useParams } from "react-router-dom";

import { useChannelsStore } from "@/features/channels";
import { ru } from "@/shared/lib/i18n/ru";
import { FullScreenSpinner } from "@/shared/ui";
import { toast } from "@/shared/ui/useToast";

import type { Channel } from "@/features/channels";
import type { ReactNode } from "react";

type RequireChannelInRoomProps = {
  children: ReactNode;
};

// Стабильная ссылка для пустого массива — чтобы селектор не вызывал ререндер.
const EMPTY_CHANNELS: Channel[] = [];

export function RequireChannelInRoom({
  children,
}: RequireChannelInRoomProps): JSX.Element {
  const { roomId = "", channelId = "" } = useParams<{
    roomId: string;
    channelId: string;
  }>();

  const channels = useChannelsStore(
    (s) => s.channelsByRoom[roomId] ?? EMPTY_CHANNELS,
  );
  const loading = useChannelsStore((s) => s.loadingByRoom[roomId] ?? false);
  const toastShownRef = useRef(false);

  const channel = channels.find((c) => c.id === channelId);
  const channelMissing = !loading && channel === undefined;
  const channelIsVoice =
    !loading && channel !== undefined && channel.kind === "voice";

  useEffect(() => {
    if (channelMissing && !toastShownRef.current) {
      toastShownRef.current = true;
      toast.error(ru.channels.notFound);
      return;
    }
    if (channelIsVoice && !toastShownRef.current) {
      toastShownRef.current = true;
      toast.info(ru.channels.voiceComingSoon);
    }
  }, [channelMissing, channelIsVoice]);

  // При смене целевого канала/комнаты — разрешаем показать toast снова.
  useEffect(() => {
    toastShownRef.current = false;
  }, [channelId, roomId]);

  if (loading) {
    return <FullScreenSpinner />;
  }
  if (channelMissing || channelIsVoice) {
    return <Navigate to={`/rooms/${roomId}`} replace />;
  }
  return <>{children}</>;
}
