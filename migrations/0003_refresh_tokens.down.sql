-- 0003_refresh_tokens.down.sql
-- Откат: удалить таблицу refresh_tokens (FK, UNIQUE, CHECK и индексы уходят вместе).

DROP TABLE IF EXISTS refresh_tokens;
