-- 0002_users.down.sql
-- Откат: удалить таблицу users и расширение citext.

DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS citext;
