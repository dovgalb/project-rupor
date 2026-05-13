-- name: InsertMessage :exec
INSERT INTO messages (id, channel_id, author_id, text, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetMessageByID :one
SELECT id, channel_id, author_id, text, created_at
FROM messages
WHERE id = $1;

-- name: ListMessagesBeforeCursor :many
SELECT m.id, m.channel_id, m.author_id, m.text, m.created_at
FROM messages AS m
WHERE m.channel_id = $1
  AND (
    sqlc.narg('before_id')::uuid IS NULL
    OR (m.created_at, m.id) < (
        SELECT b.created_at, b.id FROM messages AS b WHERE b.id = sqlc.narg('before_id')
    )
  )
ORDER BY m.created_at DESC, m.id DESC
LIMIT sqlc.arg('limit')::int;

-- name: GetChannelKind :one
SELECT id, room_id, kind
FROM channels
WHERE id = $1;
