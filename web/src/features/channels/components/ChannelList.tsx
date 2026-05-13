// ChannelList: список каналов активной комнаты (text + voice).
// Источник: docs/3_5_frontend/07-ui-contract.md (<ChannelList />).
//
// Состояния: loading / empty / error / ready.
// D-12: voice-каналы отображаются disabled.
// canManage = role owner|admin → кнопка "+" и empty CTA.

import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { ChannelListItem } from "@/features/channels/components/ChannelListItem";
import { CreateChannelModal } from "@/features/channels/components/CreateChannelModal";
import { useChannelsStore } from "@/features/channels/store";
import { useRoomsStore } from "@/features/rooms";
import { ru } from "@/shared/lib/i18n/ru";
import { Button } from "@/shared/ui/Button";
import { Spinner } from "@/shared/ui/Spinner";

import styles from "./ChannelList.module.css";

import type { Channel } from "@/features/channels/types";

type ChannelListProps = {
  roomId: string;
};

// Стабильная ссылка для пустого массива — чтобы селектор не вызывал ререндер.
const EMPTY_CHANNELS: Channel[] = [];

export function ChannelList({ roomId }: ChannelListProps): JSX.Element {
  const navigate = useNavigate();
  const { channelId: activeChannelId } = useParams<{ channelId: string }>();

  const channels = useChannelsStore(
    (s) => s.channelsByRoom[roomId] ?? EMPTY_CHANNELS,
  );
  const loading = useChannelsStore((s) => s.loadingByRoom[roomId] ?? false);
  const error = useChannelsStore((s) => s.errorByRoom[roomId] ?? null);
  const role = useRoomsStore(
    (s) => s.rooms.find((r) => r.id === roomId)?.role,
  );
  const canManage = role === "owner" || role === "admin";

  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    void useChannelsStore.getState().loadChannels(roomId);
  }, [roomId]);

  const textChannels = channels.filter((c) => c.kind === "text");
  const voiceChannels = channels.filter((c) => c.kind === "voice");

  function renderBody(): JSX.Element {
    if (loading && channels.length === 0) {
      return (
        <div className={styles.center}>
          <Spinner size="sm" />
        </div>
      );
    }
    if (error !== null && channels.length === 0) {
      return (
        <div className={styles.error}>
          <span>{ru.channels.loadError}</span>
          <Button
            variant="secondary"
            size="sm"
            onClick={() =>
              void useChannelsStore.getState().loadChannels(roomId)
            }
          >
            {ru.channels.retry}
          </Button>
        </div>
      );
    }
    if (channels.length === 0) {
      return (
        <div className={styles.empty}>
          <span>{ru.channels.emptyDescription}</span>
          {canManage ? (
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              {ru.channels.createBtn}
            </Button>
          ) : null}
        </div>
      );
    }
    return (
      <>
        {textChannels.length > 0 ? (
          <div className={styles.section}>
            <h4 className={styles.sectionTitle}>{ru.channels.textChannels}</h4>
            <ul className={styles.list}>
              {textChannels.map((c) => (
                <ChannelListItem
                  key={c.id}
                  channel={c}
                  active={c.id === activeChannelId}
                  onSelect={() =>
                    navigate(`/rooms/${roomId}/channels/${c.id}`)
                  }
                />
              ))}
            </ul>
          </div>
        ) : null}
        {voiceChannels.length > 0 ? (
          <div className={styles.section}>
            <h4 className={styles.sectionTitle}>
              {ru.channels.voiceChannels}{" "}
              <span className={styles.comingSoon}>
                {ru.channels.comingSoon}
              </span>
            </h4>
            <ul className={styles.list}>
              {voiceChannels.map((c) => (
                <ChannelListItem
                  key={c.id}
                  channel={c}
                  active={false}
                  disabled
                  onSelect={() => {
                    /* voice disabled: no-op */
                  }}
                />
              ))}
            </ul>
          </div>
        ) : null}
      </>
    );
  }

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <h3 className={styles.title}>{ru.channels.channels}</h3>
        {canManage ? (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setCreateOpen(true)}
            aria-label={ru.channels.createChannel}
          >
            +
          </Button>
        ) : null}
      </div>
      {renderBody()}
      <CreateChannelModal
        open={createOpen}
        roomId={roomId}
        onClose={() => setCreateOpen(false)}
        onCreated={(channel) => {
          if (channel.kind === "text") {
            navigate(`/rooms/${roomId}/channels/${channel.id}`);
          }
        }}
      />
    </div>
  );
}
