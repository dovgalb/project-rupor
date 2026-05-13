// RoomList: список моих комнат в Sidebar + CTA для Create / Join.
// Грузит rooms при mount (если idle). Состояния: loading / error / empty / ready.

import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { CreateRoomModal } from "@/features/rooms/components/CreateRoomModal";
import { JoinByCodeModal } from "@/features/rooms/components/JoinByCodeModal";
import { RoomListItem } from "@/features/rooms/components/RoomListItem";
import { useRoomsStore } from "@/features/rooms/store";
import { ru } from "@/shared/lib/i18n/ru";
import { Button } from "@/shared/ui/Button";
import { Spinner } from "@/shared/ui/Spinner";

import styles from "./RoomList.module.css";

export function RoomList(): JSX.Element {
  const navigate = useNavigate();
  const { roomId: activeRoomId } = useParams<{ roomId: string }>();

  const rooms = useRoomsStore((s) => s.rooms);
  const status = useRoomsStore((s) => s.status);

  const [createOpen, setCreateOpen] = useState(false);
  const [joinOpen, setJoinOpen] = useState(false);

  useEffect(() => {
    if (status === "idle") {
      void useRoomsStore.getState().loadRooms();
    }
  }, [status]);

  function handleCreated(roomId: string): void {
    navigate(`/rooms/${roomId}`);
  }

  function handleJoined(roomId: string): void {
    navigate(`/rooms/${roomId}`);
  }

  function renderBody(): JSX.Element {
    if (status === "idle" || status === "loading") {
      return (
        <div className={styles.center}>
          <Spinner size="sm" />
        </div>
      );
    }
    if (status === "error") {
      return (
        <div className={styles.error}>
          <span>{ru.rooms.loadError}</span>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => void useRoomsStore.getState().loadRooms()}
          >
            {ru.rooms.retry}
          </Button>
        </div>
      );
    }
    if (rooms.length === 0) {
      return (
        <div className={styles.empty}>
          <span>{ru.rooms.emptyDescription}</span>
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              {ru.rooms.createRoom}
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => setJoinOpen(true)}
            >
              {ru.rooms.joinByCode}
            </Button>
          </div>
        </div>
      );
    }
    return (
      <ul className={styles.list}>
        {rooms.map((r) => (
          <RoomListItem key={r.id} room={r} active={r.id === activeRoomId} />
        ))}
      </ul>
    );
  }

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <h2 className={styles.title}>{ru.rooms.myRooms}</h2>
        <div className={styles.actions}>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setCreateOpen(true)}
            aria-label={ru.rooms.createRoom}
          >
            +
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setJoinOpen(true)}
            aria-label={ru.rooms.joinByCode}
          >
            #
          </Button>
        </div>
      </div>
      {renderBody()}
      <CreateRoomModal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        onCreated={(room) => handleCreated(room.id)}
      />
      <JoinByCodeModal
        open={joinOpen}
        onClose={() => setJoinOpen(false)}
        onJoined={(room) => handleJoined(room.id)}
      />
    </div>
  );
}
