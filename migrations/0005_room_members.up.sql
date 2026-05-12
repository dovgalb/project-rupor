-- 0005_room_members.up.sql
-- Создание таблицы room_members с ролями owner/admin/member

CREATE TABLE room_members (
    room_id     uuid        NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        text        NOT NULL,
    joined_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (room_id, user_id),
    CONSTRAINT room_members_role_check CHECK (role IN ('owner', 'admin', 'member'))
);

-- Гарантия "ровно один owner на комнату"
CREATE UNIQUE INDEX room_members_one_owner_per_room
    ON room_members(room_id)
    WHERE role = 'owner';

-- Для ListUserRooms
CREATE INDEX room_members_user_id_idx ON room_members(user_id);
