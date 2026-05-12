-- name: InsertRoom :exec
INSERT INTO rooms (id, owner_id, name, created_at)
VALUES ($1, $2, $3, $4);

-- name: GetRoomByID :one
SELECT id, owner_id, name, created_at
FROM rooms
WHERE id = $1;

-- name: ListRoomsByMember :many
SELECT r.id, r.owner_id, r.name, r.created_at, m.role
FROM rooms r
JOIN room_members m ON m.room_id = r.id
WHERE m.user_id = $1
ORDER BY r.created_at DESC;

-- name: DeleteRoomByID :execrows
DELETE FROM rooms
WHERE id = $1;
