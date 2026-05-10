-- name: InsertUser :exec
INSERT INTO users (id, email, password_hash, username, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetUserByID :one
SELECT id, email, password_hash, username, created_at
FROM   users
WHERE  id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, username, created_at
FROM   users
WHERE  email = $1;
