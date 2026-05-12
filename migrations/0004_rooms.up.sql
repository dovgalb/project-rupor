-- 0004_rooms.up.sql
-- Создание таблицы rooms (комнаты — аналог серверов в Discord)

CREATE TABLE rooms (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT rooms_name_length_check CHECK (char_length(name) BETWEEN 1 AND 64)
);

CREATE INDEX rooms_owner_id_idx ON rooms(owner_id);
