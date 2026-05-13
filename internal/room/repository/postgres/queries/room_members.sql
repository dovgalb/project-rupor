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

-- name: GetMemberForChannel :one
SELECT rm.role
FROM channels AS c
JOIN room_members AS rm ON rm.room_id = c.room_id
WHERE c.id = $1 AND rm.user_id = $2;

-- name: ChannelExists :one
SELECT EXISTS(SELECT 1 FROM channels WHERE id = $1) AS exists;
