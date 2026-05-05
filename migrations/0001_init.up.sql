-- 0001_init.up.sql
-- Initial migration: enable pgcrypto for gen_random_uuid().

CREATE EXTENSION IF NOT EXISTS pgcrypto;
