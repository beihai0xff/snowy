-- name: GetUserByID :one
SELECT id, phone, nickname, role, avatar_url, last_login_at, created_at, updated_at
FROM users
WHERE id = ?;

-- name: GetUserByPhone :one
SELECT id, phone, nickname, role, avatar_url, last_login_at, created_at, updated_at
FROM users
WHERE phone = ?;

-- name: CreateUser :exec
INSERT INTO users (id, phone, nickname, role, avatar_url, last_login_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = ?;

