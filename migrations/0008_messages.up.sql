-- 0008_messages.up.sql
-- Создание таблицы messages (сообщения в text-каналах).

CREATE TABLE messages (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id  uuid        NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id   uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT messages_text_length_check CHECK (char_length(text) BETWEEN 1 AND 16000)
);

CREATE INDEX messages_channel_created_id_idx
    ON messages (channel_id, created_at DESC, id DESC);

CREATE INDEX messages_author_id_idx
    ON messages (author_id);
