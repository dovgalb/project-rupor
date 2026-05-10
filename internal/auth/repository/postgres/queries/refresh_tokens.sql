-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
FROM   refresh_tokens
WHERE  token_hash = $1;

-- name: RevokeRefreshTokenByHash :execrows
UPDATE refresh_tokens
SET    revoked_at = $2
WHERE  token_hash = $1
  AND  revoked_at IS NULL;
