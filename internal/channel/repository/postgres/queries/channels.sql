-- name: InsertChannel :exec
INSERT INTO channels (id, room_id, name, kind, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ListChannelsByRoom :many
SELECT id, room_id, name, kind, created_at
FROM channels
WHERE room_id = $1
ORDER BY created_at ASC;

-- name: DeleteChannelInRoom :execrows
DELETE FROM channels
WHERE id = $1 AND room_id = $2;
