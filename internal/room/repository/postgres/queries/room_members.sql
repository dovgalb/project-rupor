-- name: InsertRoomMember :exec
INSERT INTO room_members (room_id, user_id, role, joined_at)
VALUES ($1, $2, $3, $4);

-- name: GetRoomMember :one
SELECT room_id, user_id, role, joined_at
FROM room_members
WHERE room_id = $1 AND user_id = $2;

-- name: ListRoomMembers :many
SELECT room_id, user_id, role, joined_at
FROM room_members
WHERE room_id = $1
ORDER BY joined_at ASC;

-- name: DeleteRoomMember :execrows
DELETE FROM room_members
WHERE room_id = $1 AND user_id = $2;
