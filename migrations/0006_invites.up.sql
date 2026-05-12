-- 0006_invites.up.sql
-- Создание таблицы invites (короткие base32-коды для вступления в комнаты)

CREATE TABLE invites (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id     uuid        NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    code        text        NOT NULL,
    created_by  uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz NULL,
    CONSTRAINT invites_code_length_check CHECK (char_length(code) = 8)
);

-- Один активный код на комнату
CREATE UNIQUE INDEX invites_active_room
    ON invites(room_id)
    WHERE revoked_at IS NULL;

-- Глобальная уникальность активного кода
CREATE UNIQUE INDEX invites_active_code
    ON invites(code)
    WHERE revoked_at IS NULL;

-- Для будущей выборки истории кодов
CREATE INDEX invites_room_id_idx ON invites(room_id);
