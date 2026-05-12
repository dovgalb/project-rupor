-- name: InsertInvite :exec
INSERT INTO invites (id, room_id, code, created_by, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: RevokeActiveInvitesByRoom :execrows
UPDATE invites
SET revoked_at = $2
WHERE room_id = $1 AND revoked_at IS NULL;

-- name: GetActiveInviteByCode :one
SELECT id, room_id, code, created_by, created_at, revoked_at
FROM invites
WHERE code = $1 AND revoked_at IS NULL;

-- name: GetActiveInviteByRoom :one
SELECT id, room_id, code, created_by, created_at, revoked_at
FROM invites
WHERE room_id = $1 AND revoked_at IS NULL;
