-- 0003_refresh_tokens.up.sql
-- Создание таблицы refresh-токенов с FK на users и индексом по user_id.

CREATE TABLE refresh_tokens (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea       NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz NULL,
    CONSTRAINT refresh_tokens_token_hash_length_check CHECK (octet_length(token_hash) = 32)
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens(user_id);
