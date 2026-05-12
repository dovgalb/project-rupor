-- 0007_channels.up.sql
-- Создание таблицы channels (text/voice внутри комнаты)

CREATE TABLE channels (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id     uuid        NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    name        text        NOT NULL,
    kind        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT channels_name_length_check CHECK (char_length(name) BETWEEN 1 AND 64),
    CONSTRAINT channels_kind_check CHECK (kind IN ('text', 'voice')),
    CONSTRAINT channels_room_id_name_key UNIQUE (room_id, name)
);

CREATE INDEX channels_room_id_idx ON channels(room_id);
