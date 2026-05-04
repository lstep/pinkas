-- name: CreateUser :exec
INSERT INTO users (id, email, name, password_hash, global_role)
VALUES (?, ?, ?, ?, ?);

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, global_role, created_at, deleted_at
FROM users
WHERE email = ? AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, name, password_hash, global_role, created_at, deleted_at
FROM users
WHERE id = ? AND deleted_at IS NULL;

-- name: CountUsers :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
VALUES (?, ?, ?, ?);

-- name: GetRefreshToken :one
SELECT id, user_id, token_hash, expires_at, created_at
FROM refresh_tokens
WHERE id = ?;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens WHERE id = ?;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens WHERE expires_at < strftime('%s', 'now');

-- name: GetSetting :one
SELECT key, value FROM settings WHERE key = ?;

-- name: UpsertSetting :exec
INSERT INTO settings (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;

-- name: ListUsers :many
SELECT id, email, name, password_hash, global_role, created_at, deleted_at
FROM users
WHERE deleted_at IS NULL
ORDER BY email;

-- name: UpdateUserRole :exec
UPDATE users SET global_role = ? WHERE id = ? AND deleted_at IS NULL;

-- name: UpdateUserName :exec
UPDATE users SET name = ? WHERE id = ? AND deleted_at IS NULL;

-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = strftime('%s', 'now') WHERE id = ?;
