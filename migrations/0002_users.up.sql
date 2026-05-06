-- 0002_users.up.sql
-- Создание таблицы пользователей с ограничениями.

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         citext      NOT NULL UNIQUE,
    password_hash text        NOT NULL,
    username      text        NOT NULL UNIQUE,
    created_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_username_length_check CHECK (char_length(username) BETWEEN 3 AND 32)
);
